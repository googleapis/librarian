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

package librarianops

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/command"
)

func TestGenerateCommand(t *testing.T) {
	t.Skip("flaky test: TODO(https://github.com/googleapis/librarian/issues/3698)")
	repoDir := t.TempDir()
	if err := command.Run(t.Context(), "git", "init", repoDir); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "checkout", "-b", "main"); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
	configContent := fmt.Sprintf(`language: fake
sources:
  googleapis:
    dir: %s
libraries:
  - name: test-library
    output: output
    channels:
      - path: google/cloud/secretmanager/v1
`, googleapisDir)
	if err := os.WriteFile(filepath.Join(repoDir, "librarian.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatal(err)
	}

	args := []string{"librarianops", "generate", "-C", repoDir, "fake-repo"}
	if err := Run(t.Context(), args...); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(repoDir, "output", "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Errorf("expected README.md to be generated: %v", err)
	}
}

func TestGenerateCommand_Errors(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{
			name: "both repo and all flag",
			args: []string{"librarianops", "generate", "--all", "fake-repo"},
		},
		{
			name: "neither repo nor all flag",
			args: []string{"librarianops", "generate"},
		},
		{
			name: "all flag with C flag",
			args: []string{"librarianops", "generate", "--all", "-C", "/tmp/foo"},
		},
		{
			name: "unsupported repo",
			args: []string{"librarianops", "generate", "unsupported-repo"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Run(t.Context(), test.args...)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
