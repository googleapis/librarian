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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp"
)

// TODO(https://github.com/googleapis/librarian/issues/202): add better tests
// for librarian.Run.
func TestRun(t *testing.T) {
	if err := Run(t.Context(), []string{"version"}...); err != nil {
		log.Fatal(err)
	}
}

func TestIsURL(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Valid HTTPS URL",
			input: "https://github.com/googleapis/librarian-go",
			want:  true,
		},
		{
			name:  "Valid HTTP URL",
			input: "http://example.com/path?query=value",
			want:  true,
		},
		{
			name:  "Valid FTP URL",
			input: "ftp://user:password@host/path",
			want:  true,
		},
		{
			name:  "URL without scheme",
			input: "google.com",
			want:  false,
		},
		{
			name:  "URL with scheme only",
			input: "https://",
			want:  false,
		},
		{
			name:  "Absolute Unix file path",
			input: "/home/user/file",
			want:  false,
		},
		{
			name:  "Relative file path",
			input: "home/user/file",
			want:  false,
		},
		{
			name:  "Empty string",
			input: "",
			want:  false,
		},
		{
			name:  "Plain string",
			input: "just-a-string",
			want:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isURL(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("isURL() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

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
		// Create a state.yaml file.
		stateDir := filepath.Join(dir, config.LibrarianDir)
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll: %v", err)
		}

		state := &config.LibrarianState{
			Image: "some/image:v1.2.3",
			Libraries: []*config.LibraryState{
				{
					ID: "some-library",
					APIs: []*config.API{
						{
							Path:          "some/api",
							ServiceConfig: "api_config.yaml",
							Status:        config.StatusExisting,
						},
					},
					SourceRoots: []string{"src/a"},
				},
			},
		}

		bytes, err := yaml.Marshal(state)
		if err != nil {
			t.Fatalf("yaml.Marshal() = %v", err)
		}
		stateFile := filepath.Join(stateDir, "state.yaml")
		if err := os.WriteFile(stateFile, bytes, 0644); err != nil {
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
