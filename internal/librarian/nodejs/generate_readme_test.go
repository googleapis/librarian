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

package nodejs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractSampleName(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "v1beta1.some_sample.js", want: "some sample"},
		{input: "foo_bar.js", want: "foo bar"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := extractSampleName(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindSampleMetadata(t *testing.T) {
	type fileInfo struct {
		path    string
		content string
	}
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, dir string) string
		want  []sampleMetadata
	}{
		{
			name: "no samples directory",
			setup: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "packages", "my-package")
			},
			want: nil,
		},
		{
			name: "collects and filters samples",
			setup: func(t *testing.T, dir string) string {
				pkgDir := filepath.Join(dir, "packages", "my-package")
				generatedDir := filepath.Join(pkgDir, "samples", "generated")
				files := []fileInfo{
					{path: "v2.do_something.js", content: "console.log('do something');"},
					{path: "ignored.ts", content: "console.log('typescript');"},
					{path: "sub/v1.nested_sample.js", content: "console.log('nested');"},
				}
				for _, file := range files {
					fullPath := filepath.Join(generatedDir, file.path)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(fullPath, []byte(file.content), 0644); err != nil {
						t.Fatal(err)
					}
				}
				return pkgDir
			},
			want: []sampleMetadata{
				{
					name:     "nested sample",
					filePath: "https://github.com/googleapis/google-cloud-node/blob/main/packages/my-package/samples/generated/sub/v1.nested_sample.js",
				},
				{
					name:     "do something",
					filePath: "https://github.com/googleapis/google-cloud-node/blob/main/packages/my-package/samples/generated/v2.do_something.js",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			targetDir := test.setup(t, tmpDir)
			got, err := findSampleMetadata(targetDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(sampleMetadata{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindSampleMetadata_Error(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "samples", "generated")
	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		t.Fatal(err)
	}
	unreadableSubdir := filepath.Join(generatedDir, "unreadable")
	if err := os.MkdirAll(unreadableSubdir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(unreadableSubdir, 0755)
	})
	_, err := findSampleMetadata(tmpDir)
	if !errors.Is(err, errorFindSampleMetadata) {
		t.Errorf("findSampleMetadata() error = %v, wantErr %v", err, errorFindSampleMetadata)
	}
}

func TestReleaseLevelMarkdown(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "stable", want: releaseLevelStable},
		{input: "preview", want: releaseLevelPreview},
		{input: "other", want: releaseLevelPreview},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := releaseLevelMarkdown(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
