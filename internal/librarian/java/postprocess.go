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

package java

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/postprocessing"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	errSyncPOMs = errors.New("failed to generate or update pom.xml files")
)

type protoFileToCopy struct {
	absolutePath string
	relativePath string
}

type postProcessParams struct {
	cfg            *config.Config
	library        *config.Library
	javaAPI        *config.JavaAPI
	metadata       *repoMetadata
	outDir         string
	apiBase        string
	protosToCopy   []protoFileToCopy
	includeSamples bool
}

type libraryPostProcessParams struct {
	cfg        *config.Config
	library    *config.Library
	outDir     string
	metadata   *repoMetadata
	transports map[string]serviceconfig.Transport
	primaryDir string
}

func postProcessLibrary(params libraryPostProcessParams) error {
	if params.library.Postprocess != nil {
		if err := postprocessing.Apply(params.outDir, params.library.Postprocess); err != nil {
			return err
		}
	}
	keepSet := toKeepSet(params.library.Keep)
	if err := renderREADME(params, keepSet); err != nil {
		return fmt.Errorf("failed to render README: %w", err)
	}

	monorepoVersion, err := findMonorepoVersion(params.cfg)
	if err != nil {
		return err
	}
	parentVersion, err := findParentPOMVersion(params.cfg)
	if err != nil {
		return err
	}
	if err := syncPOMs(syncPOMsParams{
		library:         params.library,
		libraryDir:      params.outDir,
		monorepoVersion: monorepoVersion,
		parentVersion:   parentVersion,
		metadata:        params.metadata,
		transports:      params.transports,
	}); err != nil {
		return fmt.Errorf("%w: %w", errSyncPOMs, err)
	}

	return nil
}

func (params postProcessParams) gapicDir() string {
	return filepath.Join(params.outDir, params.apiBase, "gapic")
}
func (params postProcessParams) gRPCDir() string {
	return filepath.Join(params.outDir, params.apiBase, "grpc")
}
func (params postProcessParams) protoDir() string {
	return filepath.Join(params.outDir, params.apiBase, "proto")
}
func (params postProcessParams) coords() apiCoordinate {
	return deriveAPICoordinates(deriveLibraryCoordinates(params.library), params.apiBase, params.javaAPI)
}

func postProcessAPI(ctx context.Context, params postProcessParams) error {
	gapicDir := params.gapicDir()
	gRPCDir := params.gRPCDir()
	protoDir := params.protoDir()
	// Unzip the temp-codegen.srcjar into temporary {gapicDir} directory.
	srcjarPath := filepath.Join(gapicDir, "temp-codegen.srcjar")
	if _, err := os.Stat(srcjarPath); err == nil {
		if err := filesystem.Unzip(ctx, srcjarPath, gapicDir); err != nil {
			return fmt.Errorf("failed to unzip %s: %w", srcjarPath, err)
		}
	}
	if err := addHeaders(params, []string{gRPCDir, protoDir}); err != nil {
		return err
	}
	if err := copyFiles(params); err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	var keepSet map[string]bool
	if params.library != nil {
		keepSet = toKeepSet(params.library.Keep)
	}
	if err := restructureToLibrary(params, params.outDir, keepSet); err != nil {
		return fmt.Errorf("failed to restructure to library root: %w", err)
	}

	coords := params.coords()
	// Generate clirr-ignored-differences.xml for the proto module.
	protoModuleRepoRoot := filepath.Join(params.outDir, coords.Proto.ArtifactID)
	shouldGenerate, err := clirrIgnoreShouldGenerate(coords.Proto.ArtifactID, protoModuleRepoRoot, params.javaAPI.Monolithic)
	if err != nil {
		return fmt.Errorf("failed to check for clirr ignore file: %w", err)
	}
	if shouldGenerate {
		if err := generateClirrIgnore(protoModuleRepoRoot); err != nil {
			return fmt.Errorf("failed to generate clirr ignore file: %w", err)
		}
	}

	// Cleanup intermediate protoc output directory after restructuring
	if err := os.RemoveAll(filepath.Join(params.outDir, params.apiBase)); err != nil {
		return fmt.Errorf("failed to cleanup intermediate files: %w", err)
	}
	return nil
}

