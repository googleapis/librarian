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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestModelRoot(t *testing.T) {
	cfg := SourceConfig{
		Sources: Sources{
			Googleapis: "googleapis",
			Showcase:   "showcase",
		},
	}
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "googleapis",
			path: "google/cloud/secretmanager/v1",
			want: "googleapis",
		},
		{
			name: "showcase",
			path: "schema/google/showcase/v1beta1",
			want: "showcase",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := cfg.ModelRoot(test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("ModelRoot() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRoot(t *testing.T) {
	cfg := SourceConfig{
		Sources: Sources{
			Googleapis: "googleapis-path",
			Discovery:  "discovery-path",
		},
	}
	for _, test := range []struct {
		name string
		root string
		want string
	}{
		{
			name: "googleapis",
			root: "googleapis",
			want: "googleapis-path",
		},
		{
			name: "discovery",
			root: "discovery",
			want: "discovery-path",
		},
		{
			name: "unknown",
			root: "unknown",
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := cfg.Root(test.root)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Root() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tempDir := t.TempDir()
	googleapis := filepath.Join(tempDir, "googleapis")
	if err := os.Mkdir(googleapis, 0755); err != nil {
		t.Fatal(err)
	}

	specPath := "google/cloud/secretmanager/v1/secretmanager.yaml"
	fullPath := filepath.Join(googleapis, specPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := SourceConfig{
		Sources: Sources{
			Googleapis: googleapis,
		},
		ActiveRoots: []string{"googleapis"},
	}

	for _, test := range []struct {
		name    string
		relPath string
		want    string
	}{
		{
			name:    "found",
			relPath: specPath,
			want:    fullPath,
		},
		{
			name:    "not found",
			relPath: "not/found",
			want:    "not/found",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := cfg.Resolve(test.relPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
