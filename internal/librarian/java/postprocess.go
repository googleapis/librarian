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
	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const owlbotTemplatesRelPath = "sdk-platform-java/hermetic_build/library_generation/owlbot/templates"

var (
	errOwlBotMissing    = errors.New("owlbot.py not found")
	errTemplatesMissing = errors.New("templates directory not found")
	errRunOwlBot        = errors.New("failed to run owlbot.py")
	errSyncPOMs         = errors.New("failed to generate or update pom.xml files")
	errInvalidVersion   = errors.New("invalid java library version")
)

type postProcessParams struct {
	cfg            *config.Config
	library        *config.Library
	metadata       *repoMetadata
	outDir         string
	version        string
	googleapisDir  string
	apiProtos      []string
	includeSamples bool
}

type libraryPostProcessParams struct {
	cfg        *config.Config
	library    *config.Library
	outDir     string
	metadata   *repoMetadata
	transports map[string]serviceconfig.Transport
}

func postProcessLibrary(ctx context.Context, p libraryPostProcessParams) error {
	// Check if owlbot.py exists in the library output directory.
	// It is required for restructuring the output and generating README files.
	owlbotPath := filepath.Join(p.outDir, "owlbot.py")
	if _, err := os.Stat(owlbotPath); err != nil {
		return fmt.Errorf("%w in %s: %w", errOwlBotMissing, p.outDir, err)
	}
	bomVersion, err := findBOMVersion(p.cfg)
	if err != nil {
		return err
	}
	if err := runOwlBot(ctx, p.library, p.outDir, bomVersion); err != nil {
		return fmt.Errorf("%w: %w", errRunOwlBot, err)
	}

	monorepoVersion, err := findMonorepoVersion(p.cfg)
	if err != nil {
		return err
	}
	if err := syncPOMs(p.library, p.outDir, monorepoVersion, p.metadata, p.transports); err != nil {
		return fmt.Errorf("%w: %w", errSyncPOMs, err)
	}

	return nil
}

func (p postProcessParams) gapicDir() string { return filepath.Join(p.outDir, p.version, "gapic") }
func (p postProcessParams) gRPCDir() string  { return filepath.Join(p.outDir, p.version, "grpc") }
func (p postProcessParams) protoDir() string { return filepath.Join(p.outDir, p.version, "proto") }
func (p postProcessParams) coords() APICoordinate {
	return DeriveAPICoordinates(DeriveLibraryCoordinates(p.library), p.version)
}

func postProcessAPI(ctx context.Context, p postProcessParams) error {
	gapicDir := p.gapicDir()
	gRPCDir := p.gRPCDir()
	protoDir := p.protoDir()
	// Unzip the temp-codegen.srcjar into temporary version/ directory.
	srcjarPath := filepath.Join(gapicDir, "temp-codegen.srcjar")
	if _, err := os.Stat(srcjarPath); err == nil {
		if err := filesystem.Unzip(ctx, srcjarPath, gapicDir); err != nil {
			return fmt.Errorf("failed to unzip %s: %w", srcjarPath, err)
		}
	}
	for _, dir := range []string{gRPCDir, protoDir} {
		if err := addMissingHeaders(dir); err != nil {
			return fmt.Errorf("failed to fix headers in %s: %w", dir, err)
		}
	}

	if err := restructureToStaging(p); err != nil {
		return fmt.Errorf("failed to restructure to staging: %w", err)
	}

	// Generate clirr-ignored-differences.xml for the proto module.
	// We target the staging directory because runOwlBot hasn't moved the files
	// to their final destination yet.
	coords := p.coords()
	protoModuleStagingRoot := filepath.Join(p.outDir, "owl-bot-staging", p.version, coords.Proto.ArtifactID)
	if err := generateClirrIfMissing(protoModuleStagingRoot); err != nil {
		return fmt.Errorf("failed to generate clirr ignore file: %w", err)
	}

	// Cleanup intermediate protoc output directory after restructuring
	if err := os.RemoveAll(filepath.Join(p.outDir, p.version)); err != nil {
		return fmt.Errorf("failed to cleanup intermediate files: %w", err)
	}
	return nil
}

