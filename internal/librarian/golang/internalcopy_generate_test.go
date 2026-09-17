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
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	dest := filepath.Join(outDir, "internal", "fastpb")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	// a.pb.go is kept, so the existing file must survive; a_vtproto.pb.go is
	// not, so an existing file is replaced by the generated one.
	if err := os.WriteFile(filepath.Join(dest, "a.pb.go"), []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "a_vtproto.pb.go"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	library := &config.Library{Name: "lib", Keep: []string{"internal/fastpb/a.pb.go"}}
	cp := &config.GoInternalCopy{ImportPath: "lib/internal/fastpb"}
	if err := moveInternalCopy(library, cp, srcDir, outDir); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]string{"a.pb.go": "kept", "a_vtproto.pb.go": "test"} {
		got, err := os.ReadFile(filepath.Join(dest, file))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s: got %q, want %q", file, got, want)
		}
	}
	if got := getFilesInDir(t, generated, generated); len(got) != 0 {
		t.Errorf("generated files left behind: %v", got)
	}
}

func TestMoveInternalCopy_Error(t *testing.T) {
	root := t.TempDir()
	library := &config.Library{Name: "lib"}
	cp := &config.GoInternalCopy{ImportPath: "other/internal/fastpb"}
	gotErr := moveInternalCopy(library, cp, filepath.Join(root, "src"), filepath.Join(root, "repo", "lib"))
	if !errors.Is(gotErr, errInternalCopyOutsideLibrary) {
		t.Errorf("moveInternalCopy error = %v, wantErr %v", gotErr, errInternalCopyOutsideLibrary)
	}
	if got := getFilesInDir(t, root, root); len(got) != 0 {
		t.Errorf("files written: %v", got)
	}
}

// TestGenerate_InternalCopyRejected checks that a bad copy configuration is
// rejected before anything is generated or removed.
func TestGenerate_InternalCopyRejected(t *testing.T) {
	root := t.TempDir()
	createFiles(t, root, []string{"secretmanager/handwritten.go", "other/valuable.pb.go"})
	library := &config.Library{
		Name: "secretmanager",
		APIs: []*config.API{{
			Path: "google/cloud/secretmanager/v1",
			Go: &config.GoAPI{
				ClientPackage:  "secretmanager",
				ImportPath:     "secretmanager/apiv1",
				InternalCopies: []*config.GoInternalCopy{{ImportPath: "other/internal/fastpb", ProtoPackage: "google.cloud.secretmanager.v1.fastinternal"}},
			},
		}},
		Output: filepath.Join(root, "secretmanager"),
	}
	gotErr := Generate(t.Context(), nil, library, &sources.Sources{Googleapis: googleapisDir})
	if !errors.Is(gotErr, errInternalCopyOutsideLibrary) {
		t.Fatalf("Generate error = %v, wantErr %v", gotErr, errInternalCopyOutsideLibrary)
	}
	got := getFilesInDir(t, root, root)
	slices.Sort(got)
	want := []string{filepath.FromSlash("other/valuable.pb.go"), filepath.FromSlash("secretmanager/handwritten.go")}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("files after rejected generation (-want +got):\n%s", diff)
	}
}

func TestGenerateInternalCopies_ConfigError(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		copies  []*config.GoInternalCopy
		wantErr error
	}{
		{
			name:    "no copies",
			apiPath: "google/cloud/secretmanager/v1",
		},
		{
			name:    "null copy",
			apiPath: "google/cloud/secretmanager/v1",
			copies:  []*config.GoInternalCopy{nil},
			wantErr: errInternalCopyNil,
		},
		{
			name:    "api directory not found",
			apiPath: "google/cloud/missing/v1",
			copies:  []*config.GoInternalCopy{{ImportPath: "secretmanager/internal/fastpb", ProtoPackage: "google.cloud.missing.v1.fastinternal"}},
			wantErr: fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			outDir := filepath.Join(root, "secretmanager")
			goAPI := &config.GoAPI{ClientPackage: "secretmanager", ImportPath: "secretmanager/apiv1", InternalCopies: test.copies}
			library := &config.Library{Name: "secretmanager", APIs: []*config.API{{Path: test.apiPath, Go: goAPI}}, Output: outDir}
			gotErr := generateInternalCopies(t.Context(), test.apiPath, goAPI, library, nil, googleapisDir, apiDescriptorSet(t, test.apiPath), t.TempDir(), outDir)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("generateInternalCopies error = %v, wantErr %v", gotErr, test.wantErr)
			}
			if got := getFilesInDir(t, root, root); len(got) != 0 {
				t.Errorf("files written: %v", got)
			}
		})
	}
}

