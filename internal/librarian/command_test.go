// Copyright 2024 Google LLC
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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	gogitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func TestCommandUsage(t *testing.T) {
	for _, c := range CmdLibrarian.Commands {
		t.Run(c.Name(), func(t *testing.T) {
			parts := strings.Fields(c.UsageLine)
			// The first word should always be "librarian".
			if parts[0] != "librarian" {
				t.Errorf("invalid usage text: %q (the first word should be `librarian`)", c.UsageLine)
			}
			if !strings.Contains(c.UsageLine, c.Name()) {
				t.Errorf("invalid usage text: %q (should contain command name %q)", c.UsageLine, c.Name())
			}
		})
	}
}

func TestFindLibraryByID(t *testing.T) {
	lib1 := &config.LibraryState{ID: "lib1"}
	lib2 := &config.LibraryState{ID: "lib2"}
	stateWithLibs := &config.LibrarianState{
		Libraries: []*config.LibraryState{lib1, lib2},
	}
	stateNoLibs := &config.LibrarianState{
		Libraries: []*config.LibraryState{},
	}

	for _, test := range []struct {
		name      string
		state     *config.LibrarianState
		libraryID string
		want      *config.LibraryState
	}{
		{
			name:      "found first library",
			state:     stateWithLibs,
			libraryID: "lib1",
			want:      lib1,
		},
		{
			name:      "found second library",
			state:     stateWithLibs,
			libraryID: "lib2",
			want:      lib2,
		},
		{
			name:      "not found",
			state:     stateWithLibs,
			libraryID: "lib3",
			want:      nil,
		},
		{
			name:      "empty libraries slice",
			state:     stateNoLibs,
			libraryID: "lib1",
			want:      nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := findLibraryByID(test.state, test.libraryID)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("findLibraryByID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveImage(t *testing.T) {
	for _, test := range []struct {
		name          string
		imageOverride string
		state         *config.LibrarianState
		want          string
	}{
		{
			name:          "with image override, nil state",
			imageOverride: "my/custom-image:v1",
			state:         nil,
			want:          "my/custom-image:v1",
		},
		{
			name:          "with image override, non-nil state",
			imageOverride: "my/custom-image:v1",
			state:         &config.LibrarianState{Image: "gcr.io/foo/bar:v1.2.3"},
			want:          "my/custom-image:v1",
		},
		{
			name:          "no override, nil state",
			imageOverride: "",
			state:         nil,
			want:          "",
		},
		{
			name:          "no override, with state",
			imageOverride: "",
			state:         &config.LibrarianState{Image: "gcr.io/foo/bar:v1.2.3"},
			want:          "gcr.io/foo/bar:v1.2.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveImage(test.imageOverride, test.state)

			if got != test.want {
				t.Errorf("deriveImage() = %q, want %q", got, test.want)
			}
		})
	}
}

// newTestGitRepoWithCommit creates a new git repository with an initial commit.
// If dir is empty, a new temporary directory is created.
// It returns the path to the repository directory.
func newTestGitRepoWithCommit(t *testing.T, dir string) string {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	} else {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", dir, err)
		}
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "tester"},
		{"config", "user.email", "tester@example.com"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	filePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	for _, args := range [][]string{
		{"add", "README.md"},
		{"commit", "-m", "initial commit"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	return dir
}

func TestCloneOrOpenLanguageRepo(t *testing.T) {
	workRoot := t.TempDir()

	cleanRepoPath := newTestGitRepoWithCommit(t, "")
	dirtyRepoPath := newTestGitRepoWithCommit(t, "")
	if err := os.WriteFile(filepath.Join(dirtyRepoPath, "untracked.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	notARepoPath := t.TempDir()

	for _, test := range []struct {
		name    string
		repo    string
		ci      string
		wantErr bool
		check   func(t *testing.T, repo *gitrepo.LocalRepository)
		setup   func(t *testing.T, workRoot string) func()
	}{
		{
			name: "with clean repoRoot",
			repo: cleanRepoPath,
			check: func(t *testing.T, repo *gitrepo.LocalRepository) {
				absWantDir, _ := filepath.Abs(cleanRepoPath)
				if repo.Dir != absWantDir {
					t.Errorf("repo.Dir got %q, want %q", repo.Dir, absWantDir)
				}
			},
		},
		{
			name: "with repoURL with trailing slash",
			repo: "https://github.com/googleapis/google-cloud-go/",
			setup: func(t *testing.T, workRoot string) func() {
				// The expected directory name is `google-cloud-go`.
				repoPath := filepath.Join(workRoot, "google-cloud-go")
				newTestGitRepoWithCommit(t, repoPath)
				return func() {
					if err := os.RemoveAll(repoPath); err != nil {
						t.Errorf("os.RemoveAll(%q) = %v; want nil", repoPath, err)
					}
				}
			},
			check: func(t *testing.T, repo *gitrepo.LocalRepository) {
				wantDir := filepath.Join(workRoot, "google-cloud-go")
				if repo.Dir != wantDir {
					t.Errorf("repo.Dir got %q, want %q", repo.Dir, wantDir)
				}
			},
		},
		{
			name:    "no repoRoot or repoURL",
			wantErr: true,
		},
		{
			name:    "with dirty repoRoot",
			repo:    dirtyRepoPath,
			wantErr: true,
		},
		{
			name:    "with repoRoot that is not a repo",
			repo:    notARepoPath,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var cleanup func()
			if test.setup != nil {
				cleanup = test.setup(t, workRoot)
			}
			defer func() {
				if cleanup != nil {
					cleanup()
				}
			}()

			repo, err := cloneOrOpenRepo(workRoot, test.repo, test.ci, "main", "")
			if test.wantErr {
				if err == nil {
					t.Error("cloneOrOpenLanguageRepo() expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("cloneOrOpenLanguageRepo() got unexpected error: %v", err)
				return
			}
			if test.check != nil {
				if repo == nil {
					t.Fatal("cloneOrOpenLanguageRepo() returned nil repo but no error")
				}
				test.check(t, repo)
			}
		})
	}
}

func TestCleanAndCopyLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		libraryID    string
		state        *config.LibrarianState
		repo         gitrepo.Repository
		outputDir    string
		setup        func(t *testing.T, repoDir, outputDir string)
		wantErr      bool
		errContains  string
		shouldCopy   []string
		shouldDelete []string
	}{
		{
			name:      "library not found",
			libraryID: "non-existent-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
					},
				},
			},
			repo:        newTestGitRepo(t),
			wantErr:     true,
			errContains: "not found during clean and copy",
		},
		{
			name:      "clean fails",
			libraryID: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:          "some-library",
						RemoveRegex: []string{"["}, // Invalid regex
					},
				},
			},
			repo:        newTestGitRepo(t),
			wantErr:     true,
			errContains: "failed to clean library",
		},
		{
			name:      "copy fails on symlink",
			libraryID: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "some-library",
						SourceRoots: []string{
							"symlink",
						},
					},
				},
			},
			repo: newTestGitRepo(t),
			setup: func(t *testing.T, repoDir, outputDir string) {
				// Create a symlink in the output directory to trigger an error.
				if err := os.Symlink("target", filepath.Join(outputDir, "symlink")); err != nil {
					t.Fatalf("os.Symlink() = %v", err)
				}
			},
			wantErr:     true,
			errContains: "failed to copy",
		},
		{
			name:      "empty RemoveRegex defaults to source root",
			libraryID: "some-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:          "some-library",
						SourceRoots: []string{"a/path"},
					},
				},
			},
			repo: newTestGitRepo(t),
			setup: func(t *testing.T, repoDir, outputDir string) {
				// Create a stale file in the repo directory to test cleaning.
				staleFile := filepath.Join(repoDir, "a/path/stale.txt")
				if err := os.MkdirAll(filepath.Dir(staleFile), 0755); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Create(staleFile); err != nil {
					t.Fatal(err)
				}

				// Create generated files in the output directory.
				filesToCreate := []string{
					"a/path/new_generated_file_to_copy.txt",
					"skipped/path/example.txt",
				}
				for _, relPath := range filesToCreate {
					fullPath := filepath.Join(outputDir, relPath)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						t.Fatal(err)
					}
					if _, err := os.Create(fullPath); err != nil {
						t.Fatal(err)
					}
				}
			},
			shouldCopy: []string{
				"a/path/new_generated_file_to_copy.txt",
			},
			shouldDelete: []string{
				"skipped/path/example.txt",
				"a/path/stale.txt",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := test.repo.GetDir()
			outputDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, repoDir, outputDir)
			}
			err := cleanAndCopyLibrary(test.state, repoDir, test.libraryID, outputDir)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				if !strings.Contains(err.Error(), test.errContains) {
					t.Errorf("want: %s, got %s", test.errContains, err.Error())
				}

				return
			}
			if err != nil {
				t.Fatal(err)
			}

			for _, file := range test.shouldCopy {
				fullPath := filepath.Join(repoDir, file)
				if _, err := os.Stat(fullPath); err != nil {
					t.Errorf("file %s is not copied to %s", file, repoDir)
				}
			}

			for _, file := range test.shouldDelete {
				fullPath := filepath.Join(repoDir, file)
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					t.Errorf("file %s should not be copied to %s", file, repoDir)
				}
			}
		})
	}
}

