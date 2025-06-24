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

	// Usage is the one line usage.
	Usage string

	// Long is the full description of the command.
	Long string

	// Run executes the command.
	Run func(ctx context.Context, cfg *config.Config) error

	// flags is the command's flag set for parsing arguments and generating
	// usage messages. This is populated for each command in init().
	flags *flag.FlagSet
}

// Parse parses the provided command-line arguments using the command's flag
// set.
func (c *Command) Parse(args []string) error {
	if c.flags == nil {
		return nil
	}
	return c.flags.Parse(args)
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
func Lookup(name string, commands []*Command) (*Command, error) {
	for _, sub := range commands {
		if sub.Name() == name {
			return sub, nil
		}
	}
	return nil, fmt.Errorf("invalid command: %q", name)
}

// SetFlags registers a list of functions that configure flags for the command.
func (c *Command) SetFlags(flagFunctions []func(fs *flag.FlagSet)) {
	initFlags(c)
	for _, fn := range flagFunctions {
		fn(c.flags)
	}
}

func (c *Command) usage(w io.Writer) {
	if c.Short == "" || c.Usage == "" || c.Long == "" {
		panic(fmt.Sprintf("command %q is missing documentation", c.Name()))
	}

	fmt.Fprintf(w, "%s\n\n", c.Long)
	fmt.Fprintf(w, "Usage:\n  %s", c.Usage)
	if hasFlags(c.flags) {
		fmt.Fprint(w, "\n\nFlags:\n")
	}
	c.flags.SetOutput(w)
	c.flags.PrintDefaults()
	fmt.Fprintf(w, "\n\n")
}

func initFlags(c *Command) *Command {
	c.flags = flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	c.flags.Usage = func() {
		c.usage(c.flags.Output())
	}
	return c
}

func hasFlags(fs *flag.FlagSet) bool {
	visited := false
	fs.VisitAll(func(f *flag.Flag) {
		visited = true
	})
	return visited
}
