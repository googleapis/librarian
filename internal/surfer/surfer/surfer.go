// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/googleapis/librarian/internal/surfer/gcloud"
	"github.com/urfave/cli/v3"
)

// Run executes the surfer CLI with the given command line arguments.
func Run(ctx context.Context, args []string) error {
	cmd := &cli.Command{
		Name:  "surfer",
		Usage: "surfer generates gcloud command YAML files",
		Commands: []*cli.Command{
			newCmdGenerate(),
		},
	}
	return cmd.Run(ctx, args)
}

func newCmdGenerate() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "generate generates gcloud commands",
		Description: `generate generates gcloud command files from protobuf API specifications,
service config yaml, and gcloud.yaml.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "googleapis",
				Value: "https://github.com/googleapis/googleapis",
				Usage: "URL or directory path to googleapis",
			},
			&cli.StringFlag{
				Name:  "out",
				Value: ".",
				Usage: "output directory",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() == 0 {
				return fmt.Errorf("path to gcloud.yaml is required")
			}
			config := cmd.Args().First()
			googleapis := cmd.String("googleapis")
			out := cmd.String("out")
			return gcloud.Generate(ctx, googleapis, config, out)
		},
	}
}