func TestCopyOneLibrary(t *testing.T) {
	t.Parallel()
	// Create files in src directory.
	setup := func(src string, files []string) {
		for _, relPath := range files {
			fullPath := filepath.Join(src, relPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Error(err)
			}

			if _, err := os.Create(fullPath); err != nil {
				t.Error(err)
			}
		}
	}
	for _, test := range []struct {
		name          string
		dst           string
		src           string
		library       *config.LibraryState
		filesToCreate []string
		wantFiles     []string
		skipFiles     []string
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name: "copied a library",
			dst:  filepath.Join(t.TempDir(), "dst"),
			src:  filepath.Join(t.TempDir(), "src"),
			library: &config.LibraryState{
				ID: "example-library",
				SourceRoots: []string{
					"a/path",
					"another/path",
				},
			},
			filesToCreate: []string{
				"a/path/example.txt",
				"another/path/example.txt",
				"skipped/path/example.txt",
			},
			wantFiles: []string{
				"a/path/example.txt",
				"another/path/example.txt",
			},
			skipFiles: []string{
				"skipped/path/example.txt",
			},
		},
		{
			name: "invalid src",
			dst:  os.TempDir(),
			src:  "/invalid-path",
			library: &config.LibraryState{
				ID: "example-library",
				SourceRoots: []string{
					"a-library/path",
				},
			},
			wantErr:    true,
			wantErrMsg: "failed to copy",
		},
		{
			name: "invalid dst",
			dst:  "/invalid-path",
			src:  os.TempDir(),
			library: &config.LibraryState{
				ID: "example-library",
				SourceRoots: []string{
					"a-library/path",
				},
			},
			wantErr:    true,
			wantErrMsg: "failed to copy",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !test.wantErr {
				setup(test.src, test.filesToCreate)
			}
			err := copyLibrary(test.dst, test.src, test.library)
			if test.wantErr {
				if err == nil {
					t.Errorf("copyOneLibrary() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("failed to run copyOneLibrary(): %s", err.Error())
			}

			for _, file := range test.wantFiles {
				fullPath := filepath.Join(test.dst, file)
				if _, err := os.Stat(fullPath); err != nil {
					t.Errorf("file %s is not copied to %s", file, test.dst)
				}
			}

			for _, file := range test.skipFiles {
				fullPath := filepath.Join(test.dst, file)
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					t.Errorf("file %s should not be copied to %s", file, test.dst)
				}
			}
		})
	}
}

