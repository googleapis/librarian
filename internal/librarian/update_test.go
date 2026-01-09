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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

type updateTestSetup struct {
	server     *httptest.Server
	configPath string
}

const (
	googleapisTestCommit   = "123456"
	discoveryTestCommit    = "abcdef"
	conformanceTestCommit  = "conformance1234"
	protobufTestCommit     = "protobuf1234"
	showcaseTestCommit     = "showcase1234"
	googleapisTestTarball  = "googleapis-tarball-content"
	discoveryTestTarball   = "discovery-tarball-content"
	conformanceTestTarball = "conformance-tarball-content"
	protobufTestTarball    = "protobuf-tarball-content"
	showcaseTestTarball    = "showcase-tarball-content"
	testBranch             = "other"
)

var (
	googleapisTestSHA  = fmt.Sprintf("%x", sha256.Sum256([]byte(googleapisTestTarball)))
	discoveryTestSHA   = fmt.Sprintf("%x", sha256.Sum256([]byte(discoveryTestTarball)))
	conformanceTestSHA = fmt.Sprintf("%x", sha256.Sum256([]byte(conformanceTestTarball)))
	protobufTestSHA    = fmt.Sprintf("%x", sha256.Sum256([]byte(protobufTestTarball)))
	showcaseTestSHA    = fmt.Sprintf("%x", sha256.Sum256([]byte(showcaseTestTarball)))
)

