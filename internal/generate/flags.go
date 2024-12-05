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

package generate

import (
	"context"
	"flag"
	"fmt"
)

type command struct {
	name  string
	short string
	flags *flag.FlagSet
	run   func(ctx context.Context) error
}

func constructUsage(fs *flag.FlagSet, name string, hasFlags bool) func() {
	output := fmt.Sprintf("Usage:\n\n  generator %s [arguments]\n", name)
	if hasFlags {
		output += "\nFlags:\n\n"
	}
	return func() {
		fmt.Fprint(fs.Output(), output)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\n\n")
	}
}

func parseArgs(args []string) (*command, error) {
	fs := flag.NewFlagSet("generator", flag.ContinueOnError)
	commands := []*command{
		generatorCreateCommand(),
		generatorGenerateCommand(),
	}

	output := `Generator generates client libraries for Google APIs.

Usage:

  generator <command> [arguments]

The commands are:
`
	for _, c := range commands {
		output += fmt.Sprintf("\n  %s  %s", c.name, c.short)
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
		return nil, fmt.Errorf("missing command")
	}

	name := fs.Args()[0]
	var cmd *command
	for _, sub := range commands {
		if sub.name == name {
			cmd = sub
		}
	}
	if cmd == nil {
		return nil, fmt.Errorf("invalid command: %q", name)
	}
	return cmd, nil
}
