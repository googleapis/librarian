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

package librarian

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"gopkg.in/yaml.v3"
)

// regexps for parsing commit messages
var nestedCommitRegex = regexp.MustCompile(`(?s)BEGIN_NESTED_COMMIT\n(.*?)\nEND_NESTED_COMMIT`)
var conventionalCommitRegex = regexp.MustCompile(`^(feat|fix|docs|chore): (.+)$`)

func TestRunGenerate(t *testing.T) {
	const (
		initialRepoStateDir = "testdata/e2e/generate/repo_init"
		localAPISource      = "testdata/e2e/generate/api_root"
	)
	t.Parallel()
	for _, test := range []struct {
		name              string
		api               string
		push              bool
		wantErr           bool
		wantInPrBody      []string
		doNotWantInPrBody []string
		commits           []struct {
			message   string
			fileToAdd string
		}
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
		{
			name: "testRunSuccess with push",
			api:  "google/cloud/pubsub/v1",
			push: true,
			commits: []struct {
				message   string
				fileToAdd string
			}{
				{
					message:   "feat: new feature pubsub",
					fileToAdd: "google/cloud/pubsub/v1/pubsub.code",
				},
				{
					message:   "feat: new feature future",
					fileToAdd: "google/cloud/future/v2/future.code",
				},
			},
			wantInPrBody:      []string{"feat: new feature pubsub"},
			doNotWantInPrBody: []string{"feat: new feature future"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workRoot := t.TempDir()
			repo := t.TempDir()
			apiSourceRepo := t.TempDir()
			if err := initRepo(t, repo, initialRepoStateDir, "initial commit"); err != nil {
				t.Fatalf("languageRepo prepare test error = %v", err)
			}
			if err := initRepo(t, apiSourceRepo, localAPISource, "initial commit"); err != nil {
				t.Fatalf("APISouceRepo prepare test error = %v", err)
			}
			if test.push {
				// Create a local bare repository to act as the remote for the push.
				bareRepoDir := filepath.Join(t.TempDir(), "remote.git")
				if err := os.MkdirAll(bareRepoDir, 0755); err != nil {
					t.Fatalf("Failed to create bare repo dir: %v", err)
				}
				runGit(t, bareRepoDir, "init", "--bare")
				runGit(t, repo, "remote", "set-url", "origin", bareRepoDir)
				// Setup state.yaml file with initial last_generated_commit, so it picks up the commits we create below.
				apiSourceSHA := runGit(t, apiSourceRepo, "rev-parse", "HEAD")
				setupStateFile(t, repo, apiSourceSHA)
				runGit(t, repo, "commit", "-a", "-m", "populate yaml file with latest googleapis commit")
				// Create commits in the api source repo to be picked in generation PR body.
				for _, commit := range test.commits {
					createCommit(t, apiSourceRepo, commit.fileToAdd, commit.message)
				}
			}
			// Setup mock GitHub server.
			server := newMockGitHubServer(t, "generate", test.wantInPrBody, test.doNotWantInPrBody)
			defer server.Close()
			cmdArgs := []string{
				"run",
				"-tags", "e2etest",
				"github.com/googleapis/librarian/cmd/librarian",
				"generate",
				fmt.Sprintf("--api=%s", test.api),
				fmt.Sprintf("--output=%s", workRoot),
				fmt.Sprintf("--repo=%s", repo),
				fmt.Sprintf("--api-source=%s", apiSourceRepo),
			}
			if test.push {
				cmdArgs = append(cmdArgs, "--push")
			}

			cmd := exec.Command("go", cmdArgs...)
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=fake-token", config.LibrarianGithubToken))
			cmd.Env = append(cmd.Env, "LIBRARIAN_GITHUB_BASE_URL="+server.URL)
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

	setupStateFile(t, repoInitDir, "")

	workRoot := t.TempDir()
	repo := t.TempDir()
	apiSourceRepo := t.TempDir()
	if err := initRepo(t, repo, repoInitDir, "initial commit"); err != nil {
		t.Fatalf("languageRepo prepare test error = %v", err)
	}
	if err := initRepo(t, apiSourceRepo, localAPISource, "initial commit"); err != nil {
		t.Fatalf("APISouceRepo prepare test error = %v", err)
	}

	cmd := exec.Command(
		"go",
		"run",
		"-tags", "e2etest",
		"github.com/googleapis/librarian/cmd/librarian",
		"generate",
		fmt.Sprintf("--api=%s", apiToGenerate),
		fmt.Sprintf("--output=%s", workRoot),
		fmt.Sprintf("--repo=%s", repo),
		fmt.Sprintf("--api-source=%s", apiSourceRepo),
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
			apiSourceRepo := t.TempDir()
			if err := initRepo(t, repo, initialRepoStateDir, "initial commit"); err != nil {
				t.Fatalf("prepare test error = %v", err)
			}
			if err := initRepo(t, apiSourceRepo, test.apiSource, "feat: add a new api\n\nPiperOrigin-RevId: 123456"); err != nil {
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
				fmt.Sprintf("--api-source=%s", apiSourceRepo),
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

func TestRunGenerate_MultipleLibraries(t *testing.T) {
	const localAPISource = "testdata/e2e/generate/api_root"

	for _, test := range []struct {
		name                string
		initialRepoStateDir string
		expectError         bool
		expectedFiles       []string
		unexpectedFiles     []string
	}{
		{
			name:                "Multiple libraries generated successfully",
			initialRepoStateDir: "testdata/e2e/generate/multi_repo_init",
			expectedFiles:       []string{"pubsub/example.txt", "future/example.txt"},
			unexpectedFiles:     []string{},
		},
		{
			name:                "One library fails to generate",
			initialRepoStateDir: "testdata/e2e/generate/multi_repo_one_fails_init",
			expectedFiles:       []string{"pubsub/example.txt"},
			unexpectedFiles:     []string{"future/example.txt"},
		},
		{
			name:                "All libraries fail to generate",
			initialRepoStateDir: "testdata/e2e/generate/multi_repo_all_fail_init",
			expectError:         true,
			expectedFiles:       []string{},
			unexpectedFiles:     []string{"future/example.txt", "another-future/example.txt"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workRoot := t.TempDir()
			repo := t.TempDir()
			apiSourceRepo := t.TempDir()

			if err := initRepo(t, repo, test.initialRepoStateDir, "initial commit"); err != nil {
				t.Fatalf("languageRepo prepare test error = %v", err)
			}
			if err := initRepo(t, apiSourceRepo, localAPISource, "initial commit"); err != nil {
				t.Fatalf("APISouceRepo prepare test error = %v", err)
			}

			cmd := exec.Command(
				"go",
				"run",
				"github.com/googleapis/librarian/cmd/librarian",
				"generate",
				fmt.Sprintf("--output=%s", workRoot),
				fmt.Sprintf("--repo=%s", repo),
				fmt.Sprintf("--api-source=%s", apiSourceRepo),
			)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			err := cmd.Run()

			if test.expectError {
				if err == nil {
					t.Fatal("librarian generate command should fail")
				}
				return
			}

			if err != nil {
				t.Fatalf("librarian generate command error = %v", err)
			}

			for _, f := range test.expectedFiles {
				if _, err := os.Stat(filepath.Join(repo, f)); os.IsNotExist(err) {
					t.Errorf("%s should have been copied", f)
				}
			}

			for _, f := range test.unexpectedFiles {
				if _, err := os.Stat(filepath.Join(repo, f)); !os.IsNotExist(err) {
					t.Errorf("%s should not have been copied", f)
				}
			}
		})
	}
}

func TestReleaseInit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		testDataDir string
		libraryID   string
		changePath  string
		tagID       string
		tagVersion  string
		tagFormat   string
		push        bool
		wantErr     bool
	}{
		{
			name:        "release with multiple commits",
			testDataDir: "testdata/e2e/release/init/multiple_commits",
			libraryID:   "go-google-cloud-pubsub-v1",
			changePath:  "google-cloud-pubsub/v1",
			tagID:       "go-google-cloud-pubsub-v1",
			tagVersion:  "1.0.0",
			tagFormat:   "%s/v%s",
		},
		{
			name:        "release with multiple commits with push",
			testDataDir: "testdata/e2e/release/init/multiple_commits",
			libraryID:   "go-google-cloud-pubsub-v1",
			changePath:  "google-cloud-pubsub/v1",
			tagID:       "go-google-cloud-pubsub-v1",
			tagVersion:  "1.0.0",
			tagFormat:   "%s/v%s",
			push:        true,
		},
		{
			name:        "release with multiple nested commits",
			testDataDir: "testdata/e2e/release/init/multiple_nested_commits",
			libraryID:   "python-google-cloud-video-live-stream-v1",
			changePath:  "packages/google-cloud-video-live-stream",
			tagID:       "python-google-cloud-video-live-stream-v1",
			tagVersion:  "v1.12.0",
			tagFormat:   "%s-%s", // Format for {id}-{version}
		},
		{
			name:        "release with single commit",
			testDataDir: "testdata/e2e/release/init/single_commit",
			libraryID:   "dlp",
			changePath:  "dlp",
			tagID:       "dlp",
			tagVersion:  "1.24.0",
			tagFormat:   "%s/v%s",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workRoot := t.TempDir()
			repo := t.TempDir()

			initialRepoStateDir := filepath.Join(test.testDataDir, "repo_init")
			updatedState := filepath.Join(test.testDataDir, "state.yaml")
			wantChangelog := filepath.Join(test.testDataDir, "CHANGELOG.md")
			commitMsgPath := filepath.Join(test.testDataDir, "commit_msg.txt")

			if err := initRepo(t, repo, initialRepoStateDir, "initial commit"); err != nil {
				t.Fatalf("prepare test error = %v", err)
			}

			if test.push {
				// Create a local bare repository to act as the remote for the push.
				bareRepoDir := filepath.Join(t.TempDir(), "remote.git")
				if err := os.MkdirAll(bareRepoDir, 0755); err != nil {
					t.Fatalf("Failed to create bare repo dir: %v", err)
				}
				runGit(t, bareRepoDir, "init", "--bare")
				runGit(t, repo, "remote", "set-url", "origin", bareRepoDir)
			}

			// Dynamically create the tag based on the format string
			tagName := fmt.Sprintf(test.tagFormat, test.tagID, test.tagVersion)
			runGit(t, repo, "tag", tagName)

			// Add a new commit to simulate a change.
			newFilePath := filepath.Join(test.changePath, "new-file.txt")
			if err := os.MkdirAll(filepath.Join(repo, test.changePath), 0755); err != nil {
				t.Fatalf("Failed to create directory for new file: %v", err)
			}
			commitMsgBytes, err := os.ReadFile(commitMsgPath)
			if err != nil {
				t.Fatalf("Failed to read commit message file: %v", err)
			}

			createCommit(t, repo, newFilePath, string(commitMsgBytes))

			prContentToMatch := parseCommitMessageForPRContent(string(commitMsgBytes))
			server := newMockGitHubServer(t, "release", prContentToMatch, []string{})
			defer server.Close()

			cmdArgs := []string{
				"run",
				"-tags", "e2etest",
				"github.com/googleapis/librarian/cmd/librarian",
				"release",
				"init",
				fmt.Sprintf("--repo=%s", repo),
				fmt.Sprintf("--output=%s", workRoot),
				fmt.Sprintf("--library=%s", test.libraryID),
			}
			if test.push {
				cmdArgs = append(cmdArgs, "--push")
			}

			cmd := exec.Command("go", cmdArgs...)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=fake-token", config.LibrarianGithubToken))
			cmd.Env = append(cmd.Env, "LIBRARIAN_GITHUB_BASE_URL="+server.URL)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to run release init: %v", err)
			}

			// Verify the state.yaml file content
			outputDir := filepath.Join(workRoot, "output")
			t.Logf("Checking for output file in: %s", filepath.Join(outputDir, ".librarian", "state.yaml"))
			gotBytes, err := os.ReadFile(filepath.Join(outputDir, ".librarian", "state.yaml"))
			if err != nil {
				t.Fatalf("Failed to read updated state.yaml from output directory: %v", err)
			}
			wantBytes, readErr := os.ReadFile(updatedState)
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

			// Use cmpopts.IgnoreFields to ignore the dynamic commit hash.
			if diff := cmp.Diff(wantState, gotState, cmpopts.IgnoreFields(config.LibraryState{}, "LastGeneratedCommit")); diff != "" {
				t.Fatalf("Generated yaml mismatch (-want +got): %s", diff)
			}

			// Verify the CHANGELOG.md file content
			gotChangelog, err := os.ReadFile(filepath.Join(outputDir, test.changePath, "CHANGELOG.md"))
			if err != nil {
				t.Fatalf("Failed to read CHANGELOG.md from output directory: %v", err)
			}
			wantChangelogBytes, err := os.ReadFile(wantChangelog)
			if err != nil {
				t.Fatalf("Failed to read expected changelog for comparison: %v", err)
			}
			if diff := cmp.Diff(string(wantChangelogBytes), string(gotChangelog)); diff != "" {
				t.Fatalf("Generated changelog mismatch (-want +got): %s", diff)
			}
		})
	}
}

func TestReleaseTagAndRelease(t *testing.T) {
	for _, test := range []struct {
		name     string
		prBody   string
		repoPath string
		repoURL  string
		push     bool
		wantErr  bool
	}{
		{
			name: "runs successfully",
			prBody: `<details><summary>go-google-cloud-pubsub-v1: v1.0.1</summary>
		### Features
		- feat: new feature
		</details>`,
			repoURL: "https://github.com/googleapis/librarian",
		},
		{
			name: "runs successfully from cloned repo",
			prBody: `<details><summary>go-google-cloud-pubsub-v1: v1.0.1</summary>
### Features
- feat: new feature
</details>`,
			repoPath: "testdata/e2e/release/init/single_commit",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			headSHA := "abcdef123456"

			// Set up a mock GitHub API server using httptest.
			// This server will intercept HTTP requests made by the librarian command
			// and provide canned responses, avoiding any real calls to the GitHub API.
			// The handlers below simulate the endpoints that 'release tag-and-release' interacts with.
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify that the GitHub token is being sent correctly.
				if r.Header.Get("Authorization") != "Bearer fake-token" {
					t.Errorf("missing or wrong authorization header: got %q", r.Header.Get("Authorization"))
				}

				const stateYAMLContent = `
image: gcr.io/some-project/some-image:latest
libraries:
- id: go-google-cloud-pubsub-v1
  source_roots:
  - google-cloud-pubsub/v1
  tag_format: go-google-cloud-pubsub-v1-{version}
`
				// The download URL can be any unique path. The mock server will handle it.
				downloadURL := server.URL + "/raw/librarian/state.yaml"

				// Handler for the .librarian DIRECTORY listing request.
				// The client sends this to find the state.yaml file.
				if r.Method == "GET" && r.URL.Path == "/repos/googleapis/librarian/contents/.librarian" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					// CRITICAL: The response for the directory listing must include the `download_url` for the file.
					fmt.Fprintf(w, `[{"name": "state.yaml", "path": ".librarian/state.yaml", "type": "file", "download_url": %q}]`, downloadURL)
					return
				}

				// Handler for the raw CONTENT download request.
				// The client hits this endpoint after extracting the download_url from the directory listing.
				if r.Method == "GET" && r.URL.Path == "/raw/librarian/state.yaml" {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, stateYAMLContent)
					return
				}

				// Mock endpoint for the .librarian directory listing.
				// This handles the preliminary request the GitHub client makes before fetching a file.
				if r.Method == "GET" && r.URL.Path == "/repos/googleapis/librarian/contents/.librarian" {
					w.WriteHeader(http.StatusOK)
					// This response tells the client that the directory contains a file named state.yaml
					fmt.Fprint(w, `[{"name": "state.yaml", "path": ".librarian/state.yaml", "type": "file"}]`)
					return
				}

				// Mock endpoint for GET /.librarian/state.yaml
				if r.Method == "GET" && strings.HasSuffix(r.URL.Path, ".librarian/state.yaml") {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, stateYAMLContent)
					return
				}

				// Mock endpoint for GET /repos/{owner}/{repo}/pulls/{number}
				if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pulls/123") {
					w.WriteHeader(http.StatusOK)
					// Return a minimal PR object with the body and merge commit SHA.
					fmt.Fprintf(w, `{"number": 123, "body": %q, "merge_commit_sha": %q, "base": {"ref": "main"}}`, test.prBody, headSHA)
					return
				}

				// Mock endpoint for POST /repos/{owner}/{repo}/git/refs (creating the release-please tag)
				if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/git/refs") {
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"ref": "refs/tags/release-please-123"}`)
					return
				}

				// Mock endpoint for POST /repos/{owner}/{repo}/releases (creating the GitHub Release)
				if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/releases") {
					var newRelease github.RepositoryRelease
					if err := json.NewDecoder(r.Body).Decode(&newRelease); err != nil {
						t.Fatalf("failed to decode request body: %v", err)
					}
					expectedTagName := "go-google-cloud-pubsub-v1-v1.0.1"
					if *newRelease.TagName != expectedTagName {
						t.Errorf("unexpected tag name: got %q, want %q", *newRelease.TagName, expectedTagName)
					}
					if *newRelease.TargetCommitish != headSHA {
						t.Errorf("unexpected commitish: got %q, want %q", *newRelease.TargetCommitish, headSHA)
					}
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"name": "v1.0.1"}`)
					return
				}

				// Mock endpoint for PUT /repos/{owner}/{repo}/issues/{number}/labels (updating labels)
				if r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "/issues/123/labels") {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `[]`)
					return
				}

				// If any other request is made, fail the test.
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}))
			defer server.Close()

			repo := test.repoURL
			if test.repoPath != "" {
				repo = t.TempDir()
				err := initRepo(t, repo, test.repoPath, "initial commit")
				if err != nil {
					t.Fatalf("error initializing fake git repo %s", err)
				}
			}
			cmdArgs := []string{
				"run",
				"github.com/googleapis/librarian/cmd/librarian",
				"release",
				"tag-and-release",
				fmt.Sprintf("--repo=%s", repo),
				fmt.Sprintf("--github-api-endpoint=%s/", server.URL),
				"--pr=https://github.com/googleapis/librarian/pull/123",
			}
			if test.push {
				cmdArgs = append(cmdArgs, "--push")
			}

			cmd := exec.Command("go", cmdArgs...)
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=fake-token", config.LibrarianGithubToken))
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			if err := cmd.Run(); err != nil {
				if !test.wantErr {
					t.Fatalf("Failed to run release tag-and-release: %v", err)
				}
			}
		})
	}
}

