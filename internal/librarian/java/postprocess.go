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

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	owlbotTemplatesRelPath = "sdk-platform-java/hermetic_build/library_generation/owlbot/templates"
	owlbotStagingDir       = "owl-bot-staging"
)

var (
	errTemplatesMissing = errors.New("templates directory not found")
	errRunOwlBot        = errors.New("failed to run owlbot.py")
	errSyncPOMs         = errors.New("failed to generate or update pom.xml files")
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
}

func postProcessLibrary(ctx context.Context, params libraryPostProcessParams) error {
	if err := createOrVerifyOwlbotPy(params.outDir); err != nil {
		return err
	}
	bomVersion, err := findBOMVersion(params.cfg)
	if err != nil {
		return err
	}
	if err := removeKeptFilesFromStaging(params.library, params.outDir); err != nil {
		return fmt.Errorf("failed to remove kept files from staging: %w", err)
	}
	if err := runOwlBot(ctx, params.library, params.outDir, bomVersion); err != nil {
		return fmt.Errorf("%w: %w", errRunOwlBot, err)
	}

	monorepoVersion, err := findMonorepoVersion(params.cfg)
	if err != nil {
		return err
	}
	if err := syncPOMs(params.library, params.outDir, monorepoVersion, params.metadata, params.transports); err != nil {
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
func (params postProcessParams) coords() APICoordinate {
	return DeriveAPICoordinates(DeriveLibraryCoordinates(params.library), params.apiBase, params.javaAPI)
}

func stagingDir(outDir string) string { return filepath.Join(outDir, owlbotStagingDir) }

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
	if err := restructureToStaging(params); err != nil {
		return fmt.Errorf("failed to restructure to staging: %w", err)
	}

	// Generate clirr-ignored-differences.xml for the proto module.
	// We target the staging directory because runOwlBot hasn't moved the files
	// to their final destination yet.
	coords := params.coords()
	protoModuleRepoRoot := filepath.Join(params.outDir, coords.Proto.ArtifactID)
	shouldGenerate, err := clirrIgnoreShouldGenerate(coords.Proto.ArtifactID, protoModuleRepoRoot, params.javaAPI.Monolithic)
	if err != nil {
		return fmt.Errorf("failed to check for clirr ignore file: %w", err)
	}
	if shouldGenerate {
		protoModuleStagingRoot := filepath.Join(stagingDir(params.outDir), params.apiBase, coords.Proto.ArtifactID)
		if err := generateClirrIgnore(protoModuleStagingRoot); err != nil {
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
		return os.WriteFile(path, append(headerText, content...), 0644)
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
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
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

// restructureToStaging moves the generated code into a temporary staging directory
// that matches the structure expected by owlbot.py. It nests modules under the
// {apiBase} directory (e.g., owl-bot-staging/v1/proto-google-cloud-chat-v1) to
// ensure synthtool preserves the module structure.
func restructureToStaging(params postProcessParams) error {
	stagingDir := stagingDir(params.outDir)
	destRoot := filepath.Join(stagingDir, params.apiBase)
	if params.javaAPI.Monolithic {
		destRoot = filepath.Join(destRoot, "src")
	}
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	return restructureModules(params, destRoot)
}

type moveAction struct {
	src, dest   string
	description string
}

func restructure(actions []moveAction) error {
	for _, action := range actions {
		if _, err := os.Stat(action.src); err == nil {
			if err := os.MkdirAll(action.dest, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", action.dest, err)
			}
			if err := filesystem.MoveAndMerge(action.src, action.dest); err != nil {
				return fmt.Errorf("failed to move %s: %w", action.description, err)
			}
		}
	}
	return nil
}

// restructureModules moves the generated code from the temporary versioned directory
// tree into the destination root directory for GAPIC, Proto, gRPC, and samples.
// It also copies the relevant proto files into the proto module.
func restructureModules(params postProcessParams, destRoot string) error {
	coords := params.coords()
	tempProtoSrcDir := params.protoDir()
	if params.library.Name != commonProtosLibrary {
		if err := removeConflictingFiles(tempProtoSrcDir); err != nil {
			return err
		}
	}

	protoDest := filepath.Join(destRoot, coords.Proto.ArtifactID, "src", "main", "java")
	grpcDest := filepath.Join(destRoot, coords.GRPC.ArtifactID, "src", "main", "java")
	gapicMainDest := filepath.Join(destRoot, coords.GAPIC.ArtifactID, "src", "main")
	gapicTestDest := filepath.Join(destRoot, coords.GAPIC.ArtifactID, "src", "test")
	protoFilesDestDir := filepath.Join(destRoot, coords.Proto.ArtifactID, "src", "main", "proto")

	if params.javaAPI.Monolithic {
		protoDest = filepath.Join(destRoot, "main", "java")
		grpcDest = filepath.Join(destRoot, "main", "java")
		gapicMainDest = filepath.Join(destRoot, "main")
		gapicTestDest = filepath.Join(destRoot, "test")
		protoFilesDestDir = filepath.Join(destRoot, "main", "proto")
	}

	var actions []moveAction
	if shouldGenerateProto(params.javaAPI) {
		actions = append(actions, moveAction{
			src:         tempProtoSrcDir,
			dest:        protoDest,
			description: "proto source",
		})
	}
	if shouldGenerateGRPC(params.javaAPI) {
		actions = append(actions, moveAction{
			src:         params.gRPCDir(),
			dest:        grpcDest,
			description: "grpc source",
		})
	}
	if shouldGenerateGAPIC(params.javaAPI) {
		actions = append(actions, []moveAction{
			{
				src:         filepath.Join(params.gapicDir(), "src", "main"),
				dest:        gapicMainDest,
				description: "gapic source",
			},
			{
				src:         filepath.Join(params.gapicDir(), "src", "test"),
				dest:        gapicTestDest,
				description: "gapic test",
			},
		}...)
	}
	if shouldGenerateResourceNames(params.javaAPI) {
		actions = append(actions, moveAction{
			src:         filepath.Join(params.gapicDir(), "proto", "src", "main", "java"),
			dest:        protoDest,
			description: "resource name source",
		})
	}
	if params.includeSamples && shouldGenerateGAPIC(params.javaAPI) {
		actions = append(actions, moveAction{
			src:         filepath.Join(params.gapicDir(), "samples", "snippets", "generated", "src", "main", "java"),
			dest:        filepath.Join(destRoot, "samples", "snippets", "generated"),
			description: "samples",
		})
	}
	if err := restructure(actions); err != nil {
		return err
	}
	// Copy proto files to proto-*/src/main/proto
	if shouldGenerateProto(params.javaAPI) {
		if err := copyProtos(params.protosToCopy, protoFilesDestDir); err != nil {
			return fmt.Errorf("failed to copy proto files: %w", err)
		}
	}
	return nil
}

// runOwlBot executes the owlbot.py script located in outDir to restructure the
// generated code and apply templates (e.g., for README.md).
//
// It assumes that:
//  1. All APIs for the library have already been generated and staged into the
//     "owl-bot-staging" directory (see restructureToStaging()).
//  2. An owlbot.py file exists in the outDir.
//  3. The SYNTHTOOL_TEMPLATES environment variable points to a valid templates
//     directory in google-cloud-java/sdk-platform-java.
//  4. python3 is available on the system PATH and has the synthtool package
//     installed (from google-cloud-java/sdk-platform-java).
func runOwlBot(ctx context.Context, library *config.Library, outDir, bomVersion string) (retErr error) {
	// Clean up the staging directory on failure to avoid leaving dirty leftovers.
	// If owlbot.py completes successfully, it is expected to clean it up.
	defer func() {
		if retErr != nil {
			_ = os.RemoveAll(stagingDir(outDir))
		}
	}()

	releasedVersion := library.Java.ReleasedVersion
	// Versions used to populate README.md file.
	env := map[string]string{
		"SYNTHTOOL_LIBRARY_VERSION":       releasedVersion,
		"SYNTHTOOL_LIBRARIES_BOM_VERSION": bomVersion,
	}
	// Path to templates used for README.md file.
	templatesDir := filepath.Join(filepath.Dir(outDir), owlbotTemplatesRelPath)
	if _, err := os.Stat(templatesDir); err != nil {
		return fmt.Errorf("%w at %s: %w", errTemplatesMissing, templatesDir, err)
	}
	env["SYNTHTOOL_TEMPLATES"] = templatesDir
	if err := command.RunInDirWithEnv(ctx, outDir, env, "python3", "owlbot.py"); err != nil {
		return err
	}
	// Staging dirs cleans up as part of owlbot.py
	return nil
}

func copyProtos(protos []protoFileToCopy, destDir string) error {
	for _, proto := range protos {
		target := filepath.Join(destDir, proto.relativePath)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(target), err)
		}
		if err := filesystem.CopyFile(proto.absolutePath, target); err != nil {
			return fmt.Errorf("failed to copy file %s to %s: %w", proto.absolutePath, target, err)
		}
	}
	return nil
}

// removeKeptFilesFromStaging removes files and directories from the staging area
// that are marked to be preserved in the library configuration.
//
// It operates on the assumption that the staging directory structure nests
// modules under an API base directory component (e.g., owl-bot-staging/v1/proto-google-cloud-library-v1/...).
// It strips this first component (the API base like "v1") from the relative
// path to reconstruct the expected path relative to the library root, which is
// then matched against the library's Keep configuration.
func removeKeptFilesFromStaging(library *config.Library, outDir string) error {
	stagingDir := stagingDir(outDir)
	if _, err := os.Stat(stagingDir); os.IsNotExist(err) {
		return nil
	}
	keepSet := make(map[string]bool)
	for _, keep := range library.Keep {
		normalized := strings.TrimSuffix(filepath.ToSlash(keep), "/")
		keepSet[normalized] = true
	}
	return filepath.WalkDir(stagingDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relToStaging, err := filepath.Rel(stagingDir, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(relToStaging)
		i := strings.Index(relSlash, "/")
		if i == -1 {
			// Skip the staging root "." and API base directories (e.g., "v1").
			return nil
		}
		keepPath := relSlash[i+1:]
		if d.IsDir() {
			if keepSet[keepPath] {
				destPath := filepath.Join(outDir, keepPath)
				if _, err := os.Stat(destPath); err == nil {
					if err := os.RemoveAll(path); err != nil {
						return fmt.Errorf("failed to remove kept dir %s from staging: %w", path, err)
					}
				}
				return filepath.SkipDir
			}
			return nil
		}
		if shouldPreserve(keepPath, keepSet) {
			destPath := filepath.Join(outDir, keepPath)
			if _, err := os.Stat(destPath); err == nil {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove kept file %s from staging: %w", path, err)
				}
			}
		}
		return nil
	})
}

// createOrVerifyOwlbotPy ensures that the post-processing script (owlbot.py) exists
// in the library's output directory. If it is missing (which is typical for newly added
// client libraries), it automatically creates it from an embedded template to allow
// OwlBot post-processing and README generation to complete successfully.
func createOrVerifyOwlbotPy(outDir string) (err error) {
	owlbotPath := filepath.Join(outDir, "owlbot.py")
	// Open with O_EXCL to atomically ensure we only create the script if it does not exist.
	// Executable permissions (0755) are set because owlbot.py is executed during post-processing.
	file, createErr := os.OpenFile(owlbotPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0755)
	if errors.Is(createErr, fs.ErrExist) {
		return nil
	}
	if createErr != nil {
		return fmt.Errorf("failed to create owlbot.py: %w", createErr)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close owlbot.py: %w", closeErr)
		}
	}()
	if executeErr := templates.ExecuteTemplate(file, "owlbot_py.tmpl", nil); executeErr != nil {
		return fmt.Errorf("failed to write owlbot.py template: %w", executeErr)
	}
	return nil
}
