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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestGoldenReferenceFiles(t *testing.T) {
	goldenDir := filepath.Join("testdata", "golden")
	var fileList []string
	err := filepath.WalkDir(goldenDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	const wantFileCount = 188
	if len(fileList) != wantFileCount {
		t.Fatalf("unexpected golden file count: want %d, got %d", wantFileCount, len(fileList))
	}

	for _, f := range fileList {
		info, err := os.Stat(f)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() == 0 {
			t.Fatalf("golden file %s is empty", f)
		}
	}

	keyFiles := []string{
		filepath.Join(goldenDir, "golden_kitchen_sink_client.h"),
		filepath.Join(goldenDir, "golden_kitchen_sink_connection.h"),
		filepath.Join(goldenDir, "golden_thing_admin_client.h"),
		filepath.Join(goldenDir, "golden_thing_admin_connection.h"),
		filepath.Join(goldenDir, "v1", "golden_kitchen_sink_client.h"),
		filepath.Join(goldenDir, "v1", "golden_kitchen_sink_client.cc"),
		filepath.Join(goldenDir, "v1", "golden_kitchen_sink_connection.h"),
		filepath.Join(goldenDir, "v1", "golden_kitchen_sink_connection.cc"),
		filepath.Join(goldenDir, "v1", "internal", "golden_kitchen_sink_stub.h"),
		filepath.Join(goldenDir, "v1", "internal", "golden_kitchen_sink_stub.cc"),
		filepath.Join(goldenDir, "v1", "mocks", "mock_golden_kitchen_sink_connection.h"),
		filepath.Join(goldenDir, "v1", "request_id_client.h"),
		filepath.Join(goldenDir, "v1", "mocks", "mock_request_id_connection.h"),
	}
	for _, kf := range keyFiles {
		if _, err := os.Stat(kf); err != nil {
			t.Fatal(err)
		}
	}
}

func TestParseTestModels(t *testing.T) {
	requireProtoc(t)

	for _, test := range []struct {
		name          string
		includeList   []string
		serviceConfig string
		wantServices  []string
	}{
		{
			name:          "test_admin_database",
			includeList:   []string{"test.proto", "backup.proto"},
			serviceConfig: "generator/integration_tests/test.yaml",
			wantServices:  []string{"GoldenThingAdmin", "GoldenKitchenSink"},
		},
		{
			name:          "test2_rest_only",
			includeList:   []string{"test2.proto"},
			serviceConfig: "",
			wantServices:  []string{"GoldenRestOnly"},
		},
		{
			name:          "test_request_id",
			includeList:   []string{"test_request_id.proto"},
			serviceConfig: "generator/integration_tests/test_request_id.yaml",
			wantServices:  []string{"RequestIdService"},
		},
		{
			name:          "test_deprecated",
			includeList:   []string{"test_deprecated.proto"},
			serviceConfig: "",
			wantServices:  []string{"DeprecatedService"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			srcs := &sources.Sources{
				Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
				ProtobufSrc: filepath.Join("testdata", "protos"),
			}
			sourceConfig := &sources.SourceConfig{
				Sources:     srcs,
				ActiveRoots: []string{"protobuf-src", "googleapis"},
				IncludeList: test.includeList,
			}

			modelConfig := &parser.ModelConfig{
				Language:            config.LanguageCpp,
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "generator/integration_tests",
				ServiceConfig:       test.serviceConfig,
				Source:              sourceConfig,
			}

			model, err := parser.CreateModel(modelConfig)
			if err != nil {
				t.Fatal(err)
			}
			if model == nil {
				t.Fatal("expected non-nil model")
			}

			if err := api.Validate(model); err != nil {
				t.Fatal(err)
			}

			for _, wantService := range test.wantServices {
				if !slices.ContainsFunc(model.Services, func(s *api.Service) bool { return s.Name == wantService }) {
					t.Fatalf("service %q not found in model %s", wantService, test.name)
				}
			}

			if test.name == "test_admin_database" {
				idx := slices.IndexFunc(model.Services, func(s *api.Service) bool {
					return s.Name == "GoldenKitchenSink"
				})
				if idx == -1 {
					t.Fatal("service GoldenKitchenSink not found in test_admin_database model")
				}
				goldenKitchenSink := model.Services[idx]

				loc, ok := model.DefinitionLocation(goldenKitchenSink.ID)
				if !ok {
					t.Fatalf("missing definition location for %q (%s)", goldenKitchenSink.Name, goldenKitchenSink.ID)
				}
				if loc.Line <= 0 {
					t.Fatalf("expected line > 0 for %q, got %d", goldenKitchenSink.Name, loc.Line)
				}
				const wantFilename = "generator/integration_tests/test.proto"
				if loc.Filename != wantFilename {
					t.Fatalf("expected filename %q for %q, got %q", wantFilename, goldenKitchenSink.Name, loc.Filename)
				}
			}
		})
	}
}

func TestGoldenLibrarianConfig(t *testing.T) {
	cfgPath := filepath.Join("testdata", "golden_librarian.yaml")
	cfg, err := yaml.Read[config.Config](cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	want := &config.Config{
		Language: config.LanguageCpp,
		Libraries: []*config.Library{
			{
				Name: "test_admin_database",
				APIs: []*config.API{
					{Path: "generator/integration_tests"},
				},
				Cpp: &config.CppLibrary{
					AdditionalProtoFiles:          []string{"generator/integration_tests/backup.proto"},
					OverrideServiceConfigYAMLName: "generator/integration_tests/test.yaml",
					InitialCopyrightYear:          "2022",
					ProductPath:                   "generator/integration_tests/golden/v1",
					ForwardingProductPath:         "generator/integration_tests/golden",
					ServiceEndpointEnvVar:         "GOLDEN_KITCHEN_SINK_ENDPOINT",
					EmulatorEndpointEnvVar:        "GOLDEN_KITCHEN_SINK_EMULATOR_HOST",
					GenerateRestTransport:         true,
					GenerateRoundRobinDecorator:   true,
					OmitRepoMetadata:              true,
					OmittedRPCs: []string{
						"Omitted1",
						"GoldenKitchenSink.Omitted2",
						"Deprecated1",
						"Deprecated2(std::string const&)",
					},
					GenAsyncRPCs: []string{
						"GetDatabase",
						"DropDatabase",
						"StreamingRead",
						"StreamingWrite",
					},
					RetryableStatusCodes: []string{
						"GoldenKitchenSink.kInternal",
						"kUnavailable",
						"GoldenThingAdmin.kDeadlineExceeded",
					},
					IdempotencyOverrides: []config.IdempotencyRule{
						{RPCName: "GoldenThingAdmin.DropDatabase", Idempotency: "IDEMPOTENT"},
						{RPCName: "GoldenKitchenSink.ListLogs", Idempotency: "NON_IDEMPOTENT"},
					},
				},
			},
			{
				Name: "test2_rest_only",
				APIs: []*config.API{
					{Path: "generator/integration_tests"},
				},
				Cpp: &config.CppLibrary{
					InitialCopyrightYear:  "2023",
					ProductPath:           "generator/integration_tests/golden/v1",
					GenerateRestTransport: true,
					GenerateGrpcTransport: new(false),
					RetryableStatusCodes: []string{
						"kUnavailable",
					},
					EndpointLocationStyle: "LOCATION_OPTIONALLY_DEPENDENT",
				},
			},
			{
				Name: "test_request_id",
				APIs: []*config.API{
					{Path: "generator/integration_tests"},
				},
				Cpp: &config.CppLibrary{
					InitialCopyrightYear:          "2024",
					ProductPath:                   "generator/integration_tests/golden/v1",
					RetryableStatusCodes:          []string{"kUnavailable"},
					GenAsyncRPCs:                  []string{"CreateFoo"},
					OverrideServiceConfigYAMLName: "generator/integration_tests/test_request_id.yaml",
				},
			},
			{
				Name: "test_deprecated",
				APIs: []*config.API{
					{Path: "generator/integration_tests"},
				},
				Cpp: &config.CppLibrary{
					InitialCopyrightYear:  "2024",
					ProductPath:           "generator/integration_tests/golden/v1",
					GenerateRestTransport: true,
					GenerateGrpcTransport: new(true),
					RetryableStatusCodes:  []string{"kUnavailable"},
				},
			},
		},
	}

	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify service_proto_path is present for all libraries in the YAML.
	type rawLibrary struct {
		Name string `yaml:"name"`
		Cpp  struct {
			ServiceProtoPath string `yaml:"service_proto_path"`
		} `yaml:"cpp"`
	}
	type rawConfig struct {
		Libraries []rawLibrary `yaml:"libraries"`
	}
	rawCfg, err := yaml.Read[rawConfig](cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	wantProtoPaths := map[string]string{
		"test_admin_database": "generator/integration_tests/test.proto",
		"test2_rest_only":     "generator/integration_tests/test2.proto",
		"test_request_id":     "generator/integration_tests/test_request_id.proto",
		"test_deprecated":     "generator/integration_tests/test_deprecated.proto",
	}
	for _, lib := range rawCfg.Libraries {
		wantPath, ok := wantProtoPaths[lib.Name]
		if !ok {
			t.Fatalf("unexpected library %q in raw config", lib.Name)
		}
		if lib.Cpp.ServiceProtoPath != wantPath {
			t.Errorf("library %s service_proto_path mismatch: want %q, got %q", lib.Name, wantPath, lib.Cpp.ServiceProtoPath)
		}
	}
}

func requireProtoc(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
}
