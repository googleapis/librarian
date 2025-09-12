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
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/internal/config"
)

func TestPreflightSuccess(t *testing.T) {
	requireGit(t)
	config := config.Config{
		Release: &config.Release{
			Preinstalled: map[string]string{
				"git":   "git",
				"cargo": "git",
			},
		},
	}
	cmdLine := CommandLine{}
	if err := rustBumpVersions(&config, &cmdLine); err != nil {
		t.Fatal(err)
	}
}

func TestPreflightMissingCommand(t *testing.T) {
	requireGit(t)
	config := config.Config{
		Release: &config.Release{
			Preinstalled: map[string]string{
				"cargo": "not-a-valid-command-bad-bad",
			},
		},
	}
	cmdLine := CommandLine{}
	if err := rustBumpVersions(&config, &cmdLine); err == nil {
		t.Errorf("expected an error in rustBumpVersions() with a bad cargo command")
	}
}
