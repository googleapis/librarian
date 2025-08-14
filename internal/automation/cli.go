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

package automation

import (
	"context"
	"flag"
	"log/slog"
)

// Run parses the command line arguments and triggers the specified command.
func Run(args []string) error {
	ctx := context.Background()
	options, err := parseFlags(args)
	if err != nil {
		slog.Error("Error parsing command", slog.Any("err", err))
		return err
	}

	err = RunCommand(ctx, options.Command, options.ProjectId)
	if err != nil {
		slog.Error("Error running command", slog.Any("err", err))
		return err
	}
	return nil
}

type runOptions struct {
	Command   string
	ProjectId string
}

func parseFlags(args []string) (*runOptions, error) {
	flagSet := flag.NewFlagSet("dispatcher", flag.ContinueOnError)
	projectId := flagSet.String("project", "cloud-sdk-librarian-prod", "GCP project ID")
	command := flagSet.String("command", "generate", "The librarian command to run")
	err := flagSet.Parse(args)
	if err != nil {
		return nil, err
	}
	return &runOptions{
		ProjectId: *projectId,
		Command:   *command,
	}, nil
}
