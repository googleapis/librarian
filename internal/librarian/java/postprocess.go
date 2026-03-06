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

	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/license"
)

type postProcessParams struct {
	outDir         string
	libraryName    string
	version        string
	googleapisDir  string
	protos         []string
	includeSamples bool
	gapicDir       string
	grpcDir        string
	protoDir       string
}

func postProcessAPI(ctx context.Context, p postProcessParams) error {
	// Unzip the temp-codegen.srcjar into temporary version/ directory.
	srcjarPath := filepath.Join(p.gapicDir, "temp-codegen.srcjar")
	if _, err := os.Stat(srcjarPath); err == nil {
		if err := filesystem.Unzip(ctx, srcjarPath, p.gapicDir); err != nil {
			return fmt.Errorf("failed to unzip %s: %w", srcjarPath, err)
		}
	}
	for _, dir := range []string{p.grpcDir, p.protoDir} {
		if err := addMissingHeaders(dir); err != nil {
			return fmt.Errorf("failed to fix headers in %s: %w", dir, err)
		}
	}
	if err := restructureOutput(p); err != nil {
		return fmt.Errorf("failed to restructure output: %w", err)
	}

	// Generate clirr-ignored-differences.xml for the proto module.
	modules := deriveModuleNames(p.libraryName, p.version)
	protoModuleRoot := filepath.Join(p.outDir, modules.proto)
	if err := GenerateClirr(protoModuleRoot); err != nil {
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

type javaModules struct {
	gapic string // e.g., google-cloud-secretmanager
	proto string // e.g., proto-google-cloud-secretmanager-v1
	grpc  string // e.g., grpc-google-cloud-secretmanager-v1
}

func deriveModuleNames(libraryID, version string) javaModules {
	name := libraryID
	if !strings.HasPrefix(name, cloudPrefix) {
		name = cloudPrefix + libraryID
	}
	return javaModules{
		gapic: name,
		proto: fmt.Sprintf("%s%s-%s", protoPrefix, name, version),
		grpc:  fmt.Sprintf("%s%s-%s", grpcPrefix, name, version),
	}
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

// restructureOutput moves the generated code from the temporary versioned directory
// tree into the final directory structure for GAPIC, Proto, gRPC, and samples.
func restructureOutput(p postProcessParams) error {
	modules := deriveModuleNames(p.libraryName, p.version)
	// Temporary source directories (from protoc/generator output)
	tempGapicSrcDir := filepath.Join(p.outDir, p.version, "gapic", "src", "main")
	tempGapicTestDir := filepath.Join(p.outDir, p.version, "gapic", "src", "test")
	tempProtoSrcDir := filepath.Join(p.outDir, p.version, "proto")
	tempGrpcSrcDir := filepath.Join(p.outDir, p.version, "grpc")
	tempResourceNameSrcDir := filepath.Join(p.outDir, p.version, "gapic", "proto", "src", "main", "java")
	tempSamplesDir := filepath.Join(p.outDir, p.version, "gapic", "samples", "snippets", "generated", "src", "main", "java")
	// Final destination directories
	gapicDestDir := filepath.Join(p.outDir, modules.gapic, "src", "main")
	gapicTestDestDir := filepath.Join(p.outDir, modules.gapic, "src", "test")
	protoDestDir := filepath.Join(p.outDir, modules.proto, "src", "main", "java")
	grpcDestDir := filepath.Join(p.outDir, modules.grpc, "src", "main", "java")
	samplesDestDir := filepath.Join(p.outDir, "samples", "snippets", "generated")

	// Ensure destination directories exist
	destDirs := []string{gapicDestDir, gapicTestDestDir, protoDestDir, grpcDestDir}
	if p.includeSamples {
		destDirs = append(destDirs, samplesDestDir)
	}
	for _, dir := range destDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := removeConflictingFiles(tempProtoSrcDir); err != nil {
		return err
	}

	type moveAction struct {
		src, dest   string
		description string
	}
	actions := []moveAction{
		{src: tempProtoSrcDir, dest: protoDestDir, description: "proto source"},
		{src: tempGrpcSrcDir, dest: grpcDestDir, description: "grpc source"},
		{src: tempGapicSrcDir, dest: gapicDestDir, description: "gapic source"},
		{src: tempGapicTestDir, dest: gapicTestDestDir, description: "gapic test"},
		{src: tempResourceNameSrcDir, dest: protoDestDir, description: "resource name source"},
	}
	if p.includeSamples {
		actions = append(actions, moveAction{src: tempSamplesDir, dest: samplesDestDir, description: "samples"})
	}
	for _, action := range actions {
		if _, err := os.Stat(action.src); err == nil {
			if err := filesystem.MoveAndMerge(action.src, action.dest); err != nil {
				return fmt.Errorf("failed to move %s: %w", action.description, err)
			}
		}
	}
	// Copy proto files to proto-*/src/main/proto
	protoFilesDestDir := filepath.Join(p.outDir, modules.proto, "src", "main", "proto")
	if err := copyProtos(p.googleapisDir, p.protos, protoFilesDestDir); err != nil {
		return fmt.Errorf("failed to copy proto files: %w", err)
	}
	return nil
}

func copyProtos(googleapisDir string, protos []string, destDir string) error {
	for _, proto := range protos {
		if strings.HasSuffix(proto, commonProtos) {
			continue
		}
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
