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

package gcloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

const testGoogleapisDir = "../../testdata/googleapis"

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")

	for _, test := range []struct {
		name    string
		library *config.Library
	}{
		{
			name: "parallelstore",
			library: &config.Library{
				Name: "parallelstore",
				APIs: []*config.API{{Path: "google/cloud/parallelstore/v1"}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := t.TempDir()
			test.library.Output = out
			if err := Generate(t.Context(), test.library,
				&sources.Sources{Googleapis: testGoogleapisDir}); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"main.go", "README.md"} {
				if _, err := os.Stat(filepath.Join(out, name)); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestGenerate_Errors(t *testing.T) {
	for _, test := range []struct {
		name       string
		googleapis string
		apiPath    string
	}{
		{
			name:       "missing googleapis dir",
			googleapis: "nonexistent_googleapis_dir",
			apiPath:    "google/cloud/parallelstore/v1",
		},
		{
			name:       "missing api path",
			googleapis: testGoogleapisDir,
			apiPath:    "google/cloud/does/not/exist/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			library := &config.Library{
				Name:   "parallelstore",
				Output: t.TempDir(),
				APIs:   []*config.API{{Path: test.apiPath}},
			}
			if err := Generate(t.Context(), library,
				&sources.Sources{Googleapis: test.googleapis}); err == nil {
				t.Error("Generate() error = nil, want error")
			}
		})
	}
}
