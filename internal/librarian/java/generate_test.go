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
	"archive/zip"
	"bytes"
	"fmt"
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
			t.Parallel()
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

func TestConstructProtocCommandArgs_Success(t *testing.T) {
	t.Parallel()
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	protocOptions := []string{"--java_out=out"}

	args, protos, err := constructProtocCommandArgs(api, googleapisDir, protocOptions)
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
		t.Errorf("mismatch in args (-want +got):\n%s", diff)
	}

	// Verify protos contains the expected files
	expectedProtos := []string{
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
		filepath.Join(googleapisDir, "google/cloud/common_resources.proto"),
	}
	sort.Strings(expectedProtos)
	sort.Strings(protos)
	if diff := cmp.Diff(expectedProtos, protos); diff != "" {
		t.Errorf("mismatch in protos (-want +got):\n%s", diff)
	}
}

func TestConstructProtocCommandArgs_Error(t *testing.T) {
	t.Parallel()
	api := &config.API{Path: "nonexistent"}
	protocOptions := []string{"--java_out=out"}
	if _, _, err := constructProtocCommandArgs(api, googleapisDir, protocOptions); err == nil {
		t.Error("constructProtocCommandArgs() expected error for nonexistent path, got nil")
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

func TestGenerate_ErrorCases(t *testing.T) {
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
			err := generate(t.Context(), test.library, googleapisDir)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("generate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestGenerateLibraries_ErrorCase(t *testing.T) {
	t.Parallel()
	libraries := []*config.Library{
		{Name: "lib1", APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}}, Output: t.TempDir()},
	}
	err := GenerateLibraries(t.Context(), libraries, googleapisDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPostProcess(t *testing.T) {
	t.Parallel()
	outdir := t.TempDir()
	libraryName := "secretmanager"
	version := "v1"
	gapicDir := filepath.Join(outdir, version, "gapic")
	grpcDir := filepath.Join(outdir, version, "grpc")
	protoDir := filepath.Join(outdir, version, "proto")
	if err := os.MkdirAll(filepath.Join(gapicDir, "src", "main", "java"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(grpcDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "package com.google.cloud.secretmanager.v1;"
	grpcFile := filepath.Join(grpcDir, "GrpcFile.java")
	if err := os.WriteFile(grpcFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatal(err)
	}
	protoFile := filepath.Join(protoDir, "ProtoFile.java")
	if err := os.WriteFile(protoFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a dummy srcjar (which is a zip)
	srcjarPath := filepath.Join(gapicDir, "temp-codegen.srcjar")
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	mainFile, err := zw.Create("src/main/java/com/google/cloud/secretmanager/v1/SomeFile.java")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mainFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	testFile, err := zw.Create("src/test/java/com/google/cloud/secretmanager/v1/SomeTest.java")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcjarPath, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	protos := []string{filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto")}
	if err := postProcess(t.Context(), outdir, libraryName, version, googleapisDir, gapicDir, grpcDir, protoDir, protos); err != nil {
		t.Fatalf("postProcess failed: %v", err)
	}

	// Verify that the file from srcjar was unzipped and moved, but NO header was added.
	unzippedPath := filepath.Join(outdir, "google-cloud-secretmanager", "src", "main", "java", "com", "google", "cloud", "secretmanager", "v1", "SomeFile.java")
	gotContent, err := os.ReadFile(unzippedPath)
	if err != nil {
		t.Errorf("expected unzipped file at %s, but it was not found: %v", unzippedPath, err)
	}
	if strings.HasPrefix(string(gotContent), "/*\n * Copyright") {
		t.Errorf("expected no header to be prepended to %s, but one was found", unzippedPath)
	}

	// Verify that the proto file HAS a header added.
	protoDestPath := filepath.Join(outdir, "proto-google-cloud-secretmanager-v1", "src", "main", "java", "ProtoFile.java")
	gotProtoContent, err := os.ReadFile(protoDestPath)
	if err != nil {
		t.Errorf("expected proto file at %s, but it was not found: %v", protoDestPath, err)
	}
	if !strings.HasPrefix(string(gotProtoContent), "/*\n * Copyright") {
		t.Errorf("expected header to be prepended to %s, but it was not found", protoDestPath)
	}

	unzippedTestPath := filepath.Join(outdir, "google-cloud-secretmanager", "src", "test", "java", "com", "google", "cloud", "secretmanager", "v1", "SomeTest.java")
	if _, err := os.Stat(unzippedTestPath); err != nil {
		t.Errorf("expected unzipped test file at %s, but it was not found: %v", unzippedTestPath, err)
	}

	// Verify that the version directory was cleaned up
	if _, err := os.Stat(filepath.Join(outdir, version)); !os.IsNotExist(err) {
		t.Errorf("expected directory %s to be removed", filepath.Join(outdir, version))
	}
}

func TestRestructureOutput(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

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
	protoPath := filepath.Join(googleapisDir, "google", "cloud", "secretmanager", "v1", "service.proto")

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

func TestCopyProtos_Success(t *testing.T) {
	t.Parallel()
	destDir := t.TempDir()
	proto1 := filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto")
	commonResources := filepath.Join(googleapisDir, "google/cloud/common_resources.proto")
	protos := []string{proto1, commonResources}
	if err := copyProtos(googleapisDir, protos, destDir); err != nil {
		t.Fatalf("copyProtos failed: %v", err)
	}
	// Verify proto1 was copied
	if _, err := os.Stat(filepath.Join(destDir, "google/cloud/secretmanager/v1/service.proto")); err != nil {
		t.Errorf("expected proto1 to be copied: %v", err)
	}
	// Verify commonResources was NOT copied
	if _, err := os.Stat(filepath.Join(destDir, "google/cloud/common_resources.proto")); !os.IsNotExist(err) {
		t.Errorf("expected commonResources to be skipped")
	}
}

func TestCopyProtos_ErrorCase(t *testing.T) {
	t.Parallel()
	destDir := t.TempDir()
	if err := copyProtos(googleapisDir, []string{"/other/path/proto.proto"}, destDir); err == nil {
		t.Error("expected error for proto not in googleapisDir, got nil")
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
		t.Errorf("collectJavaFiles() mismatch (-want +got):\n%s", diff)
	}
}

func TestAddMissingHeaders(t *testing.T) {
	for _, test := range []struct {
		name       string
		filename   string
		content    string
		wantHeader bool
	}{
		{
			name:       "file without header",
			filename:   "NoHeader.java",
			content:    "package com.example;\npublic class NoHeader {}",
			wantHeader: true,
		},
		{
			name:       "file with existing header",
			filename:   "WithHeader.java",
			content:    "/*\n * Copyright 2024 Google LLC\n */\npackage com.example;\npublic class WithHeader {}",
			wantHeader: true,
		},
		{
			name:     "non-java file",
			filename: "test.txt",
			content:  "some text",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, test.filename)
			if err := os.WriteFile(path, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			if err := addMissingHeaders(tmpDir); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			hasHeader := strings.HasPrefix(string(got), "/*\n * Copyright")
			if hasHeader != test.wantHeader {
				t.Errorf("header presence = %v, want %v", hasHeader, test.wantHeader)
			}
			if !strings.Contains(string(got), test.content) {
				t.Error("original content not preserved")
			}
		})
	}
}
