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
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

const (
	localRepoDir       = "../../testdata/e2e/generate/repo"
	localRepoBackupDir = "../../testdata/e2e/generate/repo_backup"
)

func TestRunGenerate(t *testing.T) {
	t.Parallel()
	rand.Seed(time.Now().UnixNano())
	for _, test := range []struct {
		name    string
		api     string
		source  string
		wantErr bool
	}{
		{
			name:   "testRunSuccess",
			api:    "google/cloud/pubsub/v1",
			source: "../../testdata/e2e/generate/api_root",
		},
		{
			name:    "testRunFailed",
			api:     "google/invalid/path",
			source:  "../../testdata/e2e/generate/api_root",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := fmt.Sprintf("%s-%d", localRepoDir, rand.Intn(10000))
			workRoot := filepath.Join(os.TempDir(), fmt.Sprintf("librarian-%d", rand.Intn(10000)))
			if err := prepareTest(repo, workRoot, localRepoBackupDir); err != nil {
				t.Fatalf("prepare test error = %v", err)
			}
			defer os.RemoveAll(repo)
			defer os.RemoveAll(workRoot)

			cmd := exec.Command(
				"../../librarian",
				"generate",
				fmt.Sprintf("--api=%s", test.api),
				fmt.Sprintf("--output=%s", workRoot),
				fmt.Sprintf("--repo=%s", repo),
				fmt.Sprintf("--source=%s", test.source),
			)
			_, err := cmd.CombinedOutput()

			if test.wantErr {
				if err == nil {
					t.Fatalf("%s should fail", test.name)
				}
			} else {
				if err != nil {
					t.Fatalf("librarian generate command error = %v", err)
				}
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

func prepareTest(repoDir, workRoot, backupDir string) error {
	if err := initTestRepo(repoDir, backupDir); err != nil {
		return err
	}
	if err := os.MkdirAll(workRoot, 0755); err != nil {
		return err
	}

	return nil
}

// initTestRepo initiates an empty git repo in the given directory, copy
// files from source directory and create a commit.
func initTestRepo(dir, source string) error {

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	storage := filesystem.NewStorage(osfs.New(filepath.Join(dir, ".git")), cache.NewObjectLRU(256))
	worktree := osfs.New(dir)
	localRepo, err := git.Init(storage, worktree)
	if err != nil {
		return err
	}

	workTree, err := localRepo.Worktree()
	if err != nil {
		return err
	}
	if err := copyDir(source, dir); err != nil {
		return err
	}
	if _, err := workTree.Add("."); err != nil {
		return err
	}

	if _, err := workTree.Commit("init test repo", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test user",
			Email: "test@github.com",
			When:  time.Now(),
		},
	}); err != nil {
		return err
	}

	return nil
}

// copyDir recursively copies a directory from src to dest.
func copyDir(src string, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		if err := os.MkdirAll(dest, srcInfo.Mode()); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			// If it's a directory, recursively call CopyDir
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			// If it's a file, copy the file
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile copies a single file from src to dest.
func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the file contents
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	// Get source file info to copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Set destination file permissions
	if err := os.Chmod(dest, srcInfo.Mode()); err != nil {
		return err
	}

	return nil
}

type genResponse struct {
	ErrorMessage string `json:"error,omitempty"`
}
