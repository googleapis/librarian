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
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestIsProductionSample(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want bool
	}{
		{
			name: "valid production sample",
			path: "samples/src/main/java/com/example/Sample.java",
			want: true,
		},
		{
			name: "valid production sample at root",
			path: "src/main/java/com/example/Sample.java",
			want: true,
		},
		{
			name: "non-java file",
			path: "samples/src/main/java/README.md",
			want: false,
		},
		{
			name: "not in src/main/java",
			path: "samples/src/test/java/com/example/Sample.java",
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isProductionSample(test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "success with standard comment",
			content: `// sample-metadata:
//   title: Standard Title`,
			want: "Standard Title",
		},
		{
			name: "success with indented comment",
			content: `//   sample-metadata:
//     title: Indented Title`,
			want: "Indented Title",
		},
		{
			name: "success with single quotes",
			content: `// sample-metadata:
//   title: 'Single Quotes Title'`,
			want: "Single Quotes Title",
		},
		{
			name: "success with double quotes",
			content: `// sample-metadata:
//   title: "Double Quotes Title"`,
			want: "Double Quotes Title",
		},
		{
			name:    "success with windows carriage returns",
			content: "// sample-metadata:\r\n//   title: Windows Title\r\n",
			want:    "Windows Title",
		},
		{
			name: "no metadata block present",
			content: `// This is a standard java file.
public class Normal {}`,
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpPath := filepath.Join(t.TempDir(), "Sample.java")
			if err := os.WriteFile(tmpPath, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := extractTitle(tmpPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractTitle_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		wantErr error
	}{
		{
			name: "missing title line returns error",
			content: `// sample-metadata:
//   description: No title line immediately following!`,
			wantErr: errMissingTitle,
		},
		{
			name: "empty title value returns error",
			content: `// sample-metadata:
//   title: ""`,
			wantErr: errEmptyTitle,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpPath := filepath.Join(t.TempDir(), "Sample.java")
			if err := os.WriteFile(tmpPath, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			_, gotErr := extractTitle(tmpPath)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("extractTitle() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestExtractSamples(t *testing.T) {
	for _, test := range []struct {
		name       string
		setupFiles func(t *testing.T, dir string)
		want       []CodeSample
	}{
		{
			name: "missing samples directory",
			setupFiles: func(t *testing.T, dir string) {
				// Do nothing, tempDir is empty.
			},
			want: nil,
		},
		{
			name: "extract successfully",
			setupFiles: func(t *testing.T, dir string) {
				samplesDir := filepath.Join(dir, "samples", "src", "main", "java")
				if err := os.MkdirAll(samplesDir, 0755); err != nil {
					t.Fatal(err)
				}
				file1 := filepath.Join(samplesDir, "RequesterPays.java")
				content1 := `// sample-metadata:
//   title: Custom Title Override
public class RequesterPays {}`
				if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
					t.Fatal(err)
				}
				file2 := filepath.Join(samplesDir, "DemoSample.java")
				content2 := `public class DemoSample {}`
				if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: []CodeSample{
				{
					Title: "Demo Sample",
					File:  "samples/src/main/java/DemoSample.java",
				},
				{
					Title: "Custom Title Override",
					File:  "samples/src/main/java/RequesterPays.java",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			test.setupFiles(t, tempDir)

			samples, err := ExtractSamples(tempDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, samples); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractSamples_Error(t *testing.T) {
	for _, test := range []struct {
		name      string
		emptyDir  bool
		content   string
		wantErr   error
		errString string
	}{
		{
			name:      "error on empty directory",
			emptyDir:  true,
			errString: "dir cannot be empty",
		},
		{
			name: "error on empty title override",
			content: `// sample-metadata:
//   title: ""
public class Invalid {}`,
			wantErr: errEmptyTitle,
		},
		{
			name: "error on capitalized Title",
			content: `// sample-metadata:
//   Title: Capitalized Title
public class Invalid {}`,
			wantErr: errMissingTitle,
		},
		{
			name: "error on missing title line immediately following sample-metadata",
			content: `// sample-metadata:
//   description: missing title line
public class Invalid {}`,
			wantErr: errMissingTitle,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.emptyDir {
				_, err := ExtractSamples("")
				if err == nil || !strings.Contains(err.Error(), test.errString) {
					t.Errorf("ExtractSamples(\"\") err = %v, want substring %q", err, test.errString)
				}
				return
			}
			tempDir := t.TempDir()
			samplesDir := filepath.Join(tempDir, "samples", "src", "main", "java")
			if err := os.MkdirAll(samplesDir, 0755); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(samplesDir, "Sample.java")
			if err := os.WriteFile(file, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := ExtractSamples(tempDir)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("ExtractSamples() err = %v, want %v", err, test.wantErr)
			}
		})
	}
}
