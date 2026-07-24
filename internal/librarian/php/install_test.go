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

package php

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"os/exec"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestInstallDir(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	got, err := InstallDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(binDir, "php_tools")
	if got != want {
		t.Errorf("InstallDir() = %q, want %q", got, want)
	}
}

func TestBinDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", dir)
	got, err := binDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "php_tools", "bin")
	if got != want {
		t.Errorf("binDir() = %q, want %q", got, want)
	}
}

func TestInstall(t *testing.T) {
	for _, test := range []struct {
		name    string
		tools   *config.Tools
		setup   func(t *testing.T)
		wantErr error
		check   func(t *testing.T)
	}{
		{
			name:  "no tools, uses fallback generator",
			tools: nil,
			setup: func(t *testing.T) {
				cache := t.TempDir()
				t.Setenv("LIBRARIAN_CACHE", cache)
				t.Setenv("LIBRARIAN_BIN", filepath.Join(cache, "bin"))
				repoDir := filepath.Join(cache, "github.com/googleapis/gapic-generator-php@v1.21.2")
				if err := os.MkdirAll(filepath.Join(repoDir, "dummy"), 0o755); err != nil {
					t.Fatal(err)
				}
				bin := t.TempDir()
				writeExecutable(t, filepath.Join(bin, "php"), "#!/bin/sh\nexit 0\n")
				writeExecutable(t, filepath.Join(bin, "composer"), "#!/bin/sh\nexit 0\n")
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
		},
		{
			name: "with composer and pip tools",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "gapic-generator-php",
						Version: "1.0.0",
						Repo:    "github.com/googleapis/gapic-generator-php",
						SHA256:  "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
					},
				},
				Pip: []*config.PipTool{
					{
						Name:    "fake-pip-tool",
						Version: "2.0.0",
					},
				},
			},
			setup: func(t *testing.T) {
				cache := t.TempDir()
				t.Setenv("LIBRARIAN_CACHE", cache)
				t.Setenv("LIBRARIAN_BIN", filepath.Join(cache, "bin"))
				repoDir := filepath.Join(cache, "github.com/googleapis/gapic-generator-php@1.0.0")
				if err := os.MkdirAll(filepath.Join(repoDir, "dummy"), 0o755); err != nil {
					t.Fatal(err)
				}

				bin := t.TempDir()
				writeExecutable(t, filepath.Join(bin, "composer"), "#!/bin/sh\nexit 0\n")
				writeExecutable(t, filepath.Join(bin, "pip"), "#!/bin/sh\nexit 0\n")
				writeExecutable(t, filepath.Join(bin, "php"), "#!/bin/sh\nexit 0\n")
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			check: func(t *testing.T) {
				binDir := filepath.Join(os.Getenv("LIBRARIAN_BIN"), "php_tools", "bin")
				wrapperPath := filepath.Join(binDir, "gapic-generator-php")
				b, err := os.ReadFile(wrapperPath)
				if err != nil {
					t.Fatal(err)
				}
				repoDir := filepath.Join(os.Getenv("LIBRARIAN_CACHE"), "github.com/googleapis/gapic-generator-php@1.0.0")
				destPath := filepath.Join(repoDir, "src", "Main.php")
				phpPath, _ := exec.LookPath("php")
				want := phpWrapperContent(phpPath, destPath)
				if diff := cmp.Diff(want, string(b)); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "with composer, pip, and pnpm tools",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "gapic-generator-php",
						Version: "1.0.0",
						Repo:    "github.com/googleapis/gapic-generator-php",
						SHA256:  "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
					},
				},
				Pip: []*config.PipTool{
					{
						Name:    "fake-pip-tool",
						Version: "2.0.0",
					},
				},
				PNPM: []*config.PNPMTool{
					{
						Name:    "fake-pnpm-tool",
						Version: "3.0.0",
					},
				},
			},
			setup: func(t *testing.T) {
				cache := t.TempDir()
				t.Setenv("LIBRARIAN_CACHE", cache)
				t.Setenv("LIBRARIAN_BIN", filepath.Join(cache, "bin"))
				repoDir := filepath.Join(cache, "github.com/googleapis/gapic-generator-php@1.0.0")
				if err := os.MkdirAll(filepath.Join(repoDir, "dummy"), 0o755); err != nil {
					t.Fatal(err)
				}

				bin := t.TempDir()
				writeExecutable(t, filepath.Join(bin, "composer"), "#!/bin/sh\nexit 0\n")
				writeExecutable(t, filepath.Join(bin, "pip"), "#!/bin/sh\nexit 0\n")
				writeExecutable(t, filepath.Join(bin, "node"), "#!/bin/sh\nexit 0\n")
				writeExecutable(t, filepath.Join(bin, "pnpm"), "#!/bin/sh\nexit 0\n")
				writeExecutable(t, filepath.Join(bin, "php"), "#!/bin/sh\nexit 0\n")
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			check: func(t *testing.T) {
				binDir := filepath.Join(os.Getenv("LIBRARIAN_BIN"), "php_tools", "bin")
				wrapperPath := filepath.Join(binDir, "gapic-generator-php")
				b, err := os.ReadFile(wrapperPath)
				if err != nil {
					t.Fatal(err)
				}
				repoDir := filepath.Join(os.Getenv("LIBRARIAN_CACHE"), "github.com/googleapis/gapic-generator-php@1.0.0")
				destPath := filepath.Join(repoDir, "src", "Main.php")
				phpPath, _ := exec.LookPath("php")
				want := phpWrapperContent(phpPath, destPath)
				if diff := cmp.Diff(want, string(b)); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "with gapic-generator-php tool",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "fake-gapic-generator",
						Version: "1.0.0",
						Repo:    "github.com/googleapis/gapic-generator-php",
						SHA256:  "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
					},
				},
			},
			setup: func(t *testing.T) {
				cache := t.TempDir()
				t.Setenv("LIBRARIAN_CACHE", cache)
				t.Setenv("LIBRARIAN_BIN", filepath.Join(cache, "bin"))
				repoDir := filepath.Join(cache, "github.com/googleapis/gapic-generator-php@1.0.0")
				if err := os.MkdirAll(filepath.Join(repoDir, "dummy"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(repoDir, "src"), 0o755); err != nil {
					t.Fatal(err)
				}
				bin := t.TempDir()
				writeExecutable(t, filepath.Join(bin, "composer"), "#!/bin/sh\nexit 0\n")
				writeExecutable(t, filepath.Join(bin, "php"), "#!/bin/sh\nexit 0\n")
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			check: func(t *testing.T) {
				binDir := filepath.Join(os.Getenv("LIBRARIAN_BIN"), "php_tools", "bin")
				wrapperPath := filepath.Join(binDir, "gapic-generator-php")
				b, err := os.ReadFile(wrapperPath)
				if err != nil {
					t.Fatal(err)
				}
				repoDir := filepath.Join(os.Getenv("LIBRARIAN_CACHE"), "github.com/googleapis/gapic-generator-php@1.0.0")
				destPath := filepath.Join(repoDir, "src", "Main.php")
				phpPath, _ := exec.LookPath("php")
				want := phpWrapperContent(phpPath, destPath)
				if diff := cmp.Diff(want, string(b)); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			err := Install(t.Context(), test.tools)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Install() error = %v, wantErr = %v", err, test.wantErr)
			}
			if test.check != nil {
				test.check(t)
			}
		})
	}
}

