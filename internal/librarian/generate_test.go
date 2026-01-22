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
)

func TestGenerateCommand(t *testing.T) {
	const (
		lib1       = "library-one"
		lib1Output = "output1"
		lib2       = "library-two"
		lib2Output = "output2"
		lib3       = "library-three"
		lib3Output = "output3"
	)
	baseTempDir := t.TempDir()
	googleapisDir := createGoogleapisServiceConfigs(t, baseTempDir, map[string]string{
		"google/cloud/speech/v1":       "speech_v1.yaml",
		"grafeas/v1":                   "grafeas_v1.yaml",
		"google/cloud/texttospeech/v1": "texttospeech_v1.yaml",
	})

	allLibraries := map[string]string{
		lib1: lib1Output,
		lib2: lib2Output,
		lib3: lib3Output,
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
		{
			name: "skip generate",
			args: []string{"librarian", "generate", lib3},
			want: []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)
			configContent := fmt.Sprintf(`language: fake
sources:
  googleapis:
    dir: %s
libraries:
  - name: %s
    output: %s
    channels:
      - path: google/cloud/speech/v1
      - path: grafeas/v1
  - name: %s
    output: %s
    channels:
      - path: google/cloud/texttospeech/v1
  - name: %s
    output: %s
    skip_generate: true
    channels:
      - path: google/cloud/speech/v1
`, googleapisDir, lib1, lib1Output, lib2, lib2Output, lib3, lib3Output)
			if err := os.WriteFile(filepath.Join(tempDir, librarianConfigPath), []byte(configContent), 0644); err != nil {
				t.Fatal(err)
			}

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
				want := fmt.Sprintf("# %s\n\nGenerated library\n\n---\nFormatted\n", libName)
				if diff := cmp.Diff(want, string(got)); diff != "" {
					t.Errorf("mismatch for %q (-want +got):\n%s", libName, diff)
				}
			}
			if test.name == "all flag" {
				postGeneratePath := filepath.Join(tempDir, "POST_GENERATE_README.md")
				if _, err := os.Stat(postGeneratePath); err != nil {
					t.Errorf("expected POST_GENERATE_README.md to exist, but got error: %v", err)
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
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	googleapisDir := createGoogleapisServiceConfigs(t, tempDir, map[string]string{
		"google/cloud/speech/v1":       "speech_v1.yaml",
		"google/cloud/texttospeech/v1": "texttospeech_v1.yaml",
	})

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
			configContent := fmt.Sprintf(`language: fake
sources:
  googleapis:
    dir: %s
libraries:
  - name: %s
    output: %s
    skip_generate: true
    channels:
      - path: google/cloud/speech/v1
  - name: %s
    output: %s
    channels:
      - path: google/cloud/texttospeech/v1
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

// createGoogleapisServiceConfigs creates a mock googleapis directory structure
// with service config files for testing purposes.
// The configs map keys are channel paths (e.g., "google/cloud/speech/v1")
// and values are the service config filenames (e.g., "speech_v1.yaml").
func createGoogleapisServiceConfigs(t *testing.T, tempDir string, configs map[string]string) string {
	t.Helper()
	googleapisDir := filepath.Join(tempDir, "googleapis")

	for channelPath, filename := range configs {
		dir := filepath.Join(googleapisDir, channelPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return googleapisDir
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
		{
			// While it would definitely be odd to use "./" here, the
			// most common case for canonicalizing is for Windows where
			// the directory separator is a backslash. This test ensures
			// the logic is tested even on Unix.
			name:  "keep entries are canonicalized",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs"},
			keep:  []string{"./Cargo.toml"},
			want:  []string{"Cargo.toml"},
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
