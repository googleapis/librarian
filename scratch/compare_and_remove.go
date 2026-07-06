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
	"fmt"
	"log"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func main() {
	// 1. Read java librarian.yaml to map API paths to versions
	javaLibYAMLPath := "/usr/local/google/home/zhumin/repos/testjava/google-cloud-java/librarian.yaml"
	javaConfig, err := yaml.Read[config.Config](javaLibYAMLPath)
	if err != nil {
		log.Fatalf("failed to read java librarian.yaml: %v", err)
	}

	apiToVersion := make(map[string]string)
	for _, lib := range javaConfig.Libraries {
		for _, api := range lib.APIs {
			apiToVersion[api.Path] = lib.Version
		}
	}

	// 2. Read service config sdk.yaml
	sdkYAMLPath := "internal/serviceconfig/sdk.yaml"
	apis, err := yaml.Read[[]serviceconfig.API](sdkYAMLPath)
	if err != nil {
		log.Fatalf("failed to read sdk.yaml: %v", err)
	}

	totalCount := 0
	removedCount := 0
	for i := range *apis {
		api := &(*apis)[i]
		if api.ReleaseLevels == nil {
			continue
		}
		explicitRL, ok := api.ReleaseLevels["java"]
		if !ok {
			continue
		}
		totalCount++

		// Retrieve version for this API path from our map, fail if not found
		version, found := apiToVersion[api.Path]
		if !found {
			log.Fatalf("version not found in google-cloud-java/librarian.yaml for API path: %s", api.Path)
		}

		// Temporarily remove the explicit java release level
		delete(api.ReleaseLevels, "java")

		// Check the derived release level without the explicit setting
		derivedRL := api.ReleaseLevel("java", version)

		if explicitRL == derivedRL {
			fmt.Printf("Removing redundant java release_level for %s: %s (derived: %s, library version: %s)\n", api.Path, explicitRL, derivedRL, version)
			if len(api.ReleaseLevels) == 0 {
				api.ReleaseLevels = nil
			}
			removedCount++
		} else {
			// Restore it
			api.ReleaseLevels["java"] = explicitRL
		}
	}

	// 3. Write updates back if any changes were made
	if removedCount > 0 {
		fmt.Printf("\nStatistics:\n")
		fmt.Printf("Total explicit Java release levels originally: %d\n", totalCount)
		fmt.Printf("Removed (redundant): %d\n", removedCount)
		fmt.Printf("Kept (necessary): %d\n", totalCount-removedCount)
		fmt.Printf("\nWriting updates back to %s...\n", sdkYAMLPath)
		if err := yaml.Write(sdkYAMLPath, *apis); err != nil {
			log.Fatalf("failed to write sdk.yaml: %v", err)
		}
	} else {
		fmt.Println("No redundant java release levels found.")
	}
}
