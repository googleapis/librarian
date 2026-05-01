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

package librarian

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/yaml"
)

type updateTestSetup struct {
	server     *httptest.Server
	configPath string
}

const (
	googleapisTestCommit   = "123456"
	discoveryTestCommit    = "abcdef"
	conformanceTestCommit  = "protobuf1234"
	protobufTestCommit     = "protobuf1234"
	showcaseTestCommit     = "showcase1234"
	librarianTestCommit    = "librarian123"
	googleapisTestTarball  = "googleapis-tarball-content"
	discoveryTestTarball   = "discovery-tarball-content"
	conformanceTestTarball = "protobuf-tarball-content"
	protobufTestTarball    = "protobuf-tarball-content"
	showcaseTestTarball    = "showcase-tarball-content"
	librarianTestTarball   = "librarian-tarball-content"
	unchangedPlaceholder   = "this-should-not-change"
)

var (
	googleapisTestSHA  = fmt.Sprintf("%x", sha256.Sum256([]byte(googleapisTestTarball)))
	discoveryTestSHA   = fmt.Sprintf("%x", sha256.Sum256([]byte(discoveryTestTarball)))
	conformanceTestSHA = fmt.Sprintf("%x", sha256.Sum256([]byte(conformanceTestTarball)))
	protobufTestSHA    = fmt.Sprintf("%x", sha256.Sum256([]byte(protobufTestTarball)))
	showcaseTestSHA    = fmt.Sprintf("%x", sha256.Sum256([]byte(showcaseTestTarball)))
)

func setupUpdateTest(t *testing.T, conf *config.Config) *updateTestSetup {
	// Update defaults to using the branch configured in [sourceRepos].
	// We set up the test server handlers accordingly.
	googleapisBranch := sourceRepos["googleapis"].Branch
	discoveryBranch := sourceRepos["discovery"].Branch
	protobufBranch := sourceRepos["protobuf"].Branch
	showcaseBranch := sourceRepos["showcase"].Branch

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/googleapis/googleapis/commits/" + googleapisBranch:
			w.Write([]byte(googleapisTestCommit))
		case "/repos/googleapis/discovery-artifact-manager/commits/" + discoveryBranch:
			w.Write([]byte(discoveryTestCommit))
		case "/repos/protocolbuffers/protobuf/commits/" + protobufBranch:
			w.Write([]byte(protobufTestCommit))
		case "/repos/googleapis/gapic-showcase/commits/" + showcaseBranch:
			w.Write([]byte(showcaseTestCommit))
		case "/repos/googleapis/librarian/commits/" + config.BranchMain:
			w.Write([]byte(librarianTestCommit))
		case "/googleapis/googleapis/archive/" + googleapisTestCommit + ".tar.gz":
			w.Write([]byte(googleapisTestTarball))
		case "/googleapis/discovery-artifact-manager/archive/" + discoveryTestCommit + ".tar.gz":
			w.Write([]byte(discoveryTestTarball))
		case "/protocolbuffers/protobuf/archive/" + protobufTestCommit + ".tar.gz":
			w.Write([]byte(protobufTestTarball))
		case "/googleapis/gapic-showcase/archive/" + showcaseTestCommit + ".tar.gz":
			w.Write([]byte(showcaseTestTarball))
		case "/googleapis/librarian/archive/" + librarianTestCommit + ".tar.gz":
			w.Write([]byte(librarianTestTarball))
		default:
			http.NotFound(w, r)
		}
	}))

	githubAPI = ts.URL
	githubDownload = ts.URL

	cp := setupTestConfig(t, conf)

	return &updateTestSetup{
		server:     ts,
		configPath: cp,
	}
}

func setupTestConfig(t *testing.T, conf *config.Config) string {
	if conf == nil {
		return ""
	}
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, config.LibrarianYAML)
	if err := yaml.Write(configPath, conf); err != nil {
		t.Fatal(err)
	}
	return configPath
}

