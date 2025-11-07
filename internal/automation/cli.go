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

// Package automation implements the command-line interface and core logic
// for Librarian's automated workflows.
package automation

import (
	"context"

	"github.com/googleapis/librarian/internal/cli"
)

func newAutomationCommand() *cli.Command {
	cmd := &cli.Command{
		Short:     "automate code generation and library release process",
		UsageLine: "test",
		Long:      "test",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			runner := newAutomationRunner(cmd.Config)
			return runner.run(ctx)
		},
	}
	cmd.Init()
	addFlagBuild(cmd.Flags, cmd.Config)
	addFlagCommand(cmd.Flags, cmd.Config)
	addFlagForceRun(cmd.Flags, cmd.Config)
	addFlagProject(cmd.Flags, cmd.Config)
	addFlagPush(cmd.Flags, cmd.Config)
	return cmd
}

// Run parses the command line arguments and triggers the specified command.
func Run(ctx context.Context, args []string) error {
	cmd := newAutomationCommand()
	return cmd.Run(ctx, args)
}
