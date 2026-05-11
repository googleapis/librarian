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

// Package main provides a standalone tool to migrate Go configuration in librarian.yaml.
package main

import (
	"flag"
	"fmt"
	"log"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func main() {
	file := flag.String("file", "librarian.yaml", "Path to librarian.yaml")
	flag.Parse()

	cfg, err := yaml.Read[config.Config](*file)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	for _, lib := range cfg.Libraries {
		if lib.Go == nil || len(lib.Go.GoAPIs) == 0 {
			continue
		}

		commonFeatures := []string{"F_open_telemetry_attributes", "F_proto_cloneof"}
		hasCommon := false
		for _, goAPI := range lib.Go.GoAPIs {
			if slices.Equal(goAPI.EnabledGeneratorFeatures, commonFeatures) {
				hasCommon = true
				break
			}
		}

		if hasCommon {
			lib.Go.DefaultEnabledGeneratorFeatures = commonFeatures
		}

		for _, api := range lib.APIs {
			for _, goAPI := range lib.Go.GoAPIs {
				if goAPI.Path == api.Path {
					goAPICopy := *goAPI
					goAPICopy.Path = ""
					if hasCommon && slices.Equal(goAPICopy.EnabledGeneratorFeatures, commonFeatures) {
						goAPICopy.EnabledGeneratorFeatures = nil
					}
					if isEmptyAPI(&goAPICopy) {
						api.Go = nil
					} else {
						api.Go = &goAPICopy
					}
					break
				}
			}
		}
		lib.Go.GoAPIs = nil
		if len(lib.Go.DefaultEnabledGeneratorFeatures) == 0 &&
			len(lib.Go.DeleteGenerationOutputPaths) == 0 &&
			lib.Go.ModulePathVersion == "" &&
			lib.Go.NestedModule == "" {
			lib.Go = nil
		}
	}

	if err := yaml.Write(*file, cfg); err != nil {
		log.Fatalf("Failed to write config: %v", err)
	}

	fmt.Printf("Successfully migrated %s\n", *file)
}

func isEmptyAPI(goAPI *config.GoAPI) bool {
	return goAPI.ClientPackage == "" &&
		!goAPI.DIREGAPIC &&
		len(goAPI.EnabledGeneratorFeatures) == 0 &&
		goAPI.ImportPath == "" &&
		len(goAPI.NestedProtos) == 0 &&
		!goAPI.NoMetadata &&
		!goAPI.NoSnippets &&
		!goAPI.ProtoOnly &&
		goAPI.ProtoPackage == ""
}
