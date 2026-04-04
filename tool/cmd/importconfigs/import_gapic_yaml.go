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
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config/gapicyaml"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

func importGAPICYAMLCommand() *cli.Command {
	return &cli.Command{
		Name:  "import-gapic-yaml",
		Usage: "import *_gapic.yaml configs from googleapis into sdk.yaml",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "googleapis",
				Usage:    "path to googleapis dir",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			googleapisDir := cmd.String("googleapis")
			return runImportGAPICYAML("internal/serviceconfig/sdk.yaml", googleapisDir)
		},
	}
}

func runImportGAPICYAML(sdkYaml, googleapisDir string) error {
	apis, err := yaml.Read[[]*serviceconfig.API](sdkYaml)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", sdkYaml, err)
	}
	apiMap := toMap(*apis)

	gapicFiles, err := findGAPICYAMLFiles(googleapisDir)
	if err != nil {
		return err
	}

	for _, relPath := range gapicFiles {
		absPath := filepath.Join(googleapisDir, relPath)
		cfg, err := yaml.Read[gapicyaml.Config](absPath)
		if err != nil {
			fmt.Printf("warning: failed to parse %s: %v\n", relPath, err)
			continue
		}

		apiPath := filepath.Dir(relPath)
		api, ok := apiMap[apiPath]
		if !ok {
			fmt.Printf("warning: skipping %s: API path %q not found in sdk.yaml\n", relPath, apiPath)
			continue
		}

		gc := &serviceconfig.GAPICYamlConfig{}
		hasData := false

		if cfg.LanguageSettings.Java != nil {
			if cfg.LanguageSettings.Java.PackageName != "" {
				gc.JavaPackageName = cfg.LanguageSettings.Java.PackageName
				hasData = true
			}
			if len(cfg.LanguageSettings.Java.InterfaceNames) > 0 {
				gc.JavaInterfaceNames = cfg.LanguageSettings.Java.InterfaceNames
				hasData = true
			}
		}

		gapicInterfaces := convertInterfaces(cfg.Interfaces)
		if len(gapicInterfaces) > 0 {
			gc.Interfaces = gapicInterfaces
			hasData = true
		}

		if hasData {
			api.GAPICYaml = gc
		}
	}

	finalAPIs := toSlice(apiMap)
	sort.Slice(finalAPIs, func(i, j int) bool {
		return finalAPIs[i].Path < finalAPIs[j].Path
	})
	return yaml.Write(sdkYaml, finalAPIs)
}

// findGAPICYAMLFiles walks the googleapis directory and returns relative paths
// to all *_gapic.yaml files.
func findGAPICYAMLFiles(googleapisDir string) ([]string, error) {
	var res []string
	err := filepath.WalkDir(filepath.Join(googleapisDir, "google"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_gapic.yaml") {
			rel, err := filepath.Rel(googleapisDir, path)
			if err != nil {
				return err
			}
			res = append(res, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// convertInterfaces converts gapicyaml.Interface values to
// serviceconfig.GAPICInterface values, keeping only interfaces that have
// methods with actual configuration (long-running or batching settings).
func convertInterfaces(ifaces []gapicyaml.Interface) []serviceconfig.GAPICInterface {
	var result []serviceconfig.GAPICInterface
	for _, iface := range ifaces {
		methods := convertMethods(iface.Methods)
		if len(methods) == 0 {
			continue
		}
		result = append(result, serviceconfig.GAPICInterface{
			Name:    iface.Name,
			Methods: methods,
		})
	}
	return result
}

// convertMethods converts gapicyaml.Method values to
// serviceconfig.GAPICMethod values, keeping only methods that have
// long-running or batching configuration.
func convertMethods(methods []gapicyaml.Method) []serviceconfig.GAPICMethod {
	var result []serviceconfig.GAPICMethod
	for _, m := range methods {
		if m.LongRunning == nil && m.Batching == nil {
			continue
		}
		gm := serviceconfig.GAPICMethod{
			Name: m.Name,
		}
		if m.LongRunning != nil {
			gm.LongRunning = &serviceconfig.GAPICLongRunning{
				InitialPollDelayMillis: m.LongRunning.InitialPollDelayMillis,
				PollDelayMultiplier:    m.LongRunning.PollDelayMultiplier,
				MaxPollDelayMillis:     m.LongRunning.MaxPollDelayMillis,
				TotalPollTimeoutMillis: m.LongRunning.TotalPollTimeoutMillis,
			}
		}
		if m.Batching != nil {
			gm.Batching = convertBatching(m.Batching)
		}
		result = append(result, gm)
	}
	return result
}

// convertBatching converts a gapicyaml.Batching to a serviceconfig.GAPICBatching.
func convertBatching(b *gapicyaml.Batching) *serviceconfig.GAPICBatching {
	gb := &serviceconfig.GAPICBatching{}
	if b.Thresholds != nil {
		gb.Thresholds = &serviceconfig.GAPICBatchingThresholds{
			ElementCountThreshold:            b.Thresholds.ElementCountThreshold,
			RequestByteThreshold:             b.Thresholds.RequestByteThreshold,
			DelayThresholdMillis:             b.Thresholds.DelayThresholdMillis,
			FlowControlElementLimit:          b.Thresholds.FlowControlElementLimit,
			FlowControlByteLimit:             b.Thresholds.FlowControlByteLimit,
			FlowControlLimitExceededBehavior: b.Thresholds.FlowControlLimitExceededBehavior,
		}
	}
	if b.BatchDescriptor != nil {
		gb.BatchDescriptor = &serviceconfig.GAPICBatchDescriptor{
			BatchedField:        b.BatchDescriptor.BatchedField,
			DiscriminatorFields: b.BatchDescriptor.DiscriminatorFields,
			SubresponseField:    b.BatchDescriptor.SubresponseField,
		}
	}
	return gb
}
