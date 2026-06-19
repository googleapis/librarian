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
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/repometadata"
)

func TestGenerateReadme(t *testing.T) {
	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	library := &config.Library{
		Name:   "google-cloud-secretmanager",
		APIs:   []*config.API{{Path: "google/cloud/secretmanager/v1"}},
		Nodejs: &config.NodejsPackage{PackageName: "@google-cloud/secret-manager"},
	}
	for _, test := range []struct {
		name           string
		setup          func(dir string)
		wantReadmePath string
	}{
		{
			name:           "secret manager, no partials",
			wantReadmePath: filepath.Join("testdata", "generate_readme", "without_partials", "google-cloud-secretmanager", "README.md"),
		},
		{
			name: "secret manager, with partials",
			setup: func(dir string) {
				partialsFile := filepath.Join("testdata", "generate_readme", "with_partials", "google-cloud-secretmanager", partials)
				if err := filesystem.CopyFile(partialsFile, filepath.Join(dir, partials)); err != nil {
					t.Fatal(err)
				}
			},
			wantReadmePath: filepath.Join("testdata", "generate_readme", "with_partials", "google-cloud-secretmanager", "README.md"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "packages", library.Name)
			sampleDir := filepath.Join(output, "samples", "generated", "v1")
			if err := os.MkdirAll(sampleDir, 0755); err != nil {
				t.Fatal(err)
			}
			for _, sample := range []string{
				"secret_manager_service.access_secret_version.js",
				"secret_manager_service.add_secret_version.js",
				"secret_manager_service.create_secret.js",
				"secret_manager_service.delete_secret.js",
			} {
				if err := os.WriteFile(filepath.Join(sampleDir, sample), []byte("example"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if test.setup != nil {
				test.setup(output)
			}
			if err := generateReadmeNew(cfg, library, absGoogleapisDir, output); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(output, "README.md"))
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(test.wantReadmePath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateReadme_Error(t *testing.T) {
	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	for _, test := range []struct {
		name          string
		library       *config.Library
		googleapisDir string
		output        func(t *testing.T) string
		wantErr       error
	}{
		{
			name:          "library has no API",
			library:       &config.Library{Name: "google-cloud-secretmanager"},
			googleapisDir: absGoogleapisDir,
			output:        func(t *testing.T) string { return t.TempDir() },
			wantErr:       repometadata.ErrNoAPIs,
		},
		{
			name: "output is not a directory",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			googleapisDir: absGoogleapisDir,
			output: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "README.md")
				if err := os.WriteFile(filePath, []byte("existing file"), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			wantErr: syscall.ENOTDIR,
		},
		{
			name: "permission denied creating readme",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			googleapisDir: absGoogleapisDir,
			output: func(t *testing.T) string {
				tempDir := t.TempDir()
				if err := os.Chmod(tempDir, 0555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					_ = os.Chmod(tempDir, 0755)
				})
				return tempDir
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputDir := test.output(t)
			err := generateReadmeNew(cfg, test.library, test.googleapisDir, outputDir)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

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
					Name:     "nested sample",
					FilePath: "https://github.com/googleapis/google-cloud-node/blob/main/packages/my-package/samples/generated/sub/v1.nested_sample.js",
				},
				{
					Name:     "do something",
					FilePath: "https://github.com/googleapis/google-cloud-node/blob/main/packages/my-package/samples/generated/v2.do_something.js",
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
	if !errors.Is(err, errFindSampleMetadata) {
		t.Errorf("findSampleMetadata() error = %v, wantErr %v", err, errFindSampleMetadata)
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
