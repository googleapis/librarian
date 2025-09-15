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
		cmdRelease,
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
		return nil
	}
	cmd, arg, err := cli.LookupCommand(CmdLibrarian, arg)
	if err != nil {
		return err
	}

	// If a command is just a container for subcommands, it won't have a
	// Run function. In that case, display its usage instructions.
	if cmd.Run == nil {
		cmd.Flags.Usage()
		return fmt.Errorf("command %q requires a subcommand", cmd.Name())
	}

	if err := cmd.Parse(arg); err != nil {
		// We expect that if cmd.Parse fails, it will already
		// have printed out a command-specific usage error,
		// so we don't need to display the general usage.
		return err
	}
	slog.Info("librarian", "arguments", arg)
	if err := cmd.Config.SetDefaults(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}
	if _, err := cmd.Config.IsValid(); err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}
	return cmd.Run(ctx, cmd.Config)
}