func TestGenerateInternalCopies(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
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
				InternalCopies: []*config.GoInternalCopy{{ImportPath: "secretmanager/internal/fastpb", ProtoPackage: "google.cloud.secretmanager.v1.fastinternal"}},
			}
			library := &config.Library{
				Name:   "secretmanager",
				APIs:   []*config.API{{Path: "google/cloud/secretmanager/v1", Go: goAPI}},
				Output: outDir,
			}
			if err := generateInternalCopies(t.Context(), "google/cloud/secretmanager/v1", goAPI, library, nil, googleapisDir, apiDescriptorSet(t, "google/cloud/secretmanager/v1"), t.TempDir(), outDir); err != nil {
				t.Fatal(err)
			}
			got := getFilesInDir(t, outDir, outDir)
			slices.Sort(got)
			want := []string{filepath.FromSlash("internal/fastpb/resources.pb.go"), filepath.FromSlash("internal/fastpb/service.pb.go")}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("generated files (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateInternalCopies_Error(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	for _, test := range []struct {
		name       string
		apiPath    string
		importPath string
		wantErr    error
		wantMsg    []string
	}{
		{
			name:       "extension of a message outside the copy",
			apiPath:    "google/cloud/customoption/v1",
			importPath: "customoption/apiv1",
			wantErr:    errInternalCopyExtension,
			wantMsg:    []string{"google/cloud/customoption/v1/custom_option.proto", "extension google.cloud.customoption.v1.note", "of google.protobuf.MessageOptions"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			name, _, _ := strings.Cut(test.importPath, "/")
			outDir := filepath.Join(root, name)
			goAPI := &config.GoAPI{
				ClientPackage:  name,
				ImportPath:     test.importPath,
				InternalCopies: []*config.GoInternalCopy{{ImportPath: name + "/internal/fastpb", ProtoPackage: strings.ReplaceAll(test.apiPath, "/", ".") + ".fastinternal"}},
			}
			library := &config.Library{
				Name:   name,
				APIs:   []*config.API{{Path: test.apiPath, Go: goAPI}},
				Output: outDir,
			}
			gotErr := generateInternalCopies(t.Context(), test.apiPath, goAPI, library, nil, googleapisDir, apiDescriptorSet(t, test.apiPath), t.TempDir(), outDir)
			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("generateInternalCopies error = %v, wantErr %v", gotErr, test.wantErr)
			}
			for _, want := range test.wantMsg {
				if !strings.Contains(gotErr.Error(), want) {
					t.Errorf("error %q does not mention %q", gotErr, want)
				}
			}
			if got := getFilesInDir(t, root, root); len(got) != 0 {
				t.Errorf("files written despite the error: %v", got)
			}
		})
	}
}

func TestGenerateInternalCopies_LayoutOption(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-go")
	testhelper.RequireCommand(t, "protoc-gen-go-vtproto")
	for _, option := range []string{
		"features=marshal+unmarshal+size,paths=source_relative",
		"features=marshal,module=cloud.google.com/go,allow-empty=true",
		"Mgoogle/cloud/secretmanager/v1/fastinternal/resources.proto=example.com/elsewhere,features=marshal",
	} {
		t.Run(option, func(t *testing.T) {
			root := t.TempDir()
			apiPath := "google/cloud/secretmanager/v1"
			goAPI := &config.GoAPI{
				ImportPath: "secretmanager/apiv1",
				InternalCopies: []*config.GoInternalCopy{{
					ImportPath:   "secretmanager/internal/fastpb",
					ProtoPackage: "google.cloud.secretmanager.v1.fastinternal",
					Plugins:      []*config.GoProtocPlugin{{Name: "go-vtproto", Options: []string{option}}},
				}},
			}
			library := &config.Library{
				Name: "secretmanager", Output: filepath.Join(root, "secretmanager"),
				APIs: []*config.API{{Path: apiPath, Go: goAPI}},
			}
			if err := generateInternalCopies(t.Context(), apiPath, goAPI, library, nil, googleapisDir, apiDescriptorSet(t, apiPath), t.TempDir(), library.Output); !errors.Is(err, errProtocPluginOption) {
				t.Errorf("generateInternalCopies error = %v, wantErr %v", err, errProtocPluginOption)
			}
			if got := getFilesInDir(t, root, root); len(got) != 0 {
				t.Errorf("files written despite invalid layout: %v", got)
			}
		})
	}
}

// apiDescriptorSet writes the descriptor set of the API's proto files the way
// generateAPI does and returns its path. When the API directory is missing or
// protoc is not installed, the returned path does not exist; tests that need
// the set require protoc first.
func apiDescriptorSet(t *testing.T, apiPath string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "api.pb")
	if _, err := exec.LookPath("protoc"); err != nil {
		return path
	}
	entries, err := os.ReadDir(filepath.Join(googleapisDir, apiPath))
	if err != nil {
		return path
	}
	args := []string{"--experimental_allow_proto3_optional", "-I=" + googleapisDir, "--include_imports", "--include_source_info", "--descriptor_set_out=" + path}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".proto" {
			args = append(args, filepath.Join(googleapisDir, apiPath, entry.Name()))
		}
	}
	if err := runProtoc(t.Context(), nil, args...); err != nil {
		t.Fatal(err)
	}
	return path
}
