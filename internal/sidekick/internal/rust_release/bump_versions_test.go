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
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/external"
)

func TestBumpVersionsSuccess(t *testing.T) {
	requireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "git",
		},
	}
	setupForVersionBump(t, "release-2001-02-03")
	if err := BumpVersions(config); err != nil {
		t.Fatal(err)
	}
}

func TestBumpVersionsPreflightError(t *testing.T) {
	config := &config.Release{
		Preinstalled: map[string]string{
			"git": "git-not-found",
		},
	}
	if err := BumpVersions(config); err == nil {
		t.Errorf("expected an error in BumpVersions() with a bad git command")
	}
}

func TestBumpVersionsLastTagError(t *testing.T) {
	const echo = "/bin/echo"
	requireCommand(t, "git")
	requireCommand(t, echo)
	config := config.Release{
		Remote: "origin",
		Branch: "invalid-branch",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	setupForVersionBump(t, "last-tag-error")
	if err := BumpVersions(&config); err == nil {
		t.Fatalf("expected an error during GetLastTag")
	}
}

func setupForVersionBump(t *testing.T, wantTag string) {
	remoteDir := t.TempDir()
	continueInNewGitRepository(t, remoteDir)
	initRepositoryContents(t)
	if err := external.Run("git", "tag", wantTag); err != nil {
		t.Fatal(err)
	}
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	if err := external.Run("git", "clone", remoteDir, "."); err != nil {
		t.Fatal(err)
	}
}
