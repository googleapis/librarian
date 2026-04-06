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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
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
	for _, artifact := range []string{"google-cloud-secretmanager", "proto-google-cloud-secretmanager-v1", "grpc-google-cloud-secretmanager-v1", "google-cloud-secretmanager-bom"} {
		if err := os.MkdirAll(filepath.Join(outdir, artifact), 0755); err != nil {
			t.Fatal(err)
		}
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
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	p := postProcessParams{
		cfg: &config.Config{
			Libraries: []*config.Library{
				{Name: "google-cloud-java", Version: "1.2.3"},
			},
		},
		outDir: outdir,
		metadata: &repoMetadata{
			NamePretty:     "Secret Manager",
			APIDescription: "Secret Manager API",
		},
		library: &config.Library{
			Name: libraryName,
			APIs: []*config.API{api},
		},
		version:        version,
		googleapisDir:  googleapisDir,
		apiProtos:      apiProtos,
		includeSamples: true,
	}
	if err := postProcessAPI(t.Context(), p); err != nil {
		t.Fatal(err)
	}

	// Verify that the file from srcjar was unzipped and moved, but NO header was added.
	unzippedPath := filepath.Join(outdir, "owl-bot-staging", version, "google-cloud-secretmanager", "src", "main", "java", "com", "google", "cloud", "secretmanager", "v1", "SomeFile.java")
	gotContent, err := os.ReadFile(unzippedPath)
	if err != nil {
		t.Errorf("expected unzipped file at %s, but it was not found: %v", unzippedPath, err)
	}
	if strings.HasPrefix(string(gotContent), "/*\n * Copyright") {
		t.Errorf("expected no header to be prepended to %s, but one was found", unzippedPath)
	}

	// Verify that the proto file HAS a header added.
	protoDestPath := filepath.Join(outdir, "owl-bot-staging", version, "proto-google-cloud-secretmanager-v1", "src", "main", "java", "ProtoFile.java")
	gotProtoContent, err := os.ReadFile(protoDestPath)
	if err != nil {
		t.Errorf("expected proto file at %s, but it was not found: %v", protoDestPath, err)
	}
	if !strings.HasPrefix(string(gotProtoContent), "/*\n * Copyright") {
		t.Errorf("expected header to be prepended to %s, but it was not found", protoDestPath)
	}

	unzippedTestPath := filepath.Join(outdir, "owl-bot-staging", version, "google-cloud-secretmanager", "src", "test", "java", "com", "google", "cloud", "secretmanager", "v1", "SomeTest.java")
	if _, err := os.Stat(unzippedTestPath); err != nil {
		t.Errorf("expected unzipped test file at %s, but it was not found: %v", unzippedTestPath, err)
	}

	// Verify that the version directory was cleaned up
	if _, err := os.Stat(filepath.Join(outdir, version)); !os.IsNotExist(err) {
		t.Errorf("expected directory %s to be removed", filepath.Join(outdir, version))
	}
}

func TestRestructureModules(t *testing.T) {
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
		library:        &config.Library{Name: libraryID},
		version:        version,
		googleapisDir:  googleapisDir,
		apiProtos:      []string{protoPath},
		includeSamples: true,
	}
	destRoot := filepath.Join(tmpDir, "dest")
	if err := restructureModules(p, destRoot); err != nil {
		t.Fatal(err)
	}

	// Verify sample file location
	wantSamplePath := filepath.Join(destRoot, "samples", "snippets", "generated", "Sample.java")
	if _, err := os.Stat(wantSamplePath); err != nil {
		t.Errorf("expected sample file at %s, but it was not found: %v", wantSamplePath, err)
	}
	// Verify reflect-config.json location
	wantReflectPath := filepath.Join(destRoot, libraryName, "src", "main", "resources", "META-INF", "native-image", "reflect-config.json")
	if _, err := os.Stat(wantReflectPath); err != nil {
		t.Errorf("expected reflect-config.json at %s, but it was not found: %v", wantReflectPath, err)
	}
	// Verify proto file location
	wantProtoPath := filepath.Join(destRoot, fmt.Sprintf("proto-%s-%s", libraryName, version), "src", "main", "proto", "google", "cloud", "secretmanager", "v1", "service.proto")
	if _, err := os.Stat(wantProtoPath); err != nil {
		t.Errorf("expected proto file at %s, but it was not found: %v", wantProtoPath, err)
	}
}

func TestRestructureModules_NoSamples(t *testing.T) {
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
		library:        &config.Library{Name: libraryID},
		version:        version,
		googleapisDir:  googleapisDir,
		apiProtos:      nil,
		includeSamples: false,
	}
	destRoot := filepath.Join(tmpDir, "dest")
	if err := restructureModules(p, destRoot); err != nil {
		t.Fatal(err)
	}
	// Verify sample file location DOES NOT exist
	wantSamplePath := filepath.Join(destRoot, "samples", "snippets", "generated", "Sample.java")
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
		t.Fatal(err)
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

