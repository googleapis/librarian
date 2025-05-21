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
	"flag"
	"fmt"
	"log/slog"

	"github.com/googleapis/librarian/internal/command"
)

func Run(ctx context.Context, arg ...string) error {
	cmd, err := parseArgs(arg)
	if err != nil {
		return err
	}
	if err := cmd.Parse(arg[1:]); err != nil {
		return err
	}
	slog.Info("librarian", "arguments", arg)
	return command.RunCommand(cmd, ctx)
}

func parseArgs(args []string) (*command.Command, error) {
	fs := flag.NewFlagSet("librarian", flag.ContinueOnError)
	output := `Librarian manages client libraries for Google APIs.

Usage:

  librarian <command> [arguments]

The commands are:
`
	for _, c := range command.Commands {
		output += fmt.Sprintf("\n  %-25s  %s", c.Name, c.Short)
	}

	fs.Usage = func() {
		fmt.Fprint(fs.Output(), output)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\n\n")
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if len(fs.Args()) == 0 {
		fs.Usage()
		return nil, fmt.Errorf("command not specified")
	}
	return command.Lookup(fs.Args()[0])
}
