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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/librarian/java"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	generationConfigFileName = "generation_config.yaml"
	managedProtoStart        = "<!-- {x-generated-proto-dependencies-start} -->"
	managedProtoEnd          = "<!-- {x-generated-proto-dependencies-end} -->"
	managedGrpcStart         = "<!-- {x-generated-grpc-dependencies-start} -->"
	managedGrpcEnd           = "<!-- {x-generated-grpc-dependencies-end} -->"
)

var (
	fetchSourceWithCommit = fetchGoogleapisWithCommit
)

type javaGAPICInfo struct {
	NoSamples        bool
	AdditionalProtos []string
}

func parseJavaBazel(googleapisDir, dir string) (*javaGAPICInfo, error) {
	file, err := parseBazel(googleapisDir, dir)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}
	info := &javaGAPICInfo{}
	// 1. From java_gapic_library
	if rules := file.Rules("java_gapic_library"); len(rules) > 0 {
		if len(rules) > 1 {
			log.Printf("Warning: multiple java_gapic_library in %s/BUILD.bazel, using first", dir)
		}
	}
	// 2. From java_gapic_assembly_gradle_pkg
	if rules := file.Rules("java_gapic_assembly_gradle_pkg"); len(rules) > 0 {
		if len(rules) > 1 {
			log.Printf("Warning: multiple java_gapic_assembly_gradle_pkg in %s/BUILD.bazel, using first", dir)
		}
		rule := rules[0]
		info.NoSamples = rule.AttrLiteral("include_samples") == "False"
	}
	// 3. From proto_library_with_info
	if rules := file.Rules("proto_library_with_info"); len(rules) > 0 {
		if len(rules) > 1 {
			log.Printf("Warning: multiple proto_library_with_info in %s/BUILD.bazel, using first", dir)
		}
		rule := rules[0]
		// Search for specific common resource targets in deps
		if deps := rule.AttrStrings("deps"); len(deps) > 0 {
			protoMappings := map[string]string{
				"//google/cloud:common_resources_proto":  "google/cloud/common_resources.proto",
				"//google/cloud/location:location_proto": "google/cloud/location/locations.proto",
				"//google/iam/v1:iam_policy_proto":       "google/iam/v1/iam_policy.proto",
			}
			for _, dep := range deps {
				if protoPath, ok := protoMappings[dep]; ok {
					info.AdditionalProtos = append(info.AdditionalProtos, protoPath)
				}
			}
		}
	}
	return info, nil
}

// GAPICConfig represents the GAPIC configuration in generation_config.yaml.
type GAPICConfig struct {
	ProtoPath string `yaml:"proto_path"`
}

// LibraryConfig represents a library entry in generation_config.yaml.
type LibraryConfig struct {
	APIDescription        string        `yaml:"api_description"`
	APIID                 string        `yaml:"api_id"`
	APIShortName          string        `yaml:"api_shortname"`
	APIReference          string        `yaml:"api_reference"`
	ClientDocumentation   string        `yaml:"client_documentation"`
	CloudAPI              *bool         `yaml:"cloud_api"`
	CodeownerTeam         string        `yaml:"codeowner_team"`
	DistributionName      string        `yaml:"distribution_name"`
	ExcludedDependencies  string        `yaml:"excluded_dependencies"`
	ExcludedPoms          string        `yaml:"excluded_poms"`
	ExtraVersionedModules string        `yaml:"extra_versioned_modules"`
	GAPICs                []GAPICConfig `yaml:"GAPICs"`
	GroupID               string        `yaml:"group_id"`
	IssueTracker          string        `yaml:"issue_tracker"`
	LibraryName           string        `yaml:"library_name"`
	LibraryType           string        `yaml:"library_type"`
	MinJavaVersion        int           `yaml:"min_java_version"`
	NamePretty            string        `yaml:"name_pretty"`
	ProductDocumentation  string        `yaml:"product_documentation"`
	RecommendedPackage    string        `yaml:"recommended_package"`
	ReleaseLevel          string        `yaml:"release_level"`
	RequiresBilling       *bool         `yaml:"requires_billing"`
	RestDocumentation     string        `yaml:"rest_documentation"`
	RpcDocumentation      string        `yaml:"rpc_documentation"`
	Transport             string        `yaml:"transport"`
}

