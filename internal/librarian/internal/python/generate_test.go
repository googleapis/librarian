// Copyright 2025 Google LLC
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

package python

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"cloud.google.com/go/librarian/internal/config"
	"github.com/google/go-cmp/cmp"
)

type CommandScript struct {
	Commands []*MockCommand
}

type MockCommand struct {
	ExpectedArgs []string
	Error        *error
	called       bool
}

func TestGetStagingChildDirectory(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		apiPath  string
		expected string
	}{
		{
			name:     "versioned path",
			apiPath:  "google/test/v1",
			expected: "v1",
		},
		{
			name:     "non-versioned path",
			apiPath:  "google/test/type",
			expected: "type-py",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getStagingChildDirectory(test.apiPath)
			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("getStagingChildDirectory(%q) returned diff (-want +got):\n%s", test.apiPath, diff)
			}
		})
	}
}

func TestCreateProtocOptions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		channel  *config.Channel
		library  *config.Library
		expected []string
		wantErr  bool
	}{
		{
			name:    "basic case",
			channel: &config.Channel{Path: "google/test/v1"},
			library: &config.Library{},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=rest-numeric-enums,metadata",
			},
		},
		{
			name:    "with transport",
			channel: &config.Channel{Path: "google/test/v1"},
			library: &config.Library{Transport: "grpc"},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=transport=grpc,rest-numeric-enums,metadata",
			},
		},
		{
			name:    "with python opts",
			channel: &config.Channel{Path: "google/test/v1"},
			library: &config.Library{
				Python: &config.PythonPackage{
					OptArgs: []string{"opt1", "opt2"},
				},
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=rest-numeric-enums,metadata,opt1,opt2",
			},
		},
		{
			name:    "with python opts by channel",
			channel: &config.Channel{Path: "google/test/v1"},
			library: &config.Library{
				Python: &config.PythonPackage{
					OptArgsByChannel: map[string][]string{
						"google/test/v1": {"opt1", "opt2"},
						"google/test/v2": {"opt3", "opt4"},
					},
				},
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=rest-numeric-enums,metadata,opt1,opt2",
			},
		},
		{
			name:    "with version",
			channel: &config.Channel{Path: "google/test/v1"},
			library: &config.Library{Version: "1.2.3"},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=rest-numeric-enums,metadata,gapic-version=1.2.3",
			},
		},
		{
			name:    "with grpc config",
			channel: &config.Channel{Path: "google/test/v2"},
			library: &config.Library{},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=rest-numeric-enums,metadata,retry-config=google/test/v2/test_grpc_service_config.json",
			},
		},
		{
			name:    "multiple grpc configs",
			channel: &config.Channel{Path: "google/test/v3"},
			library: &config.Library{},
			wantErr: true,
		},
		{
			name: "with service config",
			channel: &config.Channel{
				Path:          "google/test/v1",
				ServiceConfig: "test_v1.yaml",
			},
			library: &config.Library{},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=rest-numeric-enums,metadata,service-yaml=google/test/v1/test_v1.yaml",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			googleapisDir := "testdata"

			got, err := createProtocOptions(test.channel, test.library, googleapisDir, "staging")
			if (err != nil) != test.wantErr {
				t.Fatalf("createProtocOptions() error = %v, wantErr %v", err, test.wantErr)
			}

			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("createProtocOptions() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSourceDir(t *testing.T) {
	originalFetchRepoDir := fetchRepoDir
	t.Cleanup(func() {
		fetchRepoDir = originalFetchRepoDir
	})

	fetchRepoDir = func(ctx context.Context, repo, commit, sha256 string) (string, error) {
		return "fetched", nil
	}
	for _, test := range []struct {
		name        string
		source      *config.Source
		expected    string
		expectedErr bool
	}{
		{
			name:     "source is nil",
			source:   nil,
			expected: "",
		},
		{
			name: "source has dir",
			source: &config.Source{
				Dir: "path/to/dir",
			},
			expected: "path/to/dir",
		},
		{
			name: "source needs fetching",
			source: &config.Source{
				Commit: "commit",
				SHA256: "sha256",
			},
			expected: "fetched",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := sourceDir(context.Background(), test.source, "repo")
			if (err != nil) != test.expectedErr {
				t.Fatalf("sourceDir() error = %v, wantErr %v", err, test.expectedErr)
			}
			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("sourceDir() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCopyReadmeToDocsDir(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name            string
		setup           func(t *testing.T, outdir string)
		expectedContent string
		expectedErr     bool
	}{
		{
			name: "no readme",
			setup: func(t *testing.T, outdir string) {
				// No setup needed
			},
		},
		{
			name: "readme is a regular file",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "README.rst"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedContent: "hello",
		},
		{
			name: "readme is a symlink",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "REAL_README.rst"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("REAL_README.rst", filepath.Join(outdir, "README.rst")); err != nil {
					t.Fatal(err)
				}
			},
			expectedContent: "hello",
		},
		{
			name: "dest is a symlink",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "README.rst"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(outdir, "docs"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../some/other/file", filepath.Join(outdir, "docs", "README.rst")); err != nil {
					t.Fatal(err)
				}
			},
			expectedContent: "hello",
		},
		{
			name: "unreadable readme",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "README.rst"), []byte("hello"), 0000); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					os.Chmod(filepath.Join(outdir, "README.rst"), 0644)
				})
			},
			expectedErr: true,
		},
		{
			name: "cannot create docs dir",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "README.rst"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(outdir, "docs"), []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()
			test.setup(t, outdir)
			err := copyReadmeToDocsDir(outdir)
			if (err != nil) != test.expectedErr {
				t.Fatalf("copyReadmeToDocsDir() error = %v, wantErr %v", err, test.expectedErr)
			}

			if test.expectedContent != "" {
				content, err := os.ReadFile(filepath.Join(outdir, "docs", "README.rst"))
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(test.expectedContent, string(content)); diff != "" {
					t.Errorf("copyReadmeToDocsDir() returned diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCleanUpFilesAfterPostProcessing(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		setup       func(t *testing.T, repoRoot string)
		expectedErr bool
	}{
		{
			name: "no staging dir",
			setup: func(t *testing.T, repoRoot string) {
				// No setup needed
			},
		},
		{
			name: "staging dir exists",
			setup: func(t *testing.T, repoRoot string) {
				if err := os.MkdirAll(filepath.Join(repoRoot, "owl-bot-staging"), 0755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "error removing",
			setup: func(t *testing.T, repoRoot string) {
				stagingDir := filepath.Join(repoRoot, "owl-bot-staging")
				if err := os.MkdirAll(stagingDir, 0755); err != nil {
					t.Fatal(err)
				}
				// Create a file in the directory
				if err := os.WriteFile(filepath.Join(stagingDir, "file"), []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				// Make the directory read-only to cause an error
				if err := os.Chmod(stagingDir, 0400); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					os.Chmod(stagingDir, 0755)
				})
			},
			expectedErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			test.setup(t, repoRoot)
			err := cleanUpFilesAfterPostProcessing(repoRoot)
			if (err != nil) != test.expectedErr {
				t.Fatalf("cleanUpFilesAfterPostProcessing() error = %v, wantErr %v", err, test.expectedErr)
			}
			if !test.expectedErr {
				if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); !os.IsNotExist(err) {
					t.Errorf("owl-bot-staging should have been removed")
				}
			}
		})
	}
}

func TestRunPostProcessor(t *testing.T) {
	pythonCode := fmt.Sprintf(`
from synthtool.languages import python_mono_repo
python_mono_repo.owlbot_main(%q)
`, "out/dir")
	for _, test := range []struct {
		name          string
		commandScript CommandScript
		wantErr       bool
	}{
		{
			name: "success",
			commandScript: CommandScript{
				Commands: []*MockCommand{
					{
						ExpectedArgs: []string{"python3", "-c", pythonCode},
					},
				},
			},
		},
		{
			name: "command fails",
			commandScript: CommandScript{
				Commands: []*MockCommand{
					{
						ExpectedArgs: []string{"python3", "-c", pythonCode},
						Error:        &exec.ErrNotFound,
					},
				},
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			originalRunCommand := runCommand
			t.Cleanup(func() {
				runCommand = originalRunCommand
			})
			runCommand = newMockRunCommand(t, test.commandScript)
			err := runPostProcessor(t.Context(), t.TempDir(), "out/dir")
			if (err != nil) != test.wantErr {
				t.Fatalf("runPostProcessor() error = %v, wantErr %v", err, test.wantErr)
			}
			verifyCommands(t, test.commandScript)
		})
	}
}

func TestGenerateChannel(t *testing.T) {
	repoRoot := t.TempDir()
	protocCommand := []string{
		"protoc",
		"google/test/v1/test.proto",
		"--python_gapic_out=" + repoRoot + "/owl-bot-staging/test/v1",
		"--python_gapic_opt=rest-numeric-enums,metadata",
	}

	for _, test := range []struct {
		name          string
		commandScript CommandScript
		wantErr       bool
	}{
		{
			name: "success",
			commandScript: CommandScript{
				Commands: []*MockCommand{
					{
						ExpectedArgs: protocCommand,
					},
				},
			},
		},
		{
			name: "protoc fails",
			commandScript: CommandScript{
				Commands: []*MockCommand{
					{
						ExpectedArgs: protocCommand,
						Error:        &exec.ErrNotFound,
					},
				},
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			originalRunCommand := runCommand
			t.Cleanup(func() {
				runCommand = originalRunCommand
			})
			runCommand = newMockRunCommand(t, test.commandScript)
			err := generateChannel(
				t.Context(),
				&config.Channel{Path: "google/test/v1"},
				&config.Library{Name: "test", Output: repoRoot},
				"testdata",
				repoRoot,
			)
			if (err != nil) != test.wantErr {
				t.Fatalf("generateChannel() error = %v, wantErr %v", err, test.wantErr)
			}
			verifyCommands(t, test.commandScript)
		})
	}
}

func TestGenerate(t *testing.T) {
	repoRoot := t.TempDir()
	outdir, err := filepath.Abs(filepath.Join(repoRoot, "packages", "test"))
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	postProcessor := fmt.Sprintf(`
from synthtool.languages import python_mono_repo
python_mono_repo.owlbot_main(%q)
`, outdir)

	commands := CommandScript{
		Commands: []*MockCommand{
			{
				ExpectedArgs: []string{
					"protoc",
					"google/test/v1/test.proto",
					"--python_gapic_out=" + repoRoot + "/owl-bot-staging/test/v1",
					"--python_gapic_opt=rest-numeric-enums,metadata,service-yaml=google/test/v1/test_v1.yaml",
				},
			},
			{
				ExpectedArgs: []string{
					"protoc",
					"google/test/v2/test.proto",
					"--python_gapic_out=" + repoRoot + "/owl-bot-staging/test/v2",
					"--python_gapic_opt=rest-numeric-enums,metadata,retry-config=google/test/v2/test_grpc_service_config.json,service-yaml=google/test/v2/test_v2.yaml",
				},
			},
			{
				ExpectedArgs: []string{"python3", "-c", postProcessor},
			},
		},
	}

	library := &config.Library{
		Name:   "test",
		Output: outdir,
		Channels: []*config.Channel{
			{
				Path:          "google/test/v1",
				ServiceConfig: "test_v1.yaml",
			},
			{
				Path:          "google/test/v2",
				ServiceConfig: "test_v2.yaml",
			},
		},
	}
	sources := &config.Sources{
		Googleapis: &config.Source{
			Dir: "testdata",
		},
	}

	originalRunCommand := runCommand
	t.Cleanup(func() {
		runCommand = originalRunCommand
	})
	runCommand = newMockRunCommand(t, commands)

	err = Generate(t.Context(), library, sources)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	verifyCommands(t, commands)

	if _, err := os.Stat(filepath.Join(outdir, ".repo-metadata.json")); err != nil {
		t.Fatalf("Generate() error checking for presence of .repo-metadata.json = %v", err)
	}
}

// newMockRunCommand creates a function with the same signature as Run,
// but which checks the given command script for any command that is executed.
// If the command doesn't exist, the test fails. Commands are marked as having
// been run, and an optional error is returned. The command script can then be
// verified with verifyCommands.
func newMockRunCommand(t *testing.T, script CommandScript) func(context.Context, []string, string) error {
	return func(ctx context.Context, args []string, workDir string) error {
		for _, command := range script.Commands {
			if !command.called && slices.Equal(command.ExpectedArgs, args) {
				command.called = true
				if command.Error == nil {
					return nil
				}
				return *command.Error
			}
		}
		t.Fatalf("no mocked commands for %v", args)
		// We won't get here.
		return exec.ErrNotFound
	}
}

// verifyCommands verifies that all commands in the specified script have been
// "mock executed".
func verifyCommands(t *testing.T, script CommandScript) {
	for _, command := range script.Commands {
		if !command.called {
			t.Fatalf("missing call with args %v", command.ExpectedArgs)
		}
	}
}
