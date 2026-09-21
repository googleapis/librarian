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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateClient(t *testing.T) {
	for _, test := range []struct {
		name         string
		setup        func() (*api.API, *config.Library)
		wantFilePath string
		wantContents []string
	}{
		{
			name: "basic sync client emission",
			setup: func() (*api.API, *config.Library) {
				req := api.NewTestMessage("GetSecretRequest").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 1)
				resp := api.NewTestMessage("Secret").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 10)
				method := api.NewTestMethod("GetSecret").
					WithInput(req).
					WithOutput(resp)
				method.Documentation = "Gets a secret."
				svc := api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 1).
					WithMethods(method)
				svc.Documentation = "Secret Manager Service API."
				svc.DefaultHost = "secretmanager.googleapis.com"
				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.secretmanager.v1")
				model.Name = "google-cloud-secretmanager"
				cfg := &config.Library{
					Name: "google-cloud-secretmanager",
				}
				return model, cfg
			},
			wantFilePath: "google/cloud/secretmanager_v1/services/secret_manager_service/client.py",
			wantContents: []string{
				"class SecretManagerServiceClient(metaclass=SecretManagerServiceClientMeta):",
				"DEFAULT_ENDPOINT = \"secretmanager.googleapis.com\"",
				"DEFAULT_CLIENT_INFO = gapic_v1.client_info.ClientInfo(",
				"def __init__(",
				"def get_secret(",
				"def __enter__(self) -> \"SecretManagerServiceClient\":",
				"def __exit__(self, type, value, traceback):",
			},
		},
		{
			name: "client with pagination",
			setup: func() (*api.API, *config.Library) {
				pageItem := api.NewTestMessage("Item").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/example.proto", 1)
				pageField := api.NewTestField("items").
					WithMessageType(pageItem).
					WithRepeated()
				nextPageTokenField := api.NewTestField("next_page_token").
					WithType(api.TypezString)
				resp := api.NewTestMessage("ListItemsResponse").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/example.proto", 10).
					WithFields(pageField, nextPageTokenField).
					WithPagination(nextPageTokenField, pageField)
				pageTokenField := api.NewTestField("page_token").
					WithType(api.TypezString)
				req := api.NewTestMessage("ListItemsRequest").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/example.proto", 20).
					WithFields(pageTokenField)
				method := api.NewTestMethod("ListItems").
					WithInput(req).
					WithOutput(resp).
					WithPagination(pageTokenField)
				svc := api.NewTestService("ExampleService").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/example.proto", 30).
					WithMethods(method)
				model := api.NewTestAPI([]*api.Message{pageItem, resp, req}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.example.v1")
				model.Name = "google-cloud-example"
				cfg := &config.Library{
					Name: "google-cloud-example",
				}
				return model, cfg
			},
			wantFilePath: "google/cloud/example_v1/services/example_service/client.py",
			wantContents: []string{
				"class ExampleServiceClient(metaclass=ExampleServiceClientMeta):",
				"def list_items(",
				"response = pagers.ListItemsPager(",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			model, cfg := test.setup()
			if err := Generate(t.Context(), model, outDir, cfg); err != nil {
				t.Fatal(err)
			}

			fullPath := filepath.Join(outDir, test.wantFilePath)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatal(err)
			}

			text := string(content)
			for _, want := range test.wantContents {
				if !strings.Contains(text, want) {
					t.Errorf("Generate() output file %s missing expected content %q", test.wantFilePath, want)
				}
			}
		})
	}
}
