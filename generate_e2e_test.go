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
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRunGenerate(t *testing.T) {
	const (
		repo                = "repo"
		initialRepoStateDir = "testdata/e2e/generate/repo_init"
		APISourceRepo       = "apisource"
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
			name:    "non existent in api source",
			api:     "google/non-existent/path",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workRoot := filepath.Join(t.TempDir())
			repo := filepath.Join(workRoot, repo)
			APISourceRepo := filepath.Join(workRoot, APISourceRepo)
			if err := prepareTest(t, repo, workRoot, initialRepoStateDir); err != nil {
				t.Fatalf("languageRepo prepare test error = %v", err)
			}
			if err := prepareTest(t, APISourceRepo, workRoot, localAPISource); err != nil {
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
				return
			}
			if err != nil {
				t.Fatalf("librarian generate command error = %v", err)
			}

			responseFile := filepath.Join(workRoot, "output", "generate-response.json")
			if _, err := os.Stat(responseFile); err != nil {
				t.Fatalf("can not find generate response, error = %v", err)
			}

			if test.wantErr {
				data, err := os.ReadFile(responseFile)
				if err != nil {
					t.Fatalf("ReadFile() error = %v", err)
				}
				content := &genResponse{}
				if err := json.Unmarshal(data, content); err != nil {
					t.Fatalf("Unmarshal() error = %v", err)
				}
				if content.ErrorMessage == "" {
					t.Fatalf("can not find error message in generate response")
				}
			}
		})
	}
}

func TestRunConfigure(t *testing.T) {
	const (
		localRepoDir        = "testdata/e2e/configure/repo"
		initialRepoStateDir = "testdata/e2e/configure/repo_init"
		repo                = "repo"
		APISourceRepo       = "apisource"
	)
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
			library:      "simulate-configure-error-id",
			apiSource:    "testdata/e2e/configure/api_root",
			updatedState: "testdata/e2e/configure/updated-state.yaml",
			wantErr:      true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			workRoot := filepath.Join(os.TempDir(), fmt.Sprintf("rand-%d", rand.Intn(1000)))
			repo := filepath.Join(workRoot, repo)
			APISourceRepo := filepath.Join(workRoot, APISourceRepo)
			if err := prepareTest(t, repo, workRoot, initialRepoStateDir); err != nil {
				t.Fatalf("prepare test error = %v", err)
			}
			if err := prepareTest(t, APISourceRepo, workRoot, test.apiSource); err != nil {
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

			if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
				t.Errorf("Generated yaml mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func prepareTest(t *testing.T, destRepoDir, workRoot, sourceRepoDir string) error {
	if err := initTestRepo(t, destRepoDir, sourceRepoDir); err != nil {
		return err
	}
	if err := os.MkdirAll(workRoot, 0755); err != nil {
		return err
	}

	return nil
}

// initTestRepo initiates an empty git repo in the given directory, copy
// files from source directory and create a commit.
func initTestRepo(t *testing.T, dir, source string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.CopyFS(dir, os.DirFS(source)); err != nil {
		return err
	}
	runGit(t, dir, "init")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "config", "user.email", "test@github.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "commit", "-m", "init test repo")
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
