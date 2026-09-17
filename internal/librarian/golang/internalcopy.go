// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package golang

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/gofeaturespb"
)

const (
	// protocPluginPrefix is the name prefix protoc expects for plugin binaries.
	protocPluginPrefix = "protoc-gen-"
	// internalPathElement is the path element that makes a Go package
	// unimportable from outside its parent tree.
	internalPathElement = "internal"
)

var (
	errInternalCopyNil            = errors.New("internal_copies entry must not be null")
	errInternalCopyImportPath     = errors.New("internal copy import_path must be a clean relative Go import path with an \"internal\" path element")
	errInternalCopyOutsideLibrary = errors.New("internal copy import_path resolves outside the library")
	errInternalCopyOverlap        = errors.New("internal copy directory overlaps another generated directory")
	errInternalCopyProtoPackage   = errors.New("internal copy proto_package must be a valid proto package that is not used by the API or another copy")
	errInternalCopyPackages       = errors.New("internal copy requires the API proto files to share one proto package")
	errInternalCopyNoPackage      = errors.New("internal copy requires the API proto files to declare a proto package")
	errInternalCopyFileNotFound   = errors.New("internal copy proto file not found in descriptor set")
	errInternalCopyExtension      = errors.New("internal copy cannot declare an extension of a message outside the copy")
	errInternalCopyAPILevel       = errors.New("internal copy requires the open protobuf API level")
	errProtocPluginNil            = errors.New("internal copy plugins entry must not be null")
	errProtocPluginName           = errors.New("protoc plugin name must consist of letters, digits, \"-\" and \"_\"")
	errProtocPluginOption         = errors.New("protoc plugin option changes the output layout")
	errProtocPluginNotFound       = errors.New("protoc plugin not found")

	protocPluginNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	// protocPluginLayoutOptions are the protoc-gen-go style options that move
	// plugin output away from the Go import path the copy is collected from.
	// M<file> import mappings override go_package and are also rejected.
	protocPluginLayoutOptions = []string{"paths", "module"}
)

// validateInternalCopies validates the lexical library layout without reading
// the filesystem. Clean and Generate additionally validate resolved destinations
// with validateInternalCopyPaths before mutating the repository.
func validateInternalCopies(library *config.Library, outDir string) error {
	type generatedDir struct {
		path   string
		what   string
		isCopy bool
	}
	var dirs []generatedDir
	protoPackages := make(map[string]string)
	for _, api := range library.APIs {
		goAPI := api.Go
		if goAPI == nil {
			continue
		}
		if importPath := clientImportPath(api.Path, goAPI); importPath != "" {
			dirs = append(dirs, generatedDir{
				path: filepath.Join(repoRootPath(outDir, library.Name), pathFromRepoRoot(library, importPath)),
				what: fmt.Sprintf("client directory of api %q", api.Path),
			})
		}
		apiPackage := goAPI.ProtoPackage
		if apiPackage == "" {
			apiPackage = strings.ReplaceAll(api.Path, "/", ".")
		}
		for _, cp := range goAPI.InternalCopies {
			if err := validateInternalCopyConfig(cp); err != nil {
				return fmt.Errorf("api %q: %w", api.Path, err)
			}
			if cp.ProtoPackage == apiPackage {
				return fmt.Errorf("api %q: %w: %q is the proto package of the API", api.Path, errInternalCopyProtoPackage, cp.ProtoPackage)
			}
			if other, ok := protoPackages[cp.ProtoPackage]; ok {
				return fmt.Errorf("api %q: %w: %q is also the proto package of internal copy %q", api.Path, errInternalCopyProtoPackage, cp.ProtoPackage, other)
			}
			protoPackages[cp.ProtoPackage] = cp.ImportPath
			dir, err := internalCopyDir(library, outDir, cp)
			if err != nil {
				return fmt.Errorf("api %q: %w", api.Path, err)
			}
			dirs = append(dirs, generatedDir{path: dir, what: fmt.Sprintf("internal copy %q", cp.ImportPath), isCopy: true})
		}
	}
	for i, a := range dirs {
		for _, b := range dirs[:i] {
			if !a.isCopy && !b.isCopy {
				continue
			}
			if overlapsPath(a.path, b.path) {
				return fmt.Errorf("%w: %s and %s", errInternalCopyOverlap, b.what, a.what)
			}
		}
	}
	return nil
}