func TestClean(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name             string
		files            map[string]string
		setup            func(t *testing.T, tmpDir string)
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
		{
			name: "remove directory",
			files: map[string]string{
				"dir/file1.txt": "",
				"dir/file2.txt": "",
			},
			removePatterns: []string{"dir"},
			wantRemaining:  []string{"."},
		},
		{
			name: "preserve file not matching remove pattern",
			files: map[string]string{
				"file1.txt": "",
				"file2.log": "",
			},
			removePatterns: []string{".*\\.txt"},
			wantRemaining:  []string{".", "file2.log"},
		},
		{
			name: "remove file fails on permission error",
			files: map[string]string{
				"readonlydir/file.txt": "content",
			},
			setup: func(t *testing.T, tmpDir string) {
				// Make the directory read-only to cause os.Remove to fail.
				readOnlyDir := filepath.Join(tmpDir, "readonlydir")
				if err := os.Chmod(readOnlyDir, 0555); err != nil {
					t.Fatalf("os.Chmod() = %v", err)
				}
				// Register a cleanup function to restore permissions so TempDir can be removed.
				t.Cleanup(func() {
					_ = os.Chmod(readOnlyDir, 0755)
				})
			},
			removePatterns: []string{"readonlydir/file.txt"},
			wantRemaining:  []string{".", "readonlydir", "readonlydir/file.txt"},
			wantErr:        true,
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
			if test.setup != nil {
				test.setup(t, tmpDir)
			}
			err := clean(tmpDir, test.removePatterns, test.preservePatterns)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			remainingPaths, err := allPaths(tmpDir)
			if err != nil {
				t.Fatalf("allPaths() = %v", err)
			}
			sort.Strings(test.wantRemaining)
			sort.Strings(remainingPaths)
			if diff := cmp.Diff(test.wantRemaining, remainingPaths); diff != "" {
				t.Errorf("clean() remaining files mismatch (-want +got):%s", diff)
			}

		})
	}
}

