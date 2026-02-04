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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestGenerateCommand(t *testing.T) {
	for _, test := range []struct {
		name    string
		verbose bool
	}{
		{"default", false},
		{"verbose", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := t.TempDir()
			if err := command.Run(t.Context(), "git", "init", repoDir); err != nil {
				t.Fatal(err)
			}
			if err := command.Run(t.Context(), "git", "-C", repoDir, "config", "user.email", "test@example.com"); err != nil {
				t.Fatal(err)
			}
			if err := command.Run(t.Context(), "git", "-C", repoDir, "config", "user.name", "Test User"); err != nil {
				t.Fatal(err)
			}
			if err := command.Run(t.Context(), "git", "-C", repoDir, "checkout", "-b", "main"); err != nil {
				t.Fatal(err)
			}

			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
			cfg := sample.Config()
			cfg.Sources.Googleapis = &config.Source{Dir: googleapisDir}
			if err := yaml.Write(filepath.Join(repoDir, "librarian.yaml"), cfg); err != nil {
				t.Fatal(err)
			}
			if err := command.Run(t.Context(), "git", "-C", repoDir, "add", "."); err != nil {
				t.Fatal(err)
			}
			if err := command.Run(t.Context(), "git", "-C", repoDir, "commit", "-m", "initial commit"); err != nil {
				t.Fatal(err)
			}

			// Rename temp dir to fake-repo so basename matches expected repo
			// name.
			fakeRepoDir := filepath.Join(filepath.Dir(repoDir), "fake-repo")
			if err := os.Rename(repoDir, fakeRepoDir); err != nil {
				t.Fatal(err)
			}
			repoDir = fakeRepoDir

			args := []string{"librarianops", "generate", "-C", repoDir}
			if test.verbose {
				args = append(args, "-v")
				command.Verbose = true
				defer func() { command.Verbose = false }()
			}

			if !test.verbose {
				if err := Run(t.Context(), args...); err != nil {
					t.Fatal(err)
				}
			} else {
				old := os.Stdout
				r, w, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				os.Stdout = w
				t.Cleanup(func() { os.Stdout = old })

				runErr := Run(t.Context(), args...)
				if err := w.Close(); err != nil {
					// Close writer to signal EOF to reader.
					t.Fatal(err)
				}

				var buf bytes.Buffer
				if _, err := io.Copy(&buf, r); err != nil {
					t.Fatalf("failed to read from pipe: %v", err)
				}
				if runErr != nil {
					t.Fatal(runErr)
				}

				output := buf.String()
				if !strings.Contains(output, "librarian@") {
					t.Errorf("expected output to contain librarian command, got: %s", output)
				}
				if !strings.Contains(output, "-v") {
					t.Errorf("expected output to contain -v flag, got: %s", output)
				}
			}

			readmePath := filepath.Join(repoDir, sample.Lib1Output, "README.md")
			if _, err := os.Stat(readmePath); err != nil {
				t.Errorf("expected README.md to be generated: %v", err)
			}
		})
	}
}

func TestGenerateCommand_Errors(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{
			name: "no repo argument",
			args: []string{"librarianops", "generate"},
		},
		{
			name: "unsupported repo",
			args: []string{"librarianops", "generate", "unsupported-repo"},
		},
		{
			name: "unsupported repo via C flag",
			args: []string{"librarianops", "generate", "-C", "/tmp/unsupported-repo"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Run(t.Context(), test.args...)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestUpdateLibrarianVersion(t *testing.T) {
	repoDir := t.TempDir()
	configPath := filepath.Join(repoDir, "librarian.yaml")
	initialConfig := &config.Config{
		Language: "rust",
		Version:  sample.LibrarianVersion,
	}
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatal(err)
	}

	newVersion := "v0.2.0"
	if err := updateLibrarianVersion(newVersion, repoDir); err != nil {
		t.Fatal(err)
	}

	got, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatal(err)
	}

	want := &config.Config{
		Language: "rust",
		Version:  newVersion,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestVerboseFlagSetsCommandVerbose(t *testing.T) {
	origVerbose := command.Verbose
	defer func() { command.Verbose = origVerbose }()

	for _, test := range []struct {
		name        string
		args        []string
		wantVerbose bool
	}{
		{
			name:        "without -v flag",
			args:        []string{"librarianops", "generate", "fake-repo"},
			wantVerbose: false,
		},
		{
			name:        "with -v flag",
			args:        []string{"librarianops", "generate", "-v", "fake-repo"},
			wantVerbose: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			command.Verbose = false
			Run(t.Context(), test.args...)
			if command.Verbose != test.wantVerbose {
				t.Errorf("command.Verbose = %v, want %v", command.Verbose, test.wantVerbose)
			}
		})
	}
}