// addMissingHeaders prepends the license header to all Java files in the given directory
// if they don't already have one.
func addMissingHeaders(dir string) error {
	year := time.Now().Year()
	licenseText := buildLicenseText(year)
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
		return os.WriteFile(path, append([]byte(licenseText), content...), 0644)
	})
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
// version directory (e.g., owl-bot-staging/v1/proto-google-cloud-chat-v1) to
// ensure synthtool preserves the module structure.
func restructureToStaging(p postProcessParams) error {
	stagingDir := filepath.Join(p.outDir, "owl-bot-staging")
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	return restructureModules(p, filepath.Join(stagingDir, p.version))
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
func restructureModules(p postProcessParams, destRoot string) error {
	coords := p.coords()
	tempProtoSrcDir := p.protoDir()
	if err := removeConflictingFiles(tempProtoSrcDir); err != nil {
		return err
	}
	actions := []moveAction{
		{
			src:         tempProtoSrcDir,
			dest:        filepath.Join(destRoot, coords.Proto.ArtifactID, "src", "main", "java"),
			description: "proto source",
		},
		{
			src:         p.gRPCDir(),
			dest:        filepath.Join(destRoot, coords.GRPC.ArtifactID, "src", "main", "java"),
			description: "grpc source",
		},
		{
			src:         filepath.Join(p.gapicDir(), "src", "main"),
			dest:        filepath.Join(destRoot, coords.GAPIC.ArtifactID, "src", "main"),
			description: "gapic source",
		},
		{
			src:         filepath.Join(p.gapicDir(), "src", "test"),
			dest:        filepath.Join(destRoot, coords.GAPIC.ArtifactID, "src", "test"),
			description: "gapic test",
		},
		{
			src:         filepath.Join(p.gapicDir(), "proto", "src", "main", "java"),
			dest:        filepath.Join(destRoot, coords.Proto.ArtifactID, "src", "main", "java"),
			description: "resource name source",
		},
	}
	if p.includeSamples {
		actions = append(actions, moveAction{
			src:         filepath.Join(p.gapicDir(), "samples", "snippets", "generated", "src", "main", "java"),
			dest:        filepath.Join(destRoot, "samples", "snippets", "generated"),
			description: "samples",
		})
	}
	if err := restructure(actions); err != nil {
		return err
	}
	// Copy proto files to proto-*/src/main/proto
	protoFilesDestDir := filepath.Join(destRoot, coords.Proto.ArtifactID, "src", "main", "proto")
	if err := copyProtos(p.googleapisDir, p.apiProtos, protoFilesDestDir); err != nil {
		return fmt.Errorf("failed to copy proto files: %w", err)
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
func runOwlBot(ctx context.Context, library *config.Library, outDir, bomVersion string) error {
	releasedVersion, err := releasedVersion(library.Version)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidVersion, library.Version)
	}
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

// releasedVersion derives the last released version from a snapshot version
// (e.g., x.y.0-SNAPSHOT) by decrementing the minor version.
//
// It returns an error if the snapshot version has a non-zero patch or a zero
// minor version, as this repository is assumed to always bump the minor version.
func releasedVersion(v string) (string, error) {
	if !strings.HasSuffix(v, "-SNAPSHOT") {
		return v, nil
	}
	sv, err := semver.Parse(strings.TrimSuffix(v, "-SNAPSHOT"))
	if err != nil {
		return "", err
	}
	if sv.Patch > 0 || (sv.Minor == 0 && sv.Patch == 0) {
		return "", errInvalidVersion
	}
	sv.Minor--
	sv.Patch = 0
	return sv.String(), nil
}

func copyProtos(googleapisDir string, protos []string, destDir string) error {
	for _, proto := range protos {
		// Calculate relative path from googleapisDir to preserve directory structure
		rel, err := filepath.Rel(googleapisDir, proto)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path for %s: %w", proto, err)
		}
		target := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(target), err)
		}
		if err := filesystem.CopyFile(proto, target); err != nil {
			return fmt.Errorf("failed to copy file %s to %s: %w", proto, target, err)
		}
	}
	return nil
}