func TestSortDirsByDepth(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		dirs []string
		want []string
	}{
		{
			name: "simple case",
			dirs: []string{
				"a/b",
				"short-dir",
				"a/b/c",
				"a",
			},
			want: []string{
				"a/b/c",
				"a/b",
				"short-dir",
				"a",
			},
		},
		{
			name: "empty",
			dirs: []string{},
			want: []string{},
		},
		{
			name: "single dir",
			dirs: []string{"a"},
			want: []string{"a"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sortDirsByDepth(tc.dirs)
			if diff := cmp.Diff(tc.want, tc.dirs); diff != "" {
				t.Errorf("sortDirsByDepth() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAllPaths(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		setup       func(t *testing.T, tmpDir string)
		wantPaths   []string
		wantErr     bool
		errorString string
	}{
		{
			name: "success",
			setup: func(t *testing.T, tmpDir string) {
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
			},
			wantPaths: []string{
				".",
				"dir1",
				"dir1/dir2",
				"dir1/dir2/file3.txt",
				"dir1/file2.txt",
				"file1.txt",
			},
		},
		{
			name: "unreadable directory",
			setup: func(t *testing.T, tmpDir string) {
				unreadableDir := filepath.Join(tmpDir, "unreadable")
				if err := os.Mkdir(unreadableDir, 0755); err != nil {
					t.Fatalf("os.Mkdir() = %v", err)
				}

				// Make the directory unreadable to trigger an error in filepath.WalkDir.
				if err := os.Chmod(unreadableDir, 0000); err != nil {
					t.Fatalf("os.Chmod() = %v", err)
				}
				// Schedule cleanup to restore permissions so TempDir can be removed.
				t.Cleanup(func() {
					_ = os.Chmod(unreadableDir, 0755)
				})
			},
			wantErr:     true,
			errorString: "unreadable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, tmpDir)
			}

			paths, err := allPaths(tmpDir)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			// Sort both slices to ensure consistent comparison.
			sort.Strings(paths)
			sort.Strings(test.wantPaths)

			if diff := cmp.Diff(test.wantPaths, paths); diff != "" {
				t.Errorf("allPaths() mismatch (-want +got):\n%s", diff)
			}
		})
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
		regexp.MustCompile(`^foo/.*\.txt$`),
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

func TestDeriveFinalPathsToRemove(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name             string
		files            map[string]string
		removePatterns   []string
		preservePatterns []string
		wantToRemove     []string
		wantErr          bool
	}{
		{
			name: "remove all txt files, preserve nothing",
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file2.txt": "",
				"dir2/file3.log": "",
			},
			removePatterns:   []string{`.*\.txt`},
			preservePatterns: []string{},
			wantToRemove:     []string{"file1.txt", "dir1/file2.txt"},
		},
		{
			name: "remove all files, preserve log files",
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file2.txt": "",
				"dir2/file3.log": "",
			},
			removePatterns:   []string{".*"},
			preservePatterns: []string{`.*\.log`},
			wantToRemove:     []string{".", "dir1", "dir2", "file1.txt", "dir1/file2.txt"},
		},
		{
			name: "remove files in dir1, preserve nothing",
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file2.txt": "",
				"dir1/file3.log": "",
				"dir2/file4.txt": "",
			},
			removePatterns:   []string{`dir1/.*`},
			preservePatterns: []string{},
			wantToRemove:     []string{"dir1/file2.txt", "dir1/file3.log"},
		},
		{
			name: "remove all, preserve files in dir2",
			files: map[string]string{
				"file1.txt":      "",
				"dir1/file2.txt": "",
				"dir2/file3.txt": "",
			},
			removePatterns:   []string{".*"},
			preservePatterns: []string{`dir2/.*`},
			wantToRemove:     []string{".", "dir1", "dir2", "file1.txt", "dir1/file2.txt"},
		},
		{
			name:             "no files",
			files:            map[string]string{},
			removePatterns:   []string{".*"},
			preservePatterns: []string{},
			wantToRemove:     []string{"."},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
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

			gotToRemove, err := deriveFinalPathsToRemove(tmpDir, test.removePatterns, test.preservePatterns)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			sort.Strings(gotToRemove)
			sort.Strings(test.wantToRemove)

			if diff := cmp.Diff(test.wantToRemove, gotToRemove); diff != "" {
				t.Errorf("deriveFinalPathsToRemove() toRemove mismatch in %s (-want +got):\n%s", test.name, diff)
			}
		})
	}
}

