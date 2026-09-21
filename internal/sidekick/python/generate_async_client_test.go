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

func TestGenerateAsyncClient(t *testing.T) {
	for _, test := range []struct {
		name         string
		setup        func() (*api.API, *config.Library)
		wantFilePath string
		wantContents []string
	}{
		{
			name: "basic async client emission",
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
			wantFilePath: "google/cloud/secretmanager_v1/services/secret_manager_service/async_client.py",
			wantContents: []string{
				"class SecretManagerServiceAsyncClient:",
				"DEFAULT_ENDPOINT = SecretManagerServiceClient.DEFAULT_ENDPOINT",
				"get_transport_class = SecretManagerServiceClient.get_transport_class",
				"def __init__(",
				"async def get_secret(",
				"async def __aenter__(self) -> \"SecretManagerServiceAsyncClient\":",
				"async def __aexit__(self, exc_type, exc, tb):",
				"__all__ = (",
				"    \"SecretManagerServiceAsyncClient\",",
			},
		},
		{
			name: "async client with pagination",
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
			wantFilePath: "google/cloud/example_v1/services/example_service/async_client.py",
			wantContents: []string{
				"class ExampleServiceAsyncClient:",
				"async def list_items(",
				"response = pagers.ListItemsAsyncPager(",
			},
		},
		{
			name: "async client with LRO",
			setup: func() (*api.API, *config.Library) {
				req := api.NewTestMessage("CreateInstanceRequest").
					WithPackage("google.cloud.redis.v1").
					WithSourceLocation("google/cloud/redis/v1/cloud_redis.proto", 1)
				resp := api.NewTestMessage("Instance").
					WithPackage("google.cloud.redis.v1").
					WithSourceLocation("google/cloud/redis/v1/cloud_redis.proto", 10)
				meta := api.NewTestMessage("OperationMetadata").
					WithPackage("google.cloud.redis.v1").
					WithSourceLocation("google/cloud/redis/v1/cloud_redis.proto", 20)
				lroMethod := api.NewTestMethod("CreateInstance").
					WithInput(req).
					WithOutput(resp).
					WithOperationInfo(&api.OperationInfo{ResponseTypeID: resp.ID, MetadataTypeID: meta.ID})
				svc := api.NewTestService("CloudRedis").
					WithPackage("google.cloud.redis.v1").
					WithSourceLocation("google/cloud/redis/v1/cloud_redis.proto", 30).
					WithMethods(lroMethod)
				model := api.NewTestAPI([]*api.Message{req, resp, meta}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.redis.v1")
				model.Name = "google-cloud-redis"
				cfg := &config.Library{
					Name: "google-cloud-redis",
				}
				return model, cfg
			},
			wantFilePath: "google/cloud/redis_v1/services/cloud_redis/async_client.py",
			wantContents: []string{
				"class CloudRedisAsyncClient:",
				"async def create_instance(",
				"response = operation_async.from_gapic(",
			},
		},
		{
			name: "async client with repeated and scalar flattened parameters",
			setup: func() (*api.API, *config.Library) {
				nameField := api.NewTestField("name").WithType(api.TypezString)
				delegatesField := api.NewTestField("delegates").WithType(api.TypezString).WithRepeated()
				req := api.NewTestMessage("SignJwtRequest").
					WithPackage("google.iam.credentials.v1").
					WithSourceLocation("google/iam/credentials/v1/iamcredentials.proto", 1).
					WithFields(nameField, delegatesField)
				resp := api.NewTestMessage("SignJwtResponse").
					WithPackage("google.iam.credentials.v1").
					WithSourceLocation("google/iam/credentials/v1/iamcredentials.proto", 10)
				method := api.NewTestMethod("SignJwt").
					WithInput(req).
					WithOutput(resp).
					WithSignatures(&api.MethodSignature{Names: []string{"name", "delegates"}})
				svc := api.NewTestService("IamCredentials").
					WithPackage("google.iam.credentials.v1").
					WithSourceLocation("google/iam/credentials/v1/iamcredentials.proto", 20).
					WithMethods(method)
				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithPackageName("google.iam.credentials.v1")
				model.Name = "google-iam-credentials"
				cfg := &config.Library{
					Name: "google-iam-credentials",
				}
				return model, cfg
			},
			wantFilePath: "google/iam/credentials_v1/services/iam_credentials/async_client.py",
			wantContents: []string{
				"class IamCredentialsAsyncClient:",
				"async def sign_jwt(",
				"name: Optional[str] = None,",
				"delegates: Optional[MutableSequence[str]] = None,",
				"if name is not None:",
				"request.name = name",
				"if delegates:",
				"request.delegates.extend(delegates)",
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
