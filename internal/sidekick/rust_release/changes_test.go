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
	"github.com/googleapis/librarian/internal/testhelpers"
)

const (
	newLibRsContents = `pub fn hello() -> &'static str { "Hello World" }`
)

func TestMatchesBranchPointSuccess(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	remoteDir := testhelpers.SetupForPublish(t, "v1.0.0")
	testhelpers.CloneRepository(t, remoteDir)
	if err := matchesBranchPoint(config); err != nil {
		t.Fatal(err)
	}
}

func TestMatchesBranchDiffError(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "not-a-valid-branch",
	}
	remoteDir := testhelpers.SetupForPublish(t, "v1.0.0")
	testhelpers.CloneRepository(t, remoteDir)
	if err := matchesBranchPoint(config); err == nil {
		t.Errorf("expected an error with an invalid branch")
	}
}

func TestMatchesDirtyCloneError(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "not-a-valid-branch",
	}
	remoteDir := testhelpers.SetupForPublish(t, "v1.0.0")
	testhelpers.CloneRepository(t, remoteDir)
	testhelpers.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := command.Run(t.Context(), "git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}

	if err := matchesBranchPoint(config); err == nil {
		t.Errorf("expected an error with a dirty clone")
	}
}
