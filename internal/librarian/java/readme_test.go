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
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractSamples(t *testing.T) {
	for _, test := range []struct {
		name       string
		setupFiles func(t *testing.T, dir string)
		want       []codeSample
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
			want: []codeSample{
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

			samples, err := extractSamples(tempDir)
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
		name       string
		setupFiles func(t *testing.T, dir string)
		dir        string
		wantErr    error
	}{
		{
			name:    "error on empty directory",
			dir:     "",
			wantErr: errEmptyDir,
		},
		{
			name: "error on empty title override",
			setupFiles: func(t *testing.T, dir string) {
				samplesDir := filepath.Join(dir, "samples", "src", "main", "java")
				if err := os.MkdirAll(samplesDir, 0755); err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(samplesDir, "Sample.java")
				content := `// sample-metadata:
//   title: ""
public class Invalid {}`
				if err := os.WriteFile(file, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errEmptyTitle,
		},
		{
			name: "error on missing title line immediately following sample-metadata",
			setupFiles: func(t *testing.T, dir string) {
				samplesDir := filepath.Join(dir, "samples", "src", "main", "java")
				if err := os.MkdirAll(samplesDir, 0755); err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(samplesDir, "Sample.java")
				content := `// sample-metadata:
//   description: missing title line
public class Invalid {}`
				if err := os.WriteFile(file, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errMissingTitle,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := test.dir
			if test.setupFiles != nil {
				dir = t.TempDir()
				test.setupFiles(t, dir)
			}
			_, err := extractSamples(dir)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("extractSamples() err = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestCollectSampleFiles(t *testing.T) {
	for _, test := range []struct {
		name       string
		setupFiles func(t *testing.T, dir string)
		want       []string
	}{
		{
			name: "missing samples directory",
			setupFiles: func(t *testing.T, dir string) {
				// Do nothing, temp dir is empty.
			},
			want: nil,
		},
		{
			name: "collects production java files only",
			setupFiles: func(t *testing.T, dir string) {
				samplesDir := filepath.Join(dir, "samples", "src", "main", "java", "com", "example")
				if err := os.MkdirAll(samplesDir, 0755); err != nil {
					t.Fatal(err)
				}
				testDir := filepath.Join(dir, "samples", "src", "test", "java", "com", "example")
				if err := os.MkdirAll(testDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(samplesDir, "SampleA.java"), []byte("public class SampleA {}"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(samplesDir, "README.md"), []byte("# Docs"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(testDir, "SampleATest.java"), []byte("public class SampleATest {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{
				filepath.Join("samples", "src", "main", "java", "com", "example", "SampleA.java"),
			},
		},
		{
			name: "collects production java files across nested packages and ignores directories",
			setupFiles: func(t *testing.T, dir string) {
				pkg1 := filepath.Join(dir, "samples", "src", "main", "java", "com", "example")
				pkg2 := filepath.Join(dir, "samples", "src", "main", "java", "com", "example", "subpkg")
				fakeJavaDir := filepath.Join(dir, "samples", "src", "main", "java", "com", "example", "FakeDir.java")
				if err := os.MkdirAll(pkg1, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(pkg2, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(fakeJavaDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(pkg1, "Alpha.java"), []byte("public class Alpha {}"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(pkg2, "Beta.java"), []byte("package com.example.subpkg; public class Beta {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{
				filepath.Join("samples", "src", "main", "java", "com", "example", "Alpha.java"),
				filepath.Join("samples", "src", "main", "java", "com", "example", "subpkg", "Beta.java"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			test.setupFiles(t, tempDir)

			got, err := collectSampleFiles(tempDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseCodeSample(t *testing.T) {
	for _, test := range []struct {
		name    string
		relPath string
		content string
		want    *codeSample
	}{
		{
			name:    "default title derived from filename",
			relPath: filepath.Join("samples", "src", "main", "java", "DemoSample.java"),
			content: "public class DemoSample {}",
			want: &codeSample{
				Title: "Demo Sample",
				File:  "samples/src/main/java/DemoSample.java",
			},
		},
		{
			name:    "custom title override from metadata",
			relPath: filepath.Join("samples", "src", "main", "java", "RequesterPays.java"),
			content: "// sample-metadata:\n//   title: Custom Title Override\npublic class RequesterPays {}",
			want: &codeSample{
				Title: "Custom Title Override",
				File:  "samples/src/main/java/RequesterPays.java",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			absPath := filepath.Join(tempDir, test.relPath)
			if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(absPath, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := parseCodeSample(tempDir, test.relPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseCodeSample_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		relPath string
		content string
		wantErr error
	}{
		{
			name:    "empty title returns error",
			relPath: filepath.Join("samples", "src", "main", "java", "InvalidSample.java"),
			content: "// sample-metadata:\n//   title: \"\"\npublic class InvalidSample {}",
			wantErr: errEmptyTitle,
		},
		{
			name:    "missing title line returns error",
			relPath: filepath.Join("samples", "src", "main", "java", "MissingTitle.java"),
			content: "// sample-metadata:\n//   description: missing\npublic class MissingTitle {}",
			wantErr: errMissingTitle,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			absPath := filepath.Join(tempDir, test.relPath)
			if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(absPath, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := parseCodeSample(tempDir, test.relPath)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("parseCodeSample() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

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
			if got != test.want {
				t.Errorf("isProductionSample() = %t, want %t", got, test.want)
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

func TestCollectSnippetFiles(t *testing.T) {
	for _, test := range []struct {
		name       string
		setupFiles func(t *testing.T, dir string)
		want       []string
	}{
		{
			name: "missing samples directory returns nil",
			setupFiles: func(t *testing.T, dir string) {
				// Empty directory.
			},
			want: nil,
		},
		{
			name: "collects java and xml files while ignoring excluded directories",
			setupFiles: func(t *testing.T, dir string) {
				validJavaDir := filepath.Join(dir, "samples", "src", "main", "java")
				validXMLDir := filepath.Join(dir, "samples", "src", "main", "resources")
				testDir := filepath.Join(dir, "samples", "src", "test", "java")
				genDir := filepath.Join(dir, "samples", "snippets", "generated")
				for _, d := range []string{validJavaDir, validXMLDir, testDir, genDir} {
					if err := os.MkdirAll(d, 0755); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(validJavaDir, "Sample.java"), []byte("public class Sample {}"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(validXMLDir, "pom.xml"), []byte("<project></project>"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(validJavaDir, "README.md"), []byte("# Ignore"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(testDir, "TestSample.java"), []byte("public class TestSample {}"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(genDir, "GenSnippet.java"), []byte("public class GenSnippet {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{
				filepath.Join("samples", "src", "main", "java", "Sample.java"),
				filepath.Join("samples", "src", "main", "resources", "pom.xml"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			test.setupFiles(t, tempDir)

			got, err := collectSnippetFiles(tempDir)
			if err != nil {
				t.Fatal(err)
			}
			var relGot []string
			for _, p := range got {
				rel, err := filepath.Rel(tempDir, p)
				if err != nil {
					t.Fatal(err)
				}
				relGot = append(relGot, rel)
			}
			if diff := cmp.Diff(test.want, relGot); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractSnippetsFromFile(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		content string
		want    map[string][]string
	}{
		{
			name: "extracts snippets and respects exclude blocks",
			content: `public class Example {
  // [START my_snippet]
  public void run() {
    // [START_EXCLUDE]
    secretInit();
    // [END_EXCLUDE]
    doWork();
  }
  // [END my_snippet]
}`,
			want: map[string][]string{
				"my_snippet": {
					"  public void run() {",
					"    doWork();",
					"  }",
				},
			},
		},
		{
			name: "multiple independent snippets in single file",
			content: `public class Multi {
  // [START first]
  void a() {}
  // [END first]
  // [START second]
  void b() {}
  // [END second]
}`,
			want: map[string][]string{
				"first": {
					"  void a() {}",
				},
				"second": {
					"  void b() {}",
				},
			},
		},
		{
			name: "nested snippets with exclude block",
			content: `public class Nested {
  // [START outer]
  void start() {
    // [START inner]
    doInner();
    // [START_EXCLUDE]
    logDebug();
    // [END_EXCLUDE]
    finishInner();
    // [END inner]
  }
  // [END outer]
}`,
			want: map[string][]string{
				"outer": {
					"  void start() {",
					"    doInner();",
					"    finishInner();",
					"  }",
				},
				"inner": {
					"    doInner();",
					"    finishInner();",
				},
			},
		},
		{
			name:    "no snippets in file",
			content: "public class Simple {}",
			want:    map[string][]string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpPath := filepath.Join(t.TempDir(), "SampleSnippet.java")
			if err := os.WriteFile(tmpPath, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := extractSnippetsFromFile(tmpPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractSnippetsFromFile_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		file    string
		wantErr error
	}{
		{
			name:    "empty file parameter returns error",
			file:    "",
			wantErr: errEmptyFile,
		},
		{
			name:    "non-existent file returns error",
			file:    "non-existent-file.java",
			wantErr: fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := extractSnippetsFromFile(test.file)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("extractSnippetsFromFile(%q) error = %v, wantErr %v", test.file, err, test.wantErr)
			}
		})
	}
}
