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

package sources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRoot(t *testing.T) {
	cfg := &SourceConfig{
		Sources: &Sources{
			Conformance: "conformance-path",
			Discovery:   "discovery-path",
			Googleapis:  "googleapis-path",
			ProtobufSrc: "protobuf-path",
			Showcase:    "showcase-path",
		},
	}
	for _, test := range []struct {
		name string
		root string
		want string
	}{
		{"googleapis", "googleapis", "googleapis-path"},
		{"discovery", "discovery", "discovery-path"},
		{"showcase", "showcase", "showcase-path"},
		{"protobuf-src", "protobuf-src", "protobuf-path"},
		{"conformance", "conformance", "conformance-path"},
		{"unknown", "unknown", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := cfg.Root(test.root)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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

	for _, test := range []struct {
		name    string
		cfg     *SourceConfig
		relPath string
		want    string
	}{
		{
			name: "found",
			cfg: &SourceConfig{
				Sources:     &Sources{Googleapis: googleapis},
				ActiveRoots: []string{"googleapis"},
			},
			relPath: specPath,
			want:    fullPath,
		},
		{
			name: "not found",
			cfg: &SourceConfig{
				Sources:     &Sources{Googleapis: googleapis},
				ActiveRoots: []string{"googleapis"},
			},
			relPath: "not/found",
			want:    "not/found",
		},
		{
			name: "unknown root",
			cfg: &SourceConfig{
				Sources:     &Sources{Googleapis: googleapis},
				ActiveRoots: []string{"unknown"},
			},
			relPath: specPath,
			want:    specPath,
		},
		{
			name:    "nil receiver",
			cfg:     nil,
			relPath: "some/path",
			want:    "some/path",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.cfg.Resolve(test.relPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveDir(t *testing.T) {
	tempDir := t.TempDir()
	googleapis := filepath.Join(tempDir, "googleapis")
	dirPath := "google/cloud/secretmanager/v1"
	fullDir := filepath.Join(googleapis, dirPath)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		cfg     *SourceConfig
		relPath string
		want    string
	}{
		{
			name: "found",
			cfg: &SourceConfig{
				Sources:     &Sources{Googleapis: googleapis},
				ActiveRoots: []string{"googleapis"},
			},
			relPath: dirPath,
			want:    fullDir,
		},
		{
			name: "not found",
			cfg: &SourceConfig{
				Sources:     &Sources{Googleapis: googleapis},
				ActiveRoots: []string{"googleapis"},
			},
			relPath: "not/found",
			want:    "not/found",
		},
		{
			name: "unknown root",
			cfg: &SourceConfig{
				Sources:     &Sources{Googleapis: googleapis},
				ActiveRoots: []string{"unknown"},
			},
			relPath: dirPath,
			want:    dirPath,
		},
		{
			name:    "nil receiver",
			cfg:     nil,
			relPath: "some/path",
			want:    "some/path",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.cfg.ResolveDir(test.relPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewSourceConfig(t *testing.T) {
	srcs := &Sources{
		Googleapis:  "ga-path",
		Discovery:   "disco-path",
		Conformance: "conf-path",
		ProtobufSrc: "pb-path",
		Showcase:    "show-path",
	}
	for _, test := range []struct {
		name        string
		activeRoots []string
		want        *SourceConfig
	}{
		{
			name:        "with roots",
			activeRoots: []string{"googleapis", "discovery"},
			want: &SourceConfig{
				Sources:     srcs,
				ActiveRoots: []string{"googleapis", "discovery"},
			},
		},
		{
			name:        "empty roots defaults to googleapis",
			activeRoots: nil,
			want: &SourceConfig{
				Sources:     srcs,
				ActiveRoots: []string{"googleapis"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := NewSourceConfig(srcs, test.activeRoots)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
