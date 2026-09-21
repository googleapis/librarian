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

package python

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestFindAuthScopes(t *testing.T) {
	for _, test := range []struct {
		name       string
		yamlBody   string
		wantScopes []string
	}{
		{
			name: "multiple scopes with comma and newline",
			yamlBody: `
authentication:
  rules:
    - selector: "*"
      oauth:
        canonical_scopes: |-
          https://www.googleapis.com/auth/cloud-platform,
          https://www.googleapis.com/auth/userinfo.email
`,
			wantScopes: []string{
				"https://www.googleapis.com/auth/cloud-platform",
				"https://www.googleapis.com/auth/userinfo.email",
			},
		},
		{
			name: "empty scopes falls back to default",
			yamlBody: `
authentication:
  rules:
    - selector: "*"
      oauth:
        canonical_scopes: ""
`,
			wantScopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
		},
		{
			name:       "missing config file falls back to default",
			yamlBody:   "",
			wantScopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			svc := api.NewTestService("ScopeService").
				WithPackage("google.cloud.scope.v1")
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			lib := &config.Library{
				Roots: []string{tempDir},
			}

			if test.yamlBody != "" {
				pkgDir := filepath.Join(tempDir, "google/cloud/scope/v1")
				if err := os.MkdirAll(pkgDir, 0o755); err != nil {
					t.Fatal(err)
				}
				configFile := filepath.Join(pkgDir, "scope_v1.yaml")
				if err := os.WriteFile(configFile, []byte(test.yamlBody), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			c := newTestCodec(t, model, lib)
			cfg, err := c.loadServiceConfig(svc)
			if err != nil {
				t.Fatal(err)
			}
			got := c.findAuthScopes(cfg)
			if diff := cmp.Diff(test.wantScopes, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsRestAsyncIOEnabled(t *testing.T) {
	for _, test := range []struct {
		name     string
		yamlBody string
		want     bool
	}{
		{
			name: "enabled in library_settings",
			yamlBody: `
publishing:
  library_settings:
    - version: google.cloud.async.v1
      python_settings:
        experimental_features:
          rest_async_io_enabled: true
`,
			want: true,
		},
		{
			name: "disabled in library_settings",
			yamlBody: `
publishing:
  library_settings:
    - version: google.cloud.async.v1
      python_settings:
        experimental_features:
          rest_async_io_enabled: false
`,
			want: false,
		},
		{
			name:     "missing config",
			yamlBody: "",
			want:     false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			svc := api.NewTestService("AsyncService").
				WithPackage("google.cloud.async.v1")
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			lib := &config.Library{
				Roots: []string{tempDir},
			}

			if test.yamlBody != "" {
				pkgDir := filepath.Join(tempDir, "google/cloud/async/v1")
				if err := os.MkdirAll(pkgDir, 0o755); err != nil {
					t.Fatal(err)
				}
				configFile := filepath.Join(pkgDir, "async_v1.yaml")
				if err := os.WriteFile(configFile, []byte(test.yamlBody), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			c := newTestCodec(t, model, lib)
			cfg, err := c.loadServiceConfig(svc)
			if err != nil {
				t.Fatal(err)
			}
			got := c.isRestAsyncIOEnabled(cfg, svc)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindConfigFileForService(t *testing.T) {
	tempDir := t.TempDir()
	apiDir := filepath.Join(tempDir, "google/cloud/myapi/v1")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(apiDir, "service_v1.yaml")
	if err := os.WriteFile(targetFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	otherDir := filepath.Join(tempDir, "google/cloud/other/v1")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	otherFile := filepath.Join(otherDir, "other_v1.yaml")
	if err := os.WriteFile(otherFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := api.NewTestService("MyService").
		WithPackage("google.cloud.myapi.v1")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	lib := &config.Library{
		Roots: []string{tempDir},
		APIs: []*config.API{
			{Path: "google/cloud/other/v1"},
			{Path: "google/cloud/myapi/v1"},
		},
	}

	c := newTestCodec(t, model, lib)
	got := c.findConfigFileForService(svc, "*_v1.yaml")
	if diff := cmp.Diff(targetFile, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateTransport_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		filename string
		content  string
		wantErr  error
	}{
		{
			name:     "malformed_service_config_yaml",
			filename: "myapi_v1.yaml",
			content:  "authentication:\n  rules: [invalid yaml",
			wantErr:  ErrLoadServiceConfig,
		},
		{
			name:     "malformed_grpc_service_config_json",
			filename: "myapi_grpc_service_config.json",
			content:  "{invalid_json",
			wantErr:  ErrLoadGRPCConfig,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			pkgDir := filepath.Join(tempDir, "google", "cloud", "myapi", "v1")
			if err := os.MkdirAll(pkgDir, 0o755); err != nil {
				t.Fatal(err)
			}
			filePath := filepath.Join(pkgDir, test.filename)
			if err := os.WriteFile(filePath, []byte(test.content), 0o644); err != nil {
				t.Fatal(err)
			}

			svc := api.NewTestService("MyService").
				WithPackage("google.cloud.myapi.v1")
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			lib := &config.Library{
				Roots: []string{tempDir},
				APIs: []*config.API{
					{Path: "google/cloud/myapi/v1"},
				},
			}

			c := newTestCodec(t, model, lib)
			_, err := c.annotateTransport(svc)
			if err == nil {
				t.Fatalf("annotateTransport(%s) error = nil, want %v", svc.Name, test.wantErr)
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("annotateTransport(%s) error = %v, want %v", svc.Name, err, test.wantErr)
			}
		})
	}
}

func TestGetGRPCStubType(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   string
	}{
		{
			name:   "unary_unary",
			method: api.NewTestMethod("Test"),
			want:   "unary_unary",
		},
		{
			name:   "unary_stream",
			method: api.NewTestMethod("Test").WithServerSideStreaming(),
			want:   "unary_stream",
		},
		{
			name:   "stream_unary",
			method: api.NewTestMethod("Test").WithClientSideStreaming(),
			want:   "stream_unary",
		},
		{
			name:   "stream_stream",
			method: api.NewTestMethod("Test").WithBidiStreaming(),
			want:   "stream_stream",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getGRPCStubType(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