// clientImportPath returns the import path of the API's generated client,
// deriving the default from the API path when the configuration has not been
// filled yet.
func clientImportPath(apiPath string, goAPI *config.GoAPI) string {
	if goAPI.ImportPath != "" {
		return goAPI.ImportPath
	}
	importPath, _ := defaultImportPathAndClientPkg(apiPath)
	return importPath
}

// validateInternalCopyConfig checks a single copy entry without looking at
// the filesystem or the proto sources.
func validateInternalCopyConfig(cp *config.GoInternalCopy) error {
	if cp == nil {
		return errInternalCopyNil
	}
	if err := validateInternalCopyImportPath(cp.ImportPath); err != nil {
		return err
	}
	if !protoreflect.FullName(cp.ProtoPackage).IsValid() {
		return fmt.Errorf("%w: %q", errInternalCopyProtoPackage, cp.ProtoPackage)
	}
	for _, plugin := range cp.Plugins {
		if plugin == nil {
			return errProtocPluginNil
		}
		if !protocPluginNameRe.MatchString(plugin.Name) {
			return fmt.Errorf("%w: %q", errProtocPluginName, plugin.Name)
		}
		for _, opt := range plugin.Options {
			for param := range strings.SplitSeq(opt, ",") {
				key, _, _ := strings.Cut(param, "=")
				if slices.Contains(protocPluginLayoutOptions, key) || strings.HasPrefix(key, "M") {
					return fmt.Errorf("%w: plugin %q option %q; the copy is collected from its Go import path under the protoc output directory", errProtocPluginOption, plugin.Name, opt)
				}
			}
		}
	}
	return nil
}

