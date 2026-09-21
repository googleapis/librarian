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
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestMoveInternalCopy(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	generated := filepath.Join(srcDir, "cloud.google.com", "go", "lib", "internal", "fastpb")
	createFiles(t, generated, []string{"a.pb.go", "a_vtproto.pb.go"})
	outDir := filepath.Join(root, "repo", "lib")
	library := &config.Library{Name: "lib"}
	cp := &config.GoInternalCopy{ImportPath: "lib/internal/fastpb"}
	if err := moveInternalCopy(library, cp, srcDir, outDir); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(outDir, "internal", "fastpb")
	got := getFilesInDir(t, dest, dest)
	slices.Sort(got)
	if diff := cmp.Diff([]string{"a.pb.go", "a_vtproto.pb.go"}, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if got := getFilesInDir(t, generated, generated); len(got) != 0 {
		t.Errorf("generated files left behind: %v", got)
	}
}

func TestProtocPluginPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping executable lookup test on Windows")
	}
	binDir := t.TempDir()
	pathDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, binDir)
	t.Setenv("PATH", pathDir)
	if err := os.MkdirAll(filepath.Join(binDir, toolsDir), 0o755); err != nil {
		t.Fatal(err)
	}
	toolPlugin := filepath.Join(binDir, toolsDir, "protoc-gen-tool")
	testhelper.WriteExecutable(t, toolPlugin, "#!/bin/sh\nexit 0\n")
	pathPlugin := filepath.Join(pathDir, "protoc-gen-path")
	testhelper.WriteExecutable(t, pathPlugin, "#!/bin/sh\nexit 0\n")
	for _, test := range []struct {
		name   string
		plugin string
		want   string
	}{
		{"go tool bin dir", "tool", toolPlugin},
		{"PATH fallback", "path", pathPlugin},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := protocPluginPath(test.plugin)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtocPluginPath_Error(t *testing.T) {
	t.Setenv(cache.EnvLibrarianBin, t.TempDir())
	t.Setenv("PATH", t.TempDir())
	_, gotErr := protocPluginPath("does-not-exist")
	if !errors.Is(gotErr, errProtocPluginNotFound) {
		t.Errorf("protocPluginPath error = %v, wantErr %v", gotErr, errProtocPluginNotFound)
	}
}

func TestGenerateInternalCopies(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-vtproto")
	for _, test := range []struct {
		name          string
		protoAPILevel string
	}{
		{name: "default api level"},
		{name: "open api level", protoAPILevel: "API_OPEN"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			outDir := filepath.Join(root, "secretmanager")
			goAPI := &config.GoAPI{
				ClientPackage:  "secretmanager",
				ImportPath:     "secretmanager/apiv1",
				ProtoAPILevel:  test.protoAPILevel,
				InternalCopies: []*config.GoInternalCopy{{ImportPath: "secretmanager/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.secretmanager.v1.fastinternal"}},
			}
			library := &config.Library{
				Name:   "secretmanager",
				APIs:   []*config.API{{Path: "google/cloud/secretmanager/v1", Go: goAPI}},
				Output: outDir,
			}
			if err := generateInternalCopies(t.Context(), "google/cloud/secretmanager/v1", goAPI, library, nil, googleapisDir, t.TempDir(), outDir); err != nil {
				t.Fatal(err)
			}
			got := getFilesInDir(t, outDir, outDir)
			slices.Sort(got)
			want := []string{
				filepath.FromSlash("internal/fastpb/resources.pb.go"),
				filepath.FromSlash("internal/fastpb/resources_vtproto.pb.go"),
				filepath.FromSlash("internal/fastpb/service.pb.go"),
				filepath.FromSlash("internal/fastpb/service_vtproto.pb.go"),
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateInternalCopies_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		requires []string
		apiPath  string
		copy     *config.GoInternalCopy
		wantErr  error
	}{
		{
			name:    "api directory not found",
			apiPath: "google/cloud/missing/v1",
			copy:    &config.GoInternalCopy{ImportPath: "missing/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.missing.v1.fastinternal"},
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "plugin not installed",
			apiPath: "google/cloud/secretmanager/v1",
			copy:    &config.GoInternalCopy{ImportPath: "secretmanager/internal/fastpb", Plugin: "does-not-exist", ProtoPackage: "google.cloud.secretmanager.v1.fastinternal"},
			wantErr: errProtocPluginNotFound,
		},
		{
			name:     "extension of a message outside the copy",
			requires: []string{"protoc", "protoc-gen-go", "protoc-gen-go-vtproto"},
			apiPath:  "google/cloud/customoption/v1",
			copy:     &config.GoInternalCopy{ImportPath: "customoption/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.customoption.v1.fastinternal"},
			wantErr:  errInternalCopyExtension,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, cmd := range test.requires {
				testhelper.RequireCommand(t, cmd)
			}
			outDir := filepath.Join(t.TempDir(), "lib")
			goAPI := &config.GoAPI{ImportPath: "lib/apiv1", InternalCopies: []*config.GoInternalCopy{test.copy}}
			library := &config.Library{Name: "lib", APIs: []*config.API{{Path: test.apiPath, Go: goAPI}}, Output: outDir}
			gotErr := generateInternalCopies(t.Context(), test.apiPath, goAPI, library, nil, googleapisDir, t.TempDir(), outDir)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("generateInternalCopies error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestAPIDescriptorSet_Error(t *testing.T) {
	t.Setenv(cache.EnvLibrarianCache, t.TempDir())
	notInstalled := &config.Protoc{Version: "0.0.0"}
	if _, err := apiDescriptorSet(t.Context(), notInstalled, googleapisDir, []string{"google/cloud/secretmanager/v1/service.proto"}, t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("apiDescriptorSet error = %v, wantErr %v", err, fs.ErrNotExist)
	}
}

func TestCollectAPIDirProtoFiles(t *testing.T) {
	got, err := collectAPIDirProtoFiles(googleapisDir, "google/cloud/secretmanager/v1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"google/cloud/secretmanager/v1/resources.proto", "google/cloud/secretmanager/v1/service.proto"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCollectAPIDirProtoFiles_Error(t *testing.T) {
	root := t.TempDir()
	createFiles(t, root, []string{"google/empty/v1/README.md", "google/empty/v1/nested/nested.proto"})
	for _, test := range []struct {
		name    string
		apiPath string
		wantErr error
	}{
		{name: "api directory not found", apiPath: "google/missing/v1", wantErr: fs.ErrNotExist},
		{name: "no proto files", apiPath: "google/empty/v1", wantErr: errAPIDirNoProtoFiles},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := collectAPIDirProtoFiles(root, test.apiPath); !errors.Is(err, test.wantErr) {
				t.Errorf("collectAPIDirProtoFiles error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// generateLibraryWithInternalCopy generates the secretmanager library with one
// internal copy into a fresh repository root and formats it, as librarian does
// after every generation.
func generateLibraryWithInternalCopy(t *testing.T) (string, *config.Library) {
	t.Helper()
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-grpc")
	testhelper.RequireCommand(t, "protoc-gen-go_gapic")
	testhelper.RequireCommand(t, "protoc-gen-go-vtproto")
	testhelper.RequireCommand(t, "goimports")
	repoRoot := t.TempDir()
	setupSnippets(t, repoRoot)
	library := &config.Library{
		Name: "secretmanager",
		APIs: []*config.API{{
			Path: "google/cloud/secretmanager/v1",
			Go: &config.GoAPI{
				ClientPackage: "secretmanager",
				ImportPath:    "secretmanager/apiv1",
				InternalCopies: []*config.GoInternalCopy{{
					ImportPath:   "secretmanager/internal/fastpb",
					ProtoPackage: "google.cloud.secretmanager.v1.fastinternal",
					Plugin:       "go-vtproto",
					PluginOptions: []string{
						"features=marshal+unmarshal+size+pool",
						"pool=cloud.google.com/go/secretmanager/internal/fastpb.Secret",
					},
				}},
			},
		}},
		Output: filepath.Join(repoRoot, "secretmanager"),
	}
	if err := Generate(t.Context(), nil, library, &sources.Sources{Googleapis: googleapisDir}); err != nil {
		t.Fatal(err)
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}
	return repoRoot, library
}

func TestGenerateLibrary_InternalCopy(t *testing.T) {
	repoRoot, _ := generateLibraryWithInternalCopy(t)
	for _, test := range []struct {
		path         string
		wantContains []string
		wantMissing  []string
	}{
		{
			path: "secretmanager/internal/fastpb/service.pb.go",
			wantContains: []string{
				"package fastpb",
				"// source: google/cloud/secretmanager/v1/fastinternal/service.proto",
			},
			wantMissing: []string{"SecretManagerServiceClient", "google.golang.org/grpc"},
		},
		{
			path:         "secretmanager/internal/fastpb/resources.pb.go",
			wantContains: []string{"package fastpb"},
		},
		{
			path:         "secretmanager/internal/fastpb/service_vtproto.pb.go",
			wantContains: []string{"func (m *GetSecretRequest) UnmarshalVT("},
		},
		{
			path:         "secretmanager/internal/fastpb/resources_vtproto.pb.go",
			wantContains: []string{"func SecretFromVTPool() *Secret"},
		},
		{
			path:         "secretmanager/apiv1/secretmanagerpb/service.pb.go",
			wantContains: []string{"package secretmanagerpb", "// source: google/cloud/secretmanager/v1/service.proto"},
		},
		{
			path:         "secretmanager/apiv1/secret_manager_client.go",
			wantContains: []string{"package secretmanager"},
		},
	} {
		t.Run(test.path, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, test.path))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.wantContains {
				if !bytes.Contains(content, []byte(want)) {
					t.Errorf("want %s to contain %q", test.path, want)
				}
			}
			for _, missing := range test.wantMissing {
				if bytes.Contains(content, []byte(missing)) {
					t.Errorf("want %s to not contain %q", test.path, missing)
				}
			}
		})
	}
}

// TestGenerateLibrary_InternalCopy_Files checks the copy directory holds the
// message code and the plugin output only: services are dropped from the copy,
// so there is no gRPC file.
func TestGenerateLibrary_InternalCopy_Files(t *testing.T) {
	_, library := generateLibraryWithInternalCopy(t)
	copyDir := filepath.Join(library.Output, "internal", "fastpb")
	got := getFilesInDir(t, copyDir, copyDir)
	slices.Sort(got)
	want := []string{"resources.pb.go", "resources_vtproto.pb.go", "service.pb.go", "service_vtproto.pb.go"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
