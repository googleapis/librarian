// Copyright 2024 Google LLC
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

// Package librarian contains the business logic for the Librarian CLI.
// Implementation details for interacting with other systems (Git, GitHub,
// Docker etc) are abstracted into other packages.
package librarian

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/googleapis/librarian/internal/cli"
)

// CmdLibrarian is the top-level command for the Librarian CLI.
var CmdLibrarian = &cli.Command{
	Short:     "librarian manages client libraries for Google APIs",
	UsageLine: "librarian <command> [arguments]",
	Long:      "Librarian manages client libraries for Google APIs.",
}

func init() {
	CmdLibrarian.Init()
	CmdLibrarian.Commands = append(CmdLibrarian.Commands,
		cmdGenerate,
		cmdUpdateImageTag,
		cmdCreateReleaseArtifacts,
		cmdPublishReleaseArtifacts,
		cmdVersion,
	)
}

// Run executes the Librarian CLI with the given command line
// arguments.
func Run(ctx context.Context, arg ...string) error {
	if err := CmdLibrarian.Parse(arg); err != nil {
		return err
	}
	if len(arg) == 0 {
		CmdLibrarian.Flags.Usage()
		return fmt.Errorf("command not specified")
	}
	cmd, err := CmdLibrarian.Lookup(arg[0])
	if err != nil {
		CmdLibrarian.Flags.Usage()
		return err
	}
	if err := cmd.Parse(arg[1:]); err != nil {
		// We expect that if cmd.Parse fails, it will already
		// have printed out a command-specific usage error,
		// so we don't need to display the general usage.
		return err
	}
	slog.Info("librarian", "arguments", arg)
	if err := cmd.Config.SetupUser(); err != nil {
		return fmt.Errorf("failed to setup user config: %w", err)
	}
	return cmd.Run(ctx, cmd.Config)
}
