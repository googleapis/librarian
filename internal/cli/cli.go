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

// Package cli defines a lightweight framework for building CLI commands.
// It's designed to be generic and self-contained, with no embedded business logic
// or dependencies on the surrounding application's configuration or behavior.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Command represents a single command that can be executed by the application.
type Command struct {
	// Short is a concise one-line description of the command.
	Short string

	// UsageLine is the one line usage.
	UsageLine string

	// Long is the full description of the command.
	Long string

	// Run executes the command.
	Run func(ctx context.Context, cfg *config.Config) error

	// Commands are the sub commands.
	Commands []*Command

	// Flags is the command's flag set for parsing arguments and generating
	// usage messages. This is populated for each command in init().
	Flags *flag.FlagSet

	// Config contains the configs for the command.
	Config *config.Config
}

// Parse parses the provided command-line arguments using the command's flag
// set.
func (c *Command) Parse(args []string) error {
	return c.Flags.Parse(args)
}

// Name is the command name. Command.Short is always expected to begin with
// this name.
func (c *Command) Name() string {
	if c.Short == "" {
		panic("command is missing documentation")
	}
	parts := strings.Fields(c.Short)
	return parts[0]
}

// Lookup finds a command by its name, and returns an error if the command is
// not found.
func (c *Command) Lookup(name string) (*Command, error) {
	for _, sub := range c.Commands {
		if sub.Name() == name {
			return sub, nil
		}
	}
	return nil, fmt.Errorf("invalid command: %q", name)
}

func (c *Command) usage(w io.Writer) {
	if c.Short == "" || c.UsageLine == "" || c.Long == "" {
		panic(fmt.Sprintf("command %q is missing documentation", c.Name()))
	}

	fmt.Fprintf(w, "%s\n\n", c.Long)
	fmt.Fprintf(w, "Usage:\n  %s", c.UsageLine)
	if len(c.Commands) > 0 {
		fmt.Fprint(w, "\n\nCommands:\n")
		for _, c := range c.Commands {
			parts := strings.Fields(c.Short)
			short := strings.Join(parts[1:], " ")
			fmt.Fprintf(w, "\n  %-25s  %s", c.Name(), short)
		}
	}
	if hasFlags(c.Flags) {
		fmt.Fprint(w, "\n\nFlags:\n")
	}
	c.Flags.SetOutput(w)
	c.Flags.PrintDefaults()
	fmt.Fprintf(w, "\n\n")
}

// InitFlags creates a new set of flags for the command and initializes
// them such that any parsing failures result in the command usage being
// displayed.
//
// TODO(https://github.com/googleapis/librarian/issues/619): rename since this
// now also initializes c.Config
func (c *Command) InitFlags() *Command {
	c.Flags = flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	c.Flags.Usage = func() {
		c.usage(c.Flags.Output())
	}
	c.Config = config.New()
	return c
}

func hasFlags(fs *flag.FlagSet) bool {
	visited := false
	fs.VisitAll(func(f *flag.Flag) {
		visited = true
	})
	return visited
}
