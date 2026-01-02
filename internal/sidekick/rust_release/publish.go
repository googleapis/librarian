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

package rustrelease

import (
	"context"

	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

// Publish finds all the crates that should be published, (optionally) runs
// `cargo semver-checks` and (optionally) publishes them.
func Publish(ctx context.Context, config *config.Release, dryRun bool, skipSemverChecks bool) error {
	if err := rust.PreFlight(ctx, config.Preinstalled, config.Remote, rust.ToConfigTools(config.Tools["cargo"])); err != nil {
		return err
	}
	files, lastTag, err := git.GetFileChangesSinceLastTag(ctx, config.Preinstalled, config.Remote, config.Branch, config.IgnoredChanges)
	if err != nil {
		return err
	}
	return rust.PublishCrates(ctx, rust.FromSidekickReleaseConfig(config), dryRun, skipSemverChecks, lastTag, files)
}