// GenerationConfig represents the root of generation_config.yaml.
type GenerationConfig struct {
	GoogleapisCommitish string          `yaml:"googleapis_commitish"`
	LibrariesBomVersion string          `yaml:"libraries_bom_version"`
	Libraries           []LibraryConfig `yaml:"libraries"`
}

func runJavaMigration(ctx context.Context, repoPath string) error {
	gen, err := readGenerationConfig(repoPath)
	if err != nil {
		return err
	}
	commit := gen.GoogleapisCommitish
	if commit == "" {
		commit = "master"
	}
	src, err := fetchSourceWithCommit(ctx, githubEndpoints, commit)
	if err != nil {
		return errFetchSource
	}
	versions, err := readVersions(filepath.Join(repoPath, "versions.txt"))
	if err != nil {
		return err
	}
	cfg := buildConfig(gen, repoPath, src, versions)
	if cfg == nil {
		return fmt.Errorf("no libraries found to migrate")
	}
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""

	if err := insertMarkers(repoPath, cfg); err != nil {
		return fmt.Errorf("failed to insert markers: %w", err)
	}

	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return errTidyFailed
	}
	log.Printf("Successfully migrated %d Java libraries", len(cfg.Libraries))
	return nil
}

func readGenerationConfig(path string) (*GenerationConfig, error) {
	return yaml.Read[GenerationConfig](filepath.Join(path, generationConfigFileName))
}

// readVersions parses versions.txt and returns a map of module names to snapshot versions.
// It expects the "module:released-version:current-version" format.
func readVersions(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	versions := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("read versions in %s: line %q has %d parts, want 3", path, line, len(parts))
		}
		versions[parts[0]] = parts[2] // snapshot-version
	}
	return versions, nil
}

// buildConfig converts a GenerationConfig to a Librarian Config.
func buildConfig(gen *GenerationConfig, repoPath string, src *config.Source, versions map[string]string) *config.Config {
	var libs []*config.Library
	if v, ok := versions["google-cloud-java"]; ok {
		libs = append(libs, &config.Library{
			Name:         "google-cloud-java",
			Version:      v,
			SkipGenerate: true,
		})
	}
	for _, l := range gen.Libraries {
		name := l.LibraryName
		if name == "" {
			name = l.APIShortName
		}
		output := "java-" + name
		artifactID := parseArtifactID(l.DistributionName, name)
		version := versions[artifactID]
		var apis []*config.API
		var javaAPIs []*config.JavaAPI
		for _, g := range l.GAPICs {
			if g.ProtoPath == "" {
				continue
			}
			apis = append(apis, &config.API{Path: g.ProtoPath})

			info, err := parseJavaBazel(src.Dir, g.ProtoPath)
			if err != nil {
				log.Printf("Warning: failed to parse BUILD.bazel for %s: %v", g.ProtoPath, err)
				continue
			}
			if info == nil {
				continue
			}
			javaAPI := &config.JavaAPI{
				Path:             g.ProtoPath,
				AdditionalProtos: info.AdditionalProtos,
				NoSamples:        info.NoSamples,
			}
			javaAPIs = append(javaAPIs, javaAPI)
		}
		libs = append(libs, &config.Library{
			Name:    name,
			Version: version,
			Keep:    parseOwlBotKeep(repoPath, output),
			APIs:    apis,
			Java: &config.JavaModule{
				APIIDOverride:                l.APIID,
				APIReference:                 l.APIReference,
				APIDescriptionOverride:       l.APIDescription,
				ClientDocumentationOverride:  l.ClientDocumentation,
				NonCloudAPI:                  invertBoolPtr(l.CloudAPI),
				CodeownerTeam:                l.CodeownerTeam,
				DistributionNameOverride:     l.DistributionName,
				ExcludedDependencies:         l.ExcludedDependencies,
				ExcludedPoms:                 l.ExcludedPoms,
				ExtraVersionedModules:        l.ExtraVersionedModules,
				JavaAPIs:                     javaAPIs,
				GroupID:                      l.GroupID,
				IssueTrackerOverride:         l.IssueTracker,
				LibraryTypeOverride:          l.LibraryType,
				MinJavaVersion:               l.MinJavaVersion,
				NamePrettyOverride:           l.NamePretty,
				ProductDocumentationOverride: l.ProductDocumentation,
				RecommendedPackage:           l.RecommendedPackage,
				BillingNotRequired:           invertBoolPtr(l.RequiresBilling),
				RestDocumentation:            l.RestDocumentation,
				RpcDocumentation:             l.RpcDocumentation,
			},
		})
	}
	if len(libs) == 0 {
		return nil
	}
	return &config.Config{
		Language: "java",
		Default: &config.Default{
			Java: &config.JavaModule{
				LibrariesBomVersion: gen.LibrariesBomVersion,
			},
		},
		Sources: &config.Sources{
			Googleapis: src,
		},
		Libraries: libs,
		Repo:      "googleapis/google-cloud-java",
	}
}

