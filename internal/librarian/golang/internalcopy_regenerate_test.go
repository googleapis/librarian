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
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerateLibrary_InternalCopy(t *testing.T) {
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
					Plugins: []*config.GoProtocPlugin{{
						Name: "go-vtproto",
						Options: []string{
							"features=marshal+unmarshal+size+pool",
							"pool=cloud.google.com/go/secretmanager/internal/fastpb.Secret",
						},
					}},
				}},
			},
		}},
		Output: filepath.Join(repoRoot, "secretmanager"),
	}
	// Generation is always followed by formatting, so the files a later run
	// finds in the repository are the formatted ones.
	if err := Generate(t.Context(), nil, library, &sources.Sources{Googleapis: googleapisDir}); err != nil {
		t.Fatal(err)
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}
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
				if !strings.Contains(string(content), want) {
					t.Errorf("want %s to contain %q", test.path, want)
				}
			}
			for _, missing := range test.wantMissing {
				if strings.Contains(string(content), missing) {
					t.Errorf("want %s to not contain %q", test.path, missing)
				}
			}
		})
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "secretmanager", "internal", "fastpb", "service_grpc.pb.go")); err == nil {
		t.Error("internal copy should not contain gRPC code")
	}
	t.Run("twin import", func(t *testing.T) {
		twinDir := filepath.Join(library.Output, "internal", "twin")
		if err := os.MkdirAll(twinDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(twinDir, "twin_test.go"), []byte(twinImportTest), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := runInDirWithEnv(t.Context(), library.Output, nil, command.Go, "test", "-count=1", "./internal/twin"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("regenerate with keep", func(t *testing.T) {
		plugin := library.APIs[0].Go.InternalCopies[0].Plugins[0]
		plugin.Options = []string{strings.Join(plugin.Options, ",")}
		kept := filepath.Join(library.Output, "internal", "fastpb", "resources.pb.go")
		content, err := os.ReadFile(kept)
		if err != nil {
			t.Fatal(err)
		}
		// Change the kept file in a way goimports leaves alone, so that its
		// survival is distinguishable from regeneration.
		content = append(content, []byte("\n// kept by the test\n")...)
		if err := os.WriteFile(kept, content, 0o644); err != nil {
			t.Fatal(err)
		}
		stale := filepath.Join(library.Output, "internal", "fastpb", "stale_vtproto.pb.go")
		if err := os.WriteFile(stale, []byte("package fastpb\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		handwritten := []string{"doc.go", "helpers.go", "auxiliary.go", "operations.go", "example_client.go", "handwritten.go", "a_custom.go"}
		for _, file := range handwritten {
			if err := os.WriteFile(filepath.Join(library.Output, "internal", "fastpb", file), []byte("// Handwritten.\npackage fastpb\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		checkHandwritten := func(t *testing.T) {
			t.Helper()
			for _, file := range handwritten {
				got, err := os.ReadFile(filepath.Join(library.Output, "internal", "fastpb", file))
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff("// Handwritten.\npackage fastpb\n", string(got)); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		}
		library.Keep = []string{"internal/fastpb/resources.pb.go"}
		if err := Clean(library); err != nil {
			t.Fatal(err)
		}
		checkHandwritten(t)
		if err := Generate(t.Context(), nil, library, &sources.Sources{Googleapis: googleapisDir}); err != nil {
			t.Fatal(err)
		}
		checkHandwritten(t)
		if err := Format(t.Context(), library); err != nil {
			t.Fatal(err)
		}
		checkHandwritten(t)
		got, err := os.ReadFile(kept)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(content) {
			t.Error("kept file was regenerated")
		}
		if _, err := os.Stat(stale); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("stale file: Stat error = %v, want not exist", err)
		}
		for _, file := range []string{"resources_vtproto.pb.go", "service.pb.go", "service_vtproto.pb.go"} {
			if _, err := os.Stat(filepath.Join(library.Output, "internal", "fastpb", file)); err != nil {
				t.Error(err)
			}
		}
	})
}

// twinImportTest is compiled inside the generated module: it links the public
// package and the internal copy into one test binary and checks that both
// register, that only the copy carries vtproto methods and that the two
// agree on the wire.
const twinImportTest = `package twin

import (
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/secretmanager/internal/fastpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type vtMarshaler interface {
	MarshalVT() ([]byte, error)
}

func TestTwin(t *testing.T) {
	for _, path := range []string{
		"google/cloud/secretmanager/v1/resources.proto",
		"google/cloud/secretmanager/v1/fastinternal/resources.proto",
	} {
		if _, err := protoregistry.GlobalFiles.FindFileByPath(path); err != nil {
			t.Errorf("FindFileByPath(%q) = %v", path, err)
		}
	}
	private := fastpb.SecretFromVTPool()
	defer private.ReturnToVTPool()
	if got, want := string(private.ProtoReflect().Descriptor().FullName()), "google.cloud.secretmanager.v1.fastinternal.Secret"; got != want {
		t.Errorf("private full name = %q, want %q", got, want)
	}
	public := &secretmanagerpb.Secret{Name: "projects/p/secrets/s"}
	if got, want := string(public.ProtoReflect().Descriptor().FullName()), "google.cloud.secretmanager.v1.Secret"; got != want {
		t.Errorf("public full name = %q, want %q", got, want)
	}
	if _, ok := any(public).(vtMarshaler); ok {
		t.Error("public Secret has vtproto methods")
	}
	wire, err := proto.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	if err := private.UnmarshalVT(wire); err != nil {
		t.Fatal(err)
	}
	if private.GetName() != public.GetName() {
		t.Errorf("private name = %q, want %q", private.GetName(), public.GetName())
	}
	back, err := private.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	got := &secretmanagerpb.Secret{}
	if err := proto.Unmarshal(back, got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(public, got) {
		t.Errorf("round trip = %v, want %v", got, public)
	}
}
`
