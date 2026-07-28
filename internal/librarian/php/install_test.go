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
	"os"
	"path/filepath"
	"testing"

	"os/exec"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/tool/composer"
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
				testhelper.WriteExecutable(t, filepath.Join(bin, "composer"), "#!/bin/sh\nexit 0\n")
				testhelper.WriteExecutable(t, filepath.Join(bin, "pip"), "#!/bin/sh\nexit 0\n")
				testhelper.WriteExecutable(t, filepath.Join(bin, "node"), "#!/bin/sh\nexit 0\n")
				testhelper.WriteExecutable(t, filepath.Join(bin, "pnpm"), "#!/bin/sh\nexit 0\n")
				testhelper.WriteExecutable(t, filepath.Join(bin, "php"), "#!/bin/sh\nexit 0\n")
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			check: func(t *testing.T) {
				binDir := filepath.Join(os.Getenv("LIBRARIAN_BIN"), "php_tools", "bin")
				wrapperPath := filepath.Join(binDir, "gapic-generator-php")
				if _, err := os.Stat(wrapperPath); err != nil {
					t.Errorf("wrapper file %s not found: %v", wrapperPath, err)
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
	testhelper.RequireCommand(t, "composer")
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
			wantErr: composer.ErrMissingRepo,
		},
		{
			name:    "no tools",
			tools:   nil,
			wantErr: errMissingTools,
		},
		{
			name: "no composer tools",
			tools: &config.Tools{
				Pip: []*config.PipTool{
					{
						Name:    "fake-pip-tool",
						Version: "2.0.0",
					},
				},
			},
			wantErr: errMissingComposer,
		},
		{
			name: "no pip tools",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "gapic-generator-php",
						Version: "1.0.0",
						Repo:    "github.com/googleapis/gapic-generator-php",
					},
				},
			},
			wantErr: errMissingPip,
		},
		{
			name: "no pnpm tools",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "gapic-generator-php",
						Version: "1.0.0",
						Repo:    "github.com/googleapis/gapic-generator-php",
					},
				},
				Pip: []*config.PipTool{
					{
						Name:    "fake-pip-tool",
						Version: "2.0.0",
					},
				},
			},
			wantErr: errMissingPNPM,
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