func TestSeparateFilesAndDirs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		setup     func(t *testing.T, tmpDir string)
		paths     []string
		wantFiles []string
		wantDirs  []string
		wantErr   bool
	}{
		{
			name: "mixed files, dirs, and non-existent path",
			setup: func(t *testing.T, tmpDir string) {
				files := []string{"file1.txt", "dir1/file2.txt"}
				dirs := []string{"dir1", "dir2"}
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
			},
			paths:     []string{"file1.txt", "dir1/file2.txt", "dir1", "dir2", "non-existent-file"},
			wantFiles: []string{"file1.txt", "dir1/file2.txt"},
			wantDirs:  []string{"dir1", "dir2"},
		},
		{
			name:    "stat error",
			paths:   []string{strings.Repeat("a", 300)},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, tmpDir)
			}

			gotFiles, gotDirs, err := separateFilesAndDirs(tmpDir, test.paths)
			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			sort.Strings(gotFiles)
			sort.Strings(gotDirs)
			sort.Strings(test.wantFiles)
			sort.Strings(test.wantDirs)

			if diff := cmp.Diff(test.wantFiles, gotFiles); diff != "" {
				t.Errorf("separateFilesAndDirs() files mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantDirs, gotDirs); diff != "" {
				t.Errorf("separateFilesAndDirs() dirs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompileRegexps(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
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
	} {
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

func TestCommitAndPush(t *testing.T) {
	for _, test := range []struct {
		name            string
		setupMockRepo   func(t *testing.T) gitrepo.Repository
		setupMockClient func(t *testing.T) GitHubClient
		state           *config.LibrarianState
		prType          string
		commit          bool
		push            bool
		wantErr         bool
		expectedErrMsg  string
	}{
		{
			name: "Push flag and Commit flag are not specified",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				repoDir := newTestGitRepoWithCommit(t, "")
				repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{Dir: repoDir})
				if err != nil {
					t.Fatalf("Failed to create test repo: %v", err)
				}
				return repo
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			prType: "generate",
		},
		{
			name: "create a commit",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})
				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return &mockGitHubClient{
					createdPR: &github.PullRequestMetadata{Number: 123, Repo: &github.Repository{Owner: "test-owner", Name: "test-repo"}},
				}
			},
			prType: "generate",
			commit: true,
		},
		{
			name: "create a generate pull request",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})
				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return &mockGitHubClient{
					createdPR: &github.PullRequestMetadata{Number: 123, Repo: &github.Repository{Owner: "test-owner", Name: "test-repo"}},
				}
			},
			state:  &config.LibrarianState{},
			prType: "generate",
			push:   true,
		},
		{
			name: "create a release pull request",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})
				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return &mockGitHubClient{
					createdPR: &github.PullRequestMetadata{Number: 123, Repo: &github.Repository{Owner: "test-owner", Name: "test-repo"}},
				}
			},
			state:  &config.LibrarianState{},
			prType: "release",
			push:   true,
		},
		{
			name: "No GitHub Remote",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{}, // No remotes
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			prType:         "generate",
			push:           true,
			wantErr:        true,
			expectedErrMsg: "could not find an 'origin' remote",
		},
		{
			name: "AddAll error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})
				return &MockRepository{
					Dir:          t.TempDir(),
					RemotesValue: []*git.Remote{remote},
					AddAllError:  errors.New("mock add all error"),
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			prType:         "generate",
			push:           true,
			wantErr:        true,
			expectedErrMsg: "mock add all error",
		},
		{
			name: "Create branch error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:                          t.TempDir(),
					AddAllStatus:                 status,
					RemotesValue:                 []*git.Remote{remote},
					CreateBranchAndCheckoutError: errors.New("create branch error"),
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			prType:         "generate",
			push:           true,
			wantErr:        true,
			expectedErrMsg: "create branch error",
		},
		{
			name: "Commit error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
					CommitError:  errors.New("commit error"),
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			prType:         "generate",
			push:           true,
			wantErr:        true,
			expectedErrMsg: "commit error",
		},
		{
			name: "Push error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
					PushError:    errors.New("push error"),
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			prType:         "generate",
			push:           true,
			wantErr:        true,
			expectedErrMsg: "push error",
		},
		{
			name: "Create PR body error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return &mockGitHubClient{}
			},
			state:          &config.LibrarianState{},
			prType:         "random",
			push:           true,
			wantErr:        true,
			expectedErrMsg: "failed to create pull request body",
		},
		{
			name: "Create pull request error",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})

				status := make(git.Status)
				status["file.txt"] = &git.FileStatus{Worktree: git.Modified}
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: status,
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return &mockGitHubClient{
					createPullRequestErr: errors.New("create pull request error"),
				}
			},
			state:          &config.LibrarianState{},
			prType:         "generate",
			push:           true,
			wantErr:        true,
			expectedErrMsg: "failed to create pull request",
		},
		{
			name: "No changes to commit",
			setupMockRepo: func(t *testing.T) gitrepo.Repository {
				remote := git.NewRemote(memory.NewStorage(), &gogitConfig.RemoteConfig{
					Name: "origin",
					URLs: []string{"https://github.com/googleapis/librarian.git"},
				})
				return &MockRepository{
					Dir:          t.TempDir(),
					AddAllStatus: git.Status{}, // Clean status
					RemotesValue: []*git.Remote{remote},
				}
			},
			setupMockClient: func(t *testing.T) GitHubClient {
				return nil
			},
			prType: "generate",
			push:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := test.setupMockRepo(t)
			client := test.setupMockClient(t)
			localConfig := &config.Config{
				Push:   test.push,
				Commit: test.commit,
			}
			commitInfo := &commitInfo{
				cfg:           localConfig,
				state:         test.state,
				repo:          repo,
				ghClient:      client,
				commitMessage: "",
				prType:        test.prType,
			}

			err := commitAndPush(context.Background(), commitInfo)

			if test.wantErr {
				if err == nil {
					t.Errorf("commitAndPush() expected error, got nil")
				} else if test.expectedErrMsg != "" && !strings.Contains(err.Error(), test.expectedErrMsg) {
					t.Errorf("commitAndPush() error = %v, expected to contain: %q", err, test.expectedErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("%s: commitAndPush() returned unexpected error: %v", test.name, err)
				return
			}
		})
	}
}

