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
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/bazelbuild/buildtools/build"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func main() {
	googleapisDir := flag.String("googleapis", "../googleapis", "Path to googleapis directory")
	sdkYamlPath := flag.String("sdk-yaml", "librarian/internal/serviceconfig/sdk.yaml", "Path to sdk.yaml")
	flag.Parse()

	apis, err := yaml.Read[[]serviceconfig.API](*sdkYamlPath)
	if err != nil {
		log.Fatalf("failed to read sdk.yaml: %v", err)
	}

	changed := false
	for i := range *apis {
		api := &(*apis)[i]
		if !isJavaAPI(api) {
			continue
		}

		info, err := parseJavaBazel(*googleapisDir, api.Path)
		if err != nil {
			log.Printf("Warning: failed to check transport in %s: %v", api.Path, err)
			continue
		}

		if info != nil {
			currentTransport := api.Transport("java")
			if info.Transport != "" {
				bazelTransport := serviceconfig.Transport(info.Transport)
				if currentTransport != bazelTransport {
					fmt.Printf("Updating %s transport: sdk.yaml effective is %q, Bazel has %q\n", api.Path, currentTransport, bazelTransport)
					if api.Transports == nil {
						api.Transports = make(map[string]serviceconfig.Transport)
					}
					api.Transports["java"] = bazelTransport
					changed = true
				}
			} else {
				fmt.Printf("Debug: %s has java_gapic_library but no transport attribute in BUILD.bazel\n", api.Path)
				if api.Transports == nil {
					api.Transports = make(map[string]serviceconfig.Transport)
				}
				if _, ok := api.Transports["java"]; !ok {
					api.Transports["java"] = serviceconfig.GRPC
					fmt.Printf("Assigned java: grpc to %s in sdk.yaml\n", api.Path)
					changed = true
				}
			}
		}
	}

	if changed {
		if err := yaml.Write(*sdkYamlPath, *apis); err != nil {
			log.Fatalf("failed to write sdk.yaml: %v", err)
		}
		fmt.Println("Successfully updated sdk.yaml")
	} else {
		fmt.Println("No changes needed in sdk.yaml")
	}
}

func isJavaAPI(api *serviceconfig.API) bool {
	return slices.Contains(api.Languages, "java") || slices.Contains(api.Languages, "all")
}

type javaGAPICInfo struct {
	Transport string
}

// parseJavaBazel replicates the logic from librarian/tool/cmd/migrate/java.go.
func parseJavaBazel(googleapisDir, dir string) (*javaGAPICInfo, error) {
	file, err := parseBazel(googleapisDir, dir)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}

	rules := file.Rules("java_gapic_library")
	if len(rules) == 0 {
		return nil, nil
	}

	info := &javaGAPICInfo{}
	rule := rules[0]
	if attr := rule.Attr("transport"); attr != nil {
		info.Transport = rule.AttrString("transport")
	}
	return info, nil
}

func parseBazel(googleapisDir, dir string) (*build.File, error) {
	path := filepath.Join(googleapisDir, dir, "BUILD.bazel")
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file, err := build.ParseBuild(path, data)
	if err != nil {
		return nil, err
	}
	return file, nil
}
