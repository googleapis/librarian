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

package cpp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

func TestGenerate_GRPCParity_RequestIdService(t *testing.T) {
	requireProtoc(t)

	srcs := &sources.Sources{
		Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
		ProtobufSrc: filepath.Join("testdata", "protos"),
	}
	sourceConfig := &sources.SourceConfig{
		Sources:     srcs,
		ActiveRoots: []string{"protobuf-src", "googleapis"},
		IncludeList: []string{"test_request_id.proto"},
	}
	modelConfig := &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: "generator/integration_tests",
		ServiceConfig:       "generator/integration_tests/test_request_id.yaml",
		Source:              sourceConfig,
	}

	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := api.Validate(model); err != nil {
		t.Fatal(err)
	}

	outdir := t.TempDir()
	libCfg := &config.CppLibrary{
		ProductPath:                   "generator/integration_tests/golden/v1",
		InitialCopyrightYear:          "2024",
		RetryableStatusCodes:          []string{"kUnavailable"},
		GenAsyncRPCs:                  []string{"CreateFoo"},
		OverrideServiceConfigYAMLName: "generator/integration_tests/test_request_id.yaml",
	}

	if err := Generate(t.Context(), model, outdir, libCfg); err != nil {
		t.Fatal(err)
	}

	goldenDir := filepath.Join("testdata", "golden", "v1")
	prefix := libCfg.ProductPath + "/"
	svc := "RequestIdService"
	expectedFiles := ServiceGeneratedFiles(libCfg.ProductPath, svc)

	for _, gen := range expectedFiles {
		t.Run(gen.OutputPath, func(t *testing.T) {
			fullPath := filepath.Join(outdir, gen.OutputPath)
			gotContent, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("missing generated file %q: %v", gen.OutputPath, err)
			}
			relPath := strings.TrimPrefix(gen.OutputPath, prefix)
			goldenPath := filepath.Join(goldenDir, relPath)
			goldenContent, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("missing golden file %q: %v", goldenPath, err)
			}

			if diff := cmp.Diff(string(goldenContent), string(gotContent)); diff != "" {
				t.Errorf("mismatch (-want golden +got generated):\n%s", diff)
			}
		})
	}
}

func TestGenerate_GRPCParity_DeprecatedService(t *testing.T) {
	requireProtoc(t)

	srcs := &sources.Sources{
		Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
		ProtobufSrc: filepath.Join("testdata", "protos"),
	}
	sourceConfig := &sources.SourceConfig{
		Sources:     srcs,
		ActiveRoots: []string{"protobuf-src", "googleapis"},
		IncludeList: []string{"test_deprecated.proto"},
	}
	modelConfig := &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: "generator/integration_tests",
		ServiceConfig:       "",
		Source:              sourceConfig,
	}

	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := api.Validate(model); err != nil {
		t.Fatal(err)
	}

	outdir := t.TempDir()
	libCfg := &config.CppLibrary{
		ProductPath:          "generator/integration_tests/golden/v1",
		InitialCopyrightYear: "2024",
		RetryableStatusCodes: []string{"kUnavailable"},
	}

	if err := Generate(t.Context(), model, outdir, libCfg); err != nil {
		t.Fatal(err)
	}

	goldenDir := filepath.Join("testdata", "golden", "v1")
	prefix := libCfg.ProductPath + "/"
	svc := "DeprecatedService"
	expectedFiles := ServiceGeneratedFiles(libCfg.ProductPath, svc)

	for _, gen := range expectedFiles {
		if strings.HasSuffix(gen.OutputPath, "sources.cc") {
			// sources.cc includes REST sources for DeprecatedService which is implemented in Phase 4.
			continue
		}
		t.Run(gen.OutputPath, func(t *testing.T) {
			fullPath := filepath.Join(outdir, gen.OutputPath)
			gotContent, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("missing generated file %q: %v", gen.OutputPath, err)
			}
			relPath := strings.TrimPrefix(gen.OutputPath, prefix)
			goldenPath := filepath.Join(goldenDir, relPath)
			goldenContent, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("missing golden file %q: %v", goldenPath, err)
			}

			if diff := cmp.Diff(string(goldenContent), string(gotContent)); diff != "" {
				t.Errorf("mismatch (-want golden +got generated):\n%s", diff)
			}
		})
	}
}
