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
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestRoot(t *testing.T) {
	cfg := SourceConfig{
		Sources: Sources{
			Googleapis: "googleapis-path",
			Discovery:  "discovery-path",
		},
	}
	for _, test := range []struct {
		name    string
		root    string
		want    string
		wantErr bool
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
				t.Errorf("Root(%q) mismatch (-want +got):\n%s", test.root, diff)
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
		wantErr bool
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
		{
			name:    "unknown root",
			relPath: specPath,
			want:    specPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "unknown root" {
				cfg.ActiveRoots = []string{"unknown"}
			}
			got, err := cfg.Resolve(test.relPath)
			if (err != nil) != test.wantErr {
				t.Fatalf("Resolve(%q) error = %v, wantErr %v", test.relPath, err, test.wantErr)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Resolve(%q) mismatch (-want +got):\n%s", test.relPath, diff)
			}
		})
	}
}

func TestSourceRoots(t *testing.T) {
	type TestCase struct {
		input map[string]string
		want  []string
	}
	testCases := []TestCase{
		{map[string]string{}, nil},
		{map[string]string{
			"googleapis-root": "foo",
			"other-root":      "bar",
			"ignored":         "baz",
		}, []string{"googleapis-root", "other-root"}},
		{map[string]string{
			"roots":           "googleapis,more",
			"googleapis-root": "foo",
			"other-root":      "bar",
			"more-root":       "bar",
			"ignored":         "baz",
		}, []string{"googleapis-root", "more-root"}},
	}

	for _, c := range testCases {
		got := SourceRoots(c.input)
		less := func(a, b string) bool { return a < b }
		if diff := cmp.Diff(c.want, got, cmpopts.SortSlices(less)); diff != "" {
			t.Errorf("AllSourceRoots mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestAllSourceRoots(t *testing.T) {
	type TestCase struct {
		input map[string]string
		want  []string
	}
	testCases := []TestCase{
		{map[string]string{}, nil},
		{map[string]string{
			"googleapis-root": "foo",
			"other-root":      "bar",
			"ignored":         "baz",
		}, []string{"googleapis-root", "other-root"}},
	}

	for _, c := range testCases {
		got := AllSourceRoots(c.input)
		less := func(a, b string) bool { return a < b }
		if diff := cmp.Diff(c.want, got, cmpopts.SortSlices(less)); diff != "" {
			t.Errorf("AllSourceRoots mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestNewSourceConfig(t *testing.T) {
	options := map[string]string{
		"googleapis-root":   "ga-path",
		"discovery-root":    "disco-path",
		"conformance-root":  "conf-path",
		"protobuf-src-root": "pb-path",
		"showcase-root":     "show-path",
		"include-list":      "file1,file2",
		"exclude-list":      "file3",
		"roots":             "googleapis,discovery",
	}
	want := SourceConfig{
		Sources: Sources{
			Googleapis:  "ga-path",
			Discovery:   "disco-path",
			Conformance: "conf-path",
			ProtobufSrc: "pb-path",
			Showcase:    "show-path",
		},
		ActiveRoots: []string{"googleapis-root", "discovery-root"},
		IncludeList: []string{"file1", "file2"},
		ExcludeList: []string{"file3"},
	}
	got := NewSourceConfig(options)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("NewSourceConfig() mismatch (-want +got):\n%s", diff)
	}
}
