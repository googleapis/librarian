// Copyright 2025 Google LLC
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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateCommand(t *testing.T) {
	const (
		lib1       = "library-one"
		lib1Output = "output1"
		lib2       = "library-two"
		lib2Output = "output2"
	)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "testdata", "googleapis")

	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)

	configContent := fmt.Sprintf(`language: testhelper
sources:
  googleapis:
    dir: %s
libraries:
  - name: %s
    output: %s
    apis:
      - path: google/cloud/speech/v1
      - path: google/cloud/speech/v1p1beta1
      - path: google/cloud/speech/v2
      - path: grafeas/v1
  - name: %s
    output: %s
`, googleapisDir, lib1, lib1Output, lib2, lib2Output)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	allLibraries := map[string]string{
		lib1: lib1Output,
		lib2: lib2Output,
	}

	for _, test := range []struct {
		name    string
		args    []string
		wantErr error
		want    []string
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "generate"},
			wantErr: errMissingLibraryOrAllFlag,
		},
		{
			name:    "both library and all flag",
			args:    []string{"librarian", "generate", "--all", lib1},
			wantErr: errBothLibraryAndAllFlag,
		},
		{
			name: "library name",
			args: []string{"librarian", "generate", lib1},
			want: []string{lib1},
		},
		{
			name: "all flag",
			args: []string{"librarian", "generate", "--all"},
			want: []string{lib1, lib2},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Run(t.Context(), test.args...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			generated := make(map[string]bool)
			for _, libName := range test.want {
				generated[libName] = true
			}
			for libName, outputDir := range allLibraries {
				readmePath := filepath.Join(tempDir, outputDir, "README.md")
				shouldExist := generated[libName]
				_, err = os.Stat(readmePath)
				if !shouldExist {
					if err == nil {
						t.Fatalf("expected file for %q to not be generated, but it exists", libName)
					}
					if !os.IsNotExist(err) {
						t.Fatalf("expected file for %q to not be generated, but got unexpected error: %v", libName, err)
					}
					return
				}
				if err != nil {
					t.Fatalf("expected file to be generated for %q, but got error: %v", libName, err)
				}

				got, err := os.ReadFile(readmePath)
				if err != nil {
					t.Fatalf("could not read generated file for %q: %v", libName, err)
				}
				want := fmt.Sprintf("# %s\n\nGenerated library\n", libName)
				if diff := cmp.Diff(want, string(got)); diff != "" {
					t.Errorf("mismatch for %q (-want +got):\n%s", libName, diff)
				}
			}
		})
	}
}

func TestGenerateSkip(t *testing.T) {
	const (
		lib1       = "library-one"
		lib1Output = "output1"
		lib2       = "library-two"
		lib2Output = "output2"
	)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "testdata", "googleapis")

	allLibraries := map[string]string{
		lib1: lib1Output,
		lib2: lib2Output,
	}

	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "skip_generate with all flag",
			args: []string{"librarian", "generate", "--all"},
			want: []string{lib2},
		},
		{
			name: "skip_generate with library name",
			args: []string{"librarian", "generate", lib1},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)
			configContent := fmt.Sprintf(`language: testhelper
sources:
  googleapis:
    dir: %s
libraries:
  - name: %s
    output: %s
    skip_generate: true
  - name: %s
    output: %s
`, googleapisDir, lib1, lib1Output, lib2, lib2Output)
			if err := os.WriteFile(filepath.Join(tempDir, librarianConfigPath), []byte(configContent), 0644); err != nil {
				t.Fatal(err)
			}
			if err := Run(t.Context(), test.args...); err != nil {
				t.Fatal(err)
			}
			generated := make(map[string]bool)
			for _, libName := range test.want {
				generated[libName] = true
			}
			for libName, outputDir := range allLibraries {
				readmePath := filepath.Join(tempDir, outputDir, "README.md")
				shouldExist := generated[libName]
				_, err := os.Stat(readmePath)
				if shouldExist && err != nil {
					t.Errorf("expected %q to be generated, but got error: %v", libName, err)
				}
				if !shouldExist {
					if err == nil {
						t.Errorf("expected %q to not be generated, but it exists", libName)
					} else if !os.IsNotExist(err) {
						t.Errorf("expected %q to not be generated, but got unexpected error: %v", libName, err)
					}
				}
			}
		})
	}
}

func TestPrepareLibrary(t *testing.T) {
	for _, test := range []struct {
		name     string
		language string
		output   string
		want     string
	}{
		{
			name:     "empty output derives path from channel",
			language: "rust",
			want:     "src/generated/cloud/secretmanager/v1",
		},
		{
			name:     "explicit output keeps explicit path",
			language: "rust",
			output:   "custom/output",
			want:     "custom/output",
		},
		{
			name:     "empty output uses default for non-rust",
			language: "go",
			want:     "src/generated",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			lib := &config.Library{
				Name:   "test-lib",
				Output: test.output,
				Channels: []*config.Channel{
					{Path: "google/cloud/secretmanager/v1"},
				},
			}
			defaults := &config.Default{
				Output: "src/generated",
			}
			got := prepareLibrary(test.language, lib, defaults)
			if got.Output != test.want {
				t.Errorf("got output %q, want %q", got.Output, test.want)
			}
		})
	}
}

func TestDeriveDefaultRustOutput(t *testing.T) {
	for _, test := range []struct {
		name    string
		channel string
		want    string
	}{
		{
			name:    "standard google cloud path",
			channel: "google/cloud/secretmanager/v1",
			want:    "src/generated/cloud/secretmanager/v1",
		},
		{
			name:    "nested api path",
			channel: "google/cloud/aiplatform/v1beta1",
			want:    "src/generated/cloud/aiplatform/v1beta1",
		},
		{
			name:    "non-cloud path",
			channel: "google/iam/v1",
			want:    "src/generated/iam/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveDefaultRustOutput(test.channel, "src/generated")
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestCleanOutput(t *testing.T) {
	for _, test := range []struct {
		name    string
		files   []string
		keep    []string
		want    []string
		wantErr bool
	}{
		{
			name:  "removes all except keep list",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs"},
			keep:  []string{"Cargo.toml"},
			want:  []string{"Cargo.toml"},
		},
		{
			name:    "empty directory with keep list",
			files:   []string{},
			keep:    []string{"Cargo.toml"},
			wantErr: true,
		},
		{
			name:  "only kept file",
			files: []string{"Cargo.toml"},
			keep:  []string{"Cargo.toml"},
			want:  []string{"Cargo.toml"},
		},
		{
			name:    "keep file not found",
			files:   []string{"README.md", "src/lib.rs"},
			keep:    []string{"Cargo.toml"},
			wantErr: true,
		},
		{
			name:  "keep multiple files",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs"},
			keep:  []string{"Cargo.toml", "README.md"},
			want:  []string{"Cargo.toml", "README.md"},
		},
		{
			name:  "empty keep list",
			files: []string{"Cargo.toml", "README.md"},
			keep:  []string{},
			want:  []string{},
		},
		{
			name:  "keep nested files",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs", "src/operation.rs", "src/endpoint.rs"},
			keep:  []string{"src/operation.rs", "src/endpoint.rs"},
			want:  []string{"src/endpoint.rs", "src/operation.rs"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range test.files {
				path := filepath.Join(dir, f)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			err := cleanOutput(dir, test.keep)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, f := range test.files {
				if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
					got = append(got, f)
				}
			}
			slices.Sort(got)
			slices.Sort(test.want)
			if !slices.Equal(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}
