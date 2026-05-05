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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
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
}

func TestModuleToModelConfig(t *testing.T) {
	src := &sources.Sources{}
	for _, test := range []struct {
		name            string
		lib             *config.Library
		module          *config.SwiftModule
		wantIncludeList []string
		wantCodec       map[string]string
	}{
		{
			name: "no include list",
			lib: &config.Library{
				Swift: &config.SwiftPackage{},
			},
			module:          &config.SwiftModule{APIPath: "foo"},
			wantIncludeList: nil,
			wantCodec: map[string]string{
				"copyright-year": "",
				"module":         "true",
			},
		},
		{
			name: "with include list",
			lib: &config.Library{
				Swift: &config.SwiftPackage{
					IncludeList: []string{"a.proto", "b.proto"},
				},
			},
			module:          &config.SwiftModule{APIPath: "foo"},
			wantIncludeList: []string{"a.proto", "b.proto"},
			wantCodec: map[string]string{
				"copyright-year": "",
				"module":         "true",
			},
		},
		{
			name:            "nil swift",
			lib:             &config.Library{},
			module:          &config.SwiftModule{APIPath: "foo"},
			wantIncludeList: nil,
			wantCodec: map[string]string{
				"copyright-year": "",
				"module":         "true",
			},
		},
		{
			name: "with copyright year",
			lib: &config.Library{
				CopyrightYear: "2038",
				Swift:         &config.SwiftPackage{},
			},
			module:          &config.SwiftModule{APIPath: "foo"},
			wantIncludeList: nil,
			wantCodec: map[string]string{
				"copyright-year": "2038",
				"module":         "true",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := moduleToModelConfig(test.lib, test.module, src)
			if diff := cmp.Diff(test.wantIncludeList, got.Source.IncludeList); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantCodec, got.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
