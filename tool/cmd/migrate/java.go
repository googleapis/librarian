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
	cfg := buildConfig(gen)
	if cfg == nil {
		return fmt.Errorf("no libraries found to migrate")
	}
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
func buildConfig(gen *GenerationConfig) *config.Config {
	var libs []*config.Library
	for _, l := range gen.Libraries {
		name := l.LibraryName
		if name == "" {
			name = l.APIShortName
		}
		var apis []*config.API
		for _, g := range l.GAPICs {
			if g.ProtoPath != "" {
				apis = append(apis, &config.API{Path: g.ProtoPath})
			}
		}
		libs = append(libs, &config.Library{
			Name:         name,
			Output:       "java-" + name,
			APIs:         apis,
			ReleaseLevel: l.ReleaseLevel,
			Transport:    l.Transport,
			Java: &config.JavaModule{
				APIID:                        l.APIID,
				APIReference:                 l.APIReference,
				APIDescriptionOverride:       l.APIDescription,
				ClientDocumentation:          l.ClientDocumentation,
				NonCloudAPI:                  invertBoolPtr(l.CloudAPI),
				CodeownerTeam:                l.CodeownerTeam,
				DistributionName:             l.DistributionName,
				ExcludedDependencies:         l.ExcludedDependencies,
				ExcludedPoms:                 l.ExcludedPoms,
				ExtraVersionedModules:        l.ExtraVersionedModules,
				GroupID:                      l.GroupID,
				IssueTracker:                 l.IssueTracker,
				LibraryType:                  l.LibraryType,
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
			// hardcoded for local testing
			Googleapis: &config.Source{Dir: "../../googleapis"},
		},
		Libraries: libs,
	}
}

func invertBoolPtr(p *bool) bool {
	return p != nil && !*p
}