func TestUpdateCommand(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		setup      func(*config.Config)
		wantConfig func(*config.Config)
		before     func(*testing.T)
	}{
		{
			name: "googleapis",
			args: []string{"librarian", "update", "googleapis"},
			setup: func(cfg *config.Config) {
				cfg.Sources.Googleapis.Commit = "this-should-be-changed"
				cfg.Sources.Googleapis.SHA256 = "this-should-be-changed"
			},
			wantConfig: func(cfg *config.Config) {
				cfg.Sources.Googleapis.Commit = googleapisTestCommit
				cfg.Sources.Googleapis.SHA256 = googleapisTestSHA
			},
		},
		{
			name: "discovery",
			args: []string{"librarian", "update", "discovery"},
			setup: func(cfg *config.Config) {
				cfg.Sources.Discovery.Commit = "this-should-be-changed"
				cfg.Sources.Discovery.SHA256 = "this-should-be-changed"
			},
			wantConfig: func(cfg *config.Config) {
				cfg.Sources.Discovery.Commit = discoveryTestCommit
				cfg.Sources.Discovery.SHA256 = discoveryTestSHA
			},
		},
		{
			name: "conformance",
			args: []string{"librarian", "update", "conformance"},
			setup: func(cfg *config.Config) {
				cfg.Sources.Conformance.Commit = "this-should-be-changed"
				cfg.Sources.Conformance.SHA256 = "this-should-be-changed"
			},
			wantConfig: func(cfg *config.Config) {
				cfg.Sources.Conformance.Commit = conformanceTestCommit
				cfg.Sources.Conformance.SHA256 = conformanceTestSHA
			},
		},
		{
			name: "protobuf",
			args: []string{"librarian", "update", "protobuf"},
			setup: func(cfg *config.Config) {
				cfg.Sources.ProtobufSrc.Commit = "this-should-change"
				cfg.Sources.ProtobufSrc.SHA256 = "this-should-change"
			},
			wantConfig: func(cfg *config.Config) {
				cfg.Sources.ProtobufSrc.Commit = protobufTestCommit
				cfg.Sources.ProtobufSrc.SHA256 = protobufTestSHA
			},
		},
		{
			name: "showcase",
			args: []string{"librarian", "update", "showcase"},
			setup: func(cfg *config.Config) {
				cfg.Sources.Showcase.Commit = "this-should-change"
				cfg.Sources.Showcase.SHA256 = "this-should-change"
			},
			wantConfig: func(cfg *config.Config) {
				cfg.Sources.Showcase.Commit = showcaseTestCommit
				cfg.Sources.Showcase.SHA256 = showcaseTestSHA
			},
		},
		{
			name: "multiple sources",
			args: []string{"librarian", "update", "discovery", "googleapis"},
			setup: func(cfg *config.Config) {
				cfg.Sources.Googleapis.Commit = "this-should-be-changed"
				cfg.Sources.Googleapis.SHA256 = "this-should-be-changed"
				cfg.Sources.Discovery.Commit = "this-should-be-changed"
				cfg.Sources.Discovery.SHA256 = "this-should-be-changed"
			},
			wantConfig: func(cfg *config.Config) {
				cfg.Sources.Googleapis.Commit = googleapisTestCommit
				cfg.Sources.Googleapis.SHA256 = googleapisTestSHA
				cfg.Sources.Discovery.Commit = discoveryTestCommit
				cfg.Sources.Discovery.SHA256 = discoveryTestSHA
			},
		},
		{
			name: "version",
			args: []string{"librarian", "update", "version"},
			setup: func(cfg *config.Config) {
				cfg.Version = "this-should-change"
			},
			wantConfig: func(cfg *config.Config) {
				cfg.Version = "v1.2.3"
			},
			before: fakeGoList("latest", "v1.2.3"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			initialConfig := updateTestConfig()
			test.setup(initialConfig)

			wantConfig := updateTestConfig()
			test.setup(wantConfig)
			test.wantConfig(wantConfig)

			setup := setupUpdateTest(t, initialConfig)
			defer setup.server.Close()

			if test.before != nil {
				test.before(t)
			}

			err := Run(t.Context(), test.args...)
			if err != nil {
				t.Fatal(err)
			}

			gotConfig, err := yaml.Read[config.Config](setup.configPath)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateCommand_Errors(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		conf    *config.Config
		wantErr error
	}{
		{
			name:    "no sources provided",
			args:    []string{"librarian", "update"},
			wantErr: errNoSourcesProvided,
		},
		{
			name:    "unknown source",
			args:    []string{"librarian", "update", "unknown"},
			wantErr: errUnknownSource,
		},
		{
			name: "empty sources",
			args: []string{"librarian", "update", "googleapis"},
			conf: func() *config.Config {
				cfg := sample.Config()
				cfg.Sources = nil
				return cfg
			}(),
			wantErr: errEmptySources,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupTestConfig(t, test.conf)
			err := Run(t.Context(), test.args...)
			if err == nil {
				t.Errorf("want error %v, got nil", test.wantErr)
			} else if !errors.Is(err, test.wantErr) {
				t.Errorf("want error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func updateTestConfig() *config.Config {
	cfg := sample.Config()
	cfg.Language = config.LanguageGo
	cfg.Sources = &config.Sources{
		Googleapis: &config.Source{
			Commit: unchangedPlaceholder,
			SHA256: unchangedPlaceholder,
		},
		Discovery: &config.Source{
			Commit: unchangedPlaceholder,
			SHA256: unchangedPlaceholder,
		},
		Conformance: &config.Source{
			Commit: unchangedPlaceholder,
			SHA256: unchangedPlaceholder,
		},
		ProtobufSrc: &config.Source{
			Commit: unchangedPlaceholder,
			SHA256: unchangedPlaceholder,
		},
		Showcase: &config.Source{
			Commit: unchangedPlaceholder,
			SHA256: unchangedPlaceholder,
		},
	}
	return cfg
}

// fakeGoList returns a function that mocks `go list` execution by creating a
// fake go binary in a temporary directory and adding it to the front of PATH.
// It matches arguments containing "list -m -f {{.Version}} github.com/googleapis/librarian@<target>"
// and returns the specified <want> version.
func fakeGoList(target, want string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		goDir := t.TempDir()
		goPath := filepath.Join(goDir, "go")
		script := fmt.Sprintf("#!/bin/bash\nif [[ \"$*\" == *\"list -m -f {{.Version}} github.com/googleapis/librarian@%s\"* ]]; then\n  echo %q\n  exit 0\nfi\nexit 1\n", target, want)
		if err := os.WriteFile(goPath, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", goDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
}
