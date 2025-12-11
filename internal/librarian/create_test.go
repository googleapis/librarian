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
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	libExists       = "library-one"
	libExistsOutput = "output1"
	newLib          = "library-two"
	newLibOutput    = "output2"
	newLibSpec      = "google/cloud/storage/v1"
	newLibSC        = "google/cloud/storage/v1/storage_v1.yaml"
)

type mockGenerator struct {
	called   bool
	callArgs struct {
		all         bool
		libraryName string
	}
}

func (m *mockGenerator) Run(ctx context.Context, all bool, libraryName string) error {
	m.called = true
	m.callArgs.all = all
	m.callArgs.libraryName = libraryName
	return nil
}

func TestCreateCommandForExistingLib(t *testing.T) {

	for _, test := range []struct {
		name     string
		libName  string
		output   string
		language string
	}{
		{
			name:     "run create for existing library",
			libName:  libExists,
			output:   libExistsOutput,
			language: "rust",
		},
	} {
		t.Run(test.name, func(t *testing.T) {

			createLibrarianYaml(t, test.libName, test.output, test.language, "")
			var gen Generator = &mockGenerator{}

			err := runCreateWithGenerator(context.Background(), test.libName, "", "", test.output, "protobuf", gen)
			if err != nil {
				t.Fatal(err)
			}

			mock := gen.(*mockGenerator)
			if !mock.called {
				t.Error("expected mockGenerator.Run to be called")
			}
			if mock.callArgs.libraryName != test.libName {
				t.Errorf("expected libraryName %s, got %s", test.libName, mock.callArgs.libraryName)
			}

		})
	}

}

func TestCreateCommand(t *testing.T) {

	for _, test := range []struct {
		name             string
		args             []string
		language         string
		skipCreatingYaml bool
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
			if !test.skipCreatingYaml {
				createLibrarianYaml(t, libExists, libExistsOutput, test.language, test.defaultOutput)
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
		})
	}
}

func createLibrarianYaml(t *testing.T, libName string, libOutput string, language string, defaultOutput string) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)
	config := config.Config{
		Language: language,
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Dir: "/googleapis/testdata",
			},
		},
		Default: &config.Default{
			Output: defaultOutput,
		},
		Libraries: []*config.Library{
			{
				Name:   libName,
				Output: libOutput,
			},
		},
	}

	configBytes, err := yaml.Marshal(&config)
	if err != nil {
		t.Fatalf("Failed to marshal YAML: %v", err)
	}

	configContent := string(configBytes)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, libOutput), 0755); err != nil {
		t.Fatal(err)
	}
}
