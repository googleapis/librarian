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
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"sort"

	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

func TestResolveGAPICOptions(t *testing.T) {
	for _, test := range []struct {
		name     string
		cfg      *config.Config
		library  *config.Library
		api      *config.API
		apiCfgs  *serviceconfig.API
		expected []string
	}{
		{
			name:    "basic case",
			cfg:     &config.Config{Repo: "googleapis/google-cloud-java"},
			library: &config.Library{Name: "secretmanager"},
			api:     &config.API{Path: "google/cloud/secretmanager/v1"},
			apiCfgs: &serviceconfig.API{Transports: map[string]serviceconfig.Transport{
				config.LanguageJava: serviceconfig.GRPCRest,
			}},
			expected: []string{
				"metadata",
				"repo=googleapis/google-cloud-java",
				"artifact=com.google.cloud:google-cloud-secretmanager",
				"gapic-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_gapic.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=grpc+rest",
				"rest-numeric-enums",
			},
		},
		{
			name:    "rest transport",
			cfg:     &config.Config{Repo: "googleapis/google-cloud-java"},
			library: &config.Library{Name: "secretmanager"},
			api:     &config.API{Path: "google/cloud/secretmanager/v1"},
			apiCfgs: &serviceconfig.API{Transports: map[string]serviceconfig.Transport{
				config.LanguageJava: serviceconfig.Rest,
			}},
			expected: []string{
				"metadata",
				"repo=googleapis/google-cloud-java",
				"artifact=com.google.cloud:google-cloud-secretmanager",
				"gapic-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_gapic.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=rest",
				"rest-numeric-enums",
			},
		},
		{
			name:    "no rest numeric enum case",
			cfg:     &config.Config{Repo: "googleapis/google-cloud-java"},
			library: &config.Library{Name: "secretmanager"},
			api:     &config.API{Path: "google/cloud/secretmanager/v1"},
			apiCfgs: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageJava: serviceconfig.GRPCRest,
				},
				NoRESTNumericEnums: map[string]bool{
					config.LanguageJava: true,
				},
			},
			expected: []string{
				"metadata",
				"repo=googleapis/google-cloud-java",
				"artifact=com.google.cloud:google-cloud-secretmanager",
				"gapic-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_gapic.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=grpc+rest",
			},
		},
		{
			name:    "default transport with no apiCfgs",
			cfg:     &config.Config{Repo: "googleapis/google-cloud-java"},
			library: &config.Library{Name: "secretmanager"},
			api:     &config.API{Path: "google/cloud/secretmanager/v1"},
			apiCfgs: &serviceconfig.API{},
			expected: []string{
				"metadata",
				"repo=googleapis/google-cloud-java",
				"artifact=com.google.cloud:google-cloud-secretmanager",
				"gapic-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_gapic.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=grpc+rest",
				"rest-numeric-enums",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveGAPICOptions(test.cfg, test.library, test.api, googleapisDir, test.apiCfgs)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveDistributionName(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    string
	}{
		{
			name:    "default case",
			library: &config.Library{Name: "secretmanager"},
			want:    "com.google.cloud:google-cloud-secretmanager",
		},
		{
			name: "groupID override",
			library: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{GroupID: "com.custom"},
			},
			want: "com.custom:google-cloud-secretmanager",
		},
		{
			name: "distributionName override",
			library: &config.Library{
				Name: "secretmanager",
				Java: &config.JavaModule{DistributionNameOverride: "com.google.cloud:google-cloud-secretmanager-v1"},
			},
			want: "com.google.cloud:google-cloud-secretmanager-v1",
		},
		{
			name:    "library name already has prefix",
			library: &config.Library{Name: "google-cloud-secretmanager"},
			want:    "com.google.cloud:google-cloud-secretmanager",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveDistributionName(test.library)
			if got != test.want {
				t.Errorf("deriveDistributionName() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveGAPICOptions_MultipleConfigsError(t *testing.T) {
	testCases := []struct {
		name    string
		files   []string
		apiPath string
		wantErr string
	}{
		{
			name:    "multiple grpc configs",
			files:   []string{"a_grpc_service_config.json", "b_grpc_service_config.json"},
			apiPath: "google/cloud/multiple/v1",
			wantErr: "multiple gRPC service config files found",
		},
		{
			name:    "multiple gapic configs",
			files:   []string{"a_gapic.yaml", "b_gapic.yaml"},
			apiPath: "google/cloud/multiplegapic/v1",
			wantErr: "multiple GAPIC config files found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			apiDir := filepath.Join(tmpDir, tc.apiPath)
			if err := os.MkdirAll(apiDir, 0755); err != nil {
				t.Fatal(err)
			}
			for _, file := range tc.files {
				content := []byte("")
				if strings.HasSuffix(file, ".json") {
					content = []byte("{}")
				}
				if err := os.WriteFile(filepath.Join(apiDir, file), content, 0644); err != nil {
					t.Fatal(err)
				}
			}

			apiCfgs := &serviceconfig.API{Transports: map[string]serviceconfig.Transport{
				config.LanguageJava: serviceconfig.GRPC,
			}}
			_, err := resolveGAPICOptions(&config.Config{Repo: "test-repo"}, &config.Library{Name: "test"}, &config.API{Path: tc.apiPath}, tmpDir, apiCfgs)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("resolveGAPICOptions() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestProtoProtocArgs(t *testing.T) {
	apiProtos := []string{
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
	}
	got := protoProtocArgs(apiProtos, googleapisDir, "proto-out")
	want := []string{
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		"--java_out=proto-out",
		apiProtos[0],
		apiProtos[1],
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGrpcProtocArgs(t *testing.T) {
	apiProtos := []string{
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
	}
	got := grpcProtocArgs(apiProtos, googleapisDir, "grpc-out")
	want := []string{
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		"--java_grpc_out=grpc-out",
		apiProtos[0],
		apiProtos[1],
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGapicProtocArgs(t *testing.T) {
	apiProtos := []string{
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
		filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
	}
	additionalProtos := []string{
		filepath.Join(googleapisDir, "google/cloud/common_resources.proto"),
	}
	got := gapicProtocArgs(apiProtos, additionalProtos, googleapisDir, "gapic-out", []string{"opt1", "opt2"})
	want := []string{
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		"--java_gapic_out=metadata:gapic-out",
		"--java_gapic_opt=opt1,opt2",
		apiProtos[0],
		apiProtos[1],
		additionalProtos[0],
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
		{
			name: "Java module exists but API not found",
			library: &config.Library{
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path: "other/api",
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
	cfg := &config.Config{
		Repo: "googleapis/google-cloud-java",
		Default: &config.Default{
			Java: &config.JavaModule{},
		},
	}
	library := &config.Library{Name: "secretmanager", Output: outdir}
	// Create owlbot.py and templates dir as they are mandatory for postProcessAPI
	if err := os.WriteFile(filepath.Join(outdir, "owlbot.py"), []byte("#!/usr/bin/env python3\npass"), 0755); err != nil {
		t.Fatal(err)
	}
	templatesDir := filepath.Join(filepath.Dir(outdir), owlbotTemplatesRelPath)
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatal(err)
	}
	err := generateAPI(
		t.Context(),
		cfg,
		&config.API{Path: "google/cloud/secretmanager/v1"},
		library,
		googleapisDir,
		outdir,
		&repoMetadata{
			NamePretty:     "Secret Manager",
			APIDescription: "Secret Manager API",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	// Verify that the output was restructured.
	// Since postProcessAPI now calls restructureToStaging, we check in staging/v1/...
	restructuredPath := filepath.Join(outdir, "owl-bot-staging", "v1", "google-cloud-secretmanager", "src", "main", "java")
	if _, err := os.Stat(restructuredPath); err != nil {
		t.Errorf("expected restructured path %s to exist: %v", restructuredPath, err)
	}
}

func TestGenerateAPI_NoTools(t *testing.T) {
	// Temporarily mock runProtoc to avoid external tool requirements.
	oldRunProtoc := runProtoc
	defer func() { runProtoc = oldRunProtoc }()
	// Capture all calls to runProtoc to verify arguments without executing the command.
	var calls [][]string
	runProtoc = func(ctx context.Context, args []string) error {
		calls = append(calls, args)
		return nil
	}
	outdir := t.TempDir()
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	cfg := &config.Config{
		Repo: "googleapis/google-cloud-java",
		Default: &config.Default{
			Java: &config.JavaModule{},
		},
	}
	library := &config.Library{
		Name:   "secretmanager",
		Output: outdir,
		APIs: []*config.API{
			api,
		},
	}
	for _, artifact := range []string{"google-cloud-secretmanager", "proto-google-cloud-secretmanager-v1", "grpc-google-cloud-secretmanager-v1"} {
		if err := os.MkdirAll(filepath.Join(outdir, artifact), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Create owlbot.py and templates dir as they are  mandatory for postProcessAPI
	if err := os.WriteFile(filepath.Join(outdir, "owlbot.py"), []byte("#!/usr/bin/env python3\npass"), 0755); err != nil {
		t.Fatal(err)
	}
	templatesDir := filepath.Join(filepath.Dir(outdir), owlbotTemplatesRelPath)
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatal(err)
	}
	err := generateAPI(t.Context(), cfg, api, library, googleapisDir, outdir, &repoMetadata{
		NamePretty:     "Secret Manager",
		APIDescription: "Secret Manager API",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify that runProtoc was called 3 times: proto, grpc, and gapic.
	if len(calls) != 3 {
		t.Errorf("expected 3 calls to runProtoc, got %d", len(calls))
	}
	// Basic validation of GAPIC generation arguments (the 3rd call).
	gapicArgs := calls[2]
	foundGapicOut := false
	for _, arg := range gapicArgs {
		if strings.HasPrefix(arg, "--java_gapic_out=") {
			foundGapicOut = true
			break
		}
	}
	if !foundGapicOut {
		t.Errorf("expected --java_gapic_out in gapicArgs, but not found: %v", gapicArgs)
	}
}

func TestGenerateLibrary_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		setup   func(t *testing.T, library *config.Library)
		wantErr error
	}{
		{
			name: "invalid version",
			library: &config.Library{
				Name:   "test",
				Output: t.TempDir(),
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager"}, // Missing version
				},
			},
			wantErr: errExtractVersion,
		},
		{
			name: "no protos found",
			library: &config.Library{
				Name:   "test",
				Output: t.TempDir(),
				APIs: []*config.API{
					{Path: "google/cloud/nonexistent/v1"},
				},
			},
			wantErr: errNoProtos,
		},
		{
			name: "mkdir failure for output dir",
			library: &config.Library{
				Name:   "test",
				Output: filepath.Join(t.TempDir(), "file_exists"),
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			setup: func(t *testing.T, library *config.Library) {
				// Create a regular file where a directory is expected to cause os.MkdirAll to fail.
				if err := os.WriteFile(library.Output, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: syscall.ENOTDIR,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t, test.library)
			}
			cfg := &config.Config{Language: "java"}
			err := Generate(t.Context(), cfg, test.library, &sources.Sources{Googleapis: googleapisDir})
			if !errors.Is(err, test.wantErr) {
				t.Errorf("generate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
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
		t.Fatal(err)
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
		t.Fatal(err)
	}
	sort.Strings(got)
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
