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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
)

func TestInstall_Error(t *testing.T) {
	for _, test := range []struct {
		name  string
		tools *config.Tools
	}{
		{"nil tools", nil},
		{"empty tools", &config.Tools{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Install(t.Context(), test.tools); !errors.Is(err, errNoToolsSpecified) {
				t.Fatalf("Install() error = %v, want %v", err, errNoToolsSpecified)
			}
		})
	}
}

func TestInstallDir(t *testing.T) {
	for _, test := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "LIBRARIAN_BIN set",
			env:  map[string]string{cache.EnvLibrarianBin: "/custom/install/dir"},
			want: "/custom/install/dir/php_tools",
		},
		{
			name: "LIBRARIAN_BIN empty",
			env:  map[string]string{cache.EnvLibrarianCache: "/my/home/cache"},
			want: "/my/home/cache/bin/php_tools",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}
			got, err := InstallDir()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepoFromPackageURL_Success(t *testing.T) {
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
	}
}

func TestRepoFromPackageURL_Error(t *testing.T) {
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
	}
}
