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
)

// This file is the whole internal copy feature, a stopgap until protobuf-go
// provides first-party support for what the extra plugin adds. Fill calls
// validateInternalCopies, Clean calls cleanInternalCopies and Generate calls
// generateInternalCopies; nothing else in the generator knows about copies.
// To remove the feature, delete this file and its tests, those three calls,
// and the GoInternalCopy configuration in internal/config.

const (
	// protocPluginPrefix is the name prefix protoc expects for plugin binaries.
	protocPluginPrefix = "protoc-gen-"
	// protocImportMappingPrefix starts a protoc-gen-go import mapping option,
	// M<proto file>=<Go import path>, which changes the Go package the types
	// of that file are generated into.
	protocImportMappingPrefix = "M"
	// internalPathElement is the path element that makes a Go package
	// unimportable from outside its parent tree.
	internalPathElement = "internal"
)

var (
	errInternalCopyNil            = errors.New("internal_copies entry must not be null")
	errInternalCopyImportPath     = errors.New("internal copy import_path must be a clean relative Go import path with an \"internal\" path element")
	errInternalCopyOutsideLibrary = errors.New("internal copy import_path resolves outside the library")
	errInternalCopyKeep           = errors.New("internal copy directory is removed on clean and cannot hold kept files")
	errInternalCopyProtoPackage   = errors.New("internal copy proto_package must be a valid proto package that is not used by the API or another copy")
	errInternalCopyPackages       = errors.New("internal copy requires the API proto files to share one proto package")
	errInternalCopyNoPackage      = errors.New("internal copy requires the API proto files to declare a proto package")
	errInternalCopyFileNotFound   = errors.New("internal copy proto file not found in descriptor set")
	errInternalCopyExtension      = errors.New("internal copy cannot declare an extension of a message outside the copy")
	errInternalCopyOverlap        = errors.New("internal copy directory overlaps another generated directory")
	errProtocPluginName           = errors.New("protoc plugin name must consist of letters, digits, \"-\" and \"_\"")
	errProtocPluginOption         = errors.New("protoc plugin option changes the output layout")
	errProtocPluginNotFound       = errors.New("protoc plugin not found")
	errAPIDirNoProtoFiles         = errors.New("no .proto files found in the API directory")

	protocPluginNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	// protocPluginLayoutOptions are the options of protoc-gen-go style
	// plugins that change where the generated files are written. They are
	// not allowed for a copy, because its output is collected from the
	// temporary output directory at the copy's Go import path. Import
	// mapping options are rejected for the same reason.
	protocPluginLayoutOptions = []string{"paths", "module"}
)

// generatedDir is a directory the generator writes into: the client directory
// of an API or an internal copy. Copies must not overlap any other generated
// directory, and the description names the directory in that error.
type generatedDir struct {
	path        string
	description string
	isCopy      bool
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

// validateInternalCopies checks the internal copies of every API in the
// library at fill time, before Clean or Generate touch the repository. Each
// copy must be well formed, use a proto package that neither the API nor
// another copy uses, and name an internal package whose directory lies
// strictly inside the library output directory and overlaps no other generated
// directory, so that cleanup and generation cannot reach a public package or
// another library. No kept file may lie under a copy directory, which Clean
// removes entirely.
func validateInternalCopies(library *config.Library) error {
	outDir := library.Output
	var dirs []generatedDir
	protoPackages := make(map[string]string)
	for _, api := range library.APIs {
		goAPI := api.Go
		dirs = append(dirs, generatedDir{
			path:        filepath.Join(repoRootPath(outDir, library.Name), pathFromRepoRoot(library, goAPI.ImportPath)),
			description: fmt.Sprintf("client directory of api %q", api.Path),
		})
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
			dir := internalCopyDir(library, outDir, cp)
			if rel, err := filepath.Rel(outDir, dir); err != nil || rel == "." || !containsPath(outDir, dir) {
				return fmt.Errorf("api %q: %w: %q resolves to %s, outside %s", api.Path, errInternalCopyOutsideLibrary, cp.ImportPath, dir, outDir)
			}
			for _, kept := range library.Keep {
				if containsPath(dir, filepath.Join(outDir, kept)) {
					return fmt.Errorf("api %q: %w: %q is under internal copy %q", api.Path, errInternalCopyKeep, kept, cp.ImportPath)
				}
			}
			dirs = append(dirs, generatedDir{path: dir, description: fmt.Sprintf("internal copy %q", cp.ImportPath), isCopy: true})
		}
	}
	for i, dir := range dirs {
		for _, earlier := range dirs[:i] {
			if (dir.isCopy || earlier.isCopy) && overlapsPath(dir.path, earlier.path) {
				return fmt.Errorf("%w: %s and %s", errInternalCopyOverlap, earlier.description, dir.description)
			}
		}
	}
	return nil
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
	if !protocPluginNameRe.MatchString(cp.Plugin) {
		return fmt.Errorf("%w: %q", errProtocPluginName, cp.Plugin)
	}
	for _, opt := range cp.PluginOptions {
		for param := range strings.SplitSeq(opt, ",") {
			key, _, _ := strings.Cut(param, "=")
			if slices.Contains(protocPluginLayoutOptions, key) || strings.HasPrefix(key, protocImportMappingPrefix) {
				return fmt.Errorf("%w: plugin %q option %q; the copy is collected from its Go import path under the protoc output directory", errProtocPluginOption, cp.Plugin, opt)
			}
		}
	}
	return nil
}

