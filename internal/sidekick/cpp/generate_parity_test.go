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
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestGenerate_ParityAll188Files(t *testing.T) {
	requireProtoc(t)

	cfgPath := filepath.Join("testdata", "golden_librarian.yaml")
	cfg, err := yaml.Read[config.Config](cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	libraryProtos := map[string][]string{
		"test_request_id":     {"test_request_id.proto"},
		"test_deprecated":     {"test_deprecated.proto"},
		"test2_rest_only":     {"test2.proto"},
		"test_admin_database": {"test.proto", "backup.proto"},
	}

	totalCheckedFiles := 0
	goldenDir := filepath.Join("testdata", "golden")

	for _, lib := range cfg.Libraries {
		t.Run(lib.Name, func(t *testing.T) {
			includeList, ok := libraryProtos[lib.Name]
			if !ok {
				t.Fatalf("no protos configured for library %q", lib.Name)
			}

			srcs := &sources.Sources{
				Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
				ProtobufSrc: filepath.Join("testdata", "protos"),
			}
			sourceConfig := &sources.SourceConfig{
				Sources:     srcs,
				ActiveRoots: []string{"protobuf-src", "googleapis"},
				IncludeList: includeList,
			}
			modelConfig := &parser.ModelConfig{
				Language:            config.LanguageCpp,
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "generator/integration_tests",
				ServiceConfig:       lib.Cpp.OverrideServiceConfigYAMLName,
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
			if err := Generate(t.Context(), model, outdir, lib.Cpp); err != nil {
				t.Fatal(err)
			}

			generateGrpc := true
			if lib.Cpp.GenerateGrpcTransport != nil {
				generateGrpc = *lib.Cpp.GenerateGrpcTransport
			}
			generateRest := lib.Cpp.GenerateRestTransport
			generateRoundRobin := lib.Cpp.GenerateRoundRobinDecorator

			var expectedFiles []string
			for _, svc := range model.Services {
				if slices.Contains(lib.Cpp.OmittedServices, svc.Name) {
					continue
				}
				for _, f := range CommonGeneratedFiles(lib.Cpp.ProductPath, svc.Name) {
					expectedFiles = append(expectedFiles, f.OutputPath)
				}
				if generateGrpc {
					for _, f := range GrpcGeneratedFiles(lib.Cpp.ProductPath, svc.Name) {
						expectedFiles = append(expectedFiles, f.OutputPath)
					}
				}
				if generateRest {
					for _, f := range RestGeneratedFiles(lib.Cpp.ProductPath, svc.Name) {
						expectedFiles = append(expectedFiles, f.OutputPath)
					}
				}
				if generateRoundRobin {
					for _, f := range RoundRobinGeneratedFiles(lib.Cpp.ProductPath, svc.Name) {
						expectedFiles = append(expectedFiles, f.OutputPath)
					}
				}
				if lib.Cpp.ForwardingProductPath != "" {
					for _, f := range ForwardingGeneratedFiles(lib.Cpp.ForwardingProductPath, svc.Name) {
						expectedFiles = append(expectedFiles, f.OutputPath)
					}
				}
			}

			for _, outPath := range expectedFiles {
				totalCheckedFiles++
				t.Run(outPath, func(t *testing.T) {
					fullPath := filepath.Join(outdir, outPath)
					gotContent, err := os.ReadFile(fullPath)
					if err != nil {
						t.Fatalf("missing generated file %q: %v", outPath, err)
					}
					if len(gotContent) == 0 {
						t.Errorf("generated file %s is empty", outPath)
					}

					relGoldenPath, ok := strings.CutPrefix(outPath, "generator/integration_tests/golden/")
					if !ok {
						t.Fatalf("unexpected output path prefix: %q", outPath)
					}
					goldenPath := filepath.Join(goldenDir, relGoldenPath)
					goldenContent, err := os.ReadFile(goldenPath)
					if err != nil {
						t.Fatalf("missing golden file %q: %v", goldenPath, err)
					}
					if len(goldenContent) == 0 {
						t.Errorf("golden file %s is empty", goldenPath)
					}

					if diff := cmp.Diff(string(goldenContent), string(gotContent)); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				})
			}
		})
	}

	const wantTotal = 188
	if totalCheckedFiles != wantTotal {
		t.Errorf("unexpected total checked files count: want %d, got %d", wantTotal, totalCheckedFiles)
	}
}
