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
	"github.com/googleapis/librarian/internal/librarian/rust"
	"gopkg.in/yaml.v3"
)

const (
	libExists         = "library-one"
	libExistsOutput   = "output1"
	newLib            = "library-two"
	newLibOutput      = "output2"
	newLibSpec        = "google/cloud/storage/v1"
	newLibSC          = "google/cloud/storage/v1/storage_v1.yaml"
	defaultSpecFormat = "protobuf"
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

type mockRustHelper struct {
	prepareCalled   bool
	prepareCallArgs struct {
		outputDir string
	}
	validateCalled   bool
	validateCallArgs struct {
		outputDir string
	}
}

func (m *mockRustHelper) HelperPrepareCargoWorkspace(ctx context.Context, outputDir string) error {
	m.prepareCalled = true
	m.prepareCallArgs.outputDir = outputDir
	return nil
}

func (m *mockRustHelper) HelperFormatAndValidateLibrary(ctx context.Context, outputDir string) error {
	m.validateCalled = true
	m.validateCallArgs.outputDir = outputDir
	return nil
}

func TestCreateLibrary(t *testing.T) {

	for _, test := range []struct {
		name             string
		libName          string
		output           string
		language         string
		wantErr          error
		skipCreatingYaml bool
	}{
		{
			name:     "run create for existing library",
			libName:  libExists,
			output:   libExistsOutput,
			language: "rust",
		},
		{
			name:     "create new library",
			language: "rust",
			libName:  newLib,
			output:   newLibOutput,
		},
		{
			name:             "no yaml",
			skipCreatingYaml: true,
			wantErr:          errNoYaml,
		},
		{
			name:     "unsupported language",
			language: "unsupported-lang",
			wantErr:  errUnsupportedLanguage,
			output:   newLibOutput,
		},
		{
			name:     "output flag required",
			language: "rust",
			wantErr:  errOutputFlagRequired,
		},
	} {
		t.Run(test.name, func(t *testing.T) {

			if !test.skipCreatingYaml {
				createLibrarianYaml(t, libExists, libExistsOutput, test.language, "")
			}
			var gen Generator = &mockGenerator{}
			var rustHelper rust.RustHelper = &mockRustHelper{}

			err := create(context.Background(), test.libName, "", "", test.output, defaultSpecFormat, gen, rustHelper)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
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
		serviceConfig    string
		specSource       string
		specFormat       string
		wantOutput       string
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "create"},
			wantErr: errMissingLibraryName,
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
		})
	}
}

func TestDeriveSpecificationSource(t *testing.T) {
	for _, test := range []struct {
		name               string
		serviceConfig      string
		specSource         string
		expectedSpecSource string
		language           string
	}{
		{
			name:               "rust missing service-config",
			language:           "rust",
			specSource:         newLibSpec,
			expectedSpecSource: newLibSpec,
		},
		{
			name:               "rust missing specification-source",
			language:           "rust",
			serviceConfig:      newLibSC,
			expectedSpecSource: "google/cloud/storage/v1",
		},
		{
			name:               "rust missing specification-source and service-config",
			language:           "rust",
			expectedSpecSource: "",
		},
		{
			name:               "non-rust language",
			language:           "other-lang",
			expectedSpecSource: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveSpecSource(test.specSource, test.serviceConfig, test.language)
			if got != test.expectedSpecSource {
				t.Errorf("want specification source %q, got %q", test.expectedSpecSource, got)
			}
		})
	}
}

func TestDeriveOutput(t *testing.T) {
	for _, test := range []struct {
		name           string
		specSource     string
		output         string
		defaultOutput  string
		expectedOutput string
		libraryName    string
		language       string
		wantErr        error
	}{

		{
			name:           "default rust output directory used with spec source",
			language:       "rust",
			specSource:     newLibSpec,
			defaultOutput:  "default",
			expectedOutput: "default/cloud/storage/v1",
		},
		{
			name:           "default rust output directory used with default package",
			language:       "rust",
			defaultOutput:  "default",
			libraryName:    "google-cloud-storage-v1",
			expectedOutput: "default/cloud/storage/v1",
		},
		{
			name:           "provided output directory used",
			language:       "rust",
			output:         "default",
			expectedOutput: "default",
		},
		{
			name:     "output flag required",
			language: "rust",
			wantErr:  errOutputFlagRequired,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := createLibrarianYaml(t, libExists, libExistsOutput, test.language, test.defaultOutput)
			got, err := deriveOutput(test.output, &cfg, test.libraryName, test.specSource, test.language)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
			if got != test.expectedOutput {
				t.Errorf("want output %q, got %q", test.expectedOutput, got)
			}
		})
	}
}

