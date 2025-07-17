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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"
)

// mockContainerClient is a mock implementation of the ContainerClient interface for testing.
type mockContainerClient struct {
	ContainerClient
	generateCalls int
	buildCalls    int
}

func (m *mockContainerClient) Generate(ctx context.Context, request *docker.GenerateRequest) error {
	m.generateCalls++
	return nil
}

func (m *mockContainerClient) Build(ctx context.Context, request *docker.BuildRequest) error {
	m.buildCalls++
	return nil
}

func TestDetectIfLibraryConfigured(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		api     string
		repo    string
		state   *config.LibrarianState
		want    bool
		wantErr bool
	}{
		{
			name: "no repo specified",
			api:  "some/api",
		},
		{
			name: "api not in state",
			api:  "other/api",
			repo: "some/repo",
			state: &config.LibrarianState{
				Image: "some/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:          "some-library",
						APIs:        []config.API{{Path: "some/api"}},
						SourcePaths: []string{"src/a"},
					},
				},
			},
		},
		{
			name: "api in state",
			api:  "some/api",
			repo: "some/repo",
			state: &config.LibrarianState{
				Image: "some/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:          "some-library",
						APIs:        []config.API{{Path: "some/api"}},
						SourcePaths: []string{"src/a"},
					},
				},
			},
			want: true,
		},
		{
			name:    "state file does not exist",
			api:     "some/api",
			repo:    "some/repo",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var repo *gitrepo.Repository
			if test.repo != "" {
				repo = newTestGitRepo(t)
				if test.state != nil {
					generatorInputDir := filepath.Join(repo.Dir, config.GeneratorInputDir)
					if err := os.MkdirAll(generatorInputDir, 0755); err != nil {
						t.Fatalf("os.MkdirAll(%q, 0755) = %v", generatorInputDir, err)
					}
					stateFile := filepath.Join(generatorInputDir, pipelineStateFile)
					b, err := yaml.Marshal(test.state)
					if err != nil {
						t.Fatalf("yaml.Marshal = %v", err)
					}
					if err := os.WriteFile(stateFile, b, 0644); err != nil {
						t.Fatalf("os.WriteFile(%q, ...) = %v", stateFile, err)
					}
				}
			}

			r := &generateRunner{
				cfg: &config.Config{
					API:  test.api,
					Repo: test.repo,
				},
			}
			if repo != nil {
				r.cfg.Repo = repo.Dir
			}

			got, err := r.detectIfLibraryConfigured(context.Background())
			if (err != nil) != test.wantErr {
				t.Errorf("detectIfLibraryConfigured() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("detectIfLibraryConfigured() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunGenerateCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		api               string
		repo              *gitrepo.Repository
		state             *config.LibrarianState
		container         *mockContainerClient
		wantLibraryID     string
		wantErr           bool
		wantGenerateCalls int
	}{
		{
			name: "works",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []config.API{{Path: "some/api"}},
					},
				},
			},
			container:         &mockContainerClient{},
			wantLibraryID:     "some-library",
			wantGenerateCalls: 1,
		},
		{
			name:      "missing repo",
			api:       "some/api",
			container: &mockContainerClient{},
			wantErr:   true,
		},
		{
			name: "library not found in state",
			api:  "other/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []config.API{{Path: "some/api"}},
					},
				},
			},
			container: &mockContainerClient{},
			wantErr:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					API:    test.api,
					Source: t.TempDir(),
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
			}

			outputDir := t.TempDir()
			gotLibraryID, err := r.runGenerateCommand(context.Background(), outputDir)
			if (err != nil) != test.wantErr {
				t.Errorf("runGenerateCommand() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.wantLibraryID, gotLibraryID); diff != "" {
				t.Errorf("runGenerateCommand() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantGenerateCalls, test.container.generateCalls); diff != "" {
				t.Errorf("runGenerateCommand() generateCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestRunBuildCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		build          bool
		libraryID      string
		repo           *gitrepo.Repository
		state          *config.LibrarianState
		container      *mockContainerClient
		wantBuildCalls int
		wantErr        bool
	}{
		{
			name:           "build flag not specified",
			build:          false,
			container:      &mockContainerClient{},
			wantBuildCalls: 0,
		},
		{
			name:      "build with library id",
			build:     true,
			libraryID: "some-library",
			repo:      newTestGitRepo(t),
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
					},
				},
			},
			container:      &mockContainerClient{},
			wantBuildCalls: 1,
		},
		{
			name:      "build with no library id",
			build:     true,
			container: &mockContainerClient{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					Build: test.build,
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
			}
			if err := r.runBuildCommand(context.Background(), test.libraryID); (err != nil) != test.wantErr {
				t.Errorf("runBuildCommand() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.container.buildCalls); diff != "" {
				t.Errorf("runBuildCommand() buildCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestNewGenerateRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "missing api flag",
			cfg:     &config.Config{Source: "some/source"},
			wantErr: true,
		},
		{
			name:    "missing source flag",
			cfg:     &config.Config{API: "some/api"},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &config.Config{
				API:      "some/api",
				Source:   t.TempDir(),
				Repo:     newTestGitRepo(t).Dir,
				WorkRoot: t.TempDir(),
				Image:    "gcr.io/test/test-image",
			},
		},
		{
			name: "missing image",
			cfg: &config.Config{

				API:      "some/api",
				Source:   t.TempDir(),
				Repo:     "https://github.com/googleapis/librarian.git",
				WorkRoot: t.TempDir(),
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// We need to create a fake state and config file for the test to pass.
			if test.cfg.Repo != "" && !isUrl(test.cfg.Repo) {
				stateFile := filepath.Join(test.cfg.Repo, config.LibrarianDir, pipelineStateFile)

				if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
					t.Fatalf("os.MkdirAll() = %v", err)
				}
				state := &config.LibrarianState{
					Image: "some/image:v1.2.3",
					Libraries: []*config.LibraryState{
						{
							ID:          "some-library",
							APIs:        []config.API{{Path: "some/api"}},
							SourcePaths: []string{"src/a"},
						},
					},
				}
				b, err := yaml.Marshal(state)
				if err != nil {
					t.Fatalf("yaml.Marshal() = %v", err)
				}
				if err := os.WriteFile(stateFile, b, 0644); err != nil {
					t.Fatalf("os.WriteFile(%q, ...) = %v", stateFile, err)
				}
				configFile := filepath.Join(test.cfg.Repo, config.LibrarianDir, pipelineConfigFile)
				if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
					t.Fatalf("os.WriteFile(%q, ...) = %v", configFile, err)
				}
				runGit(t, test.cfg.Repo, "add", ".")
				runGit(t, test.cfg.Repo, "commit", "-m", "add config")
			}

			_, err := newGenerateRunner(test.cfg)
			if (err != nil) != test.wantErr {
				t.Errorf("newGenerateRunner() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// newTestGitRepo creates a new git repository in a temporary directory.
func newTestGitRepo(t *testing.T) *gitrepo.Repository {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")
	repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: dir})
	if err != nil {
		t.Fatalf("gitrepo.Open(%q) = %v", dir, err)
	}
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func TestGenerateRun(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		api               string
		repo              *gitrepo.Repository
		state             *config.LibrarianState
		container         *mockContainerClient
		build             bool
		wantErr           bool
		wantGenerateCalls int
		wantBuildCalls    int
	}{
		{
			name: "regeneration of API",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []config.API{{Path: "some/api"}},
					},
				},
			},
			container:         &mockContainerClient{},
			build:             true,
			wantGenerateCalls: 1,
			wantBuildCalls:    1,
		},
		{
			name: "symlink in output",
			api:  "some/api",
			repo: newTestGitRepo(t),
			state: &config.LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []*config.LibraryState{
					{
						ID:   "some-library",
						APIs: []config.API{{Path: "some/api"}},
					},
				},
			},
			container:         &mockContainerClient{},
			build:             true,
			wantGenerateCalls: 1,
			wantErr:           true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := &generateRunner{
				cfg: &config.Config{
					API:    test.api,
					Source: t.TempDir(),
					Build:  test.build,
				},
				repo:            test.repo,
				state:           test.state,
				containerClient: test.container,
				workRoot:        t.TempDir(),
			}

			// Create a symlink in the output directory to trigger an error.
			if test.name == "symlink in output" {
				outputDir := filepath.Join(r.workRoot, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					t.Fatalf("os.MkdirAll() = %v", err)
				}
				if err := os.Symlink("target", filepath.Join(outputDir, "symlink")); err != nil {
					t.Fatalf("os.Symlink() = %v", err)
				}
			}

			err := r.run(context.Background())
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantGenerateCalls, test.container.generateCalls); diff != "" {
				t.Errorf("run() generateCalls mismatch (-want +got):%s", diff)
			}
			if diff := cmp.Diff(test.wantBuildCalls, test.container.buildCalls); diff != "" {
				t.Errorf("run() buildCalls mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestClean(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name             string
		files            map[string]string
		symlinks         map[string]string
		removePatterns   []string
		preservePatterns []string
		wantRemaining    []string
		wantErr          bool
	}{
		{
			name: "remove all",
			files: map[string]string{
				"file1.txt": "",
				"file2.txt": "",
			},
			removePatterns: []string{".*\\.txt"},
			wantRemaining:  []string{"."},
		},
		{
			name: "preserve all",
			files: map[string]string{
				"file1.txt": "",
				"file2.txt": "",
			},
			removePatterns:   []string{".*"},
			preservePatterns: []string{".*"},
			wantRemaining:    []string{".", "file1.txt", "file2.txt"},
		},
		{
			name: "remove some",
			files: map[string]string{
				"foo/file1.txt": "",
				"foo/file2.txt": "",
				"bar/file3.txt": "",
			},
			removePatterns: []string{"foo/.*"},
			wantRemaining:  []string{".", "bar", "bar/file3.txt", "foo"},
		},
		{
			name: "invalid remove pattern",
			files: map[string]string{
				"file1.txt": "",
			},
			removePatterns: []string{"["}, // Invalid regex
			wantErr:        true,
		},
		{
			name: "invalid preserve pattern",
			files: map[string]string{
				"file1.txt": "",
			},
			removePatterns:   []string{".*"},
			preservePatterns: []string{"["}, // Invalid regex
			wantErr:          true,
		},
		{
			name: "remove symlink",
			files: map[string]string{
				"file1.txt": "content",
			},
			symlinks: map[string]string{
				"symlink_to_file1": "file1.txt",
			},
			removePatterns: []string{"symlink_to_file1"},
			wantRemaining:  []string{".", "file1.txt"},
		},
		{
			name: "remove file symlinked to",
			files: map[string]string{
				"file1.txt": "content",
			},
			symlinks: map[string]string{
				"symlink_to_file1": "file1.txt",
			},
			removePatterns: []string{"file1.txt"},
			// The symlink should remain, even though it's now broken, because
			// it was not targeted for removal.
			wantRemaining: []string{".", "symlink_to_file1"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			for path, content := range test.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("os.MkdirAll() = %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("os.WriteFile() = %v", err)
				}
			}
			for link, target := range test.symlinks {
				linkPath := filepath.Join(tmpDir, link)
				if err := os.Symlink(target, linkPath); err != nil {
					t.Fatalf("os.Symlink() = %v", err)
				}
			}
			err := clean(tmpDir, test.removePatterns, test.preservePatterns)
			if (err != nil) != test.wantErr {
				t.Errorf("clean() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if !test.wantErr {
				var remainingPaths []string
				err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					relPath, err := filepath.Rel(tmpDir, path)
					if err != nil {
						return err
					}
					remainingPaths = append(remainingPaths, relPath)
					return nil
				})
				if err != nil {
					t.Fatalf("filepath.Walk() = %v", err)
				}
				sort.Strings(test.wantRemaining)
				sort.Strings(remainingPaths)
				if diff := cmp.Diff(test.wantRemaining, remainingPaths); diff != "" {
					t.Errorf("clean() remaining files mismatch (-want +got):%s", diff)
				}
			}
		})
	}
}

