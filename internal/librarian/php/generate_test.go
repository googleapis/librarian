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

package php

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow integration test")
	}
	requirePHPGenerator(t)
	// Use mock googleapis checked in as test data
	googleapisDir := "../../testdata/googleapis"
	absGoogleapis, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}
	absOwlbotCopy, err := filepath.Abs(filepath.Join("testdata", "owlbot_copy.py"))
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	destDir := filepath.Join(repoRoot, "output")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Symlink mock owlbot.py. Tests use a simplified copy-only stub to
	// avoid Node.js/prettier dependencies.
	if err := os.Symlink(absOwlbotCopy, filepath.Join(destDir, "owlbot.py")); err != nil {
		t.Fatal(err)
	}
	library := &config.Library{
		Name:   "secretmanager",
		Output: destDir,
		APIs: []*config.API{
			{
				Path: "google/cloud/secretmanager/v1",
				PHP: &config.PHPAPI{
					CommonResources: new(true),
					StagingSubdir:   "v1",
				},
			},
		},
	}
	cfg := &config.Config{
		Language: config.LanguagePhp,
	}
	err = Generate(t.Context(), cfg, library, &sources.Sources{Googleapis: absGoogleapis})
	if err != nil {
		t.Fatal(err)
	}
	// Verify output
	outputDirs := []string{"src", "tests", "samples", "fragments"}
	for _, dir := range outputDirs {
		p := filepath.Join(library.Output, dir)
		if stat, err := os.Stat(p); err != nil || !stat.IsDir() {
			t.Errorf("expected directory %s to exist and be a directory", p)
		}
	}
}

func requirePHPGenerator(t *testing.T) {
	t.Helper()
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "python3")
	testhelper.RequireCommand(t, "php")
	dir, err := generatorDir(context.Background())
	if err != nil {
		t.Skipf("skipping test: failed to locate PHP generator bin dir: %v", err)
	}
	wrapperPath := filepath.Join(dir, "wrapper.sh")
	if _, err := os.Stat(wrapperPath); err != nil {
		t.Skip("skipping test: PHP generator is not installed (run 'librarian install php' first)")
	}
}

