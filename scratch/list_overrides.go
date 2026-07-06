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
	"os"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func main() {
	sdkYAMLPath := "internal/serviceconfig/sdk.yaml"
	apis, err := yaml.Read[[]serviceconfig.API](sdkYAMLPath)
	if err != nil {
		log.Fatalf("failed to read sdk.yaml: %v", err)
	}

	outputPath := "scratch/kept_java_overrides.txt"
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "Kept (necessary) Java Release Level Overrides:")
	fmt.Fprintln(file, "=============================================")

	count := 0
	for _, api := range *apis {
		if api.ReleaseLevels == nil {
			continue
		}
		if rl, ok := api.ReleaseLevels["java"]; ok {
			line := fmt.Sprintf("- Path: %s\n  Release Level: %s\n", api.Path, rl)
			fmt.Print(line)
			fmt.Fprint(file, line)
			count++
		}
	}

	fmt.Printf("\nSaved %d overrides to %s\n", count, outputPath)
}
