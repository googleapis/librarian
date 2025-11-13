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
	"errors"
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

	// Action executes the command.
	Action func(context.Context, *Command) error

	// Commands are the sub commands.
	Commands []*Command

	// Flags is the command's flag set for parsing arguments and generating
	// usage messages. This is populated for each command in init().
	Flags *flag.FlagSet

	// Config contains the configs for the command.
	Config *config.Config
}

// NewCommandSet creates and initializes a root *Command object. It automatically appends a "version" subcommand to
// the list.
func NewCommandSet(commands []*Command, short, usageLine, long string) *Command {
	cmd := &Command{
		Short:     short,
		UsageLine: usageLine,
		Long:      long,
	}
	verifyCommandDocs(cmd)
	pkg := strings.Split(cmd.Short, " ")[0]
	cmd.Commands = append(make([]*Command, 0, len(commands)+1), commands...)
	cmd.Commands = append(cmd.Commands, newCmdVersion(pkg))

	cmd.Init()
	return cmd
}

func newCmdVersion(pkg string) *Command {
	cmdVersion := &Command{
		Short:     "version prints the version information",
		UsageLine: fmt.Sprintf("%s version", pkg),
		Long:      fmt.Sprintf("Version prints version information for the %s binary.", pkg),
		Action: func(ctx context.Context, cmd *Command) error {
			fmt.Println(Version())
			return nil
		},
	}
	cmdVersion.Init()
	return cmdVersion
}

// Run executes the command with the provided arguments.
func (c *Command) Run(ctx context.Context, args []string) error {
	cmd, remaining, err := lookupCommand(c, args)
	if err != nil {
		return err
	}
	if err := cmd.Flags.Parse(remaining); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if cmd.Action == nil {
		cmd.Flags.Usage()
		if len(cmd.Commands) > 0 {
			return nil
		}
		return fmt.Errorf("no action defined for command %q", cmd.Name())
	}
	return cmd.Action(ctx, cmd)
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

func (c *Command) usage(w io.Writer) {
	verifyCommandDocs(c)
	fmt.Fprintf(w, "%s\n\n", c.Long)
	fmt.Fprintf(w, "Usage:\n\n  %s", c.UsageLine)
	if len(c.Commands) > 0 {
		fmt.Fprint(w, "\n\nCommands:\n")
		for _, c := range c.Commands {
			parts := strings.Fields(c.Short)
			short := strings.Join(parts[1:], " ")
			fmt.Fprintf(w, "\n  %-25s  %s", c.Name(), short)
		}
	}
	if hasFlags(c.Flags) {
		fmt.Fprint(w, "\n\nFlags:\n\n")
	}
	c.Flags.SetOutput(w)
	c.Flags.PrintDefaults()
	fmt.Fprintf(w, "\n\n")
}

// Init creates a new set of flags for the command and initializes
// them such that any parsing failures result in the command usage being
// displayed.
func (c *Command) Init() *Command {
	c.Flags = flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	c.Flags.Usage = func() {
		c.usage(c.Flags.Output())
	}
	c.Config = config.New(c.Name())
	return c
}

func hasFlags(fs *flag.FlagSet) bool {
	visited := false
	fs.VisitAll(func(f *flag.Flag) {
		visited = true
	})
	return visited
}

func verifyCommandDocs(c *Command) {
	if c.Short == "" || c.UsageLine == "" || c.Long == "" {
		panic(fmt.Sprintf("command %q is missing documentation", c.Name()))
	}
}

// lookupCommand looks up the command specified by the given arguments.
// It returns the command, the remaining arguments, and an error if the command
// is not found.
func lookupCommand(cmd *Command, args []string) (*Command, []string, error) {
	// If the first character of the argument is '-' assume it is a flag.
	for len(args) > 0 && args[0][0] != '-' {
		name := args[0]
		var found *Command
		for _, sub := range cmd.Commands {
			if sub.Name() == name {
				found = sub
				break
			}
		}
		if found == nil {
			if len(cmd.Commands) == 0 {
				break
			}
			cmd.Flags.Usage()
			return nil, nil, fmt.Errorf("invalid command: %q", name)
		}
		cmd = found
		args = args[1:]
	}
	return cmd, args, nil
}
