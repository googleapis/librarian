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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

type postProcessParams struct {
	outDir         string
	libraryName    string
	version        string
	googleapisDir  string
	apiProtos      []string
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
	if err := copyProtos(p.googleapisDir, p.apiProtos, protoFilesDestDir); err != nil {
		return fmt.Errorf("failed to copy proto files: %w", err)
	}
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

// postProcessLibrary coordinates all library-level post-processing tasks,
// such as generating .repo-metadata.json.
func postProcessLibrary(cfg *config.Config, library *config.Library, outDir, googleapisDir string) error {
	// TODO(https://github.com/googleapis/librarian/issues/4217): update pom files.
	// TODO(https://github.com/googleapis/librarian/issues/4218): generate README.md
	metadata, err := deriveRepoMetadata(cfg, library, googleapisDir)
	if err != nil {
		return fmt.Errorf("failed to derive repo metadata: %w", err)
	}
	if err := metadata.write(outDir); err != nil {
		return fmt.Errorf("failed to write .repo-metadata.json: %w", err)
	}
	return nil
}

// deriveRepoMetadata constructs the repoMetadata for a Java library using
// information from the primary service configuration and library-level overrides.
func deriveRepoMetadata(cfg *config.Config, library *config.Library, googleapisDir string) (*repoMetadata, error) {
	serviceconfig.SortAPIs(library.APIs)
	api, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, config.LanguageJava)
	if err != nil {
		return nil, fmt.Errorf("failed to find primary API for path %s: %w", library.APIs[0].Path, err)
	}
	if api == nil {
		return nil, fmt.Errorf("failed to find primary API for path %s: not found", library.APIs[0].Path)
	}
	sharedMetadata := repometadata.FromAPI(cfg, api, library)

	metadata := &repoMetadata{
		APIShortname:         sharedMetadata.APIShortname,
		NamePretty:           sharedMetadata.NamePretty,
		ProductDocumentation: sharedMetadata.ProductDocumentation,
		APIDescription:       sharedMetadata.APIDescription,
		ReleaseLevel:         sharedMetadata.ReleaseLevel,
		Language:             config.LanguageJava,
		Repo:                 sharedMetadata.Repo,
		RepoShort:            fmt.Sprintf("%s-%s", config.LanguageJava, library.Name),
		DistributionName:     sharedMetadata.DistributionName,
		APIID:                sharedMetadata.APIID,
		LibraryType:          repometadata.GAPICAutoLibraryType,
		RequiresBilling:      true,
	}

	// Java-specific overrides and optional fields
	if library.Java.APIIDOverride != "" {
		metadata.APIID = library.Java.APIIDOverride
	}
	if library.Java.APIDescriptionOverride != "" {
		metadata.APIDescription = library.Java.APIDescriptionOverride
	}
	if library.Java.DistributionNameOverride != "" {
		metadata.DistributionName = library.Java.DistributionNameOverride
	}
	if library.Java.IssueTrackerOverride != "" {
		metadata.IssueTracker = library.Java.IssueTrackerOverride
	}
	if library.Java.LibraryTypeOverride != "" {
		metadata.LibraryType = library.Java.LibraryTypeOverride
	}
	if library.Java.NamePrettyOverride != "" {
		metadata.NamePretty = library.Java.NamePrettyOverride
	}
	if library.Java.ProductDocumentationOverride != "" {
		metadata.ProductDocumentation = library.Java.ProductDocumentationOverride
	}
	if library.Java.ClientDocumentationOverride != "" {
		metadata.ClientDocumentation = library.Java.ClientDocumentationOverride
	}
	metadata.RequiresBilling = !library.Java.BillingNotRequired
	// Java only fields
	metadata.CodeownerTeam = library.Java.CodeownerTeam
	metadata.ExtraVersionedModules = library.Java.ExtraVersionedModules
	metadata.ExcludedDependencies = library.Java.ExcludedDependencies
	metadata.ExcludedPoms = library.Java.ExcludedPoms
	metadata.MinJavaVersion = library.Java.MinJavaVersion
	metadata.RecommendedPackage = library.Java.RecommendedPackage
	metadata.RestDocumentation = library.Java.RestDocumentation
	metadata.RpcDocumentation = library.Java.RpcDocumentation

	// distribution_name default for Java is groupId:artifactId
	if !strings.Contains(metadata.DistributionName, ":") {
		groupID := "com.google.cloud"
		if library.Java != nil && library.Java.GroupID != "" {
			groupID = library.Java.GroupID
		}
		artifactID := library.Name
		if !strings.HasPrefix(artifactID, cloudPrefix) {
			artifactID = cloudPrefix + artifactID
		}
		metadata.DistributionName = fmt.Sprintf("%s:%s", groupID, artifactID)
	}
	// Default ClientDocumentation uses artifact ID
	if metadata.ClientDocumentation == "" {
		parts := strings.Split(metadata.DistributionName, ":")
		artifactID := parts[len(parts)-1]
		metadata.ClientDocumentation = fmt.Sprintf("https://cloud.google.com/java/docs/reference/%s/latest/overview", artifactID)
	}
	// transport
	apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageJava)
	if err != nil {
		return nil, fmt.Errorf("failed to find api config: %w", err)
	}
	transport := serviceconfig.GRPCRest
	if apiCfg != nil {
		transport = apiCfg.Transport(config.LanguageJava)
	}
	switch transport {
	case "grpc":
		metadata.Transport = "grpc"
	case "rest":
		metadata.Transport = "http"
	default:
		metadata.Transport = "both"
	}
	return metadata, nil
}
