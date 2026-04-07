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

package swift

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

func TestFromProtobuf(t *testing.T) {
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       "google/type/type.yaml",
		SpecificationSource: "google/type",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: filepath.Join(testdataDir, "googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"copyright-year":      "2038",
			"not-for-publication": "true",
			"version":             "0.1.0",
			"skip-format":         "true",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(outDir, "README.md")
	stat, err := os.Stat(filename)
	if os.IsNotExist(err) {
		t.Errorf("missing %s: %s", filename, err)
	}
	if stat.Mode().Perm()|0666 != 0666 {
		t.Errorf("generated files should just be read-write %s: %o", filename, stat.Mode())
	}
}

func TestGenerateMessageFiles(t *testing.T) {
	outDir := t.TempDir()

	secret := &api.Message{Name: "Secret", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Secret"}
	volume := &api.Message{Name: "Volume", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Volume"}

	model := api.NewTestAPI([]*api.Message{secret, volume}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	for _, expected := range []string{"Secret.swift", "Volume.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
	}
}

func TestGenerateServiceFiles(t *testing.T) {
	outDir := t.TempDir()

	iam := &api.Service{Name: "IAM"}
	secretManager := &api.Service{Name: "SecretManagerService"}

	model := api.NewTestAPI(nil, nil, []*api.Service{iam, secretManager})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	for _, expected := range []string{"IAM.swift", "SecretManagerService.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
	}
}
