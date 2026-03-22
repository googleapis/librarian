// Copyright 2025 Google LLC
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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/provider"
)

func TestSurfaceTreeRender(t *testing.T) {
	tmpDir := t.TempDir()

	service := &api.Service{
		DefaultHost: "parallelstore.googleapis.com",
		Package:     "google.cloud.parallelstore.v1",
	}
	model := &api.API{Title: "Parallelstore API"}
	overrides := &provider.Config{}

	methods := []*provider.MethodAdapter{
		{
			Method: &api.Method{
				Name: "CreateInstance",
				Service: &api.Service{Package: "google.cloud.parallelstore.v1"},
				InputType: &api.Message{},
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						PathTemplate: &api.PathTemplate{},
					}},
				},
			},
		},
	}
	// For testing, artificially set the plural resource name if GetPluralResourceName was mocked
	// Since we can't easily mock the full resource extraction here, we manually construct the SurfaceNode.

	node := &SurfaceNode{
		Name:             "instances",
		Path:             filepath.Join(tmpDir, "instances"),
		Methods:          methods,
		ServiceTitle:     "Parallelstore API",
		ResourceSingular: "instance",
		Tracks:           []string{"GA"},
	}

	err := renderTree(node, overrides, model, service)
	if err != nil {
		t.Fatalf("renderTree() errored = %v", err)
	}

	mainFile := filepath.Join(tmpDir, "instances", "create.yaml")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", mainFile)
	}

	content, _ := os.ReadFile(mainFile)
	wantContent := "_PARTIALS_: true\n"
	if diff := cmp.Diff(wantContent, string(content)); diff != "" {
		t.Errorf("main file content mismatch (-want +got):\n%s", diff)
	}

	initFile := filepath.Join(tmpDir, "instances", "__init__.py")
	if _, err := os.Stat(initFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", initFile)
	}
	initContent, _ := os.ReadFile(initFile)
	if !bytes.Contains(initContent, []byte(`"""Manage Parallelstore API instance resources."""`)) {
		t.Errorf("__init__.py content missing expected docstring:\n%s", string(initContent))
	}
}
