// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateCommand(t *testing.T) {
	const (
		lib1       = "library-one"
		lib1Output = "output1"
		lib2       = "library-two"
		lib2Output = "output2"
	)
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := fmt.Sprintf(`language: testhelper
libraries:
  - name: %s
    output: %s
  - name: %s
    output: %s
`, lib1, lib1Output, lib2, lib2Output)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	allLibraries := map[string]string{
		lib1: lib1Output,
		lib2: lib2Output,
	}

	for _, test := range []struct {
		name          string
		args          []string
		wantErr       error
		wantGenerated []string
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
			name:          "library name",
			args:          []string{"librarian", "generate", lib1},
			wantGenerated: []string{lib1},
		},
		{
			name:          "all flag",
			args:          []string{"librarian", "generate", "--all"},
			wantGenerated: []string{lib1, lib2},
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
			for _, libName := range test.wantGenerated {
				generated[libName] = true
			}
			for libName, outputDir := range allLibraries {
				readmePath := filepath.Join(tempDir, outputDir, "README.md")
				shouldExist := generated[libName]
				content, err := os.ReadFile(readmePath)
				if err != nil && shouldExist {
					t.Fatal(err)
				}
				if err == nil && !shouldExist {
					t.Errorf("expected %s to NOT be generated but file exists", libName)
				}
				if shouldExist {
					want := fmt.Sprintf("# %s\n\nGenerated library\n", libName)
					if diff := cmp.Diff(want, string(content)); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}