func TestSortDirsByDepth(t *testing.T) {
	t.Parallel()
	dirs := []string{
		"a/b",
		"short-dir",
		"a/b/c",
		"a",
	}
	want := []string{
		"a/b/c",
		"a/b",
		"short-dir",
		"a",
	}
	sortDirsByDepth(dirs)
	if diff := cmp.Diff(want, dirs); diff != "" {
		t.Errorf("sortDirsByDepth() mismatch (-want +got):\n%s", diff)
	}
}

func TestGetAllPaths(t *testing.T) {
	t.Parallel()
	// Create a temporary directory with some files and subdirectories.
	tmpDir := t.TempDir()
	files := []string{
		"file1.txt",
		"dir1/file2.txt",
		"dir1/dir2/file3.txt",
	}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("os.MkdirAll() = %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("os.WriteFile() = %v", err)
		}
	}

	paths, err := getAllPaths(tmpDir)
	if err != nil {
		t.Fatalf("getAllPaths() = %v", err)
	}

	wantPaths := []string{
		".",
		"dir1",
		"dir1/dir2",
		"dir1/dir2/file3.txt",
		"dir1/file2.txt",
		"file1.txt",
	}

	// Sort both slices to ensure consistent comparison.
	sort.Strings(paths)
	sort.Strings(wantPaths)

	if diff := cmp.Diff(wantPaths, paths); diff != "" {
		t.Errorf("getAllPaths() mismatch (-want +got):\n%s", diff)
	}
}

