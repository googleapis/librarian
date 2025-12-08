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
	"context"

	"github.com/googleapis/librarian/internal/sidekick/config"
	rustrelease "github.com/googleapis/librarian/internal/sidekick/rust_release"
)

var (
	skipSemverChecks bool
)

func init() {
	newCommand(
		"sidekick rust-publish",
		"Publish all crates changed since the last release.",
		`
Find the last release tag, and the crates that changed since then. Validates
the list against 'cargo workspaces plan', and stops if there are any
differences. The runs 'cargo semver-checks' for each crate to be published.
Finally publishes the crates using 'cargo workspaces publish' as that preserves
the dependency order.
`,
		cmdSidekick,
		rustPublish,
	).addFlagBool(&skipSemverChecks, "skip-semver-checks", false, "skip 'cargo semver-checks' for changed crates.")
}

// rustPublish finds all the crates that should be published, (optionally) runs
// `cargo semver-checks` and (optionally) publishes them.
func rustPublish(ctx context.Context, rootConfig *config.Config, cmdLine *CommandLine) error {
	return rustrelease.Publish(ctx, rootConfig.Release, cmdLine.DryRun, skipSemverChecks)
}
