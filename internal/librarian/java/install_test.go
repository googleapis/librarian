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

package java

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetBinDir(t *testing.T) {
	tmpDir := t.TempDir()
	for _, test := range []struct {
		name           string
		librarianBin   string
		librarianCache string
		want           string
	}{
		{
			name:         "LIBRARIAN_BIN is set",
			librarianBin: tmpDir,
			want:         filepath.Join(tmpDir, "java_tools", "bin"),
		},
		{
			name:           "LIBRARIAN_CACHE is set",
			librarianCache: tmpDir,
			want:           filepath.Join(tmpDir, "bin", "java_tools", "bin"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LIBRARIAN_BIN", test.librarianBin)
			t.Setenv("LIBRARIAN_CACHE", test.librarianCache)
			got, err := getBinDir()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetLibDir(t *testing.T) {
	tmpDir := t.TempDir()
	for _, test := range []struct {
		name           string
		librarianBin   string
		librarianCache string
		want           string
	}{
		{
			name:         "LIBRARIAN_BIN is set",
			librarianBin: tmpDir,
			want:         filepath.Join(tmpDir, "java_tools", "lib"),
		},
		{
			name:           "LIBRARIAN_CACHE is set",
			librarianCache: tmpDir,
			want:           filepath.Join(tmpDir, "bin", "java_tools", "lib"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LIBRARIAN_BIN", test.librarianBin)
			t.Setenv("LIBRARIAN_CACHE", test.librarianCache)
			got, err := getLibDir()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetToolsEnv(t *testing.T) {
	tmpDir := t.TempDir()
	for _, test := range []struct {
		name           string
		librarianBin   string
		librarianCache string
		want           map[string]string
	}{
		{
			name:         "LIBRARIAN_BIN is set",
			librarianBin: tmpDir,
			want:         map[string]string{"PATH": filepath.Join(tmpDir, "java_tools", "bin")},
		},
		{
			name:           "LIBRARIAN_CACHE is set",
			librarianCache: tmpDir,
			want:           map[string]string{"PATH": filepath.Join(tmpDir, "bin", "java_tools", "bin")},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LIBRARIAN_BIN", test.librarianBin)
			t.Setenv("LIBRARIAN_CACHE", test.librarianCache)
			got, err := getToolsEnv()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