func TestFilterPaths(t *testing.T) {
	t.Parallel()
	paths := []string{
		"foo/file1.txt",
		"foo/file2.log",
		"bar/file3.txt",
		"bar/file4.log",
	}
	regexps := []*regexp.Regexp{
		regexp.MustCompile(`^foo/.*.txt$`),
		regexp.MustCompile(`^bar/.*`),
	}

	filtered := filterPaths(paths, regexps)

	wantFiltered := []string{
		"foo/file1.txt",
		"bar/file3.txt",
		"bar/file4.log",
	}

	sort.Strings(filtered)
	sort.Strings(wantFiltered)

	if diff := cmp.Diff(wantFiltered, filtered); diff != "" {
		t.Errorf("filterPaths() mismatch (-want +got):%s", diff)
	}
}

func TestRemovePreservedPaths(t *testing.T) {
	t.Parallel()
	remove := []string{"a", "b", "c"}
	preserve := []string{"b", "d"}

	final := removePreservedPaths(remove, preserve)

	wantFinal := []string{"a", "c"}

	if diff := cmp.Diff(wantFinal, final); diff != "" {
		t.Errorf("removePreservedPaths() mismatch (-want +got):%s", diff)
	}
}

func TestSeparateFilesAndDirs(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	files := []string{
		"file1.txt",
		"dir1/file2.txt",
	}
	dirs := []string{
		"dir1",
		"dir2",
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("os.MkdirAll() = %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("os.WriteFile() = %v", err)
		}
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("os.MkdirAll() = %v", err)
		}
	}

	paths := append(files, dirs...)
	gotFiles, gotDirs := separateFilesAndDirs(tmpDir, paths)

	// Sort for consistent comparison.
	sort.Strings(gotFiles)
	sort.Strings(gotDirs)
	sort.Strings(files)
	sort.Strings(dirs)

	if diff := cmp.Diff(files, gotFiles); diff != "" {
		t.Errorf("separateFilesAndDirs() files mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(dirs, gotDirs); diff != "" {
		t.Errorf("separateFilesAndDirs() dirs mismatch (-want +got):\n%s", diff)
	}
}

func TestCompileRegexps(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		patterns []string
		wantErr  bool
	}{
		{
			name: "valid patterns",
			patterns: []string{
				`^foo.*`,
				`\\.txt$`,
			},
			wantErr: false,
		},
		{
			name:     "empty patterns",
			patterns: []string{},
			wantErr:  false,
		},
		{
			name: "invalid pattern",
			patterns: []string{
				`[`,
			},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid patterns",
			patterns: []string{
				`^foo.*`,
				`[`,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			regexps, err := compileRegexps(tc.patterns)
			if (err != nil) != tc.wantErr {
				t.Fatalf("compileRegexps() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if len(regexps) != len(tc.patterns) {
					t.Errorf("compileRegexps() len = %d, want %d", len(regexps), len(tc.patterns))
				}
			}
		})
	}
}
