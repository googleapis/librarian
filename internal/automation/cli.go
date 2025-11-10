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
	"flag"
	"log/slog"
)

// runCommandFn is a function type that matches RunCommand, for mocking in tests.
var runCommandFn = RunCommand

// Run parses the command line arguments and triggers the specified command.
func Run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == publishCmdName {
		cmd := newAutomationCommand()
		return cmd.Run(ctx, args)
	}

	options, err := parseFlags(args)
	if err != nil {
		slog.Error("error parsing command", slog.Any("err", err))
		return err
	}

	err = runCommandFn(ctx, options.Command, options.ProjectId, options.Push, options.Build)
	if err != nil {
		slog.Error("error running command", slog.Any("err", err))
		return err
	}
	return nil
}

type runOptions struct {
	Command   string
	ProjectId string
	Push      bool
	Build     bool
}

func parseFlags(args []string) (*runOptions, error) {
	flagSet := flag.NewFlagSet("dispatcher", flag.ContinueOnError)
	projectId := flagSet.String("project", "cloud-sdk-librarian-prod", "GCP project ID")
	command := flagSet.String("command", "generate", "The librarian command to run")
	push := flagSet.Bool("push", true, "The _PUSH flag (true/false) to Librarian CLI's -push option")
	build := flagSet.Bool("build", true, "The _BUILD flag (true/false) to Librarian CLI's -build option")
	err := flagSet.Parse(args)
	if err != nil {
		return nil, err
	}
	return &runOptions{
		ProjectId: *projectId,
		Command:   *command,
		Push:      *push,
		Build:     *build,
	}, nil
}
