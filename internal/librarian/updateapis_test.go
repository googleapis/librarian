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
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

const (
	failingLibraryID    = "Google.Failing.V1"
	succeedingLibraryID = "Google.Succeeding.V1"
)

// fakeDockerScript stands in for the docker binary. Every container command
// succeeds without output, except build-library for failingLibraryID.
const fakeDockerScript = `#!/bin/sh
build=false
failing=false
for arg in "$@"; do
  case "$arg" in
    build-library) build=true ;;
    --library-id=` + failingLibraryID + `) failing=true ;;
  esac
done
if [ "$build" = true ] && [ "$failing" = true ]; then
  echo "fake build failure" >&2
  exit 1
fi
exit 0
`

// installFakeDocker puts a fake docker binary first on PATH for this test.
func installFakeDocker(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "docker"), []byte(fakeDockerScript), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// runGit runs a git command in dir and returns its trimmed stdout.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// writeAndCommit writes content to relPath in the repo at dir and commits it.
func writeAndCommit(t *testing.T, dir, relPath, content, msg string) string {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", msg)
	return runGit(t, dir, "rev-parse", "HEAD")
}

func TestUpdateLibrary_BuildFailureDoesNotLeakLastGeneratedCommit(t *testing.T) {
	ctx := context.Background()
	installFakeDocker(t)

	// The API repo has a base commit, then one new commit for each library.
	apiDir := newTestGitRepoWithCommit(t, "")
	baseAPICommit := writeAndCommit(t, apiDir, "google/base.txt", "base", "base")
	failingAPICommit := writeAndCommit(t, apiDir, "google/failing/v1/api.proto", "failing", "change failing API")
	succeedingAPICommit := writeAndCommit(t, apiDir, "google/succeeding/v1/api.proto", "succeeding", "change succeeding API")
	apiRepo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: apiDir})
	if err != nil {
		t.Fatalf("NewRepository(api) error = %v", err)
	}

	// The failing library comes first, as Compute.V1 came before NetworkServices.V1 in b/563001842.
	pipelineState := &statepb.PipelineState{
		Libraries: []*statepb.LibraryState{
			{
				Id:                        failingLibraryID,
				GenerationAutomationLevel: statepb.AutomationLevel_AUTOMATION_LEVEL_AUTOMATIC,
				ApiPaths:                  []string{"google/failing/v1"},
				LastGeneratedCommit:       baseAPICommit,
			},
			{
				Id:                        succeedingLibraryID,
				GenerationAutomationLevel: statepb.AutomationLevel_AUTOMATION_LEVEL_AUTOMATIC,
				ApiPaths:                  []string{"google/succeeding/v1"},
				LastGeneratedCommit:       baseAPICommit,
			},
		},
	}

	languageDir := newTestGitRepoWithCommit(t, "")
	languageRepo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: languageDir})
	if err != nil {
		t.Fatalf("NewRepository(language) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(languageDir, config.GeneratorInputDir), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	workRoot := t.TempDir()
	containerConfig, err := docker.New(workRoot, "fake-image", "", "", "", nil)
	if err != nil {
		t.Fatalf("docker.New() error = %v", err)
	}
	state := &commandState{
		workRoot:        workRoot,
		languageRepo:    languageRepo,
		pipelineState:   pipelineState,
		containerConfig: containerConfig,
	}
	if err := savePipelineState(state); err != nil {
		t.Fatalf("savePipelineState() error = %v", err)
	}
	runGit(t, languageDir, "add", "-A")
	runGit(t, languageDir, "commit", "-m", "initial state")
	initialLanguageHead := runGit(t, languageDir, "rev-parse", "HEAD")

	outputRoot := filepath.Join(workRoot, "output")
	for _, dir := range []string{outputRoot, filepath.Join(workRoot, config.GeneratorInputDir)} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
	}

	// Capture logs to check that the real build error is reported.
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	cfg := &config.Config{GitUserName: "tester", GitUserEmail: "tester@example.com"}
	prContent := new(PullRequestContent)
	for _, library := range pipelineState.Libraries {
		if err := updateLibrary(ctx, state, cfg, apiRepo, outputRoot, library, prContent); err != nil {
			t.Fatalf("updateLibrary(%s) error = %v", library.Id, err)
		}
	}

	wantErrors := []string{"Error while building " + failingLibraryID}
	if strings.Join(prContent.Errors, "|") != strings.Join(wantErrors, "|") {
		t.Errorf("prContent.Errors = %v, want %v", prContent.Errors, wantErrors)
	}
	wantSuccesses := []string{"Generated " + succeedingLibraryID}
	if strings.Join(prContent.Successes, "|") != strings.Join(wantSuccesses, "|") {
		t.Errorf("prContent.Successes = %v, want %v", prContent.Successes, wantSuccesses)
	}

	// Only the succeeding library has a commit on top of the initial state.
	if got := runGit(t, languageDir, "rev-list", "--count", initialLanguageHead+"..HEAD"); got != "1" {
		t.Errorf("commits after initial state = %s, want 1", got)
	}

	// The committed state must keep the old commit for the failing library.
	savedState, err := loadRepoPipelineState(languageRepo)
	if err != nil {
		t.Fatalf("loadRepoPipelineState() error = %v", err)
	}
	wantCommits := map[string]string{
		failingLibraryID:    baseAPICommit,
		succeedingLibraryID: succeedingAPICommit,
	}
	for _, library := range savedState.Libraries {
		if got, want := library.LastGeneratedCommit, wantCommits[library.Id]; got != want {
			t.Errorf("saved LastGeneratedCommit for %s = %s, want %s", library.Id, got, want)
		}
	}
	// The in-memory state must also be restored.
	if got := pipelineState.Libraries[0].LastGeneratedCommit; got != baseAPICommit {
		t.Errorf("in-memory LastGeneratedCommit for %s = %s, want %s (not %s)", failingLibraryID, got, baseAPICommit, failingAPICommit)
	}

	if !strings.Contains(logs.String(), "exit status 1") {
		t.Errorf("logs do not contain the build error; got:\n%s", logs.String())
	}
}
