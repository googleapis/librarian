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
)

// Command represents a single command that can be executed by the application.
type Command struct {
	// Name is the unique identifier for the command.
	Name string

	// Short is a concise description shown in the 'librarian -h' output.
	Short string

	// Run executes the command.
	//
	// TODO(https://github.com/googleapis/librarian/issues/194): migrate all
	// commands to implement this method.
	Run func(ctx context.Context) error

	// flags is the command's flag set for parsing arguments and generating
	// usage messages. This is populated for each command in init().
	flags *flag.FlagSet
}

// Parse parses the provided command-line arguments using the command's flag
// set.
func (c *Command) Parse(args []string) error {
	return c.flags.Parse(args)
}

// Lookup finds a command by its name, and returns an error if the command is
// not found.
func Lookup(name string, commands []*Command) (*Command, error) {
	var cmd *Command
	for _, sub := range commands {
		if sub.Name == name {
			cmd = sub
		}
	}
	if cmd == nil {
		return nil, fmt.Errorf("invalid command: %q", name)
	}
	return cmd, nil
}

// SetFlags registers a list of functions that configure flags for the command.
func (c *Command) SetFlags(flagFunctions []func(fs *flag.FlagSet)) {
	c.flags = flag.NewFlagSet(c.Name, flag.ContinueOnError)
	c.flags.Usage = constructUsage(c.flags, c.Name)
	for _, fn := range flagFunctions {
		fn(c.flags)
	}
}

// TODO(https://github.com/googleapis/librarian/issues/205): clean up this
// function so that "librarian" is not hardcoded
func constructUsage(fs *flag.FlagSet, name string) func() {
	output := fmt.Sprintf("Usage:\n\n  librarian %s [arguments]\n", name)
	output += "\nFlags:\n\n"
	return func() {
		fmt.Fprint(fs.Output(), output)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\n\n")
	}
}
