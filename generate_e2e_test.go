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

func TestCleanAndCopy(t *testing.T) {
	const (
		localAPISource = "testdata/e2e/generate/api_root"
		apiToGenerate  = "google/cloud/pubsub/v1"
	)
	// create a temp directory for writing files, so we don't have to create testdata files.
	repoInitDir := t.TempDir()

	// within the source root, create a file to be removed,
	// then create a sub dir with 2 files, on of them should be preserved.
	pubsubDir := filepath.Join(repoInitDir, "pubsub")
	if err := os.MkdirAll(filepath.Join(pubsubDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pubsubDir, "file_to_remove.txt"), []byte("remove me"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pubsubDir, "sub", "file_to_preserve.txt"), []byte("preserve me"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pubsubDir, "sub", "another_file_to_remove.txt"), []byte("remove me"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a file outside the source root to ensure it's not touched.
	otherDir := filepath.Join(repoInitDir, "other_dir")
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, "file_to_keep.txt"), []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}

	// create a state file with remove and preserve regex.
	state := &config.LibrarianState{
		Image: "test-image:latest",
		Libraries: []*config.LibraryState{
			{
				ID:      "go-google-cloud-pubsub-v1",
				Version: "v1.0.0",
				APIs: []*config.API{
					{
						Path: "google/cloud/pubsub/v1",
					},
				},
				SourceRoots: []string{"pubsub"},
				RemoveRegex: []string{
					"pubsub/file_to_remove.txt",
					"^pubsub/sub/.*.txt",
				},
				PreserveRegex: []string{
					"pubsub/sub/file_to_preserve.txt",
				},
			},
		},
	}
	stateBytes, err := yaml.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoInitDir, ".librarian"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoInitDir, ".librarian", "state.yaml"), stateBytes, 0644); err != nil {
		t.Fatal(err)
	}

	workRoot := t.TempDir()
	repo := t.TempDir()
	APISourceRepo := t.TempDir()
	if err := initRepo(t, repo, repoInitDir); err != nil {
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
		fmt.Sprintf("--api=%s", apiToGenerate),
		fmt.Sprintf("--output=%s", workRoot),
		fmt.Sprintf("--repo=%s", repo),
		fmt.Sprintf("--api-source=%s", APISourceRepo),
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("librarian generate command error = %v", err)
	}

	// Check that the file to remove is gone.
	if _, err := os.Stat(filepath.Join(repo, "pubsub", "file_to_remove.txt")); !os.IsNotExist(err) {
		t.Errorf("pubsub/file_to_remove.txt should have been removed")
	}
	// Check that the other file to remove is gone.
	if _, err := os.Stat(filepath.Join(repo, "pubsub", "sub", "another_file_to_remove.txt")); !os.IsNotExist(err) {
		t.Errorf("pubsub/sub/another_file_to_remove.txt should have been removed")
	}
	// Check that the file to preserve is still there.
	if _, err := os.Stat(filepath.Join(repo, "pubsub", "sub", "file_to_preserve.txt")); os.IsNotExist(err) {
		t.Errorf("pubsub/sub/file_to_preserve.txt should have been preserved")
	}
	// Check that the file outside the source root is still there.
	if _, err := os.Stat(filepath.Join(repo, "other_dir", "file_to_keep.txt")); os.IsNotExist(err) {
		t.Errorf("other_dir/file_to_keep.txt should have been preserved")
	}
	// check that the new files are copied. The fake generator creates a file called "example.txt".
	if _, err := os.Stat(filepath.Join(repo, "pubsub", "example.txt")); os.IsNotExist(err) {
		t.Errorf("pubsub/example.txt should have been copied")
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
