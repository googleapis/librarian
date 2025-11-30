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

package bazel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	got := mustParse(t, `
go_grpc_library(
    name = "asset_go_proto",
    importpath = "cloud.google.com/go/asset/apiv1/assetpb",
    protos = [":asset_proto"],
)

go_gapic_library(
    name = "asset_go_gapic",
    srcs = [":asset_proto_with_info"],
    grpc_service_config = "cloudasset_grpc_service_config.json",
    importpath = "cloud.google.com/go/asset/apiv1;asset",
    metadata = True,
    release_level = "ga",
    rest_numeric_enums = True,
    service_yaml = "cloudasset_v1.yaml",
    transport = "grpc+rest",
    diregapic = False,
)
`)

	want := &Config{
		HasGAPIC:          true,
		GAPICImportPath:   "cloud.google.com/go/asset/apiv1;asset",
		ServiceYAML:       "cloudasset_v1.yaml",
		GRPCServiceConfig: "cloudasset_grpc_service_config.json",
		Transport:         "grpc+rest",
		ReleaseLevel:      "ga",
		Metadata:          true,
		DIREGAPIC:         false,
		RESTNumericEnums:  true,
		HasGoGRPC:         true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_Misconfiguration(t *testing.T) {
	content := `
go_grpc_library()

go_proto_library()
`
	tmpDir := t.TempDir()
	buildPath := filepath.Join(tmpDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := Parse(buildPath); err == nil {
		t.Error("Parse() succeeded; want error")
	}
}

func TestParse_ServiceConfigIsTarget(t *testing.T) {
	got := mustParse(t, `
go_grpc_library(
    name = "asset_go_proto",
    importpath = "cloud.google.com/go/asset/apiv1/assetpb",
    protos = [":asset_proto"],
)

go_gapic_library(
    name = "asset_go_gapic",
    srcs = [":asset_proto_with_info"],
    grpc_service_config = "cloudasset_grpc_service_config.json",
    importpath = "cloud.google.com/go/asset/apiv1;asset",
    metadata = True,
    release_level = "ga",
    rest_numeric_enums = True,
    service_yaml = ":cloudasset_v1.yaml",
    transport = "grpc+rest",
    diregapic = False,
)
`)

	want := &Config{
		HasGAPIC:          true,
		GAPICImportPath:   "cloud.google.com/go/asset/apiv1;asset",
		ServiceYAML:       "cloudasset_v1.yaml",
		GRPCServiceConfig: "cloudasset_grpc_service_config.json",
		Transport:         "grpc+rest",
		ReleaseLevel:      "ga",
		Metadata:          true,
		RESTNumericEnums:  true,
		HasGoGRPC:         true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_Errors(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid GAPIC",
			content: `go_gapic_library(
    importpath = "cloud.google.com/go/test",
    service_yaml = "test.yaml",
)`,
		},
		{
			name:    "valid non-GAPIC",
			content: `go_grpc_library()`,
		},
		{
			name: "gRPC service config and transport are optional",
			content: `go_gapic_library(
    importpath = "cloud.google.com/go/test",
    service_yaml = "test.yaml",
)`,
		},
		{
			name: "missing GAPICImportPath",
			content: `go_gapic_library(
    service_yaml = "test.yaml",
)`,
			wantErr: true,
		},
		{
			name: "missing ServiceYAML",
			content: `go_gapic_library(
    importpath = "cloud.google.com/go/test",
)`,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			buildPath := filepath.Join(tmpDir, "BUILD.bazel")
			if err := os.WriteFile(buildPath, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := Parse(buildPath); err != nil {
				if !test.wantErr {
					t.Errorf("Parse() error = %v, wantErr %v", err, test.wantErr)
				}
			}
		})
	}
}

func TestParse_NoGAPIC(t *testing.T) {
	got := mustParse(t, `
go_grpc_library(
    name = "asset_go_proto",
    importpath = "cloud.google.com/go/asset/apiv1/assetpb",
    protos = [":asset_proto"],
)
`)

	want := &Config{
		HasGoGRPC: true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_LegacyProtocPluginNoGRPC(t *testing.T) {
	got := mustParse(t, `
go_proto_library(
    name = "asset_go_proto",
    importpath = "cloud.google.com/go/asset/apiv1/assetpb",
    protos = [":asset_proto"],
)

go_gapic_library(
    name = "asset_go_gapic",
    srcs = [":asset_proto_with_info"],
    grpc_service_config = "cloudasset_grpc_service_config.json",
    importpath = "cloud.google.com/go/asset/apiv1;asset",
    metadata = True,
    release_level = "ga",
    rest_numeric_enums = True,
    service_yaml = "cloudasset_v1.yaml",
    transport = "grpc+rest",
    diregapic = False,
)
`)

	want := &Config{
		HasGAPIC:          true,
		GAPICImportPath:   "cloud.google.com/go/asset/apiv1;asset",
		ServiceYAML:       "cloudasset_v1.yaml",
		GRPCServiceConfig: "cloudasset_grpc_service_config.json",
		Transport:         "grpc+rest",
		ReleaseLevel:      "ga",
		Metadata:          true,
		RESTNumericEnums:  true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_LegacyProtocPluginWithGRPC(t *testing.T) {
	got := mustParse(t, `
go_proto_library(
    name = "asset_go_proto",
	compilers = ["@io_bazel_rules_go//proto:go_grpc"],
    importpath = "cloud.google.com/go/asset/apiv1/assetpb",
    protos = [":asset_proto"],
)

go_gapic_library(
    name = "asset_go_gapic",
    srcs = [":asset_proto_with_info"],
    grpc_service_config = "cloudasset_grpc_service_config.json",
    importpath = "cloud.google.com/go/asset/apiv1;asset",
    metadata = True,
    release_level = "ga",
    rest_numeric_enums = True,
    service_yaml = "cloudasset_v1.yaml",
    transport = "grpc+rest",
    diregapic = False,
)
`)

	want := &Config{
		HasGAPIC:          true,
		GAPICImportPath:   "cloud.google.com/go/asset/apiv1;asset",
		ServiceYAML:       "cloudasset_v1.yaml",
		GRPCServiceConfig: "cloudasset_grpc_service_config.json",
		Transport:         "grpc+rest",
		ReleaseLevel:      "ga",
		Metadata:          true,
		RESTNumericEnums:  true,
		HasLegacyGRPC:     true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func mustParse(t *testing.T, content string) *Config {
	t.Helper()
	tmpDir := t.TempDir()
	buildPath := filepath.Join(tmpDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Parse(buildPath)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
