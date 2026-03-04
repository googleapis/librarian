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
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	generationConfigFileName = "generation_config.yaml"
)

type javaGAPICInfo struct {
	NoRestNumericEnums bool
	NoSamples          bool
	AdditionalProtos   []string
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
		rule := rules[0]
		info.NoRestNumericEnums = rule.AttrLiteral("rest_numeric_enums") == "False"
	}
	// 2. From java_gapic_assembly_gradle_pkg
	if rules := file.Rules("java_gapic_assembly_gradle_pkg"); len(rules) > 0 {
		rule := rules[0]
		info.NoSamples = rule.AttrLiteral("include_samples") == "False"
	}
	// 3. From proto_library_with_info
	if rules := file.Rules("proto_library_with_info"); len(rules) > 0 {
		rule := rules[0]
		// Search for specific common resource targets in deps
		if deps := rule.AttrStrings("deps"); len(deps) > 0 {
			for _, dep := range deps {
				switch dep {
				case "//google/cloud:common_resources_proto":
					info.AdditionalProtos = append(info.AdditionalProtos, "google/cloud/common_resources.proto")
				case "//google/cloud/location:location_proto":
					info.AdditionalProtos = append(info.AdditionalProtos, "google/cloud/location/locations.proto")
				case "//google/iam/v1:iam_policy_proto":
					info.AdditionalProtos = append(info.AdditionalProtos, "google/iam/v1/iam_policy.proto")
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
	Libraries []LibraryConfig `yaml:"libraries"`
}

func runJavaMigration(ctx context.Context, repoPath string) error {
	gen, err := readGenerationConfig(repoPath)
	if err != nil {
		return err
	}
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}
	cfg := buildConfig(gen, src.Dir)
	if cfg == nil {
		return fmt.Errorf("no libraries found to migrate")
	}
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""
	if err := librarian.RunTidyOnConfig(ctx, cfg); err != nil {
		return errTidyFailed
	}
	log.Printf("Successfully migrated %d Java libraries", len(cfg.Libraries))
	return nil
}

func readGenerationConfig(path string) (*GenerationConfig, error) {
	return yaml.Read[GenerationConfig](filepath.Join(path, generationConfigFileName))
}

// buildConfig converts a GenerationConfig to a Librarian Config.
func buildConfig(gen *GenerationConfig, googleapisDir string) *config.Config {
	var libs []*config.Library
	for _, l := range gen.Libraries {
		name := l.LibraryName
		if name == "" {
			name = l.APIShortName
		}
		var apis []*config.API
		var javaAPIs []*config.JavaAPI
		for _, g := range l.GAPICs {
			if g.ProtoPath == "" {
				continue
			}
			apis = append(apis, &config.API{Path: g.ProtoPath})

			info, err := parseJavaBazel(googleapisDir, g.ProtoPath)
			if err != nil {
				log.Printf("Warning: failed to parse BUILD.bazel for %s: %v", g.ProtoPath, err)
				continue
			}
			if info == nil {
				continue
			}
			javaAPI := &config.JavaAPI{
				Path:               g.ProtoPath,
				NoRestNumericEnums: info.NoRestNumericEnums,
				AdditionalProtos:   info.AdditionalProtos,
				NoSamples:          info.NoSamples,
			}
			javaAPIs = append(javaAPIs, javaAPI)
		}
		libs = append(libs, &config.Library{
			Name:         name,
			Output:       "java-" + name,
			APIs:         apis,
			ReleaseLevel: l.ReleaseLevel,
			Transport:    l.Transport,
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
		Default:  &config.Default{},
		Sources: &config.Sources{
			Googleapis: &config.Source{Dir: googleapisDir},
		},
		Libraries: libs,
	}
}

func invertBoolPtr(p *bool) bool {
	return p != nil && !*p
}
