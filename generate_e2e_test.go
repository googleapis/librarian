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

//go:build e2e
// +build e2e

package librarian

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"gopkg.in/yaml.v3"
)

func TestRunGenerate(t *testing.T) {
	const (
		initialRepoStateDir = "testdata/e2e/generate/repo_init"
		localAPISource      = "testdata/e2e/generate/api_root"
	)
	t.Parallel()
	for _, test := range []struct {
		name    string
		api     string
		wantErr bool
	}{
		{
			name: "testRunSuccess",
			api:  "google/cloud/pubsub/v1",
		},
		{
			name:    "failed due to simulated error in generate command",
			api:     "google/cloud/future/v2",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workRoot := t.TempDir()
			repo := t.TempDir()
			APISourceRepo := t.TempDir()
			if err := initRepo(t, repo, initialRepoStateDir); err != nil {
				t.Fatalf("languageRepo prepare test error = %v", err)
			}
			if err := initRepo(t, APISourceRepo, localAPISource); err != nil {
				t.Fatalf("APISouceRepo prepare test error = %v", err)
			}

			cmd := exec.Command(
				"go",
				"run",
				"github.com/googleapis/librarian/cmd/librarian",
				"generate",
				fmt.Sprintf("--api=%s", test.api),
				fmt.Sprintf("--output=%s", workRoot),
				fmt.Sprintf("--repo=%s", repo),
				fmt.Sprintf("--api-source=%s", APISourceRepo),
			)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			err := cmd.Run()
			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should fail", test.name)
				}

				// the exact message is not populated here, but we can check it's
				// indeed an error returned from docker container.
				if g, w := err.Error(), "exit status 1"; !strings.Contains(g, w) {
					t.Fatalf("got %q, wanted it to contain %q", g, w)
				}

				return
			}

			if err != nil {
				t.Fatalf("librarian generate command error = %v", err)
			}
		})
	}
}

func TestRunConfigure(t *testing.T) {
	const (
		localRepoDir        = "testdata/e2e/configure/repo"
		initialRepoStateDir = "testdata/e2e/configure/repo_init"
	)
	t.Parallel()
	for _, test := range []struct {
		name         string
		api          string
		library      string
		apiSource    string
		updatedState string
		wantErr      bool
	}{
		{
			name:         "runs successfully",
			api:          "google/cloud/new-library-path/v2",
			library:      "new-library",
			apiSource:    "testdata/e2e/configure/api_root",
			updatedState: "testdata/e2e/configure/updated-state.yaml",
		},
		{
			name:         "failed due to simulated error in configure command",
			api:          "google/cloud/another-library/v3",
			library:      "simulate-command-error-id",
			apiSource:    "testdata/e2e/configure/api_root",
			updatedState: "testdata/e2e/configure/updated-state.yaml",
			wantErr:      true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workRoot := t.TempDir()
			repo := t.TempDir()
			APISourceRepo := t.TempDir()
			if err := initRepo(t, repo, initialRepoStateDir); err != nil {
				t.Fatalf("prepare test error = %v", err)
			}
			if err := initRepo(t, APISourceRepo, test.apiSource); err != nil {
				t.Fatalf("APISouceRepo prepare test error = %v", err)
			}

			cmd := exec.Command(
				"go",
				"run",
				"github.com/googleapis/librarian/cmd/librarian",
				"generate",
				fmt.Sprintf("--api=%s", test.api),
				fmt.Sprintf("--output=%s", workRoot),
				fmt.Sprintf("--repo=%s", repo),
				fmt.Sprintf("--api-source=%s", APISourceRepo),
				fmt.Sprintf("--library=%s", test.library),
			)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			err := cmd.Run()
			if test.wantErr {
				if err == nil {
					t.Fatal("Configure command should fail")
				}

				// the exact message is not populated here, but we can check it's
				// indeed an error returned from docker container.
				if g, w := err.Error(), "exit status 1"; !strings.Contains(g, w) {
					t.Errorf("got %q, wanted it to contain %q", g, w)
				}
				return
			}
			if err != nil {
				t.Fatalf("Failed to run configure: %v", err)
			}

			// Verify the file content
			gotBytes, err := os.ReadFile(filepath.Join(repo, ".librarian", "state.yaml"))
			if err != nil {
				t.Fatalf("Failed to read configure response file: %v", err)
			}
			wantBytes, readErr := os.ReadFile(test.updatedState)
			if readErr != nil {
				t.Fatalf("Failed to read expected state for comparison: %v", readErr)
			}
			var gotState *config.LibrarianState
			if err := yaml.Unmarshal(gotBytes, &gotState); err != nil {
				t.Fatalf("Failed to unmarshal configure response file: %v", err)
			}
			var wantState *config.LibrarianState
			if err := yaml.Unmarshal(wantBytes, &wantState); err != nil {
				t.Fatalf("Failed to unmarshal expected state: %v", err)
			}

			if diff := cmp.Diff(wantState, gotState, cmpopts.IgnoreFields(config.LibraryState{}, "LastGeneratedCommit")); diff != "" {
				t.Fatalf("Generated yaml mismatch (-want +got):\n%s", diff)
			}
			for _, lib := range gotState.Libraries {
				if lib.ID == test.library && lib.LastGeneratedCommit == "" {
					t.Fatal("LastGeneratedCommit should not be empty")
				}
			}

		})
	}
}

// initRepo initiates a git repo in the given directory, copy
// files from source directory and create a commit.
func initRepo(t *testing.T, dir, source string) error {
	if err := os.CopyFS(dir, os.DirFS(source)); err != nil {
		return err
	}
	runGit(t, dir, "init")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "config", "user.email", "test@github.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "commit", "-m", "init test repo")
	runGit(t, dir, "remote", "add", "origin", "https://github.com/googleapis/librarian.git")
	return nil
}

type genResponse struct {
	ErrorMessage string `json:"error,omitempty"`
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}
