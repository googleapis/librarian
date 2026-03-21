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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "goimports")
	for _, test := range []struct {
		name       string
		library    *config.Library
		goFilePath []string
	}{
		{
			name: "library path and snippet directory exist",
			library: &config.Library{
				Name: "example",
				APIs: []*config.API{
					{Path: "example/v1"},
				},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{Path: "example/v1", ImportPath: "example/apiv1"},
					},
				},
			},
			goFilePath: []string{
				"example",
				"internal/generated/snippets/example/apiv1",
			},
		},
		{
			name: "root module",
			library: &config.Library{
				Name: rootModule,
			},
		},
		{
			name: "proto only API",
			library: &config.Library{
				Name: "example",
				APIs: []*config.API{
					{Path: "example/common"},
				},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{Path: "example/common", ProtoOnly: true},
					},
				},
			},
			goFilePath: []string{
				"example",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			test.library.Output = filepath.Join(repoRoot, test.library.Name)
			unformatted := `package main

import (
"fmt"
"os"
)

func main() {
fmt.Println("Hello World")
}
`
			want := `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello World")
}
`
			for _, aPath := range test.goFilePath {
				path := filepath.Join(repoRoot, aPath)
				if err := os.MkdirAll(path, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, "example.go"), []byte(unformatted), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := Format(t.Context(), test.library); err != nil {
				t.Fatal(err)
			}
			for _, aPath := range test.goFilePath {
				goFile := filepath.Join(repoRoot, aPath, "example.go")
				gotBytes, err := os.ReadFile(goFile)
				if err != nil {
					t.Fatal(err)
				}
				got := string(gotBytes)
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBuildFormatArgs(t *testing.T) {
	for _, test := range []struct {
		name     string
		goModule *config.GoModule
		want     []string
	}{
		{
			name: "library with a GAPIC API",
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{Path: "example/v1", ImportPath: "example/apiv1"},
				},
			},
			want: []string{"-w", "repo/example", "repo/internal/generated/snippets/example/apiv1"},
		},
		{
			name: "library with a proto only API",
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{
					{Path: "example/v1", ProtoOnly: true},
				},
			},
			want: []string{"-w", "repo/example"},
		},
		{
			name: "snippet directory is one of the deleted path after generation",
			goModule: &config.GoModule{
				// DeleteGenerationOutputPaths should relative to library output directory.
				DeleteGenerationOutputPaths: []string{"../internal/generated/snippets/example"},
				GoAPIs: []*config.GoAPI{
					{Path: "example/v1", ImportPath: "example/apiv1"},
				},
			},
			want: []string{"-w", "repo/example"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			library := createLibrary(t)
			library.Go = test.goModule
			got, err := buildFormatArgs(library)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildFormatArgs_Error(t *testing.T) {
	library := &config.Library{
		Name: "example",
		APIs: []*config.API{
			{Path: "example/v1"},
		},
	}
	_, err := buildFormatArgs(library)
	if !errors.Is(err, errGoAPINotFound) {
		t.Errorf("got %v, want errGoAPINotFound", err)
	}
}

func createLibrary(t *testing.T) *config.Library {
	t.Helper()
	return &config.Library{
		Name:   "example",
		Output: "repo/example",
		APIs: []*config.API{
			{Path: "example/v1"},
		},
	}
}
