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

package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestBuildCppConfig(t *testing.T) {
	input := `
service {
  service_proto_path: "google/cloud/secretmanager/v1/service.proto"
  product_path: "google/cloud/secretmanager/v1"
  initial_copyright_year: "2021"
  forwarding_product_path: "google/cloud/secretmanager"
  retryable_status_codes: ["kUnavailable"]
}
`
	src := &config.Source{
		Commit: "1234567",
		SHA256: "abcdef",
		Dir:    "/some/dir",
	}

	got, err := buildCppConfig([]byte(input), src)
	if err != nil {
		t.Fatal(err)
	}

	want := &config.Config{
		Language: config.LanguageCpp,
		Repo:     "googleapis/google-cloud-cpp",
		Sources: &config.Sources{
			Googleapis: src,
		},
		Libraries: []*config.Library{
			{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Cpp: &config.CppLibrary{
					ProductPath:           "google/cloud/secretmanager/v1",
					InitialCopyrightYear:  "2021",
					ForwardingProductPath: "google/cloud/secretmanager",
					RetryableStatusCodes:  []string{"kUnavailable"},
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildCppConfig_Error(t *testing.T) {
	src := &config.Source{Commit: "1234567"}
	for _, test := range []struct {
		name  string
		input string
	}{
		{
			name:  "invalid textproto syntax",
			input: "service { unclosed string: \"abc }",
		},
		{
			name:  "unknown field",
			input: "service { unknown_field: true }",
		},
		{
			name:  "missing service_proto_path",
			input: "service { product_path: \"foo\" }",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildCppConfig([]byte(test.input), src)
			if err == nil {
				t.Fatalf("got nil error, want error, config: %+v", got)
			}
		})
	}
}

func TestReadCppGeneratorConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoPath string
		wantErr  error
	}{
		{
			name:     "valid file",
			repoPath: "testdata/run/success-cpp",
		},
		{
			name:     "missing file",
			repoPath: "testdata/run/no-config",
			wantErr:  fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := readCppGeneratorConfig(test.repoPath)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got err %v, want %v", err, test.wantErr)
			}
			if err == nil && len(got) == 0 {
				t.Error("expected non-empty generator_config.textproto data")
			}
		})
	}
}

func TestRunCppMigration(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	origFetchSource := fetchSource
	t.Cleanup(func() {
		fetchSource = origFetchSource
	})

	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    filepath.Join(wd, "../../internal/testdata/googleapis"),
		}, nil
	}

	for _, test := range []struct {
		name     string
		repoPath string
		wantErr  error
	}{
		{
			name:     "success",
			repoPath: "testdata/run/success-cpp",
		},
		{
			name:     "tidy_failed",
			repoPath: "testdata/run/tidy-fails-cpp",
			wantErr:  errTidyFailed,
		},
		{
			name:     "invalid_config",
			repoPath: "testdata/run/invalid-config-cpp",
			wantErr:  errConvertConfig,
		},
		{
			name:     "missing_file",
			repoPath: "testdata/run/no-config",
			wantErr:  fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(test.repoPath)); err != nil {
				t.Fatal(err)
			}
			err := runCppMigration(t.Context(), dir)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got err %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil {
				yamlPath := filepath.Join(dir, config.LibrarianYAML)
				if _, err := os.Stat(yamlPath); err != nil {
					t.Fatalf("expected %s to be created: %v", config.LibrarianYAML, err)
				}
			}
		})
	}
}

func TestRunCppMigration_FetchError(t *testing.T) {
	origFetchSource := fetchSource
	t.Cleanup(func() {
		fetchSource = origFetchSource
	})

	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return nil, errors.New("network error")
	}

	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/run/success-cpp")); err != nil {
		t.Fatal(err)
	}

	err := runCppMigration(t.Context(), dir)
	if !errors.Is(err, errFetchSource) {
		t.Fatalf("got err %v, want %v", err, errFetchSource)
	}
}

func TestRun_Cpp(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	origFetchSource := fetchSource
	t.Cleanup(func() {
		fetchSource = origFetchSource
	})

	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    filepath.Join(wd, "../../internal/testdata/googleapis"),
		}, nil
	}

	dir := t.TempDir()
	cppRepoDir := filepath.Join(dir, "google-cloud-cpp")
	if err := os.CopyFS(cppRepoDir, os.DirFS("testdata/run/success-cpp")); err != nil {
		t.Fatal(err)
	}

	if err := run(t.Context(), []string{cppRepoDir}); err != nil {
		t.Fatal(err)
	}

	yamlPath := filepath.Join(cppRepoDir, config.LibrarianYAML)
	if _, err := os.Stat(yamlPath); err != nil {
		t.Fatalf("expected %s to be created: %v", config.LibrarianYAML, err)
	}
}