// newMockGitHubServer creates a mock GitHub API server for testing --push functionality.
func newMockGitHubServer(t *testing.T, prTitleFragment string, expectedContentInPr []string, notExpectedContentInPr []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fake-token" {
			t.Errorf("missing or wrong authorization header: got %q", r.Header.Get("Authorization"))
		}

		// Mock endpoint for POST /repos/{owner}/{repo}/pulls
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pulls") {
			var newPR github.NewPullRequest
			if err := json.NewDecoder(r.Body).Decode(&newPR); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			expectedTitle := fmt.Sprintf("chore: librarian %s pull request", prTitleFragment)
			if !strings.Contains(*newPR.Title, expectedTitle) {
				t.Errorf("unexpected PR title: got %q, want to contain %q", *newPR.Title, expectedTitle)
			}
			for _, expectedContent := range expectedContentInPr {
				htmlEscapedContent := html.EscapeString(expectedContent)
				if !strings.Contains(*newPR.Body, htmlEscapedContent) {
					t.Errorf("unexpected PR description: got %q, missing %q", *newPR.Body, htmlEscapedContent)
				}
			}
			for _, notExpectedContent := range notExpectedContentInPr {
				if strings.Contains(*newPR.Body, notExpectedContent) {
					t.Errorf("unexpected PR description: got %q,  should not contain %q", *newPR.Body, notExpectedContent)
				}
			}
			if *newPR.Base != "main" {
				t.Errorf("unexpected PR base: got %q", *newPR.Base)
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"number": 123, "html_url": "https://github.com/googleapis/librarian/pull/123"}`)
			return
		}

		// Mock endpoint for POST /repos/{owner}/{repo}/issues/{number}/labels
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/issues/123/labels") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[]`)
			return
		}

		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
}

