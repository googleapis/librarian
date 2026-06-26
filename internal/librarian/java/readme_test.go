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
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecamelize(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "camel case",
			input: "CamelCase",
			want:  "Camel Case",
		},
		{
			name:  "simple word",
			input: "Word",
			want:  "Word",
		},
		{
			name:  "already separated",
			input: "Camel Case",
			want:  "Camel Case",
		},
		{
			name:  "java acronym IamPolicy",
			input: "IamPolicy",
			want:  "Iam Policy",
		},
		{
			name:  "java acronym GcsBucket",
			input: "GcsBucket",
			want:  "Gcs Bucket",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := decamelize(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// mockDirEntry is a mock implementation of os.DirEntry for testing.
type mockDirEntry struct {
	isDir bool
}

func (m mockDirEntry) Name() string               { return "" }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() os.FileMode          { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestIsProductionSample(t *testing.T) {
	for _, test := range []struct {
		name  string
		entry mockDirEntry
		path  string
		want  bool
	}{
		{
			name:  "valid production sample",
			entry: mockDirEntry{isDir: false},
			path:  "samples/src/main/java/com/example/Sample.java",
			want:  true,
		},
		{
			name:  "valid production sample at root",
			entry: mockDirEntry{isDir: false},
			path:  "src/main/java/com/example/Sample.java",
			want:  true,
		},
		{
			name:  "directory instead of file",
			entry: mockDirEntry{isDir: true},
			path:  "samples/src/main/java",
			want:  false,
		},
		{
			name:  "non-java file",
			entry: mockDirEntry{isDir: false},
			path:  "samples/src/main/java/README.md",
			want:  false,
		},
		{
			name:  "not in src/main/java",
			entry: mockDirEntry{isDir: false},
			path:  "samples/src/test/java/com/example/Sample.java",
			want:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isProductionSample(test.entry, test.path)
			if got != test.want {
				t.Errorf("isProductionSample() = %t, want %t", got, test.want)
			}
		})
	}
}
