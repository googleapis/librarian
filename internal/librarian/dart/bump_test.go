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

package dart

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestUpdateChangelog_New(t *testing.T) {
	tempDir := t.TempDir()

	err := updateChangelog(context.Background(), tempDir, "1.2.3", "", true)
	if err != nil {
		t.Fatalf("updateChangelog failed: %v", err)
	}

	changelogPath := filepath.Join(tempDir, "CHANGELOG.md")
	content, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	want := `# Changelog

## 1.2.3

- chore: update cloud dependencies

`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdateChangelog_Existing(t *testing.T) {
	tempDir := t.TempDir()
	changelogPath := filepath.Join(tempDir, "CHANGELOG.md")
	initialContent := `# Changelog

## 1.2.2

- chore: release 1.2.2
`
	if err := os.WriteFile(changelogPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := updateChangelog(context.Background(), tempDir, "1.2.3", "", true)
	if err != nil {
		t.Fatalf("updateChangelog failed: %v", err)
	}

	content, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	want := `# Changelog

## 1.2.3

- chore: update cloud dependencies

## 1.2.2

- chore: release 1.2.2
`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdateChangelog_WithCommits(t *testing.T) {
	tempDir := t.TempDir()

	testhelper.ContinueInNewGitRepository(t, tempDir)
	t.Chdir(tempDir)

	if err := os.WriteFile("file.txt", []byte("init"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "Initial release.")

	tagCommit, err := git.GetCommitHash(context.Background(), command.Git, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("file.txt", []byte("feat 1"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added support for something")

	if err := os.WriteFile("file.txt", []byte("fix 1"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "fix: resolved a bug")

	err = updateChangelog(context.Background(), tempDir, "1.2.3", tagCommit, false)
	if err != nil {
		t.Fatalf("updateChangelog failed: %v", err)
	}

	changelogPath := filepath.Join(tempDir, "CHANGELOG.md")
	content, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	want := `# Changelog

## 1.2.3

- fix: resolved a bug
- feat: added support for something

`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
