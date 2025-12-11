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
	"testing"
)

const (
	libExists       = "library-one"
	libExistsOutput = "output1"
	newLib          = "library-two"
	newLibOutput    = "output2"
	newLibSpec      = "google/cloud/storage/v1"
	newLibSC        = "google/cloud/storage/v1/storage_v1.yaml"
)

func TestCreateCommand(t *testing.T) {

	for _, test := range []struct {
		name             string
		args             []string
		language         string
		skipCreatingYaml bool
		wantLibs         []string
		wantErr          error
		defaultOutput    string
		libOutputFolder  string
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "create"},
			wantErr: errMissingNameFlag,
		},
		{
			name:            "run create for existing library",
			args:            []string{"librarian", "create", "--name", libExists, "--output", libExistsOutput},
			wantLibs:        []string{libExists},
			language:        "testhelper",
			libOutputFolder: libExistsOutput,
		},
		{
			name:     "missing service-config",
			args:     []string{"librarian", "create", "--name", newLib, "--output", newLibOutput, "--specification-source", newLibSpec},
			language: "rust",
			wantErr:  errServiceConfigAndSpecRequired,
		},
		{
			name:     "missing specification-source",
			args:     []string{"librarian", "create", "--name", newLib, "--output", newLibOutput, "--service-config", newLibSC},
			language: "rust",
			wantErr:  errServiceConfigAndSpecRequired,
		},
		{
			name:     "create new library",
			args:     []string{"librarian", "create", "--name", newLib, "--output", newLibOutput, "--service-config", newLibSC, "--specification-source", newLibSpec},
			language: "rust",
		},
		{
			name:             "no yaml",
			args:             []string{"librarian", "create", "--name", newLib},
			skipCreatingYaml: true,
			wantErr:          errNoYaml,
		},
		{
			name:     "unsupported language",
			args:     []string{"librarian", "create", "--name", newLib},
			language: "unsupported-lang",
			wantErr:  errUnsupportedLanguage,
		},
		{
			name:     "output flag required",
			args:     []string{"librarian", "create", "--name", newLib, "--service-config", newLibSC, "--specification-source", newLibSpec},
			language: "rust",
			wantErr:  errOutputFlagRequired,
		},
		{
			name:          "default output directory used",
			args:          []string{"librarian", "create", "--name", newLib, "--service-config", newLibSC, "--specification-source", newLibSpec},
			language:      "rust",
			defaultOutput: "default",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			googleapisDir := filepath.Join(wd, "testdata", "googleapis")
			tempDir := t.TempDir()
			if !test.skipCreatingYaml {
				// Create a temporary librarian.yaml for the test
				t.Chdir(tempDir)
				configPath := filepath.Join(tempDir, librarianConfigPath)
				configContent := fmt.Sprintf(`language: %s
sources:
  googleapis:
    dir: %s
default:
  output: %s
libraries:
  - name: %s
    output: %s
    apis:
      - path: google/cloud/speech/v1
      - path: google/cloud/speech/v1p1beta1
      - path: google/cloud/speech/v2
      - path: grafeas/v1
`, test.language, googleapisDir, test.defaultOutput, libExists, libExistsOutput)
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatal(err)
				}

				if test.libOutputFolder != "" {
					if err := os.MkdirAll(filepath.Join(tempDir, test.libOutputFolder), 0755); err != nil {
						t.Fatal(err)
					}
				}
			}
			err = Run(t.Context(), test.args...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			allLibraries := map[string]string{
				libExists: libExistsOutput,
				newLib:    newLibOutput,
			}
			validateGeneration(t, test.wantLibs, allLibraries, tempDir)
		})
	}
}
