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

package golang

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const testdataDir = "../../testdata/googleapis"

func TestBuildProtocCommand(t *testing.T) {
	for _, test := range []struct {
		name            string
		cfg             *protocConfig
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "proto only",
			cfg: &protocConfig{
				OutputDir:     "/output",
				GoogleapisDir: testdataDir,
				APIPath:       "google/cloud/secretmanager/v1",
			},
			wantContains: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"-I=" + testdataDir,
			},
			wantNotContains: []string{
				"--go_gapic_out",
				"--go-grpc_out",
			},
		},
		{
			name: "with GAPIC",
			cfg: &protocConfig{
				OutputDir:       "/output",
				GoogleapisDir:   testdataDir,
				APIPath:         "google/cloud/secretmanager/v1",
				HasGAPIC:        true,
				GAPICImportPath: "cloud.google.com/go/secretmanager/apiv1",
				ServiceYAML:     "secretmanager_v1.yaml",
			},
			wantContains: []string{
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1",
				"--go_gapic_opt=api-service-config=secretmanager_v1.yaml",
			},
		},
		{
			name: "with go-grpc",
			cfg: &protocConfig{
				OutputDir:     "/output",
				GoogleapisDir: testdataDir,
				APIPath:       "google/cloud/secretmanager/v1",
				HasGoGRPC:     true,
			},
			wantContains: []string{
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
			},
		},
		{
			name: "with service configs",
			cfg: &protocConfig{
				OutputDir:         "/output",
				GoogleapisDir:     testdataDir,
				APIPath:           "google/cloud/secretmanager/v1",
				HasGAPIC:          true,
				GAPICImportPath:   "cloud.google.com/go/secretmanager/apiv1",
				ServiceYAML:       "secretmanager_v1.yaml",
				GRPCServiceConfig: "secretmanager_grpc_service_config.json",
			},
			wantContains: []string{
				"--go_gapic_opt=api-service-config=secretmanager_v1.yaml",
				"--go_gapic_opt=grpc-service-config=secretmanager_grpc_service_config.json",
			},
		},
		{
			name: "with transport and release level",
			cfg: &protocConfig{
				OutputDir:       "/output",
				GoogleapisDir:   testdataDir,
				APIPath:         "google/cloud/secretmanager/v1",
				HasGAPIC:        true,
				GAPICImportPath: "cloud.google.com/go/secretmanager/apiv1",
				ServiceYAML:     "secretmanager_v1.yaml",
				Transport:       "grpc+rest",
				ReleaseLevel:    "ga",
			},
			wantContains: []string{
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=ga",
			},
		},
		{
			name: "with all GAPIC options",
			cfg: &protocConfig{
				OutputDir:        "/output",
				GoogleapisDir:    testdataDir,
				APIPath:          "google/cloud/secretmanager/v1",
				HasGAPIC:         true,
				GAPICImportPath:  "cloud.google.com/go/secretmanager/apiv1",
				ServiceYAML:      "secretmanager_v1.yaml",
				Metadata:         true,
				DIREGAPIC:        true,
				RESTNumericEnums: true,
			},
			wantContains: []string{
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=diregapic",
				"--go_gapic_opt=rest-numeric-enums",
			},
		},
		{
			name: "with nested protos",
			cfg: &protocConfig{
				OutputDir:     "/output",
				GoogleapisDir: testdataDir,
				APIPath:       "google/cloud/secretmanager/v1",
				NestedProtos:  []string{"nested/extra.proto"},
			},
			wantContains: []string{
				filepath.Join(testdataDir, "google/cloud/secretmanager/v1/nested/extra.proto"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			args, err := buildProtocCommand(test.cfg)
			if err != nil {
				t.Fatal(err)
			}

			for _, want := range test.wantContains {
				if !slices.Contains(args, want) {
					t.Errorf("missing %q\ngot: %v", want, args)
				}
			}
			for _, notWant := range test.wantNotContains {
				if slices.ContainsFunc(args, func(arg string) bool {
					return strings.HasPrefix(arg, notWant)
				}) {
					t.Errorf("should not contain %q\ngot: %v", notWant, args)
				}
			}
		})
	}
}

func TestBuildProtocCommand_NoProtos(t *testing.T) {
	cfg := &protocConfig{
		OutputDir:     "/output",
		GoogleapisDir: testdataDir,
		APIPath:       "google/cloud",
	}
	_, err := buildProtocCommand(cfg)
	if err == nil {
		t.Fatal("expected error for directory with no proto files")
	}
	if !strings.Contains(err.Error(), "no .proto files found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCollectProtoFiles(t *testing.T) {
	files, err := collectProtoFiles(testdataDir, "google/cloud/secretmanager/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("got %d files, want 2", len(files))
	}
}

func TestBuildGAPICOpts(t *testing.T) {
	cfg := &protocConfig{
		HasGAPIC:          true,
		GAPICImportPath:   "cloud.google.com/go/test/apiv1",
		ServiceYAML:       "test_v1.yaml",
		GRPCServiceConfig: "test_grpc.json",
		Transport:         "grpc+rest",
		ReleaseLevel:      "ga",
		Metadata:          true,
		DIREGAPIC:         true,
		RESTNumericEnums:  true,
	}

	opts := buildGAPICOpts(cfg)
	want := []string{
		"go-gapic-package=cloud.google.com/go/test/apiv1",
		"api-service-config=test_v1.yaml",
		"grpc-service-config=test_grpc.json",
		"transport=grpc+rest",
		"release-level=ga",
		"metadata",
		"diregapic",
		"rest-numeric-enums",
	}
	if len(opts) != len(want) {
		t.Fatalf("got %d opts, want %d", len(opts), len(want))
	}
	for i, opt := range opts {
		if opt != want[i] {
			t.Errorf("opts[%d] = %q, want %q", i, opt, want[i])
		}
	}
}
