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

package java

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

func TestCreateProtocOptions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		api      *config.API
		library  *config.Library
		expected []string
		wantErr  bool
	}{
		{
			name:    "basic case",
			api:     &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{},
			expected: []string{
				"--java_out=proto-out",
				"--java_grpc_out=grpc-out",
				"--java_gapic_out=metadata:gapic-out",
				"--java_gapic_opt=metadata,api-service-config=../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml,grpc-service-config=../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,transport=grpc+rest,rest-numeric-enums",
			},
		},
		{
			name: "rest transport",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Transport: "rest",
			},
			expected: []string{
				"--java_out=proto-out",
				"--java_gapic_out=metadata:gapic-out",
				"--java_gapic_opt=metadata,api-service-config=../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml,grpc-service-config=../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,transport=rest,rest-numeric-enums",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := createProtocOptions(test.api, test.library, googleapisDir, "proto-out", "grpc-out", "gapic-out")
			if (err != nil) != test.wantErr {
				t.Fatalf("createProtocOptions() error = %v, wantErr %v", err, test.wantErr)
			}

			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("createProtocOptions() returned diff (-want +got): %s", diff)
			}
		})
	}
}

func TestGenerateAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow test: Java GAPIC code generation")
	}

	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-java_gapic")
	testhelper.RequireCommand(t, "protoc-gen-java_grpc")

	outdir := t.TempDir()
	err := generateAPI(
		t.Context(),
		&config.API{Path: "google/cloud/secretmanager/v1"},
		&config.Library{Name: "secretmanager", Output: outdir},
		googleapisDir,
		outdir,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the output was restructured.
	restructuredPath := filepath.Join(outdir, "google-cloud-secretmanager", "src", "main", "java")
	if _, err := os.Stat(restructuredPath); err != nil {
		t.Errorf("expected restructured path %s to exist: %v", restructuredPath, err)
	}
}

func TestRestructureOutput(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	googleapisDir := t.TempDir()

	version := "v1"
	libraryID := "secretmanager"
	libraryName := "google-cloud-secretmanager"

	// Create a dummy structure to mimic generator output
	dirs := []string{
		filepath.Join(tmpDir, version, "gapic", "src", "main", "java"),
		filepath.Join(tmpDir, version, "gapic", "src", "main", "resources", "META-INF", "native-image"),
		filepath.Join(tmpDir, version, "gapic", "samples", "snippets", "generated", "src", "main", "java"),
		filepath.Join(tmpDir, version, "proto"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create dummy proto in googleapisDir
	protoPath := filepath.Join(googleapisDir, "google", "cloud", "secretmanager", "v1", "service.proto")
	if err := os.MkdirAll(filepath.Dir(protoPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(protoPath, []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a dummy sample file
	sampleFile := filepath.Join(tmpDir, version, "gapic", "samples", "snippets", "generated", "src", "main", "java", "Sample.java")
	if err := os.WriteFile(sampleFile, []byte("public class Sample {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a dummy reflect-config.json
	reflectConfigPath := filepath.Join(tmpDir, version, "gapic", "src", "main", "resources", "META-INF", "native-image", "reflect-config.json")
	if err := os.WriteFile(reflectConfigPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := restructureOutput(tmpDir, libraryID, version, googleapisDir, []string{protoPath}); err != nil {
		t.Fatalf("restructureOutput failed: %v", err)
	}

	// Verify sample file location
	wantSamplePath := filepath.Join(tmpDir, "samples", "snippets", "generated", "Sample.java")
	if _, err := os.Stat(wantSamplePath); err != nil {
		t.Errorf("expected sample file at %s, but it was not found: %v", wantSamplePath, err)
	}

	// Verify reflect-config.json location
	wantReflectPath := filepath.Join(tmpDir, libraryName, "src", "main", "resources", "META-INF", "native-image", "reflect-config.json")
	if _, err := os.Stat(wantReflectPath); err != nil {
		t.Errorf("expected reflect-config.json at %s, but it was not found: %v", wantReflectPath, err)
	}

	// Verify proto file location
	wantProtoPath := filepath.Join(tmpDir, fmt.Sprintf("proto-%s-%s", libraryName, version), "src", "main", "proto", "google", "cloud", "secretmanager", "v1", "service.proto")
	if _, err := os.Stat(wantProtoPath); err != nil {
		t.Errorf("expected proto file at %s, but it was not found: %v", wantProtoPath, err)
	}
}

func TestFormat(t *testing.T) {
	library := &config.Library{Name: "test-lib"}

	if err := Format(t.Context(), library); err != nil {
		t.Errorf("Format() error = %v, want nil", err)
	}
}

func TestClean(t *testing.T) {
	library := &config.Library{Name: "test-lib"}

	if err := Clean(library); err != nil {
		t.Errorf("Clean() error = %v, want nil", err)
	}
}
