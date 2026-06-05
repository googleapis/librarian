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

package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDirectory(t *testing.T) {
	for _, test := range []struct {
		name    string
		env     string
		wantDir string
	}{
		{
			name:    "uses LIBRARIAN_CACHE when set",
			env:     "/custom/cache",
			wantDir: "/custom/cache",
		},
		{
			name: "uses UserCacheDir/librarian when LIBRARIAN_CACHE not set",
			env:  "",
			wantDir: func() string {
				cache, _ := os.UserCacheDir()
				return filepath.Join(cache, "librarian")
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.env != "" {
				t.Setenv(EnvLibrarianCache, test.env)
			}
			got, err := Directory()
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantDir, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBinDirectory(t *testing.T) {
	for _, test := range []struct {
		name     string
		binEnv   string
		cacheEnv string
		wantDir  string
	}{
		{
			name:    "uses LIBRARIAN_BIN when set",
			binEnv:  "/custom/bin",
			wantDir: "/custom/bin",
		},
		{
			name:     "uses LIBRARIAN_CACHE/bin when LIBRARIAN_BIN not set",
			cacheEnv: "/custom/cache",
			wantDir:  "/custom/cache/bin",
		},
		{
			name: "uses UserCacheDir/librarian/bin when neither set",
			wantDir: func() string {
				cache, _ := os.UserCacheDir()
				return filepath.Join(cache, "librarian", "bin")
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(EnvLibrarianBin, test.binEnv)
			t.Setenv(EnvLibrarianCache, test.cacheEnv)
			got, err := BinDirectory()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantDir, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