// validateInternalCopyImportPath checks that the import path is relative, in
// canonical form (path.Clean would leave it unchanged and it has no "." or
// ".." elements) and contains an "internal" element. Canonical form matters
// because the path is joined onto the output directory: "foo/internal/../bar"
// would otherwise generate into a public package.
func validateInternalCopyImportPath(importPath string) error {
	elements := strings.Split(importPath, "/")
	switch {
	case importPath == "":
		return fmt.Errorf("%w: import_path is empty", errInternalCopyImportPath)
	case path.IsAbs(importPath), strings.Contains(importPath, `\`):
		return fmt.Errorf("%w: %q is not relative", errInternalCopyImportPath, importPath)
	case path.Clean(importPath) != importPath, slices.Contains(elements, "."), slices.Contains(elements, ".."):
		return fmt.Errorf("%w: %q is not in canonical form", errInternalCopyImportPath, importPath)
	case !slices.Contains(elements, internalPathElement):
		return fmt.Errorf("%w: %q has no %q element", errInternalCopyImportPath, importPath, internalPathElement)
	}
	return nil
}

// internalCopyDir returns the directory the copy is generated into. It is
// derived from the import path like the client directory and must lie strictly
// inside outDir, the library output directory, so that neither cleanup nor
// generation can reach into another library.
func internalCopyDir(library *config.Library, outDir string, cp *config.GoInternalCopy) (string, error) {
	dir := filepath.Join(repoRootPath(outDir, library.Name), pathFromRepoRoot(library, cp.ImportPath))
	rel, err := filepath.Rel(outDir, dir)
	if err != nil || rel == "." || !containsPath(outDir, dir) {
		return "", fmt.Errorf("%w: %q resolves to %s, outside %s", errInternalCopyOutsideLibrary, cp.ImportPath, dir, outDir)
	}
	return dir, nil
}

// overlapsPath reports whether two generated directories are the same or one
// lies below the other. The comparison ignores letter case so that two paths
// which differ only by case, and would merge on a case-insensitive
// filesystem, are still reported as overlapping.
func overlapsPath(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	return containsPath(a, b) || containsPath(b, a)
}

// containsPath reports whether child is dir itself or lies below it.
func containsPath(dir, child string) bool {
	rel, err := filepath.Rel(dir, child)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// generateInternalCopies generates every internal copy configured for the API.
//
// A copy is a second generation of the API's messages into an internal Go
// package. It is produced from a descriptor set rather than from the proto
// sources so that the proto package and file paths can be renamed first:
// protobuf-go registers every file by path and every message by full name in
// a global registry and panics on conflicts, so a verbatim copy could never be
// linked next to the public package.
func generateInternalCopies(ctx context.Context, apiPath string, goAPI *config.GoAPI, library *config.Library, pc *config.Protoc, googleapisDir, tempDir, outDir string) error {
	if len(goAPI.InternalCopies) == 0 {
		return nil
	}
	// Generate has already checked the paths on disk; this only guards the
	// configuration for callers that skip Generate.
	if err := validateInternalCopies(library, outDir); err != nil {
		return err
	}
	protoFiles, err := collectProtoFiles(googleapisDir, apiPath, goAPI.NestedProtos)
	if err != nil {
		return err
	}
	// Only the files directly in the API directory are copied. Nested protos
	// are supplied to protoc as dependencies only, whatever proto package they
	// declare, so their messages keep their public Go types and get no plugin
	// output.
	apiDir := filepath.Join(googleapisDir, apiPath)
	var apiFiles []string
	for _, f := range protoFiles {
		if filepath.Dir(f) != apiDir {
			continue
		}
		rel, err := filepath.Rel(googleapisDir, f)
		if err != nil {
			return err
		}
		apiFiles = append(apiFiles, filepath.ToSlash(rel))
	}
	for _, cp := range goAPI.InternalCopies {
		if err := generateInternalCopy(ctx, cp, goAPI, library, pc, googleapisDir, protoFiles, apiFiles, tempDir, outDir); err != nil {
			return fmt.Errorf("internal copy %q: %w", cp.ImportPath, err)
		}
	}
	return nil
}

func generateInternalCopy(ctx context.Context, cp *config.GoInternalCopy, goAPI *config.GoAPI, library *config.Library, pc *config.Protoc, googleapisDir string, protoFiles, apiFiles []string, tempDir, outDir string) error {
	plugins, err := resolveProtocPlugins(cp.Plugins)
	if err != nil {
		return err
	}
	copyDir, err := os.MkdirTemp(tempDir, "internal-copy-")
	if err != nil {
		return err
	}
	// Source info is kept so that the proto comments end up in the generated
	// Go code, as they do for the public package.
	apiSet := filepath.Join(copyDir, "api.pb")
	args := []string{
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		"--include_imports",
		"--include_source_info",
		"--descriptor_set_out=" + apiSet,
	}
	args = append(args, protoFiles...)
	if err := runProtoc(ctx, pc, args...); err != nil {
		return err
	}
	fds, err := readDescriptorSet(apiSet)
	if err != nil {
		return err
	}
	if err := checkInternalCopyAPILevel(fds, apiFiles, goAPI.ProtoAPILevel); err != nil {
		return err
	}
	renamed, err := rewriteDescriptorSet(fds, apiFiles, cp)
	if err != nil {
		return err
	}
	copySet := filepath.Join(copyDir, "copy.pb")
	if err := writeDescriptorSet(copySet, fds); err != nil {
		return err
	}
	args = []string{
		"--experimental_allow_proto3_optional",
		"--descriptor_set_in=" + copySet,
		"--go_out=" + copyDir,
	}
	if goAPI.ProtoAPILevel != "" {
		args = append(args, "--go_opt=default_api_level="+goAPI.ProtoAPILevel)
	}
	for _, plugin := range cp.Plugins {
		args = append(args, "--plugin="+protocPluginPrefix+plugin.Name+"="+plugins[plugin.Name])
		args = append(args, "--"+plugin.Name+"_out="+copyDir)
		for _, opt := range plugin.Options {
			args = append(args, "--"+plugin.Name+"_opt="+opt)
		}
	}
	args = append(args, renamed...)
	if err := runProtoc(ctx, pc, args...); err != nil {
		return err
	}
	return moveInternalCopy(library, cp, copyDir, outDir)
}

// resolveProtocPlugins resolves the binary of every plugin. It returns the
// resolved binary path per plugin name.
func resolveProtocPlugins(plugins []*config.GoProtocPlugin) (map[string]string, error) {
	binaries := make(map[string]string, len(plugins))
	for _, plugin := range plugins {
		if plugin == nil {
			return nil, errProtocPluginNil
		}
		binary, err := protocPluginPath(plugin.Name)
		if err != nil {
			return nil, err
		}
		binaries[plugin.Name] = binary
	}
	return binaries, nil
}

// protocPluginPath resolves the binary of the named protoc plugin. Plugins are
// looked up in the Go tool bin directory first, then on the PATH, matching how
// protoc resolves the plugins used for the public API.
func protocPluginPath(name string) (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	binary := protocPluginPrefix + name
	if p, err := exec.LookPath(filepath.Join(installDir, binary)); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath(binary); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%w: %s is not installed in %s; declare the Go module providing it under tools.go in librarian.yaml and run \"librarian install go\"", errProtocPluginNotFound, binary, installDir)
}

// apiFileDescriptors returns the descriptors of the API files in the order of
// apiFiles and checks that they declare a single proto package.
func apiFileDescriptors(fds *descriptorpb.FileDescriptorSet, apiFiles []string) ([]*descriptorpb.FileDescriptorProto, error) {
	byName := make(map[string]*descriptorpb.FileDescriptorProto, len(fds.GetFile()))
	for _, fd := range fds.GetFile() {
		byName[fd.GetName()] = fd
	}
	var apiFds []*descriptorpb.FileDescriptorProto
	for _, name := range apiFiles {
		fd, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", errInternalCopyFileNotFound, name)
		}
		if len(apiFds) > 0 && fd.GetPackage() != apiFds[0].GetPackage() {
			return nil, fmt.Errorf("%w: %q and %q", errInternalCopyPackages, apiFds[0].GetPackage(), fd.GetPackage())
		}
		apiFds = append(apiFds, fd)
	}
	if len(apiFds) == 0 {
		return nil, errInternalCopyFileNotFound
	}
	return apiFds, nil
}

// checkInternalCopyAPILevel verifies that protoc-gen-go would generate the
// open struct API for every message of the API files. Internal copies are
// only supported with the open API because plugins such as
// protoc-gen-go-vtproto access message fields directly, which does not
// compile against opaque structs or hybrid structs built with protoopaque.
// The effective level is resolved the way protoc-gen-go resolves it: a file
// or message api_level feature wins, then the edition default (opaque from
// edition 2024 on), then
// protoAPILevel, the configured default_api_level, and finally open.
func checkInternalCopyAPILevel(fds *descriptorpb.FileDescriptorSet, apiFiles []string, protoAPILevel string) error {
	defaultLevel := gofeaturespb.GoFeatures_API_OPEN
	defaultSource := "the protoc-gen-go default"
	if protoAPILevel != "" {
		level, ok := gofeaturespb.GoFeatures_APILevel_value[protoAPILevel]
		if !ok || level == int32(gofeaturespb.GoFeatures_API_LEVEL_UNSPECIFIED) {
			return fmt.Errorf("%w: unknown proto_api_level %q", errInternalCopyAPILevel, protoAPILevel)
		}
		defaultLevel = gofeaturespb.GoFeatures_APILevel(level)
		defaultSource = "proto_api_level"
	}
	apiFds, err := apiFileDescriptors(fds, apiFiles)
	if err != nil {
		return err
	}
	for _, fd := range apiFds {
		level, source := defaultLevel, defaultSource
		if explicit := featureAPILevel(fd.GetOptions().GetFeatures()); explicit != gofeaturespb.GoFeatures_API_LEVEL_UNSPECIFIED {
			level, source = explicit, "the file's api_level feature"
		} else if fd.GetEdition() >= descriptorpb.Edition_EDITION_2024 {
			level, source = gofeaturespb.GoFeatures_API_OPAQUE, "the default of "+fd.GetEdition().String()
		}
		if level != gofeaturespb.GoFeatures_API_OPEN {
			return apiLevelError(fd.GetName(), "", level, source)
		}
		if err := checkMessagesAPILevel(fd.GetName(), "."+fd.GetPackage(), fd.GetMessageType()); err != nil {
			return err
		}
	}
	return nil
}

// checkMessagesAPILevel rejects messages, including nested ones, whose
// api_level feature selects another API than the open one. A message without
// the feature inherits its parent's level, which the caller has checked.
func checkMessagesAPILevel(file, scope string, messages []*descriptorpb.DescriptorProto) error {
	for _, message := range messages {
		name := scope + "." + message.GetName()
		level := featureAPILevel(message.GetOptions().GetFeatures())
		if level != gofeaturespb.GoFeatures_API_LEVEL_UNSPECIFIED && level != gofeaturespb.GoFeatures_API_OPEN {
			return apiLevelError(file, strings.TrimPrefix(name, "."), level, "the message's api_level feature")
		}
		if err := checkMessagesAPILevel(file, name, message.GetNestedType()); err != nil {
			return err
		}
	}
	return nil
}

func apiLevelError(file, message string, level gofeaturespb.GoFeatures_APILevel, source string) error {
	what := "file " + file
	if message != "" {
		what = "message " + message + " in " + file
	}
	return fmt.Errorf("%w: %s would be generated with %s because of %s; plugins such as protoc-gen-go-vtproto access message fields directly, which requires API_OPEN to compile without and with the protoopaque build tag", errInternalCopyAPILevel, what, level, source)
}

// featureAPILevel returns the Go api_level feature set on the given feature
// set, or unspecified when there is none.
func featureAPILevel(features *descriptorpb.FeatureSet) gofeaturespb.GoFeatures_APILevel {
	if features == nil || !proto.HasExtension(features, gofeaturespb.E_Go) {
		return gofeaturespb.GoFeatures_API_LEVEL_UNSPECIFIED
	}
	goFeatures, ok := proto.GetExtension(features, gofeaturespb.E_Go).(*gofeaturespb.GoFeatures)
	if !ok {
		return gofeaturespb.GoFeatures_API_LEVEL_UNSPECIFIED
	}
	return goFeatures.GetApiLevel()
}

// rewriteDescriptorSet rewrites the given API files in place for the internal
// copy: it renames the proto package, moves each file under the directory
// matching the new package, points go_package at the copy's import path and
// drops service definitions. Every reference to a renamed type across the set
// is updated. It returns the new file names in the order of apiFiles.
//
// Only go_package is rewritten; every other option, annotation and comment
// keeps its original value, so resource names, HTTP bindings and similar
// metadata in the copy still describe the public API.
func rewriteDescriptorSet(fds *descriptorpb.FileDescriptorSet, apiFiles []string, cp *config.GoInternalCopy) ([]string, error) {
	apiFds, err := apiFileDescriptors(fds, apiFiles)
	if err != nil {
		return nil, err
	}
	if !protoreflect.FullName(cp.ProtoPackage).IsValid() {
		return nil, fmt.Errorf("%w: %q", errInternalCopyProtoPackage, cp.ProtoPackage)
	}
	// The copy's files are registered by path and its types by full name, both
	// derived from the proto package, so the package must be new to the set.
	for _, fd := range fds.GetFile() {
		if fd.GetPackage() == cp.ProtoPackage {
			return nil, fmt.Errorf("%w: %q is the proto package of %s", errInternalCopyProtoPackage, cp.ProtoPackage, fd.GetName())
		}
	}
	oldPackage := apiFds[0].GetPackage()
	// Full names are built from the package, so a file without one would
	// yield names such as "..Foo" that match nothing in the descriptor set.
	if oldPackage == "" {
		return nil, fmt.Errorf("%w: %s", errInternalCopyNoPackage, apiFds[0].GetName())
	}

	// Type references are fully qualified names such as ".google.spanner.v1.Type".
	// Collect the exact names defined by the API files rather than rewriting by
	// prefix so that a sibling package sharing the prefix is left untouched.
	renamedTypes := make(map[string]string)
	for _, fd := range apiFds {
		collectRenamedTypes(renamedTypes, "."+oldPackage, "."+cp.ProtoPackage, fd.GetMessageType(), fd.GetEnumType())
	}
	// An extension is registered on its extendee by field number, and renaming
	// the extension does not change that key: a copy extending a message that
	// is not itself copied would register a second extension with the same
	// number and panic at init as soon as both packages are linked.
	for _, fd := range apiFds {
		if err := checkExtendees(renamedTypes, fd.GetName(), "."+oldPackage, fd.GetExtension(), fd.GetMessageType()); err != nil {
			return nil, err
		}
	}
	renamedFiles := make(map[string]string, len(apiFds))
	newDir := strings.ReplaceAll(cp.ProtoPackage, ".", "/")
	goPackage := modulePathPrefix + cp.ImportPath + ";" + path.Base(cp.ImportPath)
	renamed := make([]string, 0, len(apiFds))
	for _, fd := range apiFds {
		newName := path.Join(newDir, path.Base(fd.GetName()))
		renamedFiles[fd.GetName()] = newName
		renamed = append(renamed, newName)
		fd.Name = new(newName)
		fd.Package = new(cp.ProtoPackage)
		if fd.Options == nil {
			fd.Options = &descriptorpb.FileOptions{}
		}
		fd.Options.GoPackage = new(goPackage)
		fd.Service = nil
	}
	for _, fd := range fds.GetFile() {
		for i, dep := range fd.GetDependency() {
			if newName, ok := renamedFiles[dep]; ok {
				fd.Dependency[i] = newName
			}
		}
		rewriteMessageTypes(renamedTypes, fd.GetMessageType())
		rewriteFieldTypes(renamedTypes, fd.GetExtension())
		for _, service := range fd.GetService() {
			for _, method := range service.GetMethod() {
				method.InputType = renameType(renamedTypes, method.InputType)
				method.OutputType = renameType(renamedTypes, method.OutputType)
			}
		}
	}
	return renamed, nil
}

// checkExtendees rejects extensions, including ones declared inside messages,
// whose extendee is not among the renamed types of the copy.
func checkExtendees(renamed map[string]string, file, scope string, extensions []*descriptorpb.FieldDescriptorProto, messages []*descriptorpb.DescriptorProto) error {
	for _, ext := range extensions {
		if _, ok := renamed[ext.GetExtendee()]; !ok {
			return fmt.Errorf("%w: %s declares extension %s of %s, which would register the same extension number on the public message; internal_copies is not supported for this API", errInternalCopyExtension, file, strings.TrimPrefix(scope+"."+ext.GetName(), "."), strings.TrimPrefix(ext.GetExtendee(), "."))
		}
	}
	for _, message := range messages {
		if err := checkExtendees(renamed, file, scope+"."+message.GetName(), message.GetExtension(), message.GetNestedType()); err != nil {
			return err
		}
	}
	return nil
}

// collectRenamedTypes records the old and new fully qualified name of every
// message and enum, including nested ones, under the given package prefixes.
func collectRenamedTypes(renamed map[string]string, oldPrefix, newPrefix string, messages []*descriptorpb.DescriptorProto, enums []*descriptorpb.EnumDescriptorProto) {
	for _, enum := range enums {
		renamed[oldPrefix+"."+enum.GetName()] = newPrefix + "." + enum.GetName()
	}
	for _, message := range messages {
		oldName := oldPrefix + "." + message.GetName()
		newName := newPrefix + "." + message.GetName()
		renamed[oldName] = newName
		collectRenamedTypes(renamed, oldName, newName, message.GetNestedType(), message.GetEnumType())
	}
}

func rewriteMessageTypes(renamed map[string]string, messages []*descriptorpb.DescriptorProto) {
	for _, message := range messages {
		rewriteFieldTypes(renamed, message.GetField())
		rewriteFieldTypes(renamed, message.GetExtension())
		rewriteMessageTypes(renamed, message.GetNestedType())
	}
}

func rewriteFieldTypes(renamed map[string]string, fields []*descriptorpb.FieldDescriptorProto) {
	for _, field := range fields {
		field.TypeName = renameType(renamed, field.TypeName)
		field.Extendee = renameType(renamed, field.Extendee)
	}
}

func renameType(renamed map[string]string, name *string) *string {
	if name == nil {
		return nil
	}
	if newName, ok := renamed[*name]; ok {
		return new(newName)
	}
	return name
}

func readDescriptorSet(path string) (*descriptorpb.FileDescriptorSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fds); err != nil {
		return nil, fmt.Errorf("failed to parse descriptor set %s: %w", path, err)
	}
	return fds, nil
}

func writeDescriptorSet(path string, fds *descriptorpb.FileDescriptorSet) error {
	data, err := proto.Marshal(fds)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// moveInternalCopy moves the generated copy from the protoc output directory to
// its destination in the repository, mirroring moveAPIDirectory. Files listed
// in the library's keep list survive the move as they survive Clean: the
// generated version is discarded and the existing file is left in place.
func moveInternalCopy(library *config.Library, cp *config.GoInternalCopy, srcDir, outDir string) error {
	src := filepath.Join(srcDir, "cloud.google.com", "go", cp.ImportPath)
	dest, err := internalCopyDir(library, outDir, cp)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	keep := make(map[string]bool, len(library.Keep))
	for _, k := range library.Keep {
		keep[path.Clean(filepath.ToSlash(k))] = true
	}
	return filesystem.MoveAndMergeWithKeep(src, dest, outDir, func(rel string) bool {
		return keep[rel]
	})
}