// internalCopyDir returns the directory the copy is generated into, derived
// from the copy's import path like the client directory.
func internalCopyDir(library *config.Library, outDir string, cp *config.GoInternalCopy) string {
	return filepath.Join(repoRootPath(outDir, library.Name), pathFromRepoRoot(library, cp.ImportPath))
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

// cleanInternalCopies removes each internal copy directory of the API. A copy
// holds generated code only, so removing the directory leaves no stale files
// behind when the API's proto files change.
func cleanInternalCopies(library *config.Library, libraryDir string, goAPI *config.GoAPI) error {
	for _, cp := range goAPI.InternalCopies {
		if err := os.RemoveAll(internalCopyDir(library, libraryDir, cp)); err != nil {
			return err
		}
	}
	return nil
}

// generateInternalCopies generates every internal copy configured for the API:
// a second generation of the API's messages into an internal Go package.
func generateInternalCopies(ctx context.Context, apiPath string, goAPI *config.GoAPI, library *config.Library, pc *config.Protoc, googleapisDir, tempDir, outDir string) error {
	if len(goAPI.InternalCopies) == 0 {
		return nil
	}
	// A copy contains only the proto files directly in the API directory.
	// Nested protos live in other directories and are imported by the API;
	// the copy keeps importing their messages from the public packages
	// rather than duplicating them.
	apiFiles, err := collectAPIDirProtoFiles(googleapisDir, apiPath)
	if err != nil {
		return err
	}
	// Resolve every plugin before protoc runs, so that a missing plugin is
	// reported without generating anything.
	plugins := make([]string, len(goAPI.InternalCopies))
	for i, cp := range goAPI.InternalCopies {
		if plugins[i], err = protocPluginPath(cp.Plugin); err != nil {
			return fmt.Errorf("internal copy %q: %w", cp.ImportPath, err)
		}
	}
	fds, err := apiDescriptorSet(ctx, pc, googleapisDir, apiFiles, tempDir)
	if err != nil {
		return err
	}
	// Each copy is generated from the descriptor set rather than from the
	// proto sources so that the proto package and file paths can be renamed
	// first: protobuf-go registers every file by path and every message by
	// full name in a global registry and panics on conflicts, so a verbatim
	// copy could never be linked next to the public package.
	for i, cp := range goAPI.InternalCopies {
		if err := generateInternalCopy(ctx, cp, plugins[i], goAPI, library, pc, fds, apiFiles, tempDir, outDir); err != nil {
			return fmt.Errorf("internal copy %q: %w", cp.ImportPath, err)
		}
	}
	return nil
}

// apiDescriptorSet runs protoc over the API files and returns their
// descriptor set with imports and source comments, the input the copies are
// rewritten from.
func apiDescriptorSet(ctx context.Context, pc *config.Protoc, googleapisDir string, apiFiles []string, tempDir string) (*descriptorpb.FileDescriptorSet, error) {
	setDir, err := os.MkdirTemp(tempDir, "internal-copy-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(setDir)
	set := filepath.Join(setDir, "api.pb")
	args := []string{
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		"--include_imports",
		"--include_source_info",
		"--descriptor_set_out=" + set,
	}
	for _, file := range apiFiles {
		args = append(args, filepath.Join(googleapisDir, filepath.FromSlash(file)))
	}
	if err := runProtoc(ctx, pc, args...); err != nil {
		return nil, err
	}
	return readDescriptorSet(set)
}

// collectAPIDirProtoFiles returns the proto files directly in the API
// directory, as slash-separated paths relative to googleapisDir.
func collectAPIDirProtoFiles(googleapisDir, apiPath string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(googleapisDir, apiPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read API directory %s: %w", filepath.Join(googleapisDir, apiPath), err)
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".proto" {
			files = append(files, apiPath+"/"+entry.Name())
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%w: %s", errAPIDirNoProtoFiles, filepath.Join(googleapisDir, apiPath))
	}
	return files, nil
}

// generateInternalCopy generates one copy from the API's descriptor set with
// protoc-gen-go and the resolved plugin binary, then moves it into place.
func generateInternalCopy(ctx context.Context, cp *config.GoInternalCopy, plugin string, goAPI *config.GoAPI, library *config.Library, pc *config.Protoc, apiSet *descriptorpb.FileDescriptorSet, apiFiles []string, tempDir, outDir string) error {
	copyDir, err := os.MkdirTemp(tempDir, "internal-copy-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(copyDir)
	// The rewrite edits the set in place, so work on a copy: the same set
	// serves every copy of the API.
	fds := proto.Clone(apiSet).(*descriptorpb.FileDescriptorSet)
	renamed, err := rewriteDescriptorSet(fds, apiFiles, cp)
	if err != nil {
		return err
	}
	copySet := filepath.Join(copyDir, "copy.pb")
	if err := writeDescriptorSet(copySet, fds); err != nil {
		return err
	}
	args := []string{
		"--experimental_allow_proto3_optional",
		"--descriptor_set_in=" + copySet,
		"--go_out=" + copyDir,
		"--plugin=" + protocPluginPrefix + cp.Plugin + "=" + plugin,
		"--" + cp.Plugin + "_out=" + copyDir,
	}
	if goAPI.ProtoAPILevel != "" {
		args = append(args, "--go_opt=default_api_level="+goAPI.ProtoAPILevel)
	}
	for _, opt := range cp.PluginOptions {
		args = append(args, "--"+cp.Plugin+"_opt="+opt)
	}
	args = append(args, renamed...)
	if err := runProtoc(ctx, pc, args...); err != nil {
		return err
	}
	return moveInternalCopy(library, cp, copyDir, outDir)
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

// moveInternalCopy moves the generated copy from the protoc output directory
// to its destination in the repository, mirroring moveAPIDirectory.
func moveInternalCopy(library *config.Library, cp *config.GoInternalCopy, srcDir, outDir string) error {
	src := filepath.Join(srcDir, "cloud.google.com", "go", cp.ImportPath)
	dest := internalCopyDir(library, outDir, cp)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	return filesystem.MoveAndMerge(src, dest)
}
