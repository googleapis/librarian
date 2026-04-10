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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/librarian/java"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	generationConfigFileName = "generation_config.yaml"
	managedProtoStartMarker  = "<!-- {x-generated-proto-dependencies-start} -->"
	managedProtoEndMarker    = "<!-- {x-generated-proto-dependencies-end} -->"
	managedGrpcStartMarker   = "<!-- {x-generated-grpc-dependencies-start} -->"
	managedGrpcEndMarker     = "<!-- {x-generated-grpc-dependencies-end} -->"

	managedDepsStartMarker = "<!-- {x-generated-dependencies-start} -->"
	managedDepsEndMarker   = "<!-- {x-generated-dependencies-end} -->"

	managedModulesStartMarker = "<!-- {x-generated-modules-start} -->"
	managedModulesEndMarker   = "<!-- {x-generated-modules-end} -->"
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
				ExcludedPOMs:                 l.ExcludedPoms,
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
				LibrariesBOMVersion: gen.LibrariesBomVersion,
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

// insertMarkers updates pom.xml files for each library to include managed markers.
func insertMarkers(repoPath string, cfg *config.Config) error {
	var clientCount, parentCount, bomCount int
	for _, lib := range cfg.Libraries {
		if lib.SkipGenerate {
			log.Printf("Debug: skipping library %s (SkipGenerate is true)", lib.Name)
			continue
		}
		libDir := filepath.Join(repoPath, "java-"+lib.Name)
		ids := getModuleArtifactIDs(lib)
		// 1. Client module pom.xml
		clientPOMPath := filepath.Join(libDir, ids.Client, "pom.xml")
		if updated, err := updatePOMMarkers(clientPOMPath, ids, "client"); err == nil {
			if updated {
				clientCount++
			}
		} else if os.IsNotExist(err) {
			log.Printf("Debug: skipping library %s (client pom.xml not found at %s)", lib.Name, clientPOMPath)
		} else {
			return err
		}
		// 2. Parent pom.xml
		parentPOMPath := filepath.Join(libDir, "pom.xml")
		if updated, err := updatePOMMarkers(parentPOMPath, ids, "parent"); err == nil {
			if updated {
				parentCount++
			}
		} else if os.IsNotExist(err) {
			log.Printf("Debug: skipping library %s (parent pom.xml not found at %s)", lib.Name, parentPOMPath)
		} else {
			return err
		}
		// 3. BOM pom.xml
		bomPOMPath := filepath.Join(libDir, ids.BOM, "pom.xml")
		if updated, err := updatePOMMarkers(bomPOMPath, ids, "bom"); err == nil {
			if updated {
				bomCount++
			}
		} else if os.IsNotExist(err) {
			log.Printf("Debug: skipping library %s (BOM pom.xml not found at %s)", lib.Name, bomPOMPath)
		} else {
			return err
		}
	}

	if clientCount > 0 {
		log.Printf("Inserted markers in %d Java client pom.xml files", clientCount)
	}
	if parentCount > 0 {
		log.Printf("Inserted markers in %d Java parent pom.xml files", parentCount)
	}
	if bomCount > 0 {
		log.Printf("Inserted markers in %d Java BOM pom.xml files", bomCount)
	}
	return nil
}

func updatePOMMarkers(pomPath string, ids moduleArtifactIDs, pomType string) (bool, error) {
	contentBytes, err := os.ReadFile(pomPath)
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(contentBytes), "\n")
	origContent := string(contentBytes)
	switch pomType {
	case "client":
		lines = wrapBlocks(wrapArgs{
			lines:       lines,
			targets:     toArtifactTags(ids.Protos),
			startMarker: managedProtoStartMarker,
			endMarker:   managedProtoEndMarker,
			startTag:    "<dependency>",
			endTag:      "</dependency>",
		})
		lines = wrapBlocks(wrapArgs{
			lines:       lines,
			targets:     toArtifactTags(ids.GRPCs),
			startMarker: managedGrpcStartMarker,
			endMarker:   managedGrpcEndMarker,
			startTag:    "<dependency>",
			endTag:      "</dependency>",
		})
	case "parent":
		// Dependency Management
		allDeps := append([]string{ids.Client, ids.BOM}, ids.Protos...)
		allDeps = append(allDeps, ids.GRPCs...)
		lines = wrapBlocks(wrapArgs{
			lines:       lines,
			targets:     toArtifactTags(allDeps),
			startMarker: managedDepsStartMarker,
			endMarker:   managedDepsEndMarker,
			startTag:    "<dependency>",
			endTag:      "</dependency>",
		})
		// Modules
		allModules := append([]string{ids.Client}, ids.Protos...)
		allModules = append(allModules, ids.GRPCs...)
		lines = wrapBlocks(wrapArgs{
			lines:       lines,
			targets:     toModuleTags(allModules),
			startMarker: managedModulesStartMarker,
			endMarker:   managedModulesEndMarker,
			startTag:    "<module>",
			endTag:      "</module>",
		})
	case "bom":
		allDeps := append([]string{ids.Client}, ids.Protos...)
		allDeps = append(allDeps, ids.GRPCs...)
		lines = wrapBlocks(wrapArgs{
			lines:       lines,
			targets:     toArtifactTags(allDeps),
			startMarker: managedDepsStartMarker,
			endMarker:   managedDepsEndMarker,
			startTag:    "<dependency>",
			endTag:      "</dependency>",
		})
	}

	newContent := strings.Join(lines, "\n")
	if newContent == origContent {
		log.Printf("Debug: no changes made to %s pom: %s (no matching targets found)", pomType, pomPath)
		return false, nil
	}

	if err := os.WriteFile(pomPath, []byte(newContent), 0644); err != nil {
		return false, err
	}
	return true, nil
}

