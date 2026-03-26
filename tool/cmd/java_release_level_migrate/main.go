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
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

// GAPICConfig represents the GAPIC configuration in generation_config.yaml.
type GAPICConfig struct {
	ProtoPath string `yaml:"proto_path"`
}

// LibraryConfig represents a library entry in generation_config.yaml.
type LibraryConfig struct {
	ReleaseLevel string        `yaml:"release_level"`
	GAPICs       []GAPICConfig `yaml:"GAPICs"`
}

// GenerationConfig represents the root of generation_config.yaml.
type GenerationConfig struct {
	Libraries []LibraryConfig `yaml:"libraries"`
}

const sdkYamlPath = "internal/serviceconfig/sdk.yaml"

func main() {
	javaRepoPath := flag.String("java-repo", "", "Path to google-cloud-java repository")
	flag.Parse()

	if *javaRepoPath == "" {
		log.Fatal("--java-repo is required")
	}

	genConfigPath := filepath.Join(*javaRepoPath, "generation_config.yaml")
	gen, err := yaml.Read[GenerationConfig](genConfigPath)
	if err != nil {
		log.Fatalf("failed to read %s: %v", genConfigPath, err)
	}

	apis, err := yaml.Read[[]*serviceconfig.API](sdkYamlPath)
	if err != nil {
		log.Fatalf("failed to read %s: %v", sdkYamlPath, err)
	}

	apiMap := make(map[string]*serviceconfig.API)
	for _, api := range *apis {
		apiMap[api.Path] = api
	}

	updatedCount := 0
	for _, l := range gen.Libraries {
		level := strings.ToLower(l.ReleaseLevel)
		if level == "" {
			level = "preview"
		}
		for _, g := range l.GAPICs {
			if g.ProtoPath == "" {
				continue
			}
			api, ok := apiMap[g.ProtoPath]
			if !ok {
				continue
			}
			if level == "stable" {
				if api.ReleaseLevels != nil {
					if _, ok := api.ReleaseLevels["java"]; ok {
						delete(api.ReleaseLevels, "java")
						if len(api.ReleaseLevels) == 0 {
							api.ReleaseLevels = nil
						}
						updatedCount++
					}
				}
				continue
			}
			if api.ReleaseLevels == nil {
				api.ReleaseLevels = make(map[string]string)
			}
			if api.ReleaseLevels["java"] != level {
				api.ReleaseLevels["java"] = level
				updatedCount++
			}
		}
	}

	if err := yaml.Write(sdkYamlPath, apis); err != nil {
		log.Fatalf("failed to write %s: %v", sdkYamlPath, err)
	}

	fmt.Printf("Successfully updated %d API entries in %s\n", updatedCount, sdkYamlPath)
}