func TestCopyLibraryFiles(t *testing.T) {
	t.Parallel()
	setup := func(src string, files []string) {
		for _, relPath := range files {
			fullPath := filepath.Join(src, relPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Error(err)
			}

			if _, err := os.Create(fullPath); err != nil {
				t.Error(err)
			}
		}
	}
	for _, test := range []struct {
		name          string
		repoDir       string
		outputDir     string
		libraryID     string
		state         *config.LibrarianState
		filesToCreate []string
		wantFiles     []string
		skipFiles     []string
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name:      "copy library files",
			repoDir:   filepath.Join(t.TempDir(), "dst"),
			outputDir: filepath.Join(t.TempDir(), "src"),
			libraryID: "example-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "example-library",
						SourceRoots: []string{
							"a/path",
							"another/path",
						},
					},
				},
			},
			filesToCreate: []string{
				"a/path/example.txt",
				"another/path/example.txt",
				"skipped/path/example.txt",
			},
			wantFiles: []string{
				"a/path/example.txt",
				"another/path/example.txt",
			},
			skipFiles: []string{
				"skipped/path/example.txt",
			},
		},
		{
			name:      "library not found",
			repoDir:   filepath.Join(t.TempDir(), "dst"),
			outputDir: filepath.Join(t.TempDir(), "src"),
			libraryID: "non-existent-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "example-library",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "not found",
		},
		{
			repoDir:   filepath.Join(t.TempDir(), "dst"),
			name:      "one source root empty",
			outputDir: filepath.Join(t.TempDir(), "src"),
			libraryID: "example-library",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID: "example-library",
						SourceRoots: []string{
							"a/path",
							"another/path",
						},
					},
				},
			},
			filesToCreate: []string{
				"a/path/example.txt",
				"skipped/path/example.txt",
			},
			wantFiles: []string{
				"a/path/example.txt",
			},
			skipFiles: []string{
				"skipped/path/example.txt",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !test.wantErr {
				setup(test.outputDir, test.filesToCreate)
			}
			err := copyLibraryFiles(test.state, test.repoDir, test.libraryID, test.outputDir)
			if test.wantErr {
				if err == nil {
					t.Errorf("copyLibraryFiles() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}
			if err != nil {
				t.Errorf("failed to run copyLibraryFiles(): %s", err.Error())
			}

			for _, file := range test.wantFiles {
				fullPath := filepath.Join(test.repoDir, file)
				if _, err := os.Stat(fullPath); err != nil {
					t.Errorf("file %s is not copied to %s", file, test.repoDir)
				}
			}

			for _, file := range test.skipFiles {
				fullPath := filepath.Join(test.repoDir, file)
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					t.Errorf("file %s should not be copied to %s", file, test.repoDir)
				}
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		dst         string
		src         string
		wantSrcFile bool
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:       "invalid src",
			src:        "/invalid-path/example.txt",
			wantErr:    true,
			wantErrMsg: "failed to open file",
		},
		{
			name:        "invalid dst path",
			src:         filepath.Join(os.TempDir(), "example.txt"),
			dst:         "/invalid-path/example.txt",
			wantSrcFile: true,
			wantErr:     true,
			wantErrMsg:  "failed to make directory",
		},
		{
			name:        "invalid dst file",
			src:         filepath.Join(os.TempDir(), "example.txt"),
			dst:         filepath.Join(os.TempDir(), "example\x00.txt"),
			wantSrcFile: true,
			wantErr:     true,
			wantErrMsg:  "failed to create file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.wantSrcFile {
				if err := os.MkdirAll(filepath.Dir(test.src), 0755); err != nil {
					t.Error(err)
				}
				sourceFile, err := os.Create(test.src)
				if err != nil {
					t.Error(err)
				}
				if err := sourceFile.Close(); err != nil {
					t.Error(err)
				}
			}
			err := copyFile(test.dst, test.src)
			if test.wantErr {
				if err == nil {
					t.Errorf("copyFile() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}

		})
	}
}
