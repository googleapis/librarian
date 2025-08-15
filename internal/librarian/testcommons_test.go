// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// newTestGitRepo creates a new git repository in a temporary directory.
func newTestGitRepo(t *testing.T) gitrepo.Repository {
	t.Helper()
	return newTestGitRepoWithState(t, true)
}

// newTestGitRepo creates a new git repository in a temporary directory.
func newTestGitRepoWithState(t *testing.T, writeState bool) gitrepo.Repository {
	t.Helper()
	dir := t.TempDir()
	remoteURL := "https://github.com/googleapis/librarian.git"
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	if writeState {
		// Create an empty state.yaml file
		stateDir := filepath.Join(dir, config.LibrarianDir)
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll: %v", err)
		}
		stateFile := filepath.Join(stateDir, "state.yaml")
		if err := os.WriteFile(stateFile, []byte(""), 0644); err != nil {
			t.Fatalf("os.WriteFile: %v", err)
		}
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")
	runGit(t, dir, "remote", "add", "origin", remoteURL)
	repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: dir})
	if err != nil {
		t.Fatalf("gitrepo.Open(%q) = %v", dir, err)
	}
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}