// parseOwlBotKeep parses the .OwlBot-hermetic.yaml file for the given library
// and extracts additional deep-preserve-regex patterns into a list of paths
// to be preserved during generation. It filters out the standard template
// patterns and ensures the paths are relative to the library's output directory.
// It assumes the regex is actually a file or dir path.
func parseOwlBotKeep(repoPath, outputDir string) []string {
	path := filepath.Join(repoPath, outputDir, ".OwlBot-hermetic.yaml")
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	content, err := yaml.Read[struct {
		DeepPreserveRegex []string `yaml:"deep-preserve-regex"`
	}](path)
	if err != nil {
		log.Printf("Warning: failed to parse %s: %v", path, err)
		return nil
	}
	var keeps []string
	prefix := "/" + outputDir + "/"
	for _, regex := range content.DeepPreserveRegex {
		// Ignore standard template pattern:
		// "/java-library-name/google-.*/src/test/java/com/google/cloud/.*/v.*/it/IT.*Test.java"
		if strings.HasPrefix(regex, prefix) && strings.HasSuffix(regex, "/src/test/java/com/google/cloud/.*/v.*/it/IT.*Test.java") {
			continue
		}
		keeps = append(keeps, strings.TrimPrefix(regex, prefix))
	}
	return keeps
}

// parseArtifactID returns the Maven artifact ID from distributionName (groupId:artifactId)
// or name. If distributionName is empty, it returns "google-cloud-" + name.
func parseArtifactID(distributionName, name string) string {
	artifactID := distributionName
	if artifactID == "" {
		artifactID = "google-cloud-" + name
	}
	if i := strings.Index(artifactID, ":"); i != -1 {
		artifactID = artifactID[i+1:]
	}
	return artifactID
}

func invertBoolPtr(p *bool) bool {
	return p != nil && !*p
}

