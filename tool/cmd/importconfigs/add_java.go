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
	"os"
	"sort"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

type LibrarianConfig struct {
	Libraries []struct {
		Apis []struct {
			Path string `yaml:"path"`
		} `yaml:"apis"`
	} `yaml:"libraries"`
}

func addJavaCommand() *cli.Command {
	return &cli.Command{
		Name:  "add-java",
		Usage: "adds java to languages in sdk.yaml based on google-cloud-java/librarian.yaml",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "librarian-yaml",
				Usage:   "path to google-cloud-java/librarian.yaml",
				Value:   "../google-cloud-java/librarian.yaml",
				Aliases: []string{"l"},
			},
			&cli.StringFlag{
				Name:    "sdk-yaml",
				Usage:   "path to librarian/internal/serviceconfig/sdk.yaml",
				Value:   "internal/serviceconfig/sdk.yaml",
				Aliases: []string{"s"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			libPath := cmd.String("librarian-yaml")
			sdkPath := cmd.String("sdk-yaml")

			// 1. Load the Java-specific librarian.yaml to find which APIs are being tracked for Java.
			libData, err := os.ReadFile(libPath)
			if err != nil {
				return err
			}

			libCfg, err := yaml.Unmarshal[LibrarianConfig](libData)
			if err != nil {
				return err
			}

			// Map for O(1) lookup of paths that should support Java.
			javaPaths := make(map[string]bool)
			for _, lib := range libCfg.Libraries {
				for _, api := range lib.Apis {
					if api.Path != "" {
						javaPaths[api.Path] = true
					}
				}
			}

			// 2. Read the central SDK configuration (the allowlist).
			apis, err := yaml.Read[[]serviceconfig.API](sdkPath)
			if err != nil {
				return err
			}

			// 3. Update the allowlist entries.
			updatedCount := 0
			for i := range *apis {
				api := &(*apis)[i]
				
				// Only process if this path is found in the Java librarian.yaml.
				if !javaPaths[api.Path] {
					continue
				}
				
				// Mandate: Only add Java if the "languages" list is already explicitly defined (not empty).
				// If empty, the API is implicitly allowed for all languages.
				if len(api.Languages) == 0 {
					continue
				}

				// Check if "java" is already present in the list.
				hasJava := false
				for _, l := range api.Languages {
					if l == "java" {
						hasJava = true
						break
					}
				}

				// Add "java" and sort the list to maintain stable formatting.
				if !hasJava {
					api.Languages = append(api.Languages, "java")
					sort.Strings(api.Languages)
					updatedCount++
					fmt.Printf("Added java to path: %s\n", api.Path)
				}
			}

			if updatedCount == 0 {
				fmt.Println("No new paths added.")
				return nil
			}

			// 4. Save the updated allowlist back to disk.
			// This uses the internal/yaml package which handles formatting and copyright headers.
			if err := yaml.Write(sdkPath, *apis); err != nil {
				return err
			}

			fmt.Printf("Updated %d entries.\n", updatedCount)
			return nil
		},
	}
}
