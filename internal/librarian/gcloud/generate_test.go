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
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

const testGoogleapisDir = "../../testdata/googleapis"

func TestGenerate(t *testing.T) {
	// sidekickgcloud.Generate calls out to protoc to build a
	// FileDescriptorSet from the protos.
	testhelper.RequireCommand(t, "protoc")

	for _, test := range []struct {
		name    string
		library *config.Library
		golden  string
	}{
		{
			name: "publicca",
			library: &config.Library{
				Name: "publicca",
				APIs: []*config.API{{Path: "google/cloud/security/publicca/v1"}},
			},
			golden: "testdata/publicca",
		},
		{
			name: "parallelstore",
			library: &config.Library{
				Name: "parallelstore",
				APIs: []*config.API{{Path: "google/cloud/parallelstore/v1"}},
			},
			golden: "testdata/parallelstore",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := t.TempDir()
			test.library.Output = out
			srcs := &sources.Sources{Googleapis: testGoogleapisDir}
			if err := Generate(t.Context(), test.library, srcs); err != nil {
				t.Fatal(err)
			}
			compareDirs(t, test.golden, out)
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
			apiPath:    "google/cloud/security/publicca/v1",
		},
		{
			name:       "missing api path",
			googleapis: testGoogleapisDir,
			apiPath:    "google/cloud/does/not/exist/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			library := &config.Library{
				Name:   "publicca",
				Output: t.TempDir(),
				APIs:   []*config.API{{Path: test.apiPath}},
			}
			srcs := &sources.Sources{Googleapis: test.googleapis}
			if err := Generate(t.Context(), library, srcs); err == nil {
				t.Error("Generate() error = nil, want error")
			}
		})
	}
}

func TestCollectProtos(t *testing.T) {
	apiPath := "google/cloud/security/publicca/v1"
	abs, err := filepath.Abs(testGoogleapisDir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := collectProtos(abs, apiPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.ToSlash(filepath.Join(apiPath, "resources.proto")),
		filepath.ToSlash(filepath.Join(apiPath, "service.proto")),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFindServiceConfig(t *testing.T) {
	apiPath := "google/cloud/security/publicca/v1"
	abs, err := filepath.Abs(testGoogleapisDir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := findServiceConfig(abs, apiPath)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(abs, apiPath, "publicca_v1.yaml")
	if got != want {
		t.Errorf("findServiceConfig() = %q, want %q", got, want)
	}
}

// compareDirs walks goldenDir and gotDir and fails the test on any file
// mismatch, missing file, or extra file.
func compareDirs(t *testing.T, goldenDir, gotDir string) {
	t.Helper()
	goldenFiles := collectFiles(t, goldenDir)
	gotFiles := collectFiles(t, gotDir)

	for rel, goldenPath := range goldenFiles {
		gotPath, ok := gotFiles[rel]
		if !ok {
			t.Errorf("%s: missing in output", rel)
			continue
		}
		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(gotPath)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", rel, diff)
		}
	}
	for rel := range gotFiles {
		if _, ok := goldenFiles[rel]; !ok {
			t.Errorf("%s: extra file generated", rel)
		}
	}
}

func collectFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = path
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
