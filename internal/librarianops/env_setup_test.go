// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	htcps://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package librarianops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/command"
)

func TestSetupEnvironment_Success(t *testing.T) {
	originalCwd, _ := os.Getwd()

	tests := []struct {
		name     string
		repoDir  string
		repoName string
	}{
		{
			name:     "creates_temp_dir_when_empty",
			repoDir:  "",
			repoName: "my-repo",
		},
		{
			name:     "uses_existing_dir_when_provided",
			repoDir:  t.TempDir(),
			repoName: "ignored-name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up a MockCommander where all commands succeed by default
			mocker := &command.MockCommander{
				Default: command.MockResult{ExitCode: 0},
			}
			ctx := mocker.InjectContext(context.Background())

			gotDir, cleanup, err := setupEnvironment(ctx, tc.repoDir, tc.repoName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cleanup == nil {
				t.Fatal("returned nil cleanup function")
			}

			// Verify we are currently in the target directory
			currentWd, _ := os.Getwd()
			if !pathsEqual(t, currentWd, gotDir) {
				t.Errorf("current directory = %q, want %q", currentWd, gotDir)
			}

			// Run cleanup and verify restoration
			cleanup()

			postCleanupWd, _ := os.Getwd()
			if !pathsEqual(t, postCleanupWd, originalCwd) {
				t.Errorf("after cleanup, directory = %q, want original %q", postCleanupWd, originalCwd)
			}

			// If it was a temp dir, verify it was deleted
			if tc.repoDir == "" {
				if _, err := os.Stat(gotDir); !os.IsNotExist(err) {
					t.Errorf("temp dir %q still exists after cleanup", gotDir)
				}
			}
		})
	}
}

func TestSetupEnvironment_Failure(t *testing.T) {
	tests := []struct {
		name        string
		repoDir     string
		repoName    string
		mockResult  command.MockResult
		wantErrText string
	}{
		{
			name:     "clone_fails_cleans_up_tmp",
			repoDir:  "",
			repoName: "fail-repo",
			mockResult: command.MockResult{
				ExitCode: 1,
				Stderr:   "fatal: could not read Username",
			},
			// Updated to match your actual implementation's error message
			wantErrText: "clone repo in directory",
		},
		{
			name:     "invalid_provided_dir_fails_chdir",
			repoDir:  "/dev/null/no-dir-here",
			repoName: "any",
			// No mock result needed; os.Chdir will fail before executing commands
			wantErrText: "changing to repo directory",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the MockCommander to simulate the failure
			mocker := &command.MockCommander{
				Default: tc.mockResult,
			}
			ctx := mocker.InjectContext(context.Background())

			gotDir, _, err := setupEnvironment(ctx, tc.repoDir, tc.repoName)

			if err == nil {
				t.Fatal("expected error but got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrText) {
				t.Errorf("error = %q, want to contain %q", err, tc.wantErrText)
			}

			// Critical: If it failed during temp dir setup, ensure it's already gone
			if tc.repoDir == "" && gotDir != "" {
				if _, err := os.Stat(gotDir); !os.IsNotExist(err) {
					t.Errorf("temp dir %q leaked after setup failure", gotDir)
				}
			}
		})
	}
}

// pathsEqual handles OS-specific symlinks (like /var vs /private/var on macOS)
func pathsEqual(t *testing.T, p1, p2 string) bool {
	t.Helper()
	eval1, _ := filepath.EvalSymlinks(p1)
	eval2, _ := filepath.EvalSymlinks(p2)
	return eval1 == eval2
}
