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

func TestRepoFromPackageURL_Success(t *testing.T) {
	packageURL := "https://github.com/googleapis/gapic-generator-php/archive/refs/tags/v1.21.2.tar.gz"
	want := "github.com/googleapis/gapic-generator-php"
	got, err := repoFromPackageURL(packageURL)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRepoFromPackageURL_Error(t *testing.T) {
	packageURL := "https://github.com/googleapis/gapic-generator-php/tarball/v1.21.2"
	if _, err := repoFromPackageURL(packageURL); !errors.Is(err, errCannotExtractRepo) {
		t.Fatalf("repoFromPackageURL() error = %v, want %v", err, errCannotExtractRepo)
	}
}

func TestInstall(t *testing.T) {
	for _, test := range []struct {
		name    string
		tools   *config.Tools
		setup   func(t *testing.T)
		wantErr error
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
				if err := os.WriteFile(filepath.Join(bin, "php"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(bin, "composer"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
		},
		{
			name: "with composer and pip tools",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "fake-composer-tool",
						Version: "1.0.0",
						Package: "https://github.com/fake/fake-tool/archive/refs/tags/1.0.0.tar.gz",
						SHA256:  "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
						Build:   []string{"echo built"},
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
				repoDir := filepath.Join(cache, "github.com/fake/fake-tool@1.0.0")
				if err := os.MkdirAll(filepath.Join(repoDir, "dummy"), 0o755); err != nil {
					t.Fatal(err)
				}

				bin := t.TempDir()
				if err := os.WriteFile(filepath.Join(bin, "sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(bin, "pip"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
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
			name: "missing package URL",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "fake-composer-tool",
						Version: "1.0.0",
					},
				},
			},
			wantErr: errMissingPackageURL,
		},
		{
			name: "invalid package URL",
			tools: &config.Tools{
				Composer: []*config.ComposerTool{
					{
						Name:    "fake-composer-tool",
						Version: "1.0.0",
						Package: "invalid-url",
					},
				},
			},
			wantErr: errCannotExtractRepo,
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
