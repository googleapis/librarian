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
	"github.com/googleapis/librarian/internal/yaml"
)

type updateTestSetup struct {
	server           *httptest.Server
	configPath       string
	initialConfig    string
	googleapisCommit string
	googleapisSHA    string
	discoveryCommit  string
	discoverySHA     string
}

func setupUpdateTest(t *testing.T) *updateTestSetup {
	const (
		googleapisCommit = "123456"
		discoveryCommit  = "abcdef"
	)
	googleapisTarball := "googleapis-tarball-content"
	discoveryTarball := "discovery-tarball-content"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/googleapis/googleapis/commits/master":
			w.Write([]byte(googleapisCommit))
		case "/repos/googleapis/discovery-artifact-manager/commits/master":
			w.Write([]byte(discoveryCommit))
		case "/googleapis/googleapis/archive/123456.tar.gz":
			w.Write([]byte(googleapisTarball))
		case "/googleapis/discovery-artifact-manager/archive/abcdef.tar.gz":
			w.Write([]byte(discoveryTarball))
		default:
			http.NotFound(w, r)
		}
	}))

	githubAPI = ts.URL
	githubDownload = ts.URL

	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := `language: go
sources:
  googleapis:
    commit: old-googleapis-commit
    sha256: old-googleapis-sha
  discovery:
    commit: old-discovery-commit
    sha256: old-discovery-sha
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	googleapisSHA := fmt.Sprintf("%x", sha256.Sum256([]byte(googleapisTarball)))
	discoverySHA := fmt.Sprintf("%x", sha256.Sum256([]byte(discoveryTarball)))

	return &updateTestSetup{
		server:           ts,
		configPath:       configPath,
		initialConfig:    configContent,
		googleapisCommit: googleapisCommit,
		googleapisSHA:    googleapisSHA,
		discoveryCommit:  discoveryCommit,
		discoverySHA:     discoverySHA,
	}
}

func TestUpdateCommand(t *testing.T) {
	setup := setupUpdateTest(t)
	defer setup.server.Close()

	for _, test := range []struct {
		name       string
		args       []string
		wantConfig *config.Config
	}{
		{
			name: "googleapis",
			args: []string{"librarian", "update", "googleapis"},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: setup.googleapisCommit,
						SHA256: setup.googleapisSHA,
					},
					Discovery: &config.Source{
						Commit: "old-discovery-commit",
						SHA256: "old-discovery-sha",
					},
				},
			},
		},
		{
			name: "discovery",
			args: []string{"librarian", "update", "discovery"},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "old-googleapis-commit",
						SHA256: "old-googleapis-sha",
					},
					Discovery: &config.Source{
						Commit: setup.discoveryCommit,
						SHA256: setup.discoverySHA,
					},
				},
			},
		},
		{
			name: "all",
			args: []string{"librarian", "update", "--all"},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: setup.googleapisCommit,
						SHA256: setup.googleapisSHA,
					},
					Discovery: &config.Source{
						Commit: setup.discoveryCommit,
						SHA256: setup.discoverySHA,
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// reset config file for each test
			if err := os.WriteFile(setup.configPath, []byte(setup.initialConfig), 0644); err != nil {
				t.Fatal(err)
			}

			err := Run(t.Context(), test.args...)
			if err != nil {
				t.Fatal(err)
			}

			gotConfig, err := yaml.Read[config.Config](setup.configPath)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantConfig, gotConfig); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateCommand_Errors(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		wantErr    error
		wantConfig *config.Config
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "update"},
			wantErr: errMissingSourceOrAllFlag,
		},
		{
			name:    "both source and all flag",
			args:    []string{"librarian", "update", "--all", "googleapis"},
			wantErr: errBothSourceAndAllFlag,
		},
		{
			name:    "unknown source",
			args:    []string{"librarian", "update", "unknown"},
			wantErr: errUnknownSource,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Run(t.Context(), test.args...)
			if err == nil {
				t.Errorf("want error %v, got nil", test.wantErr)
			} else if !errors.Is(err, test.wantErr) {
				t.Errorf("want error %v, got %v", test.wantErr, err)
			}
		})
	}
}