func addHeaders(params postProcessParams, dirs []string) error {
	if params.javaAPI.Monolithic && (params.library.Java == nil || params.library.Java.AlternateHeaders == "") {
		return nil
	}
	for _, dir := range dirs {
		if err := addMissingHeaders(params, dir); err != nil {
			return fmt.Errorf("failed to fix headers in %s: %w", dir, err)
		}
	}
	return nil
}

// addMissingHeaders prepends the license header to all Java files in the given directory
// if they don't already have one.
func addMissingHeaders(params postProcessParams, dir string) error {
	headerText, err := getLicenseText(params)
	if err != nil {
		return err
	}
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() || filepath.Ext(path) != ".java" {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if license.HasHeader(content) {
			return nil
		}
		return os.WriteFile(path, append(headerText, content...), 0o644)
	})
}

// getLicenseText reads the contents of the alternate_header property (a filepath)
// if a library has an alternate header file. Otherwise it will grab the default license
// header.
func getLicenseText(params postProcessParams) ([]byte, error) {
	if params.library == nil || params.library.Java == nil || params.library.Java.AlternateHeaders == "" {
		year := time.Now().Year()
		return []byte(buildLicenseText(year)), nil
	}
	headerPath := filepath.Join(params.outDir, params.library.Java.AlternateHeaders)
	b, err := os.ReadFile(headerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read alternate header file %s: %w", headerPath, err)
	}
	// Ensure the alternate header ends with a newline before it is prepended.
	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	return b, nil
}

func copyFiles(params postProcessParams) error {
	if params.javaAPI == nil || len(params.javaAPI.CopyFiles) == 0 {
		return nil
	}
	gapicDir := params.gapicDir()
	for _, c := range params.javaAPI.CopyFiles {
		src := filepath.Join(gapicDir, c.Source)
		dest := filepath.Join(gapicDir, c.Destination)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("failed to stat copy source %s: %w", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("failed to create destination directory for %s: %w", dest, err)
		}
		if err := filesystem.CopyFile(src, dest); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", src, dest, err)
		}
	}
	return nil
}

// buildLicenseText constructs the complete license header text for the given year.
func buildLicenseText(year int) string {
	lines := license.Header(strconv.Itoa(year))
	var b strings.Builder
	b.WriteString("/*\n")
	for _, line := range lines {
		b.WriteString(" *")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(" */\n")
	return b.String()
}

func removeConflictingFiles(protoSrcDir string) error {
	// These files are removed because they are often duplicated across
	// multiple artifacts in the Google Cloud Java ecosystem, leading
	// to classpath conflicts.
	if err := os.RemoveAll(filepath.Join(protoSrcDir, "com", "google", "cloud", "location")); err != nil {
		return fmt.Errorf("failed to remove location classes: %w", err)
	}
	if err := os.Remove(filepath.Join(protoSrcDir, "google", "cloud", "CommonResources.java")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove CommonResources.java: %w", err)
	}
	return nil
}

type moveAction struct {
	src, dest   string
	description string
}

func copyProtos(protos []protoFileToCopy, destDir string) error {
	for _, proto := range protos {
		target := filepath.Join(destDir, proto.relativePath)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(target), err)
		}
		if err := filesystem.CopyFile(proto.absolutePath, target); err != nil {
			return fmt.Errorf("failed to copy file %s to %s: %w", proto.absolutePath, target, err)
		}
	}
	return nil
}

// ApplyMoveActionsToLibrary moves generated code to the repository directory
// structure, merging directories and preserving files matching the keepSet.
func ApplyMoveActionsToLibrary(actions []moveAction, destRoot string, keepSet map[string]bool) error {
	for _, action := range actions {
		if _, err := os.Stat(action.src); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// TODO(https://github.com/googleapis/librarian/issues/6752): Return an error here once owlbot.py is removed.
				continue
			}
			return fmt.Errorf("failed to check source directory %s: %w", action.src, err)
		}
		if err := os.MkdirAll(action.dest, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", action.dest, err)
		}
		err := filesystem.MoveAndMergeWithKeep(action.src, action.dest, destRoot, func(rel string) bool {
			return shouldPreserve(rel, keepSet)
		})
		if err != nil {
			return fmt.Errorf("failed to move %s: %w", action.description, err)
		}
	}
	return nil
}

