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
	"errors"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/surfer/gcloud"
)

// Run executes the surfer CLI with the given command line arguments.
func Run(ctx context.Context, args []string) error {
	cmd := &cli.Command{
		Short:     "surfer generates gcloud command YAML files",
		UsageLine: "surfer generate [arguments]",
		Long:      "surfer generates gcloud command YAML files",
		Commands: []*cli.Command{
			newCmdGenerate(),
		},
	}
	cmd.Init()
	return cmd.Run(ctx, args)
}

var errMissingConfigFlag = errors.New("--config is required")

func newCmdGenerate() *cli.Command {
	var (
		googleapis string
		config     string
		out        string
	)

	cmdGenerate := &cli.Command{
		Short:     "generate generates gcloud commands",
		UsageLine: "surfer generate --config <path> --googleapis <path> [--out <path>]",
		Long: `generate generates gcloud commands

generate generates gcloud command files from protobuf API specifications,
service config yaml, and gcloud.yaml.

Example:
  surfer generate --config ./gcloud.yaml --googleapis ./googleapis --out ./output
`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if config == "" {
				return errMissingConfigFlag
			}

			return gcloud.Generate(ctx, googleapis, config, out)
		},
	}
	cmdGenerate.Init()
	cmdGenerate.Flags.StringVar(&config, "config", "", "path to gcloud.yaml configuration file (required)")
	cmdGenerate.Flags.StringVar(&googleapis, "googleapis", "https://github.com/googleapis/googleapis", "URL or directory path to googleapis")
	cmdGenerate.Flags.StringVar(&out, "out", ".", "output directory")
	return cmdGenerate
}
