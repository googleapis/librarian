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
	"os"
	"path/filepath"
	"strings"

	"sort"

	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

func TestCreateProtocOptions(t *testing.T) {
	for _, test := range []struct {
		name     string
		api      *config.API
		javaAPI  *config.JavaAPI
		expected []string
		wantErr  bool
	}{
		{
			name: "basic case",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			javaAPI: &config.JavaAPI{
				Path: "google/cloud/secretmanager/v1",
			},
			expected: []string{
				"--java_out=proto-out",
				"--java_grpc_out=grpc-out",
				"--java_gapic_out=metadata:gapic-out",
				"--java_gapic_opt=metadata,api-service-config=../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml,grpc-service-config=../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,transport=grpc+rest,rest-numeric-enums",
			},
		},
		{
			name: "no rest numeric enum case",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			javaAPI: &config.JavaAPI{
				Path:               "google/cloud/secretmanager/v1",
				NoRestNumericEnums: true,
			},
			expected: []string{
				"--java_out=proto-out",
				"--java_grpc_out=grpc-out",
				"--java_gapic_out=metadata:gapic-out",
				"--java_gapic_opt=metadata,api-service-config=../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml,grpc-service-config=../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,transport=grpc+rest",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := createProtocOptions(test.api, test.javaAPI, googleapisDir, "proto-out", "grpc-out", "gapic-out")
			if (err != nil) != test.wantErr {
				t.Fatalf("createProtocOptions() error = %v, wantErr %v", err, test.wantErr)
			}

			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConstructProtocCommandArgs_Success(t *testing.T) {
	t.Parallel()
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	javaAPI := &config.JavaAPI{
		Path:             api.Path,
		AdditionalProtos: []string{commonProtos},
	}
	protocOptions := []string{"--java_out=out"}

	args, apiProtos, err := constructProtocCommandArgs(api, javaAPI, googleapisDir, protocOptions)
	if err != nil {
		t.Fatalf("constructProtocCommandArgs() unexpected error: %v", err)
	}

	expectedArgs := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
		filepath.Join(googleapisDir, "google/cloud/common_resources.proto"),
		"--java_out=out",
	}

	if diff := cmp.Diff(expectedArgs, args); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify apiProtos contains the expected files
	expectedApiProtos := []string{
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
	}
	sort.Strings(expectedApiProtos)
	sort.Strings(apiProtos)
	if diff := cmp.Diff(expectedApiProtos, apiProtos); diff != "" {
		t.Errorf("mismatch apiProtos (-want +got):\n%s", diff)
	}
}

func TestConstructProtocCommandArgs_AdditionalProtos(t *testing.T) {
	t.Parallel()
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	javaAPI := &config.JavaAPI{
		AdditionalProtos: []string{"google/cloud/common_resources.proto", "google/cloud/location/locations.proto"},
	}
	protocOptions := []string{"--java_out=out"}
	args, apiProtos, err := constructProtocCommandArgs(api, javaAPI, googleapisDir, protocOptions)
	if err != nil {
		t.Fatalf("constructProtocCommandArgs() unexpected error: %v", err)
	}

	expectedArgs := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
		filepath.Join(googleapisDir, "google/cloud/common_resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/location/locations.proto"),
		"--java_out=out",
	}

	if diff := cmp.Diff(expectedArgs, args); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	expectedApiProtos := []string{
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
	}
	sort.Strings(expectedApiProtos)
	sort.Strings(apiProtos)
	if diff := cmp.Diff(expectedApiProtos, apiProtos); diff != "" {
		t.Errorf("mismatch apiProtos (-want +got):\n%s", diff)
	}
}

func TestConstructProtocCommandArgs_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		api     *config.API
		wantErr string
	}{
		{
			name:    "nonexistent path",
			api:     &config.API{Path: "nonexistent"},
			wantErr: "no protos found in api \"nonexistent\"",
		},
		{
			name:    "malformed path",
			api:     &config.API{Path: "malformed["},
			wantErr: "failed to find protos",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			protocOptions := []string{"--java_out=out"}
			javaAPI := &config.JavaAPI{Path: test.api.Path}
			_, _, err := constructProtocCommandArgs(test.api, javaAPI, googleapisDir, protocOptions)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("constructProtocCommandArgs() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestResolveJavaAPI(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		api     *config.API
		want    *config.JavaAPI
	}{
		{
			name:    "not found, returns defaults",
			library: &config.Library{},
			api:     &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.JavaAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{commonProtos},
			},
		},
		{
			name: "found in config",
			library: &config.Library{
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path:             "google/cloud/secretmanager/v1",
							AdditionalProtos: []string{"other.proto"},
							NoSamples:        true,
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.JavaAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{"other.proto"},
				NoSamples:        true,
			},
		},
		{
			name: "found in config, empty additional protos defaults to commonProtos",
			library: &config.Library{
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path: "google/cloud/secretmanager/v1",
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.JavaAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{commonProtos},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := resolveJavaAPI(test.library, test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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

func TestGenerateLibrary_ErrorCases(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		wantErr string
	}{
		{
			name:    "no apis",
			library: &config.Library{Name: "test"},
			wantErr: "no apis configured for library \"test\"",
		},
		{
			name: "invalid version",
			library: &config.Library{
				Name:   "test",
				Output: t.TempDir(),
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager"}, // Missing version
				},
			},
			wantErr: "failed to extract version from api path",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := generateLibrary(t.Context(), test.library, googleapisDir)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("generate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestGenerate_ErrorCase(t *testing.T) {
	t.Parallel()
	libraries := []*config.Library{
		{Name: "lib1", APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}}, Output: t.TempDir()},
	}
	err := Generate(t.Context(), libraries, googleapisDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFormat_Success(t *testing.T) {
	testhelper.RequireCommand(t, "google-java-format")
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{
			name: "successful format",
			setup: func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "SomeClass.java"), []byte("public class SomeClass {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:  "no files found",
			setup: func(t *testing.T, root string) {},
		},
		{
			name: "nested files in subdirectories",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "sub", "dir")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "Nested.java"), []byte("public class Nested {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "files in excluded samples path are ignored",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "samples", "snippets", "generated")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				// This file should NOT be passed to the formatter.
				if err := os.WriteFile(filepath.Join(dir, "Ignored.java"), []byte("public class Ignored {}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			test.setup(t, tmpDir)
			if err := Format(t.Context(), &config.Library{Output: tmpDir}); err != nil {
				t.Errorf("Format() error = %v, want nil", err)
			}
		})
	}
}

func TestFormat_LookPathError(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "SomeClass.java"), []byte("public class SomeClass {}"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	err := Format(t.Context(), &config.Library{Output: tmpDir})
	if err == nil {
		t.Fatal("Format() error = nil, want error")
	}
}

func TestCollectJavaFiles(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	// Create a mix of files
	filesToCreate := []string{
		"Root.java",
		"subdir/Nested.java",
		"subdir/NotJava.txt",
		"samples/snippets/generated/Ignored.java",
		"another/dir/More.java",
	}
	for _, f := range filesToCreate {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{
		filepath.Join(tmpDir, "Root.java"),
		filepath.Join(tmpDir, "subdir", "Nested.java"),
		filepath.Join(tmpDir, "another", "dir", "More.java"),
	}
	got, err := collectJavaFiles(tmpDir)
	if err != nil {
		t.Fatalf("collectJavaFiles() error = %v", err)
	}
	sort.Strings(got)
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
