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

package swift

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerateModule(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	library := &config.Library{
		Name:          "GoogleTypeModule",
		CopyrightYear: "2038",
		Swift:         defaultSwiftConfig(t),
		Output:        outDir,
	}
	library.Swift.Modules = []*config.SwiftModule{
		{
			APIPath: "google/type",
			Output:  filepath.Join(outDir, "ProtoJSON"),
		},
		{
			APIPath:    "google/type",
			Output:     filepath.Join(outDir, "ProtoJSONDefault"),
			ModuleType: "default",
		},
	}
	src := &sources.Sources{
		Googleapis: googleapisDir,
	}
	cfg := &config.Config{}

	if err := Generate(t.Context(), cfg, library, src); err != nil {
		t.Fatal(err)
	}

	expectedFile := filepath.Join(outDir, "ProtoJSON", "Expr.swift")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Error(err)
	}

	expectedDefaultFile := filepath.Join(outDir, "ProtoJSONDefault", "Expr.swift")
	if _, err := os.Stat(expectedDefaultFile); err != nil {
		t.Error(err)
	}
}

func TestGenerateModule_SwiftProtobuf(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-swift")
	testhelper.RequireCommand(t, "protoc-gen-grpc-swift")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	library := &config.Library{
		Name:          "GoogleTypeModule",
		CopyrightYear: "2038",
		Swift:         defaultSwiftConfig(t),
		Output:        outDir,
	}
	library.Swift.Modules = []*config.SwiftModule{
		{
			APIPath:    "google/type",
			Output:     filepath.Join(outDir, "ProtoJSON"),
			ModuleType: "swift-protobuf",
		},
	}
	src := &sources.Sources{
		Googleapis: googleapisDir,
	}
	cfg := &config.Config{}

	if err := Generate(t.Context(), cfg, library, src); err != nil {
		t.Fatal(err)
	}

	expectedFile := filepath.Join(outDir, "ProtoJSON", "google", "type", "expr.pb.swift")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Error(err)
	}
}

func TestModuleToModelConfig(t *testing.T) {
	src := &sources.Sources{}
	for _, test := range []struct {
		name   string
		lib    *config.Library
		module *config.SwiftModule
		want   *parser.ModelConfig
	}{
		{
			name: "no include list",
			lib: &config.Library{
				Swift: &config.SwiftPackage{},
			},
			module: &config.SwiftModule{APIPath: "foo"},
			want: &parser.ModelConfig{
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "foo",
				Source: &sources.SourceConfig{
					Sources:     &sources.Sources{},
					ActiveRoots: []string{"googleapis"},
				},
				Codec: map[string]string{
					"copyright-year": "",
					"module":         "true",
				},
			},
		},
		{
			name: "with include list",
			lib: &config.Library{
				Swift: &config.SwiftPackage{
					IncludeList: []string{"a.proto", "b.proto"},
				},
			},
			module: &config.SwiftModule{APIPath: "foo"},
			want: &parser.ModelConfig{
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "foo",
				Source: &sources.SourceConfig{
					Sources:     &sources.Sources{},
					ActiveRoots: []string{"googleapis"},
					IncludeList: []string{"a.proto", "b.proto"},
				},
				Codec: map[string]string{
					"copyright-year": "",
					"module":         "true",
				},
			},
		},
		{
			name:   "nil swift",
			lib:    &config.Library{},
			module: &config.SwiftModule{APIPath: "foo"},
			want: &parser.ModelConfig{
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "foo",
				Source: &sources.SourceConfig{
					Sources:     &sources.Sources{},
					ActiveRoots: []string{"googleapis"},
				},
				Codec: map[string]string{
					"copyright-year": "",
					"module":         "true",
				},
			},
		},
		{
			name: "with copyright year",
			lib: &config.Library{
				CopyrightYear: "2038",
				Swift:         &config.SwiftPackage{},
			},
			module: &config.SwiftModule{APIPath: "foo"},
			want: &parser.ModelConfig{
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "foo",
				Source: &sources.SourceConfig{
					Sources:     &sources.Sources{},
					ActiveRoots: []string{"googleapis"},
				},
				Codec: map[string]string{
					"copyright-year": "2038",
					"module":         "true",
				},
			},
		},
		{
			name: "discovery",
			lib: &config.Library{
				CopyrightYear:       "2038",
				Swift:               &config.SwiftPackage{},
				SpecificationFormat: config.SpecDiscovery,
				Roots:               []string{"discovery"},
			},
			module: &config.SwiftModule{APIPath: "dir/foo.v1.json"},
			want: &parser.ModelConfig{
				SpecificationFormat: config.SpecDiscovery,
				SpecificationSource: "dir/foo.v1.json",
				Source: &sources.SourceConfig{
					Sources:     &sources.Sources{},
					ActiveRoots: []string{"discovery"},
				},
				Codec: map[string]string{
					"copyright-year": "2038",
					"module":         "true",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := moduleToModelConfig(test.lib, test.module, src)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateModule_UnsupportedModuleType(t *testing.T) {
	library := &config.Library{
		Name:          "UnsupportedModule",
		CopyrightYear: "2038",
		Swift:         defaultSwiftConfig(t),
		Output:        t.TempDir(),
	}
	library.Swift.Modules = []*config.SwiftModule{
		{
			APIPath:    "google/type",
			Output:     filepath.Join(library.Output, "ProtoJSON"),
			ModuleType: "unsupported",
		},
	}
	src := &sources.Sources{}
	cfg := &config.Config{}

	err := Generate(t.Context(), cfg, library, src)
	if err == nil {
		t.Fatal("Generate did not return an error for unsupported module type 'unsupported'")
	}
	expectedErr := `unknown module type "unsupported"`
	if err.Error() != expectedErr {
		t.Errorf("got error %q, want %q", err.Error(), expectedErr)
	}
}

func TestGenerateModule_NoProtos(t *testing.T) {
	library := &config.Library{
		Name:          "NoProtosModule",
		CopyrightYear: "2038",
		Swift:         defaultSwiftConfig(t),
		Output:        t.TempDir(),
	}
	googleapisDir := t.TempDir()
	emptyAPIPath := "google/empty"
	if err := os.MkdirAll(filepath.Join(googleapisDir, emptyAPIPath), 0o755); err != nil {
		t.Fatal(err)
	}

	library.Swift.Modules = []*config.SwiftModule{
		{
			APIPath:    emptyAPIPath,
			Output:     filepath.Join(library.Output, "ProtoJSON"),
			ModuleType: "swift-protobuf",
		},
	}
	src := &sources.Sources{
		Googleapis: googleapisDir,
	}
	cfg := &config.Config{}

	err := Generate(t.Context(), cfg, library, src)
	if err == nil {
		t.Fatal("Generate did not return an error when no proto files were found")
	}
	if !strings.Contains(err.Error(), "no proto files found in") {
		t.Errorf("got error %q, want it to contain 'no proto files found in'", err.Error())
	}
}
