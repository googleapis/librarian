// Copyright 2025 Google LLC
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

// Package surfer provides the core implementation for the surfer CLI tool.
package surfer

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// Run executes the surfer CLI with the given command line arguments.
func Run(ctx context.Context, args ...string) error {
	cmd := &cli.Command{
		Name:        "surfer",
		Usage:       "generates gcloud command YAML files",
		UsageText:   "surfer [command]",
		Description: "surfer generates gcloud command YAML files",
		Commands: []*cli.Command{
			generateCommand(),
		},
	}
	return cmd.Run(ctx, args)
}

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generates gcloud commands",
		UsageText: "surfer generate <path to gcloud.yaml> --googleapis <path>",
		Description: `generate generates gcloud command files from protobuf API specifications,
service config yaml, and gcloud.yaml.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "out",
				Value: ".",
				Usage: "output directory",
			},
			&cli.StringFlag{
				Name:  "service-config",
				Usage: "path to the api service config",
			},
			&cli.StringFlag{
				Name:  "proto-files-include-list",
				Value: "google/cloud/parallelstore/v1/parallelstore.proto",
				Usage: "comma-separated list of protobuf files used to generate the gcloud commands",
			},
			&cli.StringFlag{
				Name:  "googleapis",
				Value: "https://github.com/googleapis/googleapis",
				Usage: "URL or directory path to googleapis",
			},
			&cli.StringFlag{
				Name:  "descriptor-files-to-generate",
				Usage: "comma-separated list of files to generate from the descriptors",
			},
			&cli.StringFlag{
				Name:  "descriptor-files",
				Usage: "comma-separated list of paths to binary FileDescriptorSet files",
			},
			&cli.StringFlag{
				Name:  "base-module",
				Value: "googlecloudsdk",
				Usage: "base python module path for surface command groups",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() == 0 {
				return fmt.Errorf("path to gcloud.yaml is required")
			}
			config := cmd.Args().First()
			googleapis := cmd.String("googleapis")
			out := cmd.String("out")
			includeList := cmd.String("proto-files-include-list")
			serviceConfig := cmd.String("service-config")
			descriptorFiles := cmd.String("descriptor-files")
			descriptorFilesToGenerate := cmd.String("descriptor-files-to-generate")
			baseModule := cmd.String("base-module")
			return generate(generateConfig{
				GcloudConfig:              config,
				ServiceConfig:             serviceConfig,
				IncludeList:               includeList,
				Googleapis:                googleapis,
				DescriptorFilesToGenerate: descriptorFilesToGenerate,
				DescriptorFiles:           descriptorFiles,
				Output:                    out,
				BaseModule:                baseModule,
			})
		},
	}
}
