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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRunUpgrade(t *testing.T) {
	t.Parallel()

	frozenTime := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)

	wantVersion := "v0.100.0"
	mocker := &command.FakeCommander{
		FakeResults: map[string]command.FakeResult{
			command.FormatCmd("go", "list", "-m", "-json", "github.com/googleapis/librarian@main"): {Stdout: `{"Version": "v0.100.0"}`},
		},
	}

	ctx := mocker.InjectContext(t.Context())
	ctx = context.WithValue(ctx, timeContextKey{}, frozenTime)

	repoDir := t.TempDir()
	configPath := generateLibrarianConfigPath(t, repoDir)
	initialConfig := sample.Config()
	initialConfig.Language = config.LanguageFake
	initialConfig.Version = "v0.1.0"
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatal(err)
	}

	gotVersion, err := runUpgrade(ctx, "fake-repo", repoDir)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantVersion, gotVersion); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotConfig, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatal(err)
	}

	wantConfig := initialConfig
	wantConfig.Version = wantVersion
	if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Pass the context so the test generates the exact same branch name
	githubDetails := createGithubDetailsForUpgrade(ctx, wantVersion)
	branchName := githubDetails.BranchName

	// Updated to match the actual execution chain
	wantCommands := [][]string{
		{"go", "list", "-m", "-json", "github.com/googleapis/librarian@main"},
		{"go", "run", fmt.Sprintf("github.com/googleapis/librarian/cmd/librarian@%s", wantVersion), "generate", "--all"},
		{"git", "checkout", "-b", branchName},
		{"git", "add", "."},
		{"git", "commit", "-m", githubDetails.PrTitle},
		{"git", "push", "-u", "origin", "HEAD"},
		{"gh", "pr", "create", "--title", githubDetails.PrTitle, "--body", githubDetails.PrBody},
	}

	// Check commands executed safely using the GotCommands field
	if diff := cmp.Diff(wantCommands, mocker.GotCommands); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunUpgrade_Error(t *testing.T) {
	t.Parallel()

	frozenTime := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name           string
		setup          func(t *testing.T, mocker *command.FakeCommander) (repoDir string)
		wantErrMessage string
	}{
		{
			name: "getLibrarianVersionAtMain error",
			setup: func(t *testing.T, mocker *command.FakeCommander) string {
				mocker.FakeResults = map[string]command.FakeResult{
					command.FormatCmd("go", "list", "-m", "-json", "github.com/googleapis/librarian@main"): {Error: fmt.Errorf("error")},
				}
				return t.TempDir()
			},
			wantErrMessage: "failed to get latest librarian version",
		},
		{
			name: "UpdateLibrarianVersion error",
			setup: func(t *testing.T, mocker *command.FakeCommander) string {
				mocker.FakeResults = map[string]command.FakeResult{
					command.FormatCmd("go", "list", "-m", "-json", "github.com/googleapis/librarian@main"): {Stdout: `{"Version": "v0.100.0"}`},
				}
				repoDir := t.TempDir()
				configPath := generateLibrarianConfigPath(t, repoDir)
				// Writing a directory where a file is expected to force an update error
				if err := os.Mkdir(configPath, 0755); err != nil {
					t.Fatal(err)
				}
				return repoDir
			},
			wantErrMessage: "failed to update librarian version",
		},
		{
			name: "setupEnvironment error",
			setup: func(t *testing.T, mocker *command.FakeCommander) string {
				mocker.FakeResults = map[string]command.FakeResult{
					command.FormatCmd("go", "list", "-m", "-json", "github.com/googleapis/librarian@main"): {Stdout: `{"Version": "v0.100.0"}`},
				}
				mocker.Default = command.FakeResult{Error: fmt.Errorf("git clone failed")}
				// an empty repoDir should trigger setupEnvironment to clone
				return ""
			},
			wantErrMessage: "failed to setup environment",
		},
		{
			name: "runLibrarianWithVersion error",
			setup: func(t *testing.T, mocker *command.FakeCommander) string {
				wantVersion := "v0.100.0"
				mocker.FakeResults = map[string]command.FakeResult{
					command.FormatCmd("go", "list", "-m", "-json", "github.com/googleapis/librarian@main"):                                            {Stdout: fmt.Sprintf(`{"Version": "%s"}`, wantVersion)},
					command.FormatCmd("go", "run", fmt.Sprintf("github.com/googleapis/librarian/cmd/librarian@%s", wantVersion), "generate", "--all"): {Error: fmt.Errorf("error")},
				}
				repoDir := t.TempDir()
				configPath := generateLibrarianConfigPath(t, repoDir)
				// Use an invalid config that will cause `librarian generate` to fail.
				// An empty language should be invalid.
				cfg := &config.Config{Language: config.LanguagePhp}
				if err := yaml.Write(configPath, cfg); err != nil {
					t.Fatal(err)
				}
				return repoDir
			},
			wantErrMessage: "failed to run librarian generate",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mocker := &command.FakeCommander{}
			repoDir := tc.setup(t, mocker)

			ctx := mocker.InjectContext(t.Context())
			ctx = context.WithValue(ctx, timeContextKey{}, frozenTime)

			_, gotErr := runUpgrade(ctx, "fake-repo", repoDir)
			if gotErr == nil {
				t.Fatal("got nil, want error")
			}
			if !strings.Contains(gotErr.Error(), tc.wantErrMessage) {
				t.Errorf("error detail mismatch\ngot:  %q\nwant substring: %q", gotErr.Error(), tc.wantErrMessage)
			}
		})
	}
}