func TestAddLibraryToLibrarianYaml(t *testing.T) {

	for _, test := range []struct {
		name          string
		output        string
		serviceConfig string
		specSource    string
		specFormat    string
		libraryName   string
		language      string
	}{

		{
			name:        "new library with no specification-source and service-config",
			libraryName: newLib,
			output:      newLibOutput,
			specFormat:  defaultSpecFormat,
			language:    "rust",
		},
		{
			name:          "new library with specification-source and service-config",
			libraryName:   newLib,
			output:        newLibOutput,
			specFormat:    defaultSpecFormat,
			specSource:    newLibSpec,
			serviceConfig: newLibSC,
			language:      "rust",
		},
		{
			name:        "new library with specification-source",
			libraryName: newLib,
			output:      newLibOutput,
			specFormat:  defaultSpecFormat,
			specSource:  newLibSpec,
			language:    "rust",
		},
		{
			name:          "new library with service-config",
			libraryName:   newLib,
			output:        newLibOutput,
			specFormat:    defaultSpecFormat,
			serviceConfig: newLibSC,
			language:      "rust",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := createLibrarianYaml(t, libExists, libExistsOutput, test.language, "")
			if err := addLibraryToLibrarianConfig(&cfg, test.libraryName, test.output, test.specSource, test.serviceConfig, test.specFormat); err != nil {
				t.Fatalf("unexpected error adding library to librarian.yaml: %v", err)
			}
			validateLibrarianYaml(t, newLib, test.output, test.specSource, test.specFormat, test.serviceConfig)
		})
	}
}

func createLibrarianYaml(t *testing.T, libName string, libOutput string, language string, defaultOutput string) config.Config {
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

	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, libOutput), 0755); err != nil {
		t.Fatal(err)
	}
	return config
}

func validateLibrarianYaml(t *testing.T, libName string, libOutput string, specSource string, specFormat string, serviceConfig string) {
	configBytes, err := os.ReadFile(librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.Config
	if err := yaml.Unmarshal(configBytes, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Libraries) != 2 {
		t.Errorf("want number of libraries in librarian.yaml to be 2, got %d", len(cfg.Libraries))
	}
	var newLib *config.Library
	for _, lib := range cfg.Libraries {
		if lib.Name == libName {
			newLib = lib
			break
		}
	}
	if newLib == nil {
		t.Fatalf("library %q not found in config after adding it", libName)
	}

	if newLib.Name != libName {
		t.Errorf("Expected Name %q, got %q", libName, newLib.Name)
	}
	if newLib.CopyrightYear == "" {
		t.Errorf("Expected CopyrightYear, got %q", newLib.CopyrightYear)
	}
	if newLib.Version == "" {
		t.Errorf("Expected Version, got %q", newLib.Version)
	}
	if newLib.SpecificationFormat != specFormat {
		t.Errorf("Expected SpecificationFormat %q, got %q", specFormat, newLib.SpecificationFormat)
	}

	if serviceConfig != "" || specSource != "" {
		if len(newLib.Channels) != 1 {
			t.Errorf("Expected 1 channel, got: %+v", newLib.Channels)
		}
		if newLib.Channels[0].ServiceConfig != serviceConfig {
			t.Errorf("Expected channel with service config %q, got: %+v", serviceConfig, newLib.Channels[0].ServiceConfig)
		}
		if newLib.Channels[0].Path != specSource {
			t.Errorf("Expected channel with specification source %q, got: %+v", specSource, newLib.Channels[0].Path)
		}
	}
	if serviceConfig == "" && specSource == "" && len(newLib.Channels) != 0 {
		t.Errorf("Expected no channels, got: %+v", newLib.Channels)
	}
	if newLib.Output != libOutput {
		t.Errorf("Expected Output %q, got %q", libOutput, newLib.Output)
	}
}
