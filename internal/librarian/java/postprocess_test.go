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

	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestPostProcessAPI(t *testing.T) {
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

	apiProtos := []string{filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto")}
	p := postProcessParams{
		outDir:         outdir,
		libraryName:    libraryName,
		version:        version,
		googleapisDir:  googleapisDir,
		apiProtos:      apiProtos,
		includeSamples: true,
		gapicDir:       gapicDir,
		grpcDir:        grpcDir,
		protoDir:       protoDir,
	}
	if err := postProcessAPI(t.Context(), p); err != nil {
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

	p := postProcessParams{
		outDir:         tmpDir,
		libraryName:    libraryID,
		version:        version,
		googleapisDir:  googleapisDir,
		apiProtos:      []string{protoPath},
		includeSamples: true,
		gapicDir:       filepath.Join(tmpDir, version, "gapic"),
		grpcDir:        filepath.Join(tmpDir, version, "grpc"),
		protoDir:       filepath.Join(tmpDir, version, "proto"),
	}
	if err := restructureOutput(p); err != nil {
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

func TestRestructureOutput_NoSamples(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	version := "v1"
	libraryID := "secretmanager"
	// Create a dummy structure to mimic generator output
	dirs := []string{
		filepath.Join(tmpDir, version, "gapic", "src", "main", "java"),
		filepath.Join(tmpDir, version, "gapic", "samples", "snippets", "generated", "src", "main", "java"),
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

	p := postProcessParams{
		outDir:         tmpDir,
		libraryName:    libraryID,
		version:        version,
		googleapisDir:  googleapisDir,
		apiProtos:      nil,
		includeSamples: false,
		gapicDir:       filepath.Join(tmpDir, version, "gapic"),
		grpcDir:        filepath.Join(tmpDir, version, "grpc"),
		protoDir:       filepath.Join(tmpDir, version, "proto"),
	}
	if err := restructureOutput(p); err != nil {
		t.Fatalf("restructureOutput failed: %v", err)
	}
	// Verify sample file location DOES NOT exist
	wantSamplePath := filepath.Join(tmpDir, "samples", "snippets", "generated", "Sample.java")
	if _, err := os.Stat(wantSamplePath); !os.IsNotExist(err) {
		t.Errorf("expected sample file at %s to be missing, but it exists", wantSamplePath)
	}
}

func TestCopyProtos_Success(t *testing.T) {
	t.Parallel()
	destDir := t.TempDir()
	proto1 := filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto")
	protos := []string{proto1}
	if err := copyProtos(googleapisDir, protos, destDir); err != nil {
		t.Fatalf("copyProtos failed: %v", err)
	}
	// Verify proto1 was copied
	if _, err := os.Stat(filepath.Join(destDir, "google/cloud/secretmanager/v1/service.proto")); err != nil {
		t.Errorf("expected proto1 to be copied: %v", err)
	}
}

func TestCopyProtos_ErrorCase(t *testing.T) {
	t.Parallel()
	destDir := t.TempDir()
	if err := copyProtos(googleapisDir, []string{"/other/path/proto.proto"}, destDir); err == nil {
		t.Error("expected error for proto not in googleapisDir, got nil")
	}
}

func TestAddMissingHeaders(t *testing.T) {
	for _, test := range []struct {
		name         string
		filename     string
		content      string
		wantModified bool
	}{
		{
			name:         "file without header",
			filename:     "NoHeader.java",
			content:      "package com.example;",
			wantModified: true,
		},
		{
			name:     "file with full header",
			filename: "WithHeader.java",
			content:  "/* Licensed under the Apache License, Version 2.0 (the \"License\") */\npackage com.example;",
		},
		{
			name:         "file with partial header",
			filename:     "PartialHeader.java",
			content:      "/* Copyright 2024 Google LLC */\npackage com.example;",
			wantModified: true,
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
			originalContent := []byte(test.content)
			if err := os.WriteFile(path, originalContent, 0644); err != nil {
				t.Fatal(err)
			}
			if err := addMissingHeaders(tmpDir); err != nil {
				t.Fatal(err)
			}

			newContent, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			wasModified := !bytes.Equal(originalContent, newContent)
			if wasModified != test.wantModified {
				t.Errorf("modification status = %v, want %v", wasModified, test.wantModified)
			}
		})
	}
}

func TestSortAPIs(t *testing.T) {
	for _, test := range []struct {
		name string
		apis []*config.API
		want []string
	}{
		{
			name: "stable before unstable",
			apis: []*config.API{
				{Path: "google/cloud/secretmanager/v1beta1"},
				{Path: "google/cloud/secretmanager/v1"},
			},
			want: []string{
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v1beta1",
			},
		},
		{
			name: "higher stable version before lower",
			apis: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
				{Path: "google/cloud/secretmanager/v2"},
			},
			want: []string{
				"google/cloud/secretmanager/v2",
				"google/cloud/secretmanager/v1",
			},
		},
		{
			name: "higher unstable version before lower (string comparison)",
			apis: []*config.API{
				{Path: "google/cloud/secretmanager/v1beta2"},
				{Path: "google/cloud/secretmanager/v1beta1"},
			},
			want: []string{
				"google/cloud/secretmanager/v1beta2",
				"google/cloud/secretmanager/v1beta1",
			},
		},
		{
			name: "no version (lower depth before higher)",
			apis: []*config.API{
				{Path: "google/cloud/secretmanager/v1/subpath"},
				{Path: "google/cloud/secretmanager"},
			},
			want: []string{
				"google/cloud/secretmanager",
				"google/cloud/secretmanager/v1/subpath",
			},
		},
		{
			name: "version before no version",
			apis: []*config.API{
				{Path: "google/cloud/secretmanager"},
				{Path: "google/cloud/secretmanager/v1"},
			},
			want: []string{
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sortAPIs(test.apis)
			var got []string
			for _, api := range test.apis {
				got = append(got, api.Path)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
