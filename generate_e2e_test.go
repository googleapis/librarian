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
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	repo               = "repo"
	localRepoBackupDir = "testdata/e2e/generate/repo_backup"
	localAPISource     = "testdata/e2e/generate/api_root"
)

func TestRunGenerate(t *testing.T) {
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
			name:    "non existant in api source",
			api:     "google/invalid/path",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workRoot := filepath.Join(t.TempDir())
			repo := filepath.Join(workRoot, repo)
			if err := prepareTest(t, repo, workRoot, localRepoBackupDir); err != nil {
				t.Fatalf("prepare test error = %v", err)
			}

			cmd := exec.Command(
				"go",
				"run",
				"github.com/googleapis/librarian/cmd/librarian",
				"generate",
				fmt.Sprintf("--api=%s", test.api),
				fmt.Sprintf("--output=%s", workRoot),
				fmt.Sprintf("--repo=%s", repo),
				fmt.Sprintf("--api-source=%s", localAPISource),
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