func TestPostProcessLibrary(t *testing.T) {
	t.Parallel()
	testhelper.RequireCommand(t, "python3")

	library := &config.Library{
		Name:    "secretmanager",
		Version: "1.2.3",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
	}
	defaultCfg := &config.Config{
		Libraries: []*config.Library{
			{Name: rootLibrary, Version: "1.0.0"},
		},
		Default: &config.Default{
			Java: &config.JavaModule{
				LibrariesBomVersion: "26.35.0",
			},
		},
	}

	for _, test := range []struct {
		name    string
		cfg     *config.Config
		setup   func(t *testing.T, outDir string)
		wantErr string
	}{
		{
			name:    "owlbot.py missing",
			cfg:     defaultCfg,
			wantErr: "owlbot.py not found",
		},
		{
			name: "findBomVersion failure",
			cfg:  &config.Config{},
			setup: func(t *testing.T, outDir string) {
				writeOwlBot(t, outDir, "sys.exit(0)")
			},
			wantErr: "libraries bom version not found in config",
		},
		{
			name: "runOwlBot failure (missing templates)",
			cfg:  defaultCfg,
			setup: func(t *testing.T, outDir string) {
				writeOwlBot(t, outDir, "sys.exit(0)")
			},
			wantErr: "templates directory not found",
		},
		{
			name: "findMonorepoVersion failure",
			cfg: &config.Config{
				Default: defaultCfg.Default,
			},
			setup: func(t *testing.T, outDir string) {
				writeOwlBot(t, outDir, "sys.exit(0)")
				if err := os.MkdirAll(filepath.Join(filepath.Dir(outDir), owlbotTemplatesRelPath), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "failed to find monorepo version",
		},
		{
			name: "runOwlBot failure (non-zero exit status)",
			cfg:  defaultCfg,
			setup: func(t *testing.T, outDir string) {
				writeOwlBot(t, outDir, "sys.exit(1)")
				if err := os.MkdirAll(filepath.Join(filepath.Dir(outDir), owlbotTemplatesRelPath), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "failed to run owlbot.py",
		},
		{
			name: "syncPoms failure (missing module directories)",
			cfg:  defaultCfg,
			setup: func(t *testing.T, outDir string) {
				writeOwlBot(t, outDir, "sys.exit(0)")
				if err := os.MkdirAll(filepath.Join(filepath.Dir(outDir), owlbotTemplatesRelPath), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "target directory",
		},
		{
			name: "success",
			cfg:  defaultCfg,
			setup: func(t *testing.T, outDir string) {
				writeOwlBot(t, outDir, "sys.exit(0)")
				if err := os.MkdirAll(filepath.Join(filepath.Dir(outDir), owlbotTemplatesRelPath), 0755); err != nil {
					t.Fatal(err)
				}
				libCoords := deriveLibCoords(library)
				apiCoords := deriveAPICoords(libCoords, "v1")
				for _, dir := range []string{
					filepath.Join(outDir, apiCoords.proto.ArtifactID),
					filepath.Join(outDir, apiCoords.grpc.ArtifactID),
					filepath.Join(outDir, apiCoords.gapic.ArtifactID),
					filepath.Join(outDir, apiCoords.parent.ArtifactID),
					filepath.Join(outDir, apiCoords.bom.ArtifactID),
				} {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatal(err)
					}
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, outDir)
			}
			params := libraryPostProcessParams{
				cfg:      test.cfg,
				library:  library,
				outDir:   outDir,
				metadata: &repoMetadata{NamePretty: "Secret Manager"},
			}
			err := postProcessLibrary(t.Context(), params)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
				}
			}
		})
	}
}

func writeOwlBot(t *testing.T, outDir, script string) {
	t.Helper()
	content := "import sys; " + script
	if err := os.WriteFile(filepath.Join(outDir, "owlbot.py"), []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestRunOwlBot(t *testing.T) {
	t.Parallel()
	testhelper.RequireCommand(t, "python3")
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	templatesDir := filepath.Join(tmp, "sdk-platform-java", "hermetic_build", "library_generation", "owlbot", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a dummy owlbot.py that checks environment variables.
	owlbotContent := `
import os
import sys

lib_version = os.environ.get("SYNTHTOOL_LIBRARY_VERSION")
bom_version = os.environ.get("SYNTHTOOL_LIBRARIES_BOM_VERSION")
templates = os.environ.get("SYNTHTOOL_TEMPLATES")

if lib_version != "1.2.3":
    print(f"Expected SYNTHTOOL_LIBRARY_VERSION=1.2.3, got {lib_version}")
    sys.exit(1)
if bom_version != "4.5.6":
    print(f"Expected SYNTHTOOL_LIBRARIES_BOM_VERSION=4.5.6, got {bom_version}")
    sys.exit(1)
if not templates or not templates.endswith("templates"):
    print(f"Expected SYNTHTOOL_TEMPLATES to be set and end with 'templates', got {templates}")
    sys.exit(1)

with open("owlbot-ran.txt", "w") as f:
    f.write("success")
`
	if err := os.WriteFile(filepath.Join(outDir, "owlbot.py"), []byte(owlbotContent), 0755); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{Version: "1.2.3"}
	if err := runOwlBot(t.Context(), library, outDir, "4.5.6"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "owlbot-ran.txt")); err != nil {
		t.Errorf("expected owlbot.py to run and create owlbot-ran.txt: %v", err)
	}
}

func TestRunOwlBot_Error(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	library := &config.Library{}
	if err := runOwlBot(t.Context(), library, outDir, ""); err == nil {
		t.Error("expected error due to missing templates directory, got nil")
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
