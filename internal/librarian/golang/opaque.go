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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	protocTool "github.com/googleapis/librarian/internal/tool/protoc"
)

const opaqueProtoPrefix = "librarian_opaque"

var (
	protoGoPackagePattern = regexp.MustCompile(`(?m)^[ \t]*option[ \t]+go_package[ \t]*=[ \t]*"[^"]*"[ \t]*;[^\n]*$`)
	protoImportPattern    = regexp.MustCompile(`(?m)^([ \t]*import(?:[ \t]+(?:public|weak))?[ \t]+")([^"]+)(";[^\n]*$)`)
	protoPackagePatternRE = regexp.MustCompile(`(?m)^[ \t]*package[ \t]+([A-Za-z_][A-Za-z0-9_.]*)[ \t]*;`)
	protoTypePattern      = regexp.MustCompile(`\b(?:message|enum)\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

type opaqueProto struct {
	path   string
	source string
	pkg    string
	types  []string
}

func generateOpaqueCopy(ctx context.Context, library *config.Library, apiPath string, goAPI *config.GoAPI, pc *config.Protoc, googleapisDir, tempDir, outDir string) error {
	copy := goAPI.OpaqueCopy
	includeDir, err := protocIncludeDir(pc)
	if err != nil {
		return err
	}
	apiFiles, err := collectProtoFiles(googleapisDir, apiPath, goAPI.NestedProtos)
	if err != nil {
		return err
	}
	defaultPackageSource, err := os.ReadFile(apiFiles[0])
	if err != nil {
		return err
	}
	defaultPackage, err := parseProtoPackage(defaultPackageSource)
	if err != nil {
		return fmt.Errorf("derive default opaque proto package: %w", err)
	}
	protos, err := collectOpaqueProtos(apiFiles, copy.ExtraProtos, googleapisDir, includeDir)
	if err != nil {
		return err
	}
	targetPackage := copy.ProtoPackage
	if targetPackage == "" {
		targetPackage = defaultPackage + ".internalopaque"
	}
	for _, proto := range protos {
		if proto.pkg == targetPackage {
			return fmt.Errorf("opaque copy proto package %q must differ from copied package %q", targetPackage, proto.pkg)
		}
	}

	sourceDir := filepath.Join(tempDir, "opaque-source")
	generatedDir := filepath.Join(tempDir, "opaque-output")
	if err := os.MkdirAll(generatedDir, 0o755); err != nil {
		return err
	}
	imports := make(map[string]string, len(protos))
	types := make(map[string][]string)
	for _, proto := range protos {
		imports[proto.path] = path.Join(opaqueProtoPrefix, proto.path)
		types[proto.pkg] = append(types[proto.pkg], proto.types...)
	}
	goImportPath := "cloud.google.com/go/" + copy.ImportPath
	goPackage := path.Base(copy.ImportPath)
	var protoPaths []string
	for _, proto := range protos {
		rewritten, err := rewriteOpaqueProto([]byte(proto.source), targetPackage, goImportPath+";"+goPackage, imports, types)
		if err != nil {
			return fmt.Errorf("rewrite %q: %w", proto.path, err)
		}
		protoPath := imports[proto.path]
		destination := filepath.Join(sourceDir, filepath.FromSlash(protoPath))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, rewritten, 0o644); err != nil {
			return err
		}
		protoPaths = append(protoPaths, protoPath)
	}
	slices.Sort(protoPaths)
	args := buildOpaqueProtocArgs(sourceDir, googleapisDir, includeDir, generatedDir, goImportPath, protoPaths)
	if err := runProtoc(ctx, pc, args...); err != nil {
		return err
	}

	generatedSource := filepath.Join(generatedDir, "cloud.google.com", "go", filepath.FromSlash(copy.ImportPath))
	if err := writeOpaqueDoc(generatedSource, goPackage, apiPath, library.CopyrightYear); err != nil {
		return err
	}
	destination := filepath.Join(repoRootPath(outDir, library.Name), clientPathFromRepoRoot(library, &config.GoAPI{ImportPath: copy.ImportPath}))
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	return filesystem.MoveAndMerge(generatedSource, destination)
}

func collectOpaqueProtos(apiFiles, extraProtos []string, googleapisDir, includeDir string) ([]opaqueProto, error) {
	files := make(map[string]string)
	for _, filename := range apiFiles {
		rel, err := filepath.Rel(googleapisDir, filename)
		if err != nil {
			return nil, err
		}
		files[filepath.ToSlash(rel)] = filename
	}
	for _, proto := range extraProtos {
		filename, err := resolveOpaqueProto(proto, googleapisDir, includeDir)
		if err != nil {
			return nil, err
		}
		files[proto] = filename
	}
	paths := make([]string, 0, len(files))
	for proto := range files {
		paths = append(paths, proto)
	}
	sort.Strings(paths)
	protos := make([]opaqueProto, 0, len(paths))
	for _, protoPath := range paths {
		source, err := os.ReadFile(files[protoPath])
		if err != nil {
			return nil, err
		}
		pkg, err := parseProtoPackage(source)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", protoPath, err)
		}
		protos = append(protos, opaqueProto{
			path:   protoPath,
			source: string(source),
			pkg:    pkg,
			types:  parseProtoTypes(source),
		})
	}
	return protos, nil
}

func resolveOpaqueProto(proto, googleapisDir, includeDir string) (string, error) {
	for _, dir := range []string{googleapisDir, includeDir} {
		filename := filepath.Join(dir, filepath.FromSlash(proto))
		info, err := os.Stat(filename)
		if err == nil && !info.IsDir() {
			return filename, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("%w: %q does not exist under googleapis or protoc include directory", errOpaqueCopyExtraProto, proto)
}

func protocIncludeDir(pc *config.Protoc) (string, error) {
	binary, err := protocTool.BinaryPathOrSystem(pc)
	if err != nil {
		return "", fmt.Errorf("find protoc: %w", err)
	}
	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return "", fmt.Errorf("resolve protoc path: %w", err)
	}
	includeDir := filepath.Clean(filepath.Join(filepath.Dir(binary), "..", "include"))
	info, err := os.Stat(includeDir)
	if err != nil {
		return "", fmt.Errorf("find protoc include directory %q: %w", includeDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("protoc include path %q is not a directory", includeDir)
	}
	return includeDir, nil
}

func parseProtoPackage(source []byte) (string, error) {
	match := protoPackagePatternRE.FindSubmatch(source)
	if match == nil {
		return "", errors.New("proto package not found")
	}
	return string(match[1]), nil
}

func parseProtoTypes(source []byte) []string {
	matches := protoTypePattern.FindAllSubmatch(source, -1)
	types := make([]string, 0, len(matches))
	for _, match := range matches {
		types = append(types, string(match[1]))
	}
	return types
}

func rewriteOpaqueProto(source []byte, targetPackage, goPackage string, imports map[string]string, types map[string][]string) ([]byte, error) {
	if _, err := parseProtoPackage(source); err != nil {
		return nil, err
	}
	source, err := stripProtoServices(source)
	if err != nil {
		return nil, err
	}
	source = protoPackagePatternRE.ReplaceAll(source, []byte("package "+targetPackage+";"))
	if protoGoPackagePattern.Match(source) {
		source = protoGoPackagePattern.ReplaceAll(source, []byte("option go_package = \""+goPackage+"\";"))
	} else {
		packageEnd := protoPackagePatternRE.FindIndex(source)
		source = bytes.Join([][]byte{source[:packageEnd[1]], []byte("\n\noption go_package = \"" + goPackage + "\";"), source[packageEnd[1]:]}, nil)
	}
	source = protoImportPattern.ReplaceAllFunc(source, func(line []byte) []byte {
		match := protoImportPattern.FindSubmatch(line)
		if rewritten, ok := imports[string(match[2])]; ok {
			return bytes.Join([][]byte{match[1], []byte(rewritten), match[3]}, nil)
		}
		return line
	})

	packages := make([]string, 0, len(types))
	for pkg := range types {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	for _, pkg := range packages {
		names := slices.Clone(types[pkg])
		slices.SortFunc(names, func(a, b string) int {
			return len(b) - len(a)
		})
		for _, name := range names {
			pattern := regexp.MustCompile(`(^|[^A-Za-z0-9_.])(\.?)` + regexp.QuoteMeta(pkg+"."+name) + `([^A-Za-z0-9_]|$)`)
			replacement := []byte("${1}${2}" + targetPackage + "." + name + "${3}")
			source = pattern.ReplaceAll(source, replacement)
		}
	}
	return source, nil
}

func buildOpaqueProtocArgs(sourceDir, googleapisDir, includeDir, outputDir, importPath string, protoPaths []string) []string {
	args := []string{
		"--experimental_allow_proto3_optional",
		"--go_out=" + outputDir,
		"--go_opt=default_api_level=API_OPAQUE",
	}
	for _, proto := range protoPaths {
		args = append(args, "--go_opt=M"+proto+"="+importPath)
	}
	args = append(args, "-I="+sourceDir, "-I="+googleapisDir, "-I="+includeDir)
	return append(args, protoPaths...)
}

func writeOpaqueDoc(dir, packageName, apiPath, copyrightYear string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`// Copyright %s Google LLC
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

// Code generated by Librarian. DO NOT EDIT.

// Package %s contains a generated internal wire-compatible opaque copy of %s.
//
// Do not import this package outside this module.
package %s
`, copyrightYear, packageName, apiPath, packageName)
	return os.WriteFile(filepath.Join(dir, "doc.go"), []byte(content), 0o644)
}

func stripProtoServices(source []byte) ([]byte, error) {
	for offset := 0; ; {
		start, body, ok := findProtoService(source, offset)
		if !ok {
			return source, nil
		}
		end, err := findProtoBlockEnd(source, body)
		if err != nil {
			return nil, err
		}
		source = append(source[:start], source[end:]...)
		offset = start
	}
}

func findProtoService(source []byte, offset int) (int, int, bool) {
	for i := offset; i < len(source); {
		if next := skipProtoCommentOrString(source, i); next != i {
			i = next
			continue
		}
		if !isProtoIdentifierStart(source[i]) {
			i++
			continue
		}
		start := i
		for i < len(source) && isProtoIdentifierPart(source[i]) {
			i++
		}
		if string(source[start:i]) != "service" {
			continue
		}
		nameStart := skipProtoSpaceAndComments(source, i)
		if nameStart >= len(source) || !isProtoIdentifierStart(source[nameStart]) {
			continue
		}
		nameEnd := nameStart + 1
		for nameEnd < len(source) && isProtoIdentifierPart(source[nameEnd]) {
			nameEnd++
		}
		body := skipProtoSpaceAndComments(source, nameEnd)
		if body < len(source) && source[body] == '{' {
			return start, body, true
		}
	}
	return 0, 0, false
}

func findProtoBlockEnd(source []byte, start int) (int, error) {
	depth := 0
	for i := start; i < len(source); {
		if next := skipProtoCommentOrString(source, i); next != i {
			i = next
			continue
		}
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
		i++
	}
	return 0, errors.New("unterminated service block")
}

func skipProtoSpaceAndComments(source []byte, offset int) int {
	for offset < len(source) {
		if strings.ContainsRune(" \t\r\n", rune(source[offset])) {
			offset++
			continue
		}
		if next := skipProtoComment(source, offset); next != offset {
			offset = next
			continue
		}
		return offset
	}
	return offset
}

func skipProtoCommentOrString(source []byte, offset int) int {
	if next := skipProtoComment(source, offset); next != offset {
		return next
	}
	if source[offset] != '\'' && source[offset] != '"' {
		return offset
	}
	quote := source[offset]
	for i := offset + 1; i < len(source); i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		if source[i] == quote {
			return i + 1
		}
	}
	return len(source)
}

func skipProtoComment(source []byte, offset int) int {
	if offset+1 >= len(source) || source[offset] != '/' {
		return offset
	}
	if source[offset+1] == '/' {
		if end := bytes.IndexByte(source[offset+2:], '\n'); end >= 0 {
			return offset + end + 3
		}
		return len(source)
	}
	if source[offset+1] == '*' {
		if end := bytes.Index(source[offset+2:], []byte("*/")); end >= 0 {
			return offset + end + 4
		}
		return len(source)
	}
	return offset
}

func isProtoIdentifierStart(ch byte) bool {
	return ch == '_' || ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z'
}

func isProtoIdentifierPart(ch byte) bool {
	return isProtoIdentifierStart(ch) || ch >= '0' && ch <= '9'
}