func TestUpgradeCommand(t *testing.T) {
	t.Parallel()

	frozenTime := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)

	mocker := &command.FakeCommander{
		FakeResults: map[string]command.FakeResult{
			command.FormatCmd("go", "list", "-m", "-json", "github.com/googleapis/librarian@main"): {Stdout: `{"Version": "v0.100.0"}`},
		},
	}

	ctx := mocker.InjectContext(t.Context())
	ctx = context.WithValue(ctx, timeContextKey{}, frozenTime)

	repoDir := t.TempDir()

	configPath := generateLibrarianConfigPath(t, repoDir)
	initialConfig := sample.Config()
	initialConfig.Language = config.LanguageFake
	initialConfig.Version = "v0.1.0"
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatal(err)
	}

	cmd := upgradeCommand()
	// Added "upgrade" as the first argument so urfave/cli parses the flags correctly
	if err := cmd.Run(ctx, []string{"upgrade", "-C", repoDir}); err != nil {
		t.Error(err)
	}
}

func TestUpgradeCommand_Error(t *testing.T) {
	frozenTime := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name           string
		args           []string
		setup          func(t *testing.T, mocker *command.FakeCommander)
		wantErrMessage string
	}{{
		name: "usage error",
		args: []string{"upgrade"}, // Added "upgrade" so parseFlags correctly evaluates empty args
		setup: func(t *testing.T, mocker *command.FakeCommander) {
			// No setup needed for usage error.
		},
		wantErrMessage: "usage:",
	},
		{
			name: "runUpgrade error",
			args: []string{"upgrade", "-C", "."}, // Added "upgrade" so urfave/cli parses flags properly
			setup: func(t *testing.T, mocker *command.FakeCommander) {
				mocker.FakeResults = map[string]command.FakeResult{
					command.FormatCmd("go", "list", "-m", "-json", "github.com/googleapis/librarian@main"): {Error: fmt.Errorf("error")},
				}
			},
			wantErrMessage: "failed to get latest librarian version",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// t.Chdir is not safe for parallel execution
			t.Chdir(t.TempDir())

			mocker := &command.FakeCommander{}
			tc.setup(t, mocker)

			ctx := mocker.InjectContext(t.Context())
			ctx = context.WithValue(ctx, timeContextKey{}, frozenTime)

			cmd := upgradeCommand()
			err := cmd.Run(ctx, tc.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrMessage) {
				t.Errorf("error mismatch\ngot: %q, want substring: %q", err.Error(), tc.wantErrMessage)
			}
		})
	}
}

func generateLibrarianConfigPath(t *testing.T, repoDir string) string {
	t.Helper()
	return filepath.Join(repoDir, "librarian.yaml")
}
