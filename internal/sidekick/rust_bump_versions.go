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

package sidekick

import (
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	rustrelease "github.com/googleapis/librarian/internal/sidekick/internal/rust_release"
)

func init() {
	newCommand(
		"sidekick rust-bump-versions",
		"Increments the version numbers as needed.",
		`
Finds all the changes since the last release and increments, if needed, the
version numbers for all crates that have changed since then. The command changes
both the '.sidekick.toml' and the 'Cargo.toml' files.

For crates where version number has already been increased, this command has no
effect.
`,
		cmdSidekick,
		rustBumpVersions,
	)
}

// rustBumpVersions increments the version numbers as needed.
func rustBumpVersions(rootConfig *config.Config, cmdLine *CommandLine) error {
	return rustrelease.BumpVersions(rootConfig.Release)
}