type moduleArtifactIDs struct {
	Client string
	BOM    string
	Protos []string
	GRPCs  []string
}

// getModuleArtifactIDs returns the proto and gRPC artifact IDs for all APIs in the library.
func getModuleArtifactIDs(lib *config.Library) moduleArtifactIDs {
	lc := java.DeriveLibraryCoordinates(lib)
	ids := moduleArtifactIDs{
		Client: lc.GAPIC.ArtifactID,
		BOM:    lc.BOM.ArtifactID,
	}
	for _, api := range lib.APIs {
		version := serviceconfig.ExtractVersion(api.Path)
		// Find Java-specific API config to handle artifact ID overrides.
		javaAPI := java.ResolveJavaAPI(lib, api)
		apiCoord := java.DeriveAPICoordinates(lc, version, javaAPI)
		ids.Protos = append(ids.Protos, apiCoord.Proto.ArtifactID)
		ids.GRPCs = append(ids.GRPCs, apiCoord.GRPC.ArtifactID)
	}
	return ids
}

type wrapArgs struct {
	lines       []string
	targets     []string
	startMarker string
	endMarker   string
	startTag    string
	endTag      string
}

// wrapBlocks inserts start and end markers around a set of matching blocks.
// If matching blocks are not contiguous, it moves them together to the
// position of the first matching block.
func wrapBlocks(args wrapArgs) []string {
	if len(args.targets) == 0 {
		return args.lines
	}
	kept, moved, insertAt := splitMatchingBlocks(args)
	if insertAt == -1 {
		return args.lines
	}

	indent := getLineIndent(moved[0])

	res := make([]string, 0, len(args.lines)+2)
	res = append(res, kept[:insertAt]...)
	res = append(res, indent+args.startMarker)
	res = append(res, moved...)
	res = append(res, indent+args.endMarker)
	res = append(res, kept[insertAt:]...)
	return res
}

// toArtifactTags converts artifact IDs into Maven <artifactId> tags.
func toArtifactTags(ids []string) []string {
	tags := make([]string, 0, len(ids))
	for _, id := range ids {
		tags = append(tags, "<artifactId>"+id+"</artifactId>")
	}
	return tags
}

// toModuleTags converts artifact IDs into Maven <module> tags.
func toModuleTags(ids []string) []string {
	tags := make([]string, 0, len(ids))
	for _, id := range ids {
		tags = append(tags, "<module>"+id+"</module>")
	}
	return tags
}

// splitMatchingBlocks partitions POM lines into 'kept' and 'moved' slices.
// 'moved' contains all blocks matching any target.
// 'kept' contains all other lines in their original relative order.
// 'insertAt' is the index in 'kept' where the first matching block was originally located,
// serving as the insertion point for the relocated blocks.
func splitMatchingBlocks(args wrapArgs) (kept, moved []string, insertAt int) {
	insertAt = -1
	for i := 0; i < len(args.lines); i++ {
		if !strings.Contains(args.lines[i], args.startTag) {
			kept = append(kept, args.lines[i])
			continue
		}

		block, nextIdx := nextBlock(args.lines, i, args.endTag)
		if containsAny(block, args.targets) {
			if insertAt == -1 {
				insertAt = len(kept)
			}
			moved = append(moved, block...)
		} else {
			kept = append(kept, block...)
		}
		i = nextIdx
	}
	return
}

// nextBlock returns the full block starting at index i and ending with endTag.
func nextBlock(lines []string, i int, endTag string) (block []string, endIdx int) {
	start := i
	for i < len(lines) && !strings.Contains(lines[i], endTag) {
		i++
	}
	if i >= len(lines) { // Malformed XML
		return lines[start:], len(lines) - 1
	}
	return lines[start : i+1], i
}

// containsAny returns true if any line in the block contains any of the target strings.
func containsAny(block, targets []string) bool {
	for _, line := range block {
		for _, t := range targets {
			if strings.Contains(line, t) {
				return true
			}
		}
	}
	return false
}

// getLineIndent returns the leading whitespace of a line.
func getLineIndent(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	return line[:len(line)-len(trimmed)]
}
