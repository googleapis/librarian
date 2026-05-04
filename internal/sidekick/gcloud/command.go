// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcloud

import (
	"fmt"
	"strings"
)

// Command represents a leaf command.
type Command struct {
	Args       []string
	Flags      []Flag
	Name       string
	PathFormat string
	PathLabel  string
	Usage      string
}

// HasPath reports whether the command composes a resource path.
func (c Command) HasPath() bool { return c.PathFormat != "" }

// PathFormatArgs returns the comma-separated cmd.String("X") arguments
// passed to the generated [fmt.Sprintf] call.
func (c Command) PathFormatArgs() string {
	parts := make([]string, len(c.Args))
	for i, a := range c.Args {
		parts[i] = fmt.Sprintf("cmd.String(%q)", a)
	}
	return strings.Join(parts, ", ")
}

// Flag represents a single CLI flag.
type Flag struct {
	// Name is the long flag name without leading dashes (e.g. "project").
	Name string

	// Kind is the urfave/cli flag type (e.g. "String", "Bool").
	Kind string

	// Required reports whether the flag must be set on the command line.
	Required bool

	// Usage is the help text shown next to the flag.
	//
	// TODO(https://github.com/googleapis/librarian/issues/5769):
	// Usage is currently a generic "The <name>." string. Source it from
	// the proto field's documentation when we wire that through.
	Usage string
}

// flag returns a Flag with canonical Usage derived from name.
func flag(name, kind string, required bool) Flag {
	return Flag{
		Name:     name,
		Kind:     kind,
		Required: required,
		Usage:    fmt.Sprintf("The %s.", strings.ReplaceAll(name, "-", " ")),
	}
}

// pathFlag returns the canonical required-string Flag for a path-derived name.
func pathFlag(name string) Flag {
	return flag(name, "String", true)
}