func TestGenerate_Error(t *testing.T) {
	requirePHPGenerator(t)
	for _, test := range []struct {
		name    string
		lib     *config.Library
		wantErr error
	}{
		{
			name: "missing PHP config (requires staging_subdir)",
			lib: &config.Library{
				Name: "SecretManager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
			wantErr: errMissingStagingSubdir,
		},
		{
			name: "missing common_resources config",
			lib: &config.Library{
				Name: "SecretManager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						PHP: &config.PHPAPI{
							StagingSubdir: "v1",
						},
					},
				},
			},
			wantErr: errCommonResourcesUnconfigured,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language: config.LanguagePhp,
			}
			err := Generate(t.Context(), cfg, test.lib, &sources.Sources{Googleapis: t.TempDir()})
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Generate() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestGatherProtos(t *testing.T) {
	tmp := t.TempDir()
	files := []struct {
		path    string
		isProto bool
	}{
		{"a.proto", true},
		{"sub/b.proto", true},
		{"c.txt", false},
		{"sub/d.proto", true},
	}
	for _, f := range files {
		p := filepath.Join(tmp, f.path)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := gatherProtos(tmp)
	if err != nil {
		t.Fatalf("gatherProtos failed: %v", err)
	}
	want := []string{
		filepath.Join(tmp, "a.proto"),
		filepath.Join(tmp, "sub/b.proto"),
		filepath.Join(tmp, "sub/d.proto"),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGapicOpts(t *testing.T) {
	for _, test := range []struct {
		name           string
		apiMetadata    *serviceconfig.API
		grpcConfigPath string
		want           []string
	}{
		{
			name: "defaults",
			want: []string{"metadata", "transport=grpc+rest", "migration-mode=NEW_SURFACE_ONLY", "generate-snippets"},
		},
		{
			name: "with grpc config and service yaml",
			apiMetadata: &serviceconfig.API{
				ServiceConfig: "service.yaml",
			},
			grpcConfigPath: "grpc_config.json",
			want: []string{
				"metadata", "transport=grpc+rest", "migration-mode=NEW_SURFACE_ONLY",
				"rest-numeric-enums", "generate-snippets",
				"grpc_service_config=grpc_config.json",
				"service_yaml=service.yaml",
			},
		},
		{
			name: "skip rest numeric enums",
			apiMetadata: &serviceconfig.API{
				SkipRESTNumericEnums: []string{"php"},
			},
			want: []string{"metadata", "transport=grpc+rest", "migration-mode=NEW_SURFACE_ONLY",
				"generate-snippets"},
		},
		{
			name: "custom transport",
			apiMetadata: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					"php": serviceconfig.Transport("rest"),
				},
			},
			want: []string{"metadata", "transport=rest", "migration-mode=NEW_SURFACE_ONLY",
				"rest-numeric-enums", "generate-snippets"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := gapicOpts(test.apiMetadata, test.grpcConfigPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGatherGAPICProtos(t *testing.T) {
	for _, test := range []struct {
		name                   string
		setupFiles             []string
		apiPath                string
		additionalProtos       []string
		includeCommonResources bool
		wantProtos             []string
	}{
		{
			name: "protos found, common resources enabled",
			setupFiles: []string{
				"google/cloud/secretmanager/v1/service.proto",
				"google/cloud/secretmanager/v1/resources.proto",
				commonResourcesProto,
			},
			apiPath:                "google/cloud/secretmanager/v1",
			includeCommonResources: true,
			wantProtos: []string{
				"google/cloud/secretmanager/v1/resources.proto",
				"google/cloud/secretmanager/v1/service.proto",
				commonResourcesProto,
			},
		},
		{
			name: "protos found, common resources disabled",
			setupFiles: []string{
				"google/cloud/secretmanager/v1/service.proto",
				"google/cloud/secretmanager/v1/resources.proto",
				commonResourcesProto,
			},
			apiPath:                "google/cloud/secretmanager/v1",
			includeCommonResources: false,
			wantProtos: []string{
				"google/cloud/secretmanager/v1/resources.proto",
				"google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name: "additional protos added",
			setupFiles: []string{
				"google/cloud/secretmanager/v1/service.proto",
				"google/cloud/secretmanager/v1/resources.proto",
				"google/cloud/location/locations.proto",
			},
			apiPath: "google/cloud/secretmanager/v1",
			additionalProtos: []string{
				"google/cloud/location/locations.proto",
			},
			wantProtos: []string{
				"google/cloud/secretmanager/v1/resources.proto",
				"google/cloud/secretmanager/v1/service.proto",
				"google/cloud/location/locations.proto",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			for _, file := range test.setupFiles {
				p := filepath.Join(tempDir, file)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := gatherGAPICProtos(tempDir, test.apiPath, test.additionalProtos, test.includeCommonResources)
			if err != nil {
				t.Fatal(err)
			}
			var want []string
			for _, file := range test.wantProtos {
				want = append(want, filepath.Join(tempDir, file))
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGatherGAPICProtos_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		setupFiles []string
		apiPath    string
		wantErr    error
	}{
		{
			name:       "directory not found",
			setupFiles: nil,
			apiPath:    "google/cloud/secretmanager/v1",
			wantErr:    errNoProtos,
		},
		{
			name:       "empty directory",
			setupFiles: []string{"google/cloud/secretmanager/v1/not-a-proto.txt"},
			apiPath:    "google/cloud/secretmanager/v1",
			wantErr:    errNoProtos,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			for _, file := range test.setupFiles {
				p := filepath.Join(tempDir, file)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := gatherGAPICProtos(tempDir, test.apiPath, nil, true)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("gatherGAPICProtos() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestBuildGapicProtocArgs(t *testing.T) {
	tempDir := t.TempDir()
	src := &sources.Sources{
		Googleapis: tempDir,
	}
	srcCfg := sources.NewSourceConfig(src, []string{"googleapis"})
	params := &generateAPIParams{
		srcCfg:      srcCfg,
		wrapperPath: "/path/to/wrapper.sh",
	}
	opts := []string{"metadata", "generate-snippets"}
	targetProtos := []string{"/path/to/proto1.proto", "/path/to/proto2.proto"}
	got := buildGapicProtocArgs(params, "/path/to/output.zip", opts, targetProtos)
	want := []string{
		"--experimental_allow_proto3_optional",
		"--plugin=protoc-gen-gapic=/path/to/wrapper.sh",
		"--gapic_out=metadata,generate-snippets:/path/to/output.zip",
		"-I", tempDir,
		"/path/to/proto1.proto",
		"/path/to/proto2.proto",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildProtoProtocArgs(t *testing.T) {
	tempDir := t.TempDir()
	src := &sources.Sources{
		Googleapis: tempDir,
	}
	srcCfg := sources.NewSourceConfig(src, []string{"googleapis"})
	params := &generateAPIParams{
		srcCfg: srcCfg,
	}
	targetProtos := []string{"/path/to/proto1.proto", "/path/to/proto2.proto"}
	got := buildProtoProtocArgs(params, "/path/to/proto.zip", targetProtos)
	want := []string{
		"--experimental_allow_proto3_optional",
		"--php_out=/path/to/proto.zip",
		"-I", tempDir,
		"/path/to/proto1.proto",
		"/path/to/proto2.proto",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name          string
		libName       string
		defaultOutput string
		want          string
	}{
		{
			name:          "standard",
			libName:       "Ces",
			defaultOutput: "packages",
			want:          "packages/Ces",
		},
		{
			name:          "empty default",
			libName:       "Ces",
			defaultOutput: "",
			want:          "Ces",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.libName, test.defaultOutput)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
