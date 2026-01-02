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
	"path"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPublishSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "/bin/echo",
		},
		Tools: map[string][]config.Tool{
			"cargo": {
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	if err := Publish(t.Context(), config, true, false); err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithLocalChangesError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "/bin/echo",
		},
		Tools: map[string][]config.Tool{
			"cargo": {
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-with-local-changes-error")
	testhelper.CloneRepository(t, remoteDir)
	// Add a local commit that is not pushed to the remote.
	testhelper.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := command.Run(t.Context(), "git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}
	if err := Publish(t.Context(), config, true, false); err == nil {
		t.Errorf("expected an error publishing a dirty local repository")
	}
}

func TestPublishPreflightError(t *testing.T) {
	config := &config.Release{
		Preinstalled: map[string]string{
			"git": "git-not-found",
		},
	}
	if err := Publish(t.Context(), config, true, false); err == nil {
		t.Errorf("expected an error in Publish() with a bad git command")
	}
}

func TestPublishLastTagError(t *testing.T) {
	const echo = "/bin/echo"
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, echo)
	config := config.Release{
		Remote: "origin",
		Branch: "invalid-branch",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	if err := Publish(t.Context(), &config, true, false); err == nil {
		t.Fatalf("expected an error during GetLastTag")
	}
}
