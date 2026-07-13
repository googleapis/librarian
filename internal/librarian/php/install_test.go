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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
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
<<<<<<< HEAD
	for _, test := range []struct {
		name       string
		packageURL string
		want       string
	}{
		{
			name:       "success",
			packageURL: "https://github.com/googleapis/gapic-generator-php/archive/refs/tags/v1.21.2.tar.gz",
			want:       "github.com/googleapis/gapic-generator-php",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := repoFromPackageURL(test.packageURL)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("repoFromPackageURL() = %q, want %q", got, test.want)
			}
		})
=======
	packageURL := "https://github.com/googleapis/gapic-generator-php/archive/refs/tags/v1.21.2.tar.gz"
	want := "github.com/googleapis/gapic-generator-php"
	got, err := repoFromPackageURL(packageURL)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
>>>>>>> e14a92a9 (feat(php): support dynamic composer installation)
	}
}

func TestRepoFromPackageURL_Error(t *testing.T) {
<<<<<<< HEAD
	for _, test := range []struct {
		name       string
		packageURL string
	}{
		{
			name:       "invalid URL",
			packageURL: "https://github.com/googleapis/gapic-generator-php/tarball/v1.21.2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := repoFromPackageURL(test.packageURL); err == nil {
				t.Fatal("repoFromPackageURL() expected error, got nil")
			}
		})
=======
	packageURL := "https://github.com/googleapis/gapic-generator-php/tarball/v1.21.2"
	if _, err := repoFromPackageURL(packageURL); !errors.Is(err, errCannotExtractRepo) {
		t.Fatalf("repoFromPackageURL() error = %v, want %v", err, errCannotExtractRepo)
>>>>>>> e14a92a9 (feat(php): support dynamic composer installation)
	}
}
