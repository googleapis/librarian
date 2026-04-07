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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/license"
)

const owlbotTemplatesRelPath = "sdk-platform-java/hermetic_build/library_generation/owlbot/templates"

type postProcessParams struct {
	cfg                 *config.Config
	library             *config.Library
	metadata            *repoMetadata
	outDir              string
	librariesBomVersion string
	version             string
	googleapisDir       string
	apiProtos           []string
	includeSamples      bool
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

	// Check if owlbot.py exists in the library output directory.
	// It is required for restructuring the output and generating README files.
	owlbotPath := filepath.Join(p.outDir, "owlbot.py")
	if _, err := os.Stat(owlbotPath); err != nil {
		return fmt.Errorf("owlbot.py not found in %s: %w", p.outDir, err)
	}
	if err := restructureToStaging(p); err != nil {
		return fmt.Errorf("failed to restructure to staging: %w", err)
	}
	if err := runOwlBot(ctx, p); err != nil {
		return fmt.Errorf("failed to run owlbot.py: %w", err)
	}

	// Generate clirr-ignored-differences.xml for the proto module.
	coords := p.coords()
	protoModuleRoot := filepath.Join(p.outDir, coords.Proto.ArtifactID)
	if err := generateClirr(protoModuleRoot); err != nil {
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
	if err := os.Remove(filepath.Join(protoSrcDir, "google", "cloud", "CommonResources.java")); err != nil && !os.IsNotExist(err) {
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
			dest:        filepath.Join(destRoot, coords.Gapic.ArtifactID, "src", "main"),
			description: "gapic source",
		},
		{
			src:         filepath.Join(p.gapicDir(), "src", "test"),
			dest:        filepath.Join(destRoot, coords.Gapic.ArtifactID, "src", "test"),
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

func runOwlBot(ctx context.Context, p postProcessParams) error {
	// Versions used to populate README.md file.
	env := map[string]string{
		"SYNTHTOOL_LIBRARY_VERSION":       p.library.Version,
		"SYNTHTOOL_LIBRARIES_BOM_VERSION": p.librariesBomVersion,
	}
	// Path to templates used for README.md file.
	templatesDir := filepath.Join(filepath.Dir(p.outDir), owlbotTemplatesRelPath)
	if _, err := os.Stat(templatesDir); err != nil {
		return fmt.Errorf("templates directory not found at %s: %w", templatesDir, err)
	}
	env["SYNTHTOOL_TEMPLATES"] = templatesDir
	if err := command.RunInDirWithEnv(ctx, p.outDir, env, "python3", "owlbot.py"); err != nil {
		return err
	}
	// Staging dirs cleans up as part of owlbot.py
	return nil
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