func TestInstall_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		tools   *config.Tools
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name: "missing repo URL",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "gapic-generator-php",
						Version: "1.0.0",
					},
				},
			},
			wantErr: errMissingRepo,
		},
		{
			name: "missing composer tool in PATH",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "gapic-generator-php",
						Version: "1.0.0",
						Repo:    "github.com/googleapis/gapic-generator-php",
					},
				},
			},
			setup: func(t *testing.T) {
				cache := t.TempDir()
				t.Setenv("LIBRARIAN_CACHE", cache)
				t.Setenv("LIBRARIAN_BIN", filepath.Join(cache, "bin"))
				repoDir := filepath.Join(cache, "github.com/googleapis/gapic-generator-php@1.0.0")
				if err := os.MkdirAll(filepath.Join(repoDir, "dummy"), 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", t.TempDir())
			},
			wantErr: exec.ErrNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			err := Install(t.Context(), test.tools)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Install() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestCreateBinWrapper(t *testing.T) {
	for _, test := range []struct {
		name        string
		wrapperName string
	}{
		{
			name:        "simple wrapper",
			wrapperName: "foo",
		},
		{
			name:        "nested wrapper",
			wrapperName: "nested/dir/foo",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			binDir := t.TempDir()
			destPath := "/path/to/dest"
			content := fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", destPath)
			if err := createBinWrapper(test.wrapperName, content, binDir); err != nil {
				t.Fatal(err)
			}
			wrapperPath := filepath.Join(binDir, test.wrapperName)
			b, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(content, string(b)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			info, err := os.Stat(wrapperPath)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o755 {
				t.Errorf("wrapper permissions = %04o, want 0755", info.Mode().Perm())
			}
		})
	}
}

//nolint:unparam // content is the same for all calls but keeping parameter for flexibility
func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
}
