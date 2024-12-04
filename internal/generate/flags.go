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
	name     string
	short    string
	usage    string
	flags    *flag.FlagSet
	commands []*command
	run      func(ctx context.Context) error
}

func (c *command) lookup(name string) *command {
	for _, sub := range c.commands {
		if sub.name == name {
			return sub
		}
	}
	return nil
}

func generatorCommand() *command {
	c := &command{
		name:  "generator",
		short: "Generator generates client libraries for Google APIs.",
		usage: "generator <command> [arguments]",
		commands: []*command{
			generatorCreateCommand(),
			generatorGenerateCommand(),
		},
		flags: flag.NewFlagSet("generator", flag.ContinueOnError),
		// run is not set for generator because it is different than the
		// others
	}
	c.flags.Usage = constructUsage(c.flags, c.short, c.usage, c.commands, false)
	return c
}

func constructUsage(fs *flag.FlagSet, short, usage string, commands []*command, hasFlags bool) func() {
	output := fmt.Sprintf("%s\n\nUsage:\n\n  %s\n", short, usage)
	if len(commands) > 0 {
		output += "\nThe commands are:\n\n"
		for _, c := range commands {
			output += fmt.Sprintf("  %s  %s\n", c.name, c.short)
		}
	}
	if hasFlags {
		output += "\nFlags:\n\n"
	}
	return func() {
		fmt.Fprint(fs.Output(), output)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\n\n")
	}
}
