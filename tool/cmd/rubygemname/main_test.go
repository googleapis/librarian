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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRun(t *testing.T) {
	googleapisDir := t.TempDir()
	writeFile(t, filepath.Join(googleapisDir, "google/cloud/workloadidentity/v1/service.proto"), `syntax = "proto3";
option ruby_package = "Google::Cloud::WorkloadIdentity::V1";
`)
	writeFile(t, filepath.Join(googleapisDir, "google/cloud/backupdr/v1beta/service.proto"), `syntax = "proto3";
option ruby_package = "Google::Cloud::BackupDR::V1beta";
`)

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "librarian.yaml")
	cfg := &config.Config{
		Language: config.LanguageRuby,
		Sources: &config.Sources{
			Googleapis: &config.Source{Dir: googleapisDir},
		},
		Libraries: []*config.Library{
			{
				Name: "google-cloud-backupdr",
				APIs: []*config.API{{Path: "google/cloud/backupdr/v1"}},
				Ruby: &config.RubyPackage{WrapperOf: []string{"v1:1.0"}},
			},
			{
				Name: "google-cloud-backupdr-v1",
				APIs: []*config.API{{Path: "google/cloud/backupdr/v1"}},
			},
		},
	}
	if err := yaml.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "net-new API with explicit googleapis flag",
			args: []string{"rubygemname", "--config", cfgPath, "--googleapis", googleapisDir, "google/cloud/workloadidentity/v1"},
			want: "google-cloud-workload_identity\ngoogle-cloud-workload_identity-v1\n",
		},
		{
			name: "existing service new version via librarian-yaml alias and sources.googleapis.dir",
			args: []string{"rubygemname", "--librarian-yaml", cfgPath, "google/cloud/backupdr/v1beta"},
			want: "google-cloud-backupdr-v1beta\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := run(t.Context(), test.args, &stdout); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, stdout.String()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunErrors(t *testing.T) {
	dir := t.TempDir()
	nonRubyCfgPath := filepath.Join(dir, "nonruby.yaml")
	if err := yaml.Write(nonRubyCfgPath, &config.Config{Language: config.LanguageGo}); err != nil {
		t.Fatal(err)
	}

	rubyNoSourceCfgPath := filepath.Join(dir, "nosource.yaml")
	if err := yaml.Write(rubyNoSourceCfgPath, &config.Config{Language: config.LanguageRuby}); err != nil {
		t.Fatal(err)
	}

	rubyEmptySourceCfgPath := filepath.Join(dir, "emptysource.yaml")
	if err := yaml.Write(rubyEmptySourceCfgPath, &config.Config{
		Language: config.LanguageRuby,
		Sources:  &config.Sources{Googleapis: &config.Source{Commit: "abc123"}},
	}); err != nil {
		t.Fatal(err)
	}

	googleapisDir := t.TempDir()
	writeFile(t, filepath.Join(googleapisDir, "google/cloud/foo/v1/service.proto"), `syntax = "proto3";
option ruby_package = "Google::Cloud::Foo::V1";
`)

	rubyExistingAPICfgPath := filepath.Join(dir, "existingapi.yaml")
	if err := yaml.Write(rubyExistingAPICfgPath, &config.Config{
		Language: config.LanguageRuby,
		Libraries: []*config.Library{
			{
				Name: "google-cloud-foo-v1",
				APIs: []*config.API{{Path: "google/cloud/foo/v1"}},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		args    []string
		wantErr error
	}{
		{
			name:    "missing API arg",
			args:    []string{"rubygemname"},
			wantErr: errWrongAPICount,
		},
		{
			name:    "too many API args",
			args:    []string{"rubygemname", "a/v1", "b/v1"},
			wantErr: errWrongAPICount,
		},
		{
			name:    "non-ruby config",
			args:    []string{"rubygemname", "--config", nonRubyCfgPath, "google/cloud/foo/v1"},
			wantErr: errNotRubyConfig,
		},
		{
			name:    "missing googleapis flag and source",
			args:    []string{"rubygemname", "--config", rubyNoSourceCfgPath, "google/cloud/foo/v1"},
			wantErr: errMissingGoogleapis,
		},
		{
			name:    "googleapis dir missing requested api",
			args:    []string{"rubygemname", "--config", rubyNoSourceCfgPath, "--googleapis", googleapisDir, "google/cloud/missing/v1"},
			wantErr: os.ErrNotExist,
		},
		{
			name:    "api already exists in library",
			args:    []string{"rubygemname", "--config", rubyExistingAPICfgPath, "--googleapis", googleapisDir, "google/cloud/foo/v1"},
			wantErr: errAPIAlreadyExists,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			err := run(t.Context(), test.args, &stdout)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got error %v, want errors.Is(_, %v)", err, test.wantErr)
			}
		})
	}

	t.Run("load sources fails when sha256 is missing", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run(t.Context(), []string{"rubygemname", "--config", rubyEmptySourceCfgPath, "google/cloud/foo/v1"}, &stdout)
		if err == nil {
			t.Fatal("expected error from LoadSources, got nil")
		}
	})

	t.Run("stdout write error is returned", func(t *testing.T) {
		err := deriveAndPrint(t.Context(), rubyNoSourceCfgPath, googleapisDir, "google/cloud/foo/v1", failingWriter{})
		if !errors.Is(err, errWriteFailed) {
			t.Fatalf("got error %v, want errors.Is(_, %v)", err, errWriteFailed)
		}
	})
}

var errWriteFailed = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

func TestDeriveAndPrintMissingConfig(t *testing.T) {
	var stdout bytes.Buffer
	err := deriveAndPrint(t.Context(), filepath.Join(t.TempDir(), "missing.yaml"), "", "google/cloud/foo/v1", &stdout)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got error %v, want os.ErrNotExist", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
