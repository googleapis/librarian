// Copyright 2024 Google LLC
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

package serviceconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/testing/protocmp"
)

const googleapisDir = "../testdata/googleapis"

func TestRead(t *testing.T) {
	got, err := Read(filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want := &Service{
		Name:  "secretmanager.googleapis.com",
		Title: "Secret Manager API",
		Documentation: &Documentation{
			Summary: "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
		},
	}
	opts := cmp.Options{
		protocmp.Transform(),
		protocmp.IgnoreFields(&Service{}, "apis", "authentication", "config_version", "http", "publishing"),
		protocmp.IgnoreFields(&Documentation{}, "overview", "rules"),
	}
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// TestNoGenprotoServiceConfigImports verifies that the genproto serviceconfig
// dependency is isolated to this package.
func TestNoGenprotoServiceConfigImports(t *testing.T) {
	const genprotoImport = "google.golang.org/genproto/googleapis/api/serviceconfig"
	root := filepath.Join("..", "..")

	var violations []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil ||
			!strings.HasSuffix(path, ".go") ||
			strings.Contains(path, "/vendor/") ||
			strings.Contains(path, "/testdata/") ||
			strings.Contains(path, "internal/serviceconfig/") {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.Contains(string(content), genprotoImport) {
			relPath, _ := filepath.Rel(root, path)
			violations = append(violations, relPath)
		}
		return nil
	})
	if len(violations) > 0 {
		t.Errorf("Found %d file(s) importing %q outside of internal/serviceconfig:\n  %s",
			len(violations), genprotoImport, strings.Join(violations, "\n  "))
	}
}

func TestFind(t *testing.T) {
	for _, test := range []struct {
		name    string
		api     string
		want    *API
		wantErr bool
	}{
		{
			name: "found with title",
			api:  "google/cloud/secretmanager/v1",
			want: &API{
				Path:             "google/cloud/secretmanager/v1",
				ServiceConfig:    "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				NewIssueURI:      "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				DocumentationURI: "https://cloud.google.com/secret-manager/docs/overview",
				APIShortName:     "secretmanager",
				OpenAPI:          "testdata/secretmanager_openapi_v1.json",
				Title:            "Secret Manager API",
			},
		},
		{
			name: "not service config has title override",
			api:  "google/cloud/orgpolicy/v1",
			want: &API{
				Path:  "google/cloud/orgpolicy/v1",
				Title: "Organization Policy Types",
			},
		},
		{
			name: "directory does not exist",
			api:  "google/cloud/nonexistent/v1",
			want: &API{
				Path: "google/cloud/nonexistent/v1",
			},
			wantErr: true,
		},
		{
			name: "service config override",
			api:  "google/cloud/aiplatform/v1/schema/predict/instance",
			want: &API{
				Path:          "google/cloud/aiplatform/v1/schema/predict/instance",
				ServiceConfig: "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
				Title:         "Vertex AI API",
			},
		},
		{
			name: "openapi",
			api:  "testdata/secretmanager_openapi_v1.json",
			want: &API{
				Path:             "google/cloud/secretmanager/v1",
				OpenAPI:          "testdata/secretmanager_openapi_v1.json",
				ServiceConfig:    "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Title:            "Secret Manager API",
				NewIssueURI:      "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
				DocumentationURI: "https://cloud.google.com/secret-manager/docs/overview",
				APIShortName:     "secretmanager",
			},
		},
		{
			name: "discovery",
			api:  "discoveries/compute.v1.json",
			want: &API{
				Path:          "google/cloud/compute/v1",
				Discovery:     "discoveries/compute.v1.json",
				ServiceConfig: "google/cloud/compute/v1/compute_v1.yaml",
				Title:         "Google Compute Engine API",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Find(googleapisDir, test.api, LangGo)
			if err != nil {
				if !test.wantErr {
					t.Fatal(err)
				}
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindGRPCServiceConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "found",
			path: "google/cloud/secretmanager/v1",
			want: "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
		},
		{
			name: "not found",
			path: "google/cloud/orgpolicy/v1",
			want: "",
		},
		{
			name: "directory does not exist",
			path: "google/cloud/nonexistent/v1",
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := FindGRPCServiceConfig(googleapisDir, test.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestFindGRPCServiceConfigMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	apiPath := "google/example/v1"
	apiDir := filepath.Join(dir, apiPath)
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"foo_grpc_service_config.json", "bar_grpc_service_config.json"} {
		if err := os.WriteFile(filepath.Join(apiDir, name), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := FindGRPCServiceConfig(dir, apiPath)
	if err == nil {
		t.Fatal("expected error for multiple gRPC service config files")
	}
}

func TestPopulateFromServiceConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		api  *API
		cfg  *Service
		want *API
	}{
		{
			name: "populate everything from service config",
			api:  &API{},
			cfg: &Service{
				Title: "service config title",
				Publishing: &annotations.Publishing{
					DocumentationUri: "service config doc uri",
					NewIssueUri:      "service config new issue uri",
					ApiShortName:     "service config short name",
				},
			},
			want: &API{
				Title:            "service config title",
				DocumentationURI: "service config doc uri",
				NewIssueURI:      "service config new issue uri",
				APIShortName:     "service config short name",
			},
		},
		{
			name: "no publishing",
			api: &API{
				DocumentationURI: "override doc uri",
			},
			cfg: &Service{
				Title: "service config title",
			},
			want: &API{
				Title:            "service config title",
				DocumentationURI: "override doc uri",
			},
		},
		{
			name: "everything overridden",
			api: &API{
				Title:            "override title",
				DocumentationURI: "override doc uri",
				NewIssueURI:      "override new issue uri",
				APIShortName:     "override short name",
			},
			cfg: &Service{
				Title: "service config title",
				Publishing: &annotations.Publishing{
					DocumentationUri: "service config doc uri",
					NewIssueUri:      "service config new issue uri",
					ApiShortName:     "service config short name",
				},
			},
			want: &API{
				Title:            "override title",
				DocumentationURI: "override doc uri",
				NewIssueURI:      "override new issue uri",
				APIShortName:     "override short name",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := populateFromServiceConfig(test.api, test.cfg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateAPI(t *testing.T) {
	for _, test := range []struct {
		name     string
		path     string
		language string
		api      *API
		wantErr  bool
	}{
		{
			name:     "api in allowlist, all languages",
			path:     "google/api",
			language: LangGo,
			api:      &API{Path: "google/api"},
			wantErr:  false,
		},
		{
			name:     "api in allowlist, restricted language allowed",
			path:     "google/cloud/aiplatform/v1beta1",
			language: LangPython,
			api:      &API{Path: "google/cloud/aiplatform/v1beta1", Languages: []string{LangPython}},
			wantErr:  false,
		},
		{
			name:     "api in allowlist, restricted language not allowed",
			path:     "google/cloud/aiplatform/v1beta1",
			language: LangGo,
			api:      &API{Path: "google/cloud/aiplatform/v1beta1", Languages: []string{LangPython}},
			wantErr:  true,
		},
		{
			name:     "api not in list, google/cloud/ prefix",
			path:     "google/cloud/newapi/v1",
			language: LangGo,
			api:      nil,
			wantErr:  false,
		},
		{
			name:     "api not in list, no google/cloud/ prefix",
			path:     "google/ads/newapi/v1",
			language: LangGo,
			api:      nil,
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := validateAPI(test.path, test.language, test.api)
			if (err != nil) != test.wantErr {
				t.Errorf("validateAPI() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !test.wantErr && got == nil {
				t.Error("validateAPI() returned nil but want non-nil API")
			}
		})
	}
}