func setupUpdateTest(t *testing.T, conf *config.Config) *updateTestSetup {
	// Source.Branch can be empty in the config file. Update should default to
	// using the branch configured in [sourceRepos], so we only set up the
	// test server handlers with Source.Branch when it is explicitly set as it
	// would be in the file on disk.
	googleapisBranch := sourceRepos["googleapis"].Branch
	if conf.Sources.Googleapis.Branch != "" {
		googleapisBranch = conf.Sources.Googleapis.Branch
	}
	discoveryBranch := sourceRepos["discovery"].Branch
	if conf.Sources.Discovery.Branch != "" {
		discoveryBranch = conf.Sources.Discovery.Branch
	}
	conformanceBranch := sourceRepos["conformance"].Branch
	if conf.Sources.Conformance.Branch != "" {
		conformanceBranch = conf.Sources.Conformance.Branch
	}
	protobufBranch := sourceRepos["protobuf"].Branch
	if conf.Sources.ProtobufSrc.Branch != "" {
		protobufBranch = conf.Sources.ProtobufSrc.Branch
	}
	showcaseBranch := sourceRepos["showcase"].Branch
	if conf.Sources.Showcase.Branch != "" {
		showcaseBranch = conf.Sources.Showcase.Branch
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/googleapis/googleapis/commits/" + googleapisBranch:
			w.Write([]byte(googleapisTestCommit))
		case "/repos/googleapis/discovery-artifact-manager/commits/" + discoveryBranch:
			w.Write([]byte(discoveryTestCommit))
		case "/repos/protocolbuffers/protobuf/commits/" + conformanceBranch:
			w.Write([]byte(conformanceTestCommit))
		case "/repos/protocolbuffers/protobuf/commits/" + protobufBranch:
			w.Write([]byte(protobufTestCommit))
		case "/repos/googleapis/gapic-showcase/commits/" + showcaseBranch:
			w.Write([]byte(showcaseTestCommit))
		case "/googleapis/googleapis/archive/" + googleapisTestCommit + ".tar.gz":
			w.Write([]byte(googleapisTestTarball))
		case "/googleapis/discovery-artifact-manager/archive/" + discoveryTestCommit + ".tar.gz":
			w.Write([]byte(discoveryTestTarball))
		case "/protocolbuffers/protobuf/archive/" + conformanceTestCommit + ".tar.gz":
			w.Write([]byte(conformanceTestTarball))
		case "/protocolbuffers/protobuf/archive/" + protobufTestCommit + ".tar.gz":
			w.Write([]byte(protobufTestTarball))
		case "/googleapis/gapic-showcase/archive/" + showcaseTestCommit + ".tar.gz":
			w.Write([]byte(showcaseTestTarball))
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
	configPath := filepath.Join(tempDir, librarianConfigPath)
	if err := yaml.Write(configPath, conf); err != nil {
		t.Fatal(err)
	}
	return configPath
}

func TestUpdateCommand(t *testing.T) {
	for _, test := range []struct {
		name          string
		args          []string
		initialConfig *config.Config
		wantConfig    *config.Config
	}{
		{
			name: "googleapis",
			args: []string{"librarian", "update", "googleapis"},
			initialConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-be-changed",
						SHA256: "this-should-be-changed",
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Conformance: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					ProtobufSrc: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Showcase: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: googleapisTestCommit,
						SHA256: googleapisTestSHA,
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Conformance: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					ProtobufSrc: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Showcase: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
		},
		{
			name: "discovery",
			args: []string{"librarian", "update", "discovery"},
			initialConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Discovery: &config.Source{
						Commit: "this-should-be-changed",
						SHA256: "this-should-be-changed",
					},
					Conformance: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					ProtobufSrc: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Showcase: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Discovery: &config.Source{
						Commit: discoveryTestCommit,
						SHA256: discoveryTestSHA,
					},
					Conformance: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					ProtobufSrc: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Showcase: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
		},
		{
			name: "conformance",
			args: []string{"librarian", "update", "conformance"},
			initialConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Conformance: &config.Source{
						Commit: "this-should-be-changed",
						SHA256: "this-should-be-changed",
					},
					ProtobufSrc: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Showcase: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Conformance: &config.Source{
						Commit: conformanceTestCommit,
						SHA256: conformanceTestSHA,
					},
					ProtobufSrc: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Showcase: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
		},
		{
			name: "protobuf",
			args: []string{"librarian", "update", "protobuf"},
			initialConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Conformance: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					ProtobufSrc: &config.Source{
						// Use a non default branch to avoid collision with
						// conformance branch.
						Branch: testBranch,
						Commit: "this-should-change",
						SHA256: "this-should-change",
					},
					Showcase: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Conformance: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					ProtobufSrc: &config.Source{
						Branch: testBranch,
						Commit: protobufTestCommit,
						SHA256: protobufTestSHA,
					},
					Showcase: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
		},
		{
			name: "showcase",
			args: []string{"librarian", "update", "showcase"},
			initialConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Conformance: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					ProtobufSrc: &config.Source{
						Branch: testBranch,
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Showcase: &config.Source{
						Commit: "this-should-change",
						SHA256: "this-should-change",
					},
				},
			},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Conformance: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					ProtobufSrc: &config.Source{
						Branch: testBranch,
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
					Showcase: &config.Source{
						Commit: showcaseTestCommit,
						SHA256: showcaseTestSHA,
					},
				},
			},
		},
		{
			name: "all",
			args: []string{"librarian", "update", "--all"},
			initialConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "this-should-be-changed",
						SHA256: "this-should-be-changed",
					},
					Discovery: &config.Source{
						Commit: "this-should-be-changed",
						SHA256: "this-should-be-changed",
					},
					Conformance: &config.Source{
						Commit: "this-should-be-changed",
						SHA256: "this-should-be-changed",
					},
					ProtobufSrc: &config.Source{
						Branch: "testBranch",
						Commit: "this-should-be-changed",
						SHA256: "this-should-be-changed",
					},
				},
			},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: googleapisTestCommit,
						SHA256: googleapisTestSHA,
					},
					Discovery: &config.Source{
						Commit: discoveryTestCommit,
						SHA256: discoveryTestSHA,
					},
					Conformance: &config.Source{
						Commit: conformanceTestCommit,
						SHA256: conformanceTestSHA,
					},
					ProtobufSrc: &config.Source{
						Branch: testBranch,
						Commit: protobufTestCommit,
						SHA256: protobufTestSHA,
					},
				},
			},
		},
		{
			name: "googleapis branch",
			args: []string{"librarian", "update", "googleapis"},
			initialConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Branch: testBranch,
						Commit: "this-should-be-changed",
						SHA256: "this-should-be-changed",
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
			wantConfig: &config.Config{
				Language: "go",
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Branch: testBranch,
						Commit: googleapisTestCommit,
						SHA256: googleapisTestSHA,
					},
					Discovery: &config.Source{
						Commit: "this-should-not-change",
						SHA256: "this-should-not-change",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setup := setupUpdateTest(t, test.initialConfig)
			defer setup.server.Close()

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
		name    string
		args    []string
		conf    *config.Config
		wantErr error
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
		{
			name: "empty sources",
			args: []string{"librarian", "update", "googleapis"},
			conf: &config.Config{
				Language: "go",
			},
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
