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
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestPostProcessAPI_SkipHeaders(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		monolithic bool
		wantHeader string
	}{
		{"default adds header", false, "/*\n * Copyright"},
		{"monolithic skips header", true, "package"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			outdir := t.TempDir()
			apiBase := "v1"
			gRPCDir := filepath.Join(outdir, apiBase, "grpc")
			if err := os.MkdirAll(gRPCDir, 0o755); err != nil {
				t.Fatal(err)
			}
			grpcFile := filepath.Join(gRPCDir, "GRPCFile.java")
			if err := os.WriteFile(grpcFile, []byte("package com.test;"), 0o644); err != nil {
				t.Fatal(err)
			}
			params := postProcessParams{
				outDir:  outdir,
				apiBase: apiBase,
				library: &config.Library{Java: &config.JavaModule{}},
				javaAPI: &config.JavaAPI{Monolithic: test.monolithic},
			}
			if err := addHeaders(params, []string{gRPCDir}); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(grpcFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(got, []byte(test.wantHeader)) {
				t.Errorf("mismatch got = %q, want %q", got, test.wantHeader)
			}
		})
	}
}

func TestPostProcessAPI_AlternateHeaders(t *testing.T) {
	t.Parallel()
	outdir := t.TempDir()
	apiBase := "v1"
	gRPCDir := filepath.Join(outdir, apiBase, "grpc")
	if err := os.MkdirAll(gRPCDir, 0o755); err != nil {
		t.Fatal(err)
	}
	grpcFile := filepath.Join(gRPCDir, "GRPCFile.java")
	if err := os.WriteFile(grpcFile, []byte("package com.test;"), 0o644); err != nil {
		t.Fatal(err)
	}
	params := postProcessParams{
		outDir:  outdir,
		apiBase: apiBase,
		library: &config.Library{Java: &config.JavaModule{}},
		javaAPI: &config.JavaAPI{},
	}
	altHeader := "/* Alternate */\n"
	headerFile := filepath.Join(outdir, "header.txt")
	if err := os.WriteFile(headerFile, []byte(altHeader), 0o644); err != nil {
		t.Fatal(err)
	}
	params.library.Java.AlternateHeaders = "header.txt"
	if err := addHeaders(params, []string{gRPCDir}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(grpcFile)
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := "/* Alternate */"
	if !bytes.HasPrefix(got, []byte(wantHeader)) {
		t.Errorf("mismatch got = %q, want %q", got, wantHeader)
	}
	if err := os.WriteFile(grpcFile, []byte("package com.test;"), 0o644); err != nil {
		t.Fatal(err)
	}
	params.javaAPI.Monolithic = true
	if err := addHeaders(params, []string{gRPCDir}); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(grpcFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte(wantHeader)) {
		t.Errorf("monolithic mismatch got = %q, want %q", got, wantHeader)
	}
}

func TestCopyProtos_Success(t *testing.T) {
	t.Parallel()
	destDir := t.TempDir()
	proto1 := filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto")
	protos := []protoFileToCopy{
		{
			absolutePath: proto1,
			relativePath: "google/cloud/secretmanager/v1/service.proto",
		},
	}
	if err := copyProtos(protos, destDir); err != nil {
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
	err := copyProtos([]protoFileToCopy{{absolutePath: "/other/path/proto.proto", relativePath: "other/path/proto.proto"}}, destDir)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("copyProtos() error = %v, wantErr %v", err, fs.ErrNotExist)
	}
}

func TestAddMissingHeaders(t *testing.T) {
	defaultHeader := buildLicenseText(time.Now().Year())
	for _, test := range []struct {
		name        string
		params      postProcessParams
		filename    string
		content     string
		wantContent string
	}{
		{
			name:        "file without header",
			filename:    "NoHeader.java",
			content:     "package com.example;",
			wantContent: defaultHeader + "package com.example;",
		},
		{
			name:        "file with full header",
			filename:    "WithHeader.java",
			content:     "/* Licensed under the Apache License, Version 2.0 (the \"License\") */\npackage com.example;",
			wantContent: "/* Licensed under the Apache License, Version 2.0 (the \"License\") */\npackage com.example;",
		},
		{
			name:        "file with partial header",
			filename:    "PartialHeader.java",
			content:     "/* Copyright 2024 Google LLC */\npackage com.example;",
			wantContent: defaultHeader + "/* Copyright 2024 Google LLC */\npackage com.example;",
		},
		{
			name:        "non-java file",
			filename:    "test.txt",
			content:     "some text",
			wantContent: "some text",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, test.filename)
			if err := os.WriteFile(path, []byte(test.content), 0o644); err != nil {
				t.Fatal(err)
			}

			params := test.params
			params.outDir = tmpDir
			if params.library != nil && params.library.Java != nil && params.library.Java.AlternateHeaders != "" {
				headerPath := filepath.Join(tmpDir, params.library.Java.AlternateHeaders)
				if err := os.MkdirAll(filepath.Dir(headerPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(headerPath, []byte("/* Alternate Header */\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if err := addMissingHeaders(params, tmpDir); err != nil {
				t.Fatal(err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantContent, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddMissingHeaders_AlternateHeaders_Error(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	params := postProcessParams{
		outDir: tmpDir,
		library: &config.Library{
			Java: &config.JavaModule{
				AlternateHeaders: "missing-header.txt",
			},
		},
	}
	err := addMissingHeaders(params, tmpDir)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("addMissingHeaders() error = %v, wantErr %v", err, fs.ErrNotExist)
	}
}

func TestCopyFiles(t *testing.T) {
	t.Parallel()
	outdir := t.TempDir()
	apiBase := "v1"
	gapicDir := filepath.Join(outdir, apiBase, "gapic")
	srcPath := "src/main/java/com/google/storage/v2/gapic_metadata.json"
	destPath := "src/main/resources/com/google/storage/v2/gapic_metadata.json"

	fullSrcPath := filepath.Join(gapicDir, srcPath)
	if err := os.MkdirAll(filepath.Dir(fullSrcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"schema": "1.0"}`
	if err := os.WriteFile(fullSrcPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	params := postProcessParams{
		outDir:  outdir,
		apiBase: apiBase,
		javaAPI: &config.JavaAPI{
			CopyFiles: []*config.JavaFileCopy{
				{
					Source:      srcPath,
					Destination: destPath,
				},
			},
		},
	}
	if err := copyFiles(params); err != nil {
		t.Fatal(err)
	}
	// Verify copy
	fullDestPath := filepath.Join(gapicDir, destPath)
	if _, err := os.Stat(fullDestPath); err != nil {
		t.Errorf("destination file %s does not exist: %v", fullDestPath, err)
	}
	if _, err := os.Stat(fullSrcPath); err != nil {
		t.Errorf("source file %s should still exist", fullSrcPath)
	}
	gotContent, err := os.ReadFile(fullDestPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(content, string(gotContent)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCopyFiles_Error(t *testing.T) {
	t.Parallel()
	outdir := t.TempDir()
	apiBase := "v1"
	params := postProcessParams{
		outDir:  outdir,
		apiBase: apiBase,
		javaAPI: &config.JavaAPI{
			CopyFiles: []*config.JavaFileCopy{
				{
					Source:      "non-existent",
					Destination: "dest",
				},
			},
		},
	}
	err := copyFiles(params)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("copyFiles() error = %v, wantErr %v", err, fs.ErrNotExist)
	}
}

func TestApplyMoveActionsToLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		srcFiles  map[string]string
		destFiles map[string]string
		keepSet   map[string]bool
		wantFiles map[string]string
	}{
		{
			name:      "new file",
			srcFiles:  map[string]string{"file.txt": "content"},
			keepSet:   nil,
			wantFiles: map[string]string{"file.txt": "content"},
		},
		{
			name:      "overwrite",
			srcFiles:  map[string]string{"file.txt": "new content"},
			destFiles: map[string]string{"file.txt": "old content"},
			keepSet:   nil,
			wantFiles: map[string]string{"file.txt": "new content"},
		},
		{
			name:      "keepset preserve",
			srcFiles:  map[string]string{"file.txt": "new content"},
			destFiles: map[string]string{"file.txt": "old content"},
			keepSet:   map[string]bool{"file.txt": true},
			wantFiles: map[string]string{"file.txt": "old content"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			srcDir := filepath.Join(dir, "src")
			destDir := filepath.Join(dir, "dest")
			writeFiles(t, srcDir, test.srcFiles)
			writeFiles(t, destDir, test.destFiles)
			actions := []moveAction{
				{src: srcDir, dest: destDir, description: "test files"},
			}
			if err := ApplyMoveActionsToLibrary(actions, destDir, test.keepSet); err != nil {
				t.Fatal(err)
			}
			got := readDirFiles(t, destDir)
			if diff := cmp.Diff(test.wantFiles, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestApplyMoveActionsToLibrary_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name              string
		files             map[string]string
		destFiles         map[string]string
		readOnlySrcParent bool
		missingSrc        bool
		wantErr           error
	}{
		{
			name:      "directory to file type mismatch",
			files:     map[string]string{"mismatch/file.txt": "content"},
			destFiles: map[string]string{"mismatch": "content"},
			wantErr:   syscall.ENOTDIR,
		},
		{
			name:              "source parent directory inaccessible",
			readOnlySrcParent: true,
			wantErr:           fs.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			srcParent := filepath.Join(dir, "src_parent")
			srcDir := filepath.Join(srcParent, "src")
			destDir := filepath.Join(dir, "dest")
			if !test.missingSrc {
				writeFiles(t, srcDir, test.files)
			}
			writeFiles(t, destDir, test.destFiles)
			if test.readOnlySrcParent {
				if err := os.Chmod(srcParent, 0o000); err != nil {
					t.Fatal(err)
				}
				defer os.Chmod(srcParent, 0o755)
			}
			actions := []moveAction{
				{src: srcDir, dest: destDir, description: "test files"},
			}
			err := ApplyMoveActionsToLibrary(actions, destDir, nil)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("ApplyMoveActionsToLibrary() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestApplyMoveActionsToLibrary_NonExistentSource(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	destDir := filepath.Join(dir, "dest")
	writeFiles(t, destDir, nil)
	actions := []moveAction{
		{src: srcDir, dest: destDir, description: "non-existent src"},
	}
	if err := ApplyMoveActionsToLibrary(actions, destDir, nil); err != nil {
		t.Fatal(err)
	}
	got := readDirFiles(t, destDir)
	want := map[string]string{}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRestructureToLibrary(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		monolithic     bool
		includeSamples bool
		filesToWrite   map[string]string
		wantFiles      map[string]string
	}{
		{
			name:           "standard multi-module",
			monolithic:     false,
			includeSamples: true,
			filesToWrite: map[string]string{
				"v1/gapic/src/main/java/Foo.java":                               "class Foo {}",
				"v1/gapic/src/test/FooTest.java":                                "class FooTest {}",
				"v1/proto/com/google/cloud/test/v1/BarProto.java":               "class BarProto {}",
				"v1/grpc/com/google/cloud/test/v1/BarGrpc.java":                 "class BarGrpc {}",
				"v1/gapic/proto/src/main/java/Resource.java":                    "class Resource {}",
				"v1/gapic/samples/snippets/generated/src/main/java/Sample.java": "class Sample {}",
			},
			wantFiles: map[string]string{
				"proto-google-cloud-test-v1/src/main/java/com/google/cloud/test/v1/BarProto.java": "class BarProto {}",
				"grpc-google-cloud-test-v1/src/main/java/com/google/cloud/test/v1/BarGrpc.java":   "class BarGrpc {}",
				"proto-google-cloud-test-v1/src/main/java/Resource.java":                          "class Resource {}",
				"google-cloud-test/src/main/java/Foo.java":                                        "class Foo {}",
				"google-cloud-test/src/test/FooTest.java":                                         "class FooTest {}",
				"samples/snippets/generated/Sample.java":                                          "class Sample {}",
			},
		},
		{
			name:           "monolithic library",
			monolithic:     true,
			includeSamples: false,
			filesToWrite: map[string]string{
				"v1/gapic/src/main/java/Gapic.java": "class Gapic {}",
				"v1/grpc/Grpc.java":                 "class Grpc {}",
				"v1/proto/Proto.java":               "class Proto {}",
			},
			wantFiles: map[string]string{
				"src/main/java/Gapic.java": "class Gapic {}",
				"src/main/java/Grpc.java":  "class Grpc {}",
				"src/main/java/Proto.java": "class Proto {}",
			},
		},
		{
			name:           "samples disabled",
			monolithic:     false,
			includeSamples: false,
			filesToWrite: map[string]string{
				"v1/gapic/src/main/java/Foo.java":                               "class Foo {}",
				"v1/gapic/samples/snippets/generated/src/main/java/Sample.java": "class Sample {}",
			},
			wantFiles: map[string]string{
				"google-cloud-test/src/main/java/Foo.java": "class Foo {}",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srcDir := t.TempDir()
			destDir := t.TempDir()
			writeFiles(t, srcDir, tc.filesToWrite)
			library := &config.Library{
				Name: "test-lib",
				APIs: []*config.API{{Path: "google/cloud/test/v1", Java: &config.JavaAPI{Monolithic: tc.monolithic}}},
				Java: &config.JavaModule{GroupID: "com.google.cloud", ArtifactID: "google-cloud-test"},
			}
			params := postProcessParams{
				cfg:            &config.Config{},
				library:        library,
				javaAPI:        library.APIs[0].Java,
				outDir:         srcDir,
				includeSamples: tc.includeSamples,
				apiBase:        "v1",
			}
			if err := restructureToLibrary(params, destDir, nil); err != nil {
				t.Fatal(err)
			}
			gotFiles := readDirFiles(t, destDir)
			if diff := cmp.Diff(tc.wantFiles, gotFiles); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRestructureToLibrary_OverwritesExistingFiles verifies that restructureToLibrary overwrites existing files in the destination.
func TestRestructureToLibrary_OverwritesExistingFiles(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	destDir := t.TempDir()
	writeFiles(t, srcDir, map[string]string{
		"v1/gapic/src/main/java/Foo.java": "class Foo { /* new content */ }",
	})
	writeFiles(t, destDir, map[string]string{
		"google-cloud-test/src/main/java/Foo.java": "class Foo { /* old content */ }",
	})
	library := &config.Library{
		Name: "test-lib",
		APIs: []*config.API{{Path: "google/cloud/test/v1", Java: &config.JavaAPI{}}},
		Java: &config.JavaModule{GroupID: "com.google.cloud", ArtifactID: "google-cloud-test"},
	}
	params := postProcessParams{
		cfg:            &config.Config{},
		library:        library,
		javaAPI:        library.APIs[0].Java,
		outDir:         srcDir,
		includeSamples: false,
		apiBase:        "v1",
	}
	// Pass nil keepSet to expect default overwriting of conflicting files.
	if err := restructureToLibrary(params, destDir, nil); err != nil {
		t.Fatal(err)
	}
	gotFiles := readDirFiles(t, destDir)
	wantFiles := map[string]string{
		"google-cloud-test/src/main/java/Foo.java": "class Foo { /* new content */ }",
	}
	if diff := cmp.Diff(wantFiles, gotFiles); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRestructureToLibrary_CommonProtos(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	destDir := t.TempDir()
	writeFiles(t, srcDir, map[string]string{
		"v1/proto/com/google/cloud/location/LocationsProto.java": "class LocationsProto {}",
	})
	library := &config.Library{
		Name: commonProtosLibrary,
		APIs: []*config.API{{Path: "google/cloud/test/v1", Java: &config.JavaAPI{ProtoArtifactIDOverride: "proto-google-common-protos"}}},
		Java: &config.JavaModule{GroupID: "com.google.cloud", ArtifactID: "google-cloud-" + commonProtosLibrary},
	}
	params := postProcessParams{
		cfg:            &config.Config{},
		library:        library,
		javaAPI:        library.APIs[0].Java,
		outDir:         srcDir,
		includeSamples: false,
		apiBase:        "v1",
	}
	if err := restructureToLibrary(params, destDir, nil); err != nil {
		t.Fatal(err)
	}
	gotFiles := readDirFiles(t, destDir)
	wantFiles := map[string]string{
		"proto-google-common-protos/src/main/java/com/google/cloud/location/LocationsProto.java": "class LocationsProto {}",
	}
	if diff := cmp.Diff(wantFiles, gotFiles); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// TestPostProcessAPI verifies that Go-native postprocessor correctly restructures
// generated Java files to their target directories and cleans up intermediate files.
func TestPostProcessAPI(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"v1/gapic/src/main/java/Foo.java": "class Foo {}",
		"v1/grpc/dummy":                   "",
		"v1/proto/dummy":                  "",
	})
	library := &config.Library{
		Name: "test-lib",
		APIs: []*config.API{{Path: "google/cloud/test/v1", Java: &config.JavaAPI{}}},
		Java: &config.JavaModule{GroupID: "com.google.cloud", ArtifactID: "google-cloud-test", ReleasedVersion: "1.2.3"},
	}
	postParams := postProcessParams{
		cfg:            &config.Config{},
		library:        library,
		javaAPI:        library.APIs[0].Java,
		outDir:         dir,
		includeSamples: true,
		apiBase:        "v1",
	}
	if err := postProcessAPI(t.Context(), postParams); err != nil {
		t.Fatal(err)
	}
	// Verify that files are relocated directly to target paths and staging is skipped.
	want := map[string]string{
		"google-cloud-test/src/main/java/Foo.java":       "class Foo {}",
		"proto-google-cloud-test-v1/src/main/java/dummy": "",
		"grpc-google-cloud-test-v1/src/main/java/dummy":  "",
	}
	got := readDirFiles(t, dir)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// TestPostProcessLibrary verifies that library-level postprocessing tasks
// (such as text replacements, POM updates, and README generation) execute
// correctly in the Go-native flow.
func TestPostProcessLibrary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"google-cloud-test/src/main/java/Foo.java": "class Foo {}"})
	library := &config.Library{
		Name: "test-lib",
		Java: &config.JavaModule{
			GroupID:    "com.google.cloud",
			ArtifactID: "google-cloud-test",
			// Disable syncPOMs to simplify config requirements.
			SkipPOMUpdates: true,
		},
		// Disable renderREADME to simplify config requirements.
		Keep: []string{"README.md"},
		Postprocess: &config.Postprocess{
			Replace: []config.ReplaceConfig{
				{Path: "google-cloud-test/src/main/java/Foo.java", Original: "class Foo", Replacement: "class RenamedFoo"},
			},
		},
	}
	params := libraryPostProcessParams{
		cfg: &config.Config{
			Libraries: []*config.Library{
				{Name: "google-cloud-java", Version: "1.2.3"},
				{Name: "google-cloud-pom-parent", Version: "1.2.3"},
			},
		},
		library:  library,
		outDir:   dir,
		metadata: &repoMetadata{},
	}
	if err := postProcessLibrary(params); err != nil {
		t.Fatal(err)
	}
	// Verify postprocessing rules were applied.
	want := map[string]string{"google-cloud-test/src/main/java/Foo.java": "class RenamedFoo {}"}
	got := readDirFiles(t, dir)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readDirFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	got := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestToKeepSet(t *testing.T) {
	t.Parallel()
	input := []string{"foo/", "bar/baz", "qux/", ""}
	got := toKeepSet(input)
	want := map[string]bool{
		"foo":     true,
		"bar/baz": true,
		"qux":     true,
		"":        true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