// initRepo initiates a git repo in the given directory, copy
// files from source directory and create a commit.
func initRepo(t *testing.T, dir, source, message string) error {
	t.Logf("initializing repo, dir %s, source %s", dir, source)
	if err := os.CopyFS(dir, os.DirFS(source)); err != nil {
		return err
	}
	runGit(t, dir, "init")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "config", "user.email", "test@github.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "commit", "-m", message)
	runGit(t, dir, "remote", "add", "origin", "https://github.com/googleapis/librarian.git")
	return nil
}

type genResponse struct {
	ErrorMessage string `json:"error,omitempty"`
}

// runGit runs a git command in the specified directory and returns the trimmed output as a string.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to run git command: %s, %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// createCommit creates a new commit in the given repo directory with a dummy file using the specified commit message.
func createCommit(t *testing.T, dir, filePath string, commitMsg string) error {
	fileToCommit := filepath.Join(dir, filePath)

	file, err := os.OpenFile(fileToCommit, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", fileToCommit, err)
	}
	defer file.Close()
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", commitMsg)
	return nil
}

// setupStateFile creates a .librarian/state.yaml file in the given repo directory with the provided lastGeneratedCommit.
func setupStateFile(t *testing.T, repoInitDir, lastGeneratedCommit string) {
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
				LastGeneratedCommit: lastGeneratedCommit,
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
}

// parseCommitMessageForPRContent extracts lines that should be included in the PR description from commit message.
func parseCommitMessageForPRContent(content string) []string {
	matches := nestedCommitRegex.FindStringSubmatch(content)
	nestedContent := content
	if len(matches) > 1 {
		nestedContent = matches[1]
	}

	var result []string
	for _, line := range strings.Split(nestedContent, "\n") {
		if match := conventionalCommitRegex.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			result = append(result, match[2])
		}
	}

	return result
}
