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
	"strings"
	"testing"
)

func TestRunGit_Success(t *testing.T) {
	t.Parallel()
	repoDir := createTempGitRepo(t)
	createDummyFile(t, repoDir, "testfile.txt")

	runGit(t, repoDir, "add", "testfile.txt")
	runGit(t, repoDir, "commit", "-m", "Add testfile")

	output, err := exec.Command("git", "-C", repoDir, "log", "--oneline").Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}
	if !strings.Contains(string(output), "Add testfile") {
		t.Errorf("Expected 'Add testfile' in git log, got: %s", string(output))
	}
}

// Helper to create a temporary git repository for testing.
func createTempGitRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	return repoDir
}

// Helper to create a dummy file in a directory.
func createDummyFile(t *testing.T, dir, filename string) {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("failed to create dummy file: %v", err)
	}
}
