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

package librarian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	testLibraryName = "google-cloud-secretmanager"
	testOutputDir   = "secretmanager-output"
	branch          = "main"
)

type wantLibrary struct {
	libraryName  string
	outputDir    string
	checkVersion bool
}

// TestFakeWorkflow tests multi-command workflows using the fake language.
func TestFakeWorkflow(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	runWorkflowTests(t, setupFakeRepo)
}

func runWorkflowTests(t *testing.T, setup func(t *testing.T) *config.Config) {
	t.Helper()

	for _, test := range []struct {
		name        string
		steps       [][]string
		modifyFiles []string
		want        wantLibrary
	}{
		{
			name: "add and generate",
			steps: [][]string{
				{"librarian", "add", "google-cloud-orgpolicy", "--output", "orgpolicy-output"},
				{"librarian", "generate", "google-cloud-orgpolicy"},
			},
			want: wantLibrary{
				libraryName: "google-cloud-orgpolicy",
				outputDir:   "orgpolicy-output",
			},
		},
		{
			name: "generate",
			steps: [][]string{
				{"librarian", "generate", testLibraryName},
			},
			want: wantLibrary{
				libraryName: testLibraryName,
				outputDir:   testOutputDir,
			},
		},
		{
			name: "generate all",
			steps: [][]string{
				{"librarian", "generate", "--all"},
			},
			want: wantLibrary{
				libraryName: testLibraryName,
				outputDir:   testOutputDir,
			},
		},
		{
			name: "release",
			steps: [][]string{
				{"librarian", "generate", testLibraryName},
				{"librarian", "release", testLibraryName},
			},
			modifyFiles: []string{
				filepath.Join(testOutputDir, "README.md"),
			},
			want: wantLibrary{
				libraryName:  testLibraryName,
				checkVersion: true,
			},
		},
		{
			name: "release all",
			steps: [][]string{
				{"librarian", "generate", "--all"},
				{"librarian", "release", "--all"},
			},
			modifyFiles: []string{
				filepath.Join(testOutputDir, "README.md"),
			},
			want: wantLibrary{
				libraryName:  testLibraryName,
				checkVersion: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setup(t)
			for i, args := range test.steps {
				if err := Run(t.Context(), args...); err != nil {
					t.Fatalf("step %d (%v) failed: %v", i, args, err)
				}
				if i == 0 && len(test.modifyFiles) > 0 {
					modifyTestFiles(t, test.modifyFiles)
				}
			}
			verifyLibrary(t, test.want)
		})
	}
}

func modifyTestFiles(t *testing.T, files []string) {
	t.Helper()
	for _, path := range files {
		if err := os.WriteFile(path, []byte("modified\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	commitAll(t, "test: modify files")
}

func setupRepo(t *testing.T, cfg *config.Config, createFiles func(t *testing.T)) {
	t.Helper()

	remoteDir := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, remoteDir)

	createFiles(t)
	commitAll(t, "initial commit")

	if err := yaml.Write(librarianConfigPath, cfg); err != nil {
		t.Fatal(err)
	}
	commitAll(t, "chore: add librarian yaml")

	if err := command.Run(t.Context(), "git", "tag", sample.InitialTag); err != nil {
		t.Fatal(err)
	}

	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	if err := command.Run(t.Context(), "git", "clone", "--branch", branch, remoteDir, "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "config", "user.email", "test@test-only.com"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "config", "user.name", "Test Account"); err != nil {
		t.Fatal(err)
	}
}

func commitAll(t *testing.T, message string) {
	t.Helper()
	if err := command.Run(t.Context(), "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", message); err != nil {
		t.Fatal(err)
	}
}

func setupFakeRepo(t *testing.T) *config.Config {
	t.Helper()
	cfg := testConfig(t, languageFake, googleapisTestDir(t))
	cfg.Libraries = []*config.Library{
		{
			Name:    testLibraryName,
			Version: sample.InitialVersion,
			Output:  testOutputDir,
		},
	}
	setupRepo(t, cfg, func(t *testing.T) {
		if err := os.WriteFile("README.md", []byte("# Test Repo"), 0644); err != nil {
			t.Fatal(err)
		}
	})
	return cfg
}

func verifyLibrary(t *testing.T, want wantLibrary) {
	t.Helper()

	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, lib := range cfg.Libraries {
		if lib.Name == want.libraryName {
			found = true
			if want.checkVersion && lib.Version == sample.InitialVersion {
				t.Errorf("%s version not bumped, still %q", lib.Name, lib.Version)
			}
			break
		}
	}
	if !found {
		t.Fatalf("library %s not found in config", want.libraryName)
	}

	if want.outputDir != "" {
		readmePath := filepath.Join(want.outputDir, "README.md")
		if _, err := os.Stat(readmePath); err != nil {
			t.Errorf("%s not generated: %v", readmePath, err)
		}
	}
}

func googleapisTestDir(t *testing.T) string {
	t.Helper()
	googleapisDir, err := filepath.Abs("../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	return googleapisDir
}

func testConfig(t *testing.T, language, googleapisDir string) *config.Config {
	t.Helper()
	return &config.Config{
		Language: language,
		Default:  &config.Default{},
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
		},
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Dir:    googleapisDir,
				Commit: "9fcfbea0aa5b50fa22e190faceb073d74504172b",
				SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
			},
		},
	}
}
