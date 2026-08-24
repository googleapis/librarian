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

package swift

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/testhelper"
)

func setupMonorepoWithRootFiles(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, repoDir)

	for _, f := range DefaultRootFiles {
		if err := os.WriteFile(f, []byte("contents of "+f), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	authDir := filepath.Join("packages", "auth")
	testhelper.AddSwiftPackage(t, authDir, "GoogleCloudAuth")
	storageDir := filepath.Join("packages", "storage")
	testhelper.AddSwiftPackage(t, storageDir, "GoogleCloudStorage")
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: initial commit with auth and storage")

	// Add a second commit to the package
	authFile := filepath.Join(authDir, "Sources", "GoogleCloudAuth", "Auth.swift")
	if err := os.WriteFile(authFile, []byte("// new auth code"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: update auth logic")

	return repoDir
}

func TestSplitSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	setupMonorepoWithRootFiles(t)

	splitSHA, err := Split(t.Context(), SplitParams{
		TargetDir: "packages/auth",
		Origin:    "HEAD",
	})
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}
	if len(splitSHA) != 40 {
		t.Fatalf("Split() = %q, want 40-character SHA", splitSHA)
	}

	// Verify root files exist in the split commit
	lsOut, err := command.Output(t.Context(), command.Git, "ls-tree", "--name-only", splitSHA)
	if err != nil {
		t.Fatal(err)
	}
	files := strings.Fields(lsOut)
	for _, expectedFile := range []string{"LICENSE", "CODE_OF_CONDUCT.md", "CONTRIBUTING.md", "Package.swift", "Sources"} {
		found := slices.Contains(files, expectedFile)
		if !found {
			t.Errorf("expected file %q in split tree, got files: %v", expectedFile, files)
		}
	}

	// Verify all commits in history contain root files
	revListOut, err := command.Output(t.Context(), command.Git, "rev-list", splitSHA)
	if err != nil {
		t.Fatal(err)
	}
	commits := strings.Fields(revListOut)
	if len(commits) < 2 {
		t.Errorf("expected at least 2 commits in split history, got %d", len(commits))
	}
	for _, c := range commits {
		tree, err := command.Output(t.Context(), command.Git, "ls-tree", "--name-only", c)
		if err != nil {
			t.Fatal(err)
		}
		cFiles := strings.Fields(tree)
		for _, rf := range DefaultRootFiles {
			found := slices.Contains(cFiles, rf)
			if !found {
				t.Errorf("commit %s missing root file %s; found: %v", c, rf, cFiles)
			}
		}
	}

	testhelper.RunGit(t, "fsck", "--full")
}

func TestSplitNoRootFiles(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	setupMonorepoWithRootFiles(t)

	splitSHA, err := Split(t.Context(), SplitParams{
		TargetDir: "packages/auth",
		Origin:    "HEAD",
		RootFiles: []string{},
	})
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	lsOut, err := command.Output(t.Context(), command.Git, "ls-tree", "--name-only", splitSHA)
	if err != nil {
		t.Fatal(err)
	}
	files := strings.Fields(lsOut)
	for _, unexpectedFile := range DefaultRootFiles {
		for _, f := range files {
			if f == unexpectedFile {
				t.Errorf("unexpected file %q in split tree when RootFiles is empty", unexpectedFile)
			}
		}
	}
}

func TestSplitEmptyTargetDir(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	setupMonorepoWithRootFiles(t)

	_, err := Split(t.Context(), SplitParams{
		TargetDir: "",
	})
	if err == nil {
		t.Fatal("expected error for empty target dir, got nil")
	}
}
