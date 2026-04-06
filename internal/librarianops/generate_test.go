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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestGenerateCommand(t *testing.T) {
	// Build the librarian binary from local source to avoid downloading
	// a published module version during tests.
	librarianBin := filepath.Join(t.TempDir(), "librarian")
	if err := command.Run(t.Context(), command.Go, "build", "-o", librarianBin, "../../cmd/librarian"); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		verbose bool
	}{
		{"default", false},
		{"verbose", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := t.TempDir()
			testhelper.RunGit(t, "init", repoDir)
			testhelper.RunGit(t, "-C", repoDir, "config", "user.email", "test@example.com")
			testhelper.RunGit(t, "-C", repoDir, "config", "user.name", "Test User")
			testhelper.RunGit(t, "-C", repoDir, "checkout", "-b", config.BranchMain)

			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
			cfg := sample.Config()
			cfg.Sources.Googleapis = &config.Source{Dir: googleapisDir}
			if err := yaml.Write(filepath.Join(repoDir, config.LibrarianYAML), cfg); err != nil {
				t.Fatal(err)
			}
			testhelper.RunGit(t, "-C", repoDir, "add", ".")
			testhelper.RunGit(t, "-C", repoDir, "commit", "-m", "initial commit")

			// Rename temp dir to fake-repo so basename matches expected repo
			// name.
			fakeRepoDir := filepath.Join(filepath.Dir(repoDir), "fake-repo")
			if err := os.Rename(repoDir, fakeRepoDir); err != nil {
				t.Fatal(err)
			}
			repoDir = fakeRepoDir

			if test.verbose {
				command.Verbose = true
				defer func() { command.Verbose = false }()
			}
			runInDocker := false
			if err := processRepo(t.Context(), repoFake, repoDir, librarianBin, test.verbose, runInDocker); err != nil {
				t.Fatal(err)
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

func TestSourcesToUpdate(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *config.Config
		want []string
	}{
		{
			name: "both sources",
			cfg: &config.Config{
				Sources: &config.Sources{
					Discovery:  &config.Source{Commit: "abc"},
					Googleapis: &config.Source{Commit: "def"},
				},
			},
			want: []string{"discovery", "googleapis"},
		},
		{
			name: "only googleapis",
			cfg: &config.Config{
				Sources: &config.Sources{
					Googleapis: &config.Source{Commit: "def"},
				},
			},
			want: []string{"googleapis"},
		},
		{
			name: "only discovery",
			cfg: &config.Config{
				Sources: &config.Sources{
					Discovery: &config.Source{Commit: "abc"},
				},
			},
			want: []string{"discovery"},
		},
		{
			name: "no sources configured",
			cfg:  &config.Config{},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := sourcesToUpdate(test.cfg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
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