// toKeepSet normalizes a list of keep paths into a lookup map.
func toKeepSet(keep []string) map[string]bool {
	keepSet := make(map[string]bool, len(keep))
	for _, k := range keep {
		normalized := strings.TrimSuffix(filepath.ToSlash(k), "/")
		keepSet[normalized] = true
	}
	return keepSet
}

// restructureToLibrary moves all generated source code to the library root directories.
// It also removes conflicting files, and copies public proto files to the library.
func restructureToLibrary(params postProcessParams, destRoot string, keepSet map[string]bool) error {
	tempProtoSrcDir := params.protoDir()
	isCommonProtos := params.library.Name == commonProtosLibrary
	if !isCommonProtos {
		if err := removeConflictingFiles(tempProtoSrcDir); err != nil {
			return err
		}
	}
	if err := moveSourcesToLibrary(params, destRoot, keepSet); err != nil {
		return err
	}
	if err := copyProtoFilesToLibrary(params, destRoot); err != nil {
		return err
	}
	return nil
}

// moveSourcesToLibrary relocates the generated Java source files (GAPIC, proto,
// gRPC, resource names, and samples) to their repository destinations.
func moveSourcesToLibrary(params postProcessParams, destRoot string, keepSet map[string]bool) error {
	coords := params.coords()
	isMonolithic := params.javaAPI.Monolithic
	var protoDest, grpcDest, gapicMainDest, gapicTestDest string
	// Determine target repository subdirectories based on library structure.
	if isMonolithic {
		protoDest = filepath.Join(destRoot, "src", "main", "java")
		grpcDest = filepath.Join(destRoot, "src", "main", "java")
		gapicMainDest = filepath.Join(destRoot, "src", "main")
		gapicTestDest = filepath.Join(destRoot, "src", "test")
	} else {
		protoDest = filepath.Join(destRoot, coords.Proto.ArtifactID, "src", "main", "java")
		grpcDest = filepath.Join(destRoot, coords.GRPC.ArtifactID, "src", "main", "java")
		gapicMainDest = filepath.Join(destRoot, coords.GAPIC.ArtifactID, "src", "main")
		gapicTestDest = filepath.Join(destRoot, coords.GAPIC.ArtifactID, "src", "test")
	}
	var actions []moveAction
	// Collect generated source directories to relocate.
	if shouldGenerateProto(params.javaAPI) {
		actions = append(actions, moveAction{src: params.protoDir(), dest: protoDest, description: "proto main source files"})
	}
	if shouldGenerateGRPC(params.javaAPI) {
		actions = append(actions, moveAction{src: params.gRPCDir(), dest: grpcDest, description: "gRPC main source files"})
	}
	if shouldGenerateGAPIC(params.javaAPI) {
		actions = append(actions,
			moveAction{src: filepath.Join(params.gapicDir(), "src", "main"), dest: gapicMainDest, description: "GAPIC main source files"},
			moveAction{src: filepath.Join(params.gapicDir(), "src", "test"), dest: gapicTestDest, description: "GAPIC test source files"},
		)
	}
	if shouldGenerateResourceNames(params.javaAPI) {
		actions = append(actions, moveAction{src: filepath.Join(params.gapicDir(), "proto", "src", "main", "java"), dest: protoDest, description: "resource name source files"})
	}
	if params.includeSamples && shouldGenerateGAPIC(params.javaAPI) {
		actions = append(actions, moveAction{src: filepath.Join(params.gapicDir(), "samples", "snippets", "generated", "src", "main", "java"), dest: filepath.Join(destRoot, "samples", "snippets", "generated"), description: "samples"})
	}
	// Relocate all collected source files to their destinations.
	return ApplyMoveActionsToLibrary(actions, destRoot, keepSet)
}

// copyProtoFilesToLibrary copies public proto definition (.proto) files from the
// generator inputs to their target directory structure.
func copyProtoFilesToLibrary(params postProcessParams, destRoot string) error {
	if !shouldGenerateProto(params.javaAPI) {
		return nil
	}
	coords := params.coords()
	var destProtoDir string
	if params.javaAPI.Monolithic {
		destProtoDir = filepath.Join(destRoot, "src", "main", "proto")
	} else {
		destProtoDir = filepath.Join(destRoot, coords.Proto.ArtifactID, "src", "main", "proto")
	}
	if err := copyProtos(params.protosToCopy, destProtoDir); err != nil {
		return fmt.Errorf("failed to copy proto files: %w", err)
	}
	return nil
}
