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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestClean(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	libraryName := "google-cloud-secretmanager"
	version := "v1"
	// Create directories to clean
	dirs := []string{
		filepath.Join(tmpDir, libraryName, "src"),
		filepath.Join(tmpDir, fmt.Sprintf("proto-%s-%s", libraryName, version), "src"),
		filepath.Join(tmpDir, fmt.Sprintf("grpc-%s-%s", libraryName, version), "src"),
		filepath.Join(tmpDir, "samples", "snippets", "generated"),
		filepath.Join(tmpDir, "kept-dir"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Create files
	files := []string{
		filepath.Join(tmpDir, libraryName, "src", "Main.java"),
		filepath.Join(tmpDir, libraryName, "src", "main", "java", "com", "google", "cloud", "secretmanager", "v1", "stub", "Version.java"),
		filepath.Join(tmpDir, libraryName, "src", "main", "java", "com", "google", "cloud", "secretmanager", "v1", "Version.java"),
		filepath.Join(tmpDir, libraryName, "src", "test", "java", "com", "google", "cloud", "secretmanager", "v1", "it", "ITSecretManagerTest.java"),
		filepath.Join(tmpDir, libraryName, "pom.xml"),
		filepath.Join(tmpDir, "kept-file.txt"),
		filepath.Join(tmpDir, "kept-dir", "file.txt"),
	}
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	lib := &config.Library{
		Name:   "secretmanager",
		Output: tmpDir,
		Keep:   []string{"kept-file.txt", "kept-dir"},
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	if err := Clean(lib); err != nil {
		t.Fatal(err)
	}

	// Verify cleaned paths
	cleanedPaths := []string{
		filepath.Join(tmpDir, libraryName, "src", "Main.java"),
		filepath.Join(tmpDir, fmt.Sprintf("proto-%s-%s", libraryName, version), "src"),
		filepath.Join(tmpDir, fmt.Sprintf("grpc-%s-%s", libraryName, version), "src"),
		filepath.Join(tmpDir, "samples", "snippets", "generated"),
		filepath.Join(tmpDir, libraryName, "src", "main", "java", "com", "google", "cloud", "secretmanager", "v1", "Version.java"),
	}
	for _, p := range cleanedPaths {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected path %s to be removed, but it still exists", p)
		}
	}
	// Verify kept paths
	keptPaths := []string{
		filepath.Join(tmpDir, "kept-file.txt"),
		filepath.Join(tmpDir, "kept-dir", "file.txt"),
		filepath.Join(tmpDir, libraryName, "pom.xml"),
		filepath.Join(tmpDir, fmt.Sprintf("proto-%s-%s", libraryName, version)),
		filepath.Join(tmpDir, fmt.Sprintf("grpc-%s-%s", libraryName, version)),
		filepath.Join(tmpDir, libraryName, "src", "main", "java", "com", "google", "cloud", "secretmanager", "v1", "stub", "Version.java"),
		filepath.Join(tmpDir, libraryName, "src", "test", "java", "com", "google", "cloud", "secretmanager", "v1", "it", "ITSecretManagerTest.java"),
	}
	for _, p := range keptPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected path %s to be kept, but it was removed: %v", p, err)
		}
	}
}

func TestIsDirNotEmpty(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("generic error"),
			want: false,
		},
		{
			name: "ENOTEMPTY",
			err:  &os.PathError{Op: "remove", Path: "/tmp", Err: syscall.ENOTEMPTY},
			want: true,
		},
		{
			name: "EEXIST",
			err:  &os.PathError{Op: "remove", Path: "/tmp", Err: syscall.EEXIST},
			want: true,
		},
		{
			name: "EACCES",
			err:  &os.PathError{Op: "remove", Path: "/tmp", Err: syscall.EACCES},
			want: false,
		},
		{
			name: "wrapped ENOTEMPTY",
			err:  fmt.Errorf("failed: %w", &os.PathError{Op: "remove", Path: "/tmp", Err: syscall.ENOTEMPTY}),
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isDirNotEmpty(test.err)
			if got != test.want {
				t.Errorf("isDirNotEmpty(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func TestCleanPatterns(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		library *config.Library
		want    map[string]bool
	}{
		{
			name: "default_case",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: map[string]bool{
				filepath.Join("google-cloud-secretmanager", "src"):          true,
				filepath.Join("proto-google-cloud-secretmanager-v1", "src"): true,
				filepath.Join("grpc-google-cloud-secretmanager-v1", "src"):  true,
				filepath.Join("samples", "snippets", "generated"):           true,
				".repo-metadata.json": true,
			},
		},
		{
			name: "with_overrides",
			library: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{
					DistributionNameOverride: "com.google.cloud:secretmanager-special",
				},
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: map[string]bool{
				filepath.Join("secretmanager-special", "src"):          true,
				filepath.Join("proto-secretmanager-special-v1", "src"): true,
				filepath.Join("grpc-secretmanager-special-v1", "src"):  true,
				filepath.Join("samples", "snippets", "generated"):      true,
				".repo-metadata.json":                                  true,
			},
		},
		{
			name: "multiple_apis",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
					{Path: "google/cloud/secretmanager/v1beta1"},
				},
			},
			want: map[string]bool{
				filepath.Join("google-cloud-secretmanager", "src"):               true,
				filepath.Join("proto-google-cloud-secretmanager-v1", "src"):      true,
				filepath.Join("grpc-google-cloud-secretmanager-v1", "src"):       true,
				filepath.Join("proto-google-cloud-secretmanager-v1beta1", "src"): true,
				filepath.Join("grpc-google-cloud-secretmanager-v1beta1", "src"):  true,
				filepath.Join("samples", "snippets", "generated"):                true,
				".repo-metadata.json": true,
			},
		},
		{
			name: "gapic_artifact_id_override",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path:                    "google/cloud/secretmanager/v1",
							GAPICArtifactIDOverride: "custom-gapic-artifact",
						},
					},
				},
			},
			want: map[string]bool{
				filepath.Join("custom-gapic-artifact", "src"):          true,
				filepath.Join("proto-custom-gapic-artifact-v1", "src"): true,
				filepath.Join("grpc-custom-gapic-artifact-v1", "src"):  true,
				filepath.Join("samples", "snippets", "generated"):      true,
				".repo-metadata.json":                                  true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := cleanPatterns(test.library)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
