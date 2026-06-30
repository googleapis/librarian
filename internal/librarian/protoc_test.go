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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInstallDir(t *testing.T) {
	for _, test := range []struct {
		name         string
		version      string
		librarianBin string
		cacheDir     string
		want         string
	}{
		{
			name:         "valid version with LIBRARIAN_BIN",
			version:      "25.1",
			librarianBin: "/custom/bin",
			want:         filepath.FromSlash("/custom/bin/protoc/v25.1"),
		},
		{
			name:     "valid version with LIBRARIAN_CACHE fallback",
			version:  "26.0-rc1",
			cacheDir: "/custom/cache",
			want:     filepath.FromSlash("/custom/cache/bin/protoc/v26.0-rc1"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.librarianBin != "" {
				t.Setenv("LIBRARIAN_BIN", test.librarianBin)
			} else {
				t.Setenv("LIBRARIAN_BIN", "")
			}

			if test.cacheDir != "" {
				t.Setenv("LIBRARIAN_CACHE", test.cacheDir)
			} else {
				t.Setenv("LIBRARIAN_CACHE", "")
			}

			got, err := installDir(test.version)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDownloadURL(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "simple version",
			version: "25.1",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v25.1/protoc-25.1-linux-x86_64.zip",
		},
		{
			name:    "release candidate",
			version: "26.0-rc1",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v26.0-rc1/protoc-26.0-rc1-linux-x86_64.zip",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := downloadURL(test.version)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
