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
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/googleapis/librarian/tool/cmd/importconfigs/bazel"
	"github.com/urfave/cli/v3"
)

var langToConstant = map[string]string{
	"all":    "LangAll",
	"csharp": "LangCsharp",
	"go":     "LangGo",
	"java":   "LangJava",
	"nodejs": "LangNodejs",
	"php":    "LangPhp",
	"python": "LangPython",
	"ruby":   "LangRuby",
}

func updateRestNumericEnumsCommand() *cli.Command {
	return &cli.Command{
		Name:  "update-rest-numeric-enums",
		Usage: "update rest numeric enums values in internal/serviceconfig/api.go from BUILD.bazel files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "googleapis",
				Usage:    "path to googleapis dir",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			googleapisDir := cmd.String("googleapis")
			return runUpdateRestNumericEnums("internal/serviceconfig/sdk.yaml", googleapisDir)
		},
	}
}

func runUpdateRestNumericEnums(sdkYaml, googleapisDir string) error {
	apis, err := yaml.Read[[]*serviceconfig.API](sdkYaml)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", sdkYaml, err)
	}
	apiMap := toMap(*apis)
	buildFilePaths, err := findBuild(googleapisDir)
	if err != nil {
		return err
	}
	var newAPIs []*serviceconfig.API
	for _, path := range buildFilePaths {
		noRESTNumericEnums := readRestNumericEnums(googleapisDir, path)
		if len(noRESTNumericEnums) == 0 {
			// No need to change the default value.
			continue
		}
		buildDir := filepath.Dir(path)
		api, ok := apiMap[buildDir]
		if ok {
			// Add the NoRESTNumericEnums to an existing API, regardless a cloud API or not.
			api.NoRESTNumericEnums = noRESTNumericEnums
			continue
		}

		// Ignore a non-cloud API that is not in sdk.yaml since it is blocked.
		if !strings.HasPrefix(buildDir, "google/cloud") {
			continue
		}
		// Add the NoRESTNumericEnums to a cloud API.
		newAPIs = append(newAPIs, &serviceconfig.API{
			// Add languages so they won't be blocked.
			Languages: []string{
				config.LanguageDart,
				config.LanguageGo,
				config.LanguageJava,
				config.LanguagePython,
				config.LanguageRust,
			},
			Path:               buildDir,
			NoRESTNumericEnums: noRESTNumericEnums,
		})
	}
	finalAPIs := toSlice(apiMap)
	finalAPIs = append(finalAPIs, newAPIs...)
	sort.Slice(finalAPIs, func(i, j int) bool {
		return finalAPIs[i].Path < finalAPIs[j].Path
	})
	return yaml.Write(sdkYaml, finalAPIs)
}

func findBuild(googleapisDir string) ([]string, error) {
	var res []string
	err := filepath.WalkDir(filepath.Join(googleapisDir, "google"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "BUILD.bazel" {
			return nil
		}

		res = append(res, strings.TrimPrefix(path, googleapisDir+"/"))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func toMap(apis []*serviceconfig.API) map[string]*serviceconfig.API {
	res := make(map[string]*serviceconfig.API)
	for _, api := range apis {
		res[api.Path] = api
	}
	return res
}

func toSlice(apis map[string]*serviceconfig.API) []*serviceconfig.API {
	var res []*serviceconfig.API
	for _, api := range apis {
		res = append(res, api)
	}
	return res
}

func readRestNumericEnums(googleapisDir, path string) map[string]bool {
	buildPath := filepath.Join(googleapisDir, path)
	if _, err := os.Stat(buildPath); os.IsNotExist(err) {
		return nil
	}
	numericEnums, err := bazel.ParseRESTNumericEnums(buildPath)
	if err != nil {
		slog.Warn("failed to parse rest numeric enums", "path", buildPath, "error", err)
		return nil
	}
	return simplifyRESTNumericEnums(numericEnums)
}

func simplifyRESTNumericEnums(restNumericEnums map[string]bool) map[string]bool {
	for lang := range langToConstant {
		if lang == "all" {
			continue
		}
		_, ok := restNumericEnums[lang]
		if !ok {
			// At least one language is not present,
			// return as-is.
			return restNumericEnums
		}
	}
	// All languages do not need rest_numeric_enums.
	// Return all: true.
	return map[string]bool{"all": true}
}
