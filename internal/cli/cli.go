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

	// FlagFunctions are functions to initialize the command's flag set.
	FlagFunctions []func(fs *flag.FlagSet)

	// Flags is the command's flag set for parsing arguments and generating
	// usage messages. This is populated for each command in init().
	Flags *flag.FlagSet
}

// Parse parses the provided command-line arguments using the command's flag
// set.
func (c *Command) Parse(args []string) error {
	return c.Flags.Parse(args)
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
