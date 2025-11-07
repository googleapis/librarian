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

	"github.com/googleapis/librarian/internal/sidekick/config"
)

func TestLastTagSuccess(t *testing.T) {
	const wantTag = "release-2001-02-03"

	const echo = "/bin/echo"
	requireCommand(t, echo)
	release := config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	setupForVersionBump(t, wantTag)
	got, err := getLastTag(&release)
	if err != nil {
		t.Fatal(err)
	}
	if got != wantTag {
		t.Errorf("tag mismatch, want=%s, got=%s", wantTag, got)
	}
}

func TestLastTagGitError(t *testing.T) {
	const echo = "/bin/echo"
	requireCommand(t, echo)
	release := config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	remoteDir := t.TempDir()
	continueInNewGitRepository(t, remoteDir)
	if got, err := getLastTag(&release); err == nil {
		t.Fatalf("expected an error, got=%s", got)
	}
}