// insertMarkers updates the pom.xml of the main client module for each library
// to include managed dependency markers for generated proto and gRPC dependencies.
func insertMarkers(repoPath string, cfg *config.Config) error {
	var totalInserts int
	for _, lib := range cfg.Libraries {
		if lib.SkipGenerate {
			log.Printf("Debug: skipping library %s (SkipGenerate=true)", lib.Name)
			continue
		}
		distName := java.DeriveDistributionName(lib)
		parts := strings.SplitN(distName, ":", 2)
		if len(parts) != 2 {
			log.Printf("Debug: skipping library %s (invalid distribution name: %s)", lib.Name, distName)
			continue
		}
		gapicArtifactID := parts[1]
		clientPomPath := filepath.Join(repoPath, "java-"+lib.Name, gapicArtifactID, "pom.xml")
		if _, err := os.Stat(clientPomPath); err != nil {
			log.Printf("Warning: pom.xml not found for library %s at %s", lib.Name, clientPomPath)
			continue
		}

		contentBytes, err := os.ReadFile(clientPomPath)
		if err != nil {
			return err
		}
		lines := strings.Split(string(contentBytes), "\n")

		protoIDs, grpcIDs := getModuleArtifactIDs(lib)
		if len(protoIDs) == 0 && len(grpcIDs) == 0 {
			log.Printf("Debug: skipping library %s (no APIs found to wrap)", lib.Name)
			continue
		}

		updatedLines := wrapDependencies(lines, protoIDs, managedProtoStart, managedProtoEnd, lib.Name, "proto")
		updatedLines = wrapDependencies(updatedLines, grpcIDs, managedGrpcStart, managedGrpcEnd, lib.Name, "grpc")

		newContent := strings.Join(updatedLines, "\n")
		if newContent != string(contentBytes) {
			if err := os.WriteFile(clientPomPath, []byte(newContent), 0644); err != nil {
				return err
			}
			totalInserts++
		} else {
			log.Printf("Debug: no changes needed for library %s (markers may already exist or dependencies not found)", lib.Name)
		}
	}
	if totalInserts > 0 {
		log.Printf("Inserted markers in %d Java pom.xml files", totalInserts)
	}
	return nil
}

// getModuleArtifactIDs returns the proto and gRPC artifact IDs for all APIs in the library.
func getModuleArtifactIDs(lib *config.Library) (protoIDs, grpcIDs []string) {
	for _, api := range lib.APIs {
		version := serviceconfig.ExtractVersion(api.Path)
		names := java.DeriveModuleNames(lib.Name, version)
		protoIDs = append(protoIDs, names.Proto)
		grpcIDs = append(grpcIDs, names.Grpc)
	}
	return
}

// wrapDependencies inserts start and end markers around the block of dependencies
// matching the provided artifact IDs. It returns the modified lines.
func wrapDependencies(lines []string, artifactIDs []string, startMarker, endMarker string, libName, depType string) []string {
	if len(artifactIDs) == 0 {
		return lines
	}
	if slices.Contains(lines, startMarker) {
		log.Printf("Debug: library %s already has %s markers", libName, depType)
		return lines
	}

	targets := make([]string, len(artifactIDs))
	for i, id := range artifactIDs {
		targets[i] = "<artifactId>" + id + "</artifactId>"
	}

	first, last := findMarkerBounds(lines, targets)
	if first == -1 {
		log.Printf("Debug: library %s: none of the %s artifact IDs %v found in pom.xml", libName, depType, artifactIDs)
		return lines
	}

	// Use the same indentation as the first dependency line.
	indent := lines[first][:len(lines[first])-len(strings.TrimLeft(lines[first], " \t"))]
	return slices.Concat(
		lines[:first],
		[]string{indent + startMarker},
		lines[first:last+1],
		[]string{indent + endMarker},
		lines[last+1:],
	)
}

// findMarkerBounds returns the starting line of the first <dependency> block
// and the ending line of the last <dependency> block that contain any of the target artifact IDs.
func findMarkerBounds(lines []string, targets []string) (first, last int) {
	first, last = -1, -1
	for i, line := range lines {
		match := false
		for _, t := range targets {
			if strings.Contains(line, t) {
				match = true
				break
			}
		}
		if !match {
			continue
		}

		// Find the start of this <dependency> block
		start := i
		for start > 0 && !strings.Contains(lines[start], "<dependency>") {
			start--
		}
		if first == -1 || start < first {
			first = start
		}

		// Find the end of this <dependency> block
		end := i
		for end < len(lines) && !strings.Contains(lines[end], "</dependency>") {
			end++
		}
		if end > last {
			last = end
		}
	}
	return
}
