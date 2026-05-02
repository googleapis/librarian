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

package gcloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFromProtobuf(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: filepath.Join(testdataDir, "googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"copyright-year": "2026",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(model, outDir); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(outDir, "README.md")
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("missing %s: %s", filename, err)
		}
		t.Fatal(err)
	}
}

func TestParallelstoreMock(t *testing.T) {
	// TODO(https://github.com/googleapis/librarian/issues/5769): once we
	// implement the model building, we should remove the hardcoded data and
	// construct it from internal/testdata/googleapis instead.
	method := &api.Method{
		Name: "CreateInstance",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					PathTemplate: (&api.PathTemplate{}).
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("locations").
						WithVariableNamed("location").
						WithLiteral("instances"),
				},
			},
		},
		InputType: &api.Message{
			Fields: []*api.Field{},
		},
	}

	service := &api.Service{
		Name:        "Parallelstore",
		DefaultHost: "parallelstore.googleapis.com",
		Methods:     []*api.Method{method},
	}
	method.Service = service

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{service},
	}

	outDir := t.TempDir()
	if err := Generate(model, outDir); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(outDir, "main.go")
	gotMain, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatal(err)
	}
	wantMain, err := os.ReadFile(filepath.Join("testdata", "parallelstore", "main.go.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(wantMain), string(gotMain)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	readmeFile := filepath.Join(outDir, "README.md")
	gotReadme, err := os.ReadFile(readmeFile)
	if err != nil {
		t.Fatal(err)
	}
	wantReadme, err := os.ReadFile(filepath.Join("testdata", "parallelstore", "README.md.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(wantReadme), string(gotReadme)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
