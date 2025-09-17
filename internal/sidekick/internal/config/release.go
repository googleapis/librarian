// Copyright 2025 Google LLC
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

package config

// Release holds the configuration parameter for any `${lang}-release` subcommand.
type Release struct {
	// Remote sets the name of the source-of-truth remote for releases, typically `upstream`.
	Remote string

	// ReleaseBranch sets the name of the release branch, typically `main`
	Branch string

	// Tools defines the list of tools to install, indexed by installer.
	Tools map[string][]Tool

	// Preinstalled tools defines the list of tools that must be pre-installed.
	//
	// This is indexed by the well-known name of the tool vs. its path, e.g.
	// [preinstalled]
	// cargo = /usr/bin/cargo
	Preinstalled map[string]string `toml:"pre-installed"`

	// IgnoredChanges defines globs that are ignored in change analysis.
	IgnoredChanges []string `toml:"ignored-changes,omitempty"`

	// An alternative location for the `roots.pem` file. If empty it has no
	// effect.
	RootsPem string
}

// Tool defines the configuration required to install helper tools.
type Tool struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}
