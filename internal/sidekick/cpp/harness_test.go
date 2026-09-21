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
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
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
		wantServices  []struct {
			name     string
			filename string
		}
	}{
		{
			name:          "test_admin_database",
			includeList:   []string{"test.proto", "backup.proto"},
			serviceConfig: "generator/integration_tests/test.yaml",
			wantServices: []struct {
				name     string
				filename string
			}{
				{"GoldenThingAdmin", "generator/integration_tests/test.proto"},
				{"GoldenKitchenSink", "generator/integration_tests/test.proto"},
			},
		},
		{
			name:          "test2_rest_only",
			includeList:   []string{"test2.proto"},
			serviceConfig: "",
			wantServices: []struct {
				name     string
				filename string
			}{
				{"GoldenRestOnly", "generator/integration_tests/test2.proto"},
			},
		},
		{
			name:          "test_request_id",
			includeList:   []string{"test_request_id.proto"},
			serviceConfig: "generator/integration_tests/test_request_id.yaml",
			wantServices: []struct {
				name     string
				filename string
			}{
				{"RequestIdService", "generator/integration_tests/test_request_id.proto"},
			},
		},
		{
			name:          "test_deprecated",
			includeList:   []string{"test_deprecated.proto"},
			serviceConfig: "",
			wantServices: []struct {
				name     string
				filename string
			}{
				{"DeprecatedService", "generator/integration_tests/test_deprecated.proto"},
			},
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

			for _, want := range test.wantServices {
				idx := slices.IndexFunc(model.Services, func(s *api.Service) bool {
					return s.Name == want.name
				})
				if idx == -1 {
					t.Fatalf("service %q not found in model %s", want.name, test.name)
				}
				svc := model.Services[idx]

				loc, ok := model.DefinitionLocation(svc.ID)
				if !ok {
					t.Fatalf("missing definition location for %q (%s)", svc.Name, svc.ID)
				}
				if loc.Line <= 0 {
					t.Fatalf("expected line > 0 for %q, got %d", svc.Name, loc.Line)
				}
				if loc.Filename != want.filename {
					t.Fatalf("expected filename %q for %q, got %q", want.filename, svc.Name, loc.Filename)
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
}

func requireProtoc(t *testing.T) {
	t.Helper()
	testhelper.RequireCommand(t, "protoc")
}
