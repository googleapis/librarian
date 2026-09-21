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

func TestGenerateMixins(t *testing.T) {
	for _, test := range []struct {
		name                 string
		setup                func() (*api.API, *config.Library)
		clientPath           string
		asyncClientPath      string
		restTransportPath    string
		wantClientTokens     []string
		wantAsyncTokens      []string
		wantRestTokens       []string
		unwantedClientTokens []string
		unwantedAsyncTokens  []string
	}{
		{
			name: "service with location and operations mixins",
			setup: func() (*api.API, *config.Library) {
				req := api.NewTestMessage("GetItemRequest").WithSourceLocation("example.proto", 1)
				resp := api.NewTestMessage("Item").WithSourceLocation("example.proto", 5)

				mainMeth := api.NewTestMethod("GetItem").
					WithInput(req).
					WithOutput(resp)

				getLocMeth := api.NewTestMethod("GetLocation")
				getLocMeth.SourceServiceID = ".google.cloud.location.Locations"
				getLocMeth.InputTypeID = ".google.cloud.location.GetLocationRequest"
				getLocMeth.OutputTypeID = ".google.cloud.location.Location"

				listLocMeth := api.NewTestMethod("ListLocations")
				listLocMeth.SourceServiceID = ".google.cloud.location.Locations"
				listLocMeth.InputTypeID = ".google.cloud.location.ListLocationsRequest"
				listLocMeth.OutputTypeID = ".google.cloud.location.ListLocationsResponse"

				getOpMeth := api.NewTestMethod("GetOperation")
				getOpMeth.SourceServiceID = ".google.longrunning.Operations"
				getOpMeth.InputTypeID = ".google.longrunning.GetOperationRequest"
				getOpMeth.OutputTypeID = ".google.longrunning.Operation"

				listOpMeth := api.NewTestMethod("ListOperations")
				listOpMeth.SourceServiceID = ".google.longrunning.Operations"
				listOpMeth.InputTypeID = ".google.longrunning.ListOperationsRequest"
				listOpMeth.OutputTypeID = ".google.longrunning.ListOperationsResponse"

				cancelOpMeth := api.NewTestMethod("CancelOperation")
				cancelOpMeth.SourceServiceID = ".google.longrunning.Operations"
				cancelOpMeth.InputTypeID = ".google.longrunning.CancelOperationRequest"
				cancelOpMeth.OutputTypeID = ".google.protobuf.Empty"

				deleteOpMeth := api.NewTestMethod("DeleteOperation")
				deleteOpMeth.SourceServiceID = ".google.longrunning.Operations"
				deleteOpMeth.InputTypeID = ".google.longrunning.DeleteOperationRequest"
				deleteOpMeth.OutputTypeID = ".google.protobuf.Empty"

				waitOpMeth := api.NewTestMethod("WaitOperation")
				waitOpMeth.SourceServiceID = ".google.longrunning.Operations"
				waitOpMeth.InputTypeID = ".google.longrunning.WaitOperationRequest"
				waitOpMeth.OutputTypeID = ".google.longrunning.Operation"

				svc := api.NewTestService("MixService").
					WithPackage("google.cloud.mix.v1").
					WithSourceLocation("example.proto", 10).
					WithMethods(mainMeth, getLocMeth, listLocMeth, getOpMeth, listOpMeth, cancelOpMeth, deleteOpMeth, waitOpMeth)
				svc.DefaultHost = "mix.googleapis.com"

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.mix.v1")
				model.Name = "google-cloud-mix"

				cfg := &config.Library{
					Name: "google-cloud-mix",
				}
				return model, cfg
			},
			clientPath:        "google/cloud/mix_v1/services/mix_service/client.py",
			asyncClientPath:   "google/cloud/mix_v1/services/mix_service/async_client.py",
			restTransportPath: "google/cloud/mix_v1/services/mix_service/transports/rest.py",
			wantClientTokens: []string{
				"def get_location(",
				"def list_locations(",
				"def get_operation(",
				"def list_operations(",
				"def cancel_operation(",
				"def delete_operation(",
				"def wait_operation(",
				"rpc = self._transport._wrapped_methods[self._transport.get_location]",
				"rpc = self._transport._wrapped_methods[self._transport.get_operation]",
				"from google.cloud.location import locations_pb2",
				"from google.longrunning import operations_pb2",
			},
			wantAsyncTokens: []string{
				"async def get_location(",
				"async def list_locations(",
				"async def get_operation(",
				"async def list_operations(",
				"async def cancel_operation(",
				"async def delete_operation(",
				"async def wait_operation(",
				"rpc = self.transport._wrapped_methods[self._client._transport.get_location]",
				"rpc = self.transport._wrapped_methods[self._client._transport.get_operation]",
			},
			wantRestTokens: []string{
				"def pre_get_location(",
				"def post_get_location(",
				"def get_location(self):",
				"class _GetLocation(_BaseMixServiceRestTransport._BaseGetLocation",
			},
		},
		{
			name: "service without mixins omits mixin methods and imports",
			setup: func() (*api.API, *config.Library) {
				req := api.NewTestMessage("SimpleRequest").WithSourceLocation("simple.proto", 1)
				resp := api.NewTestMessage("SimpleResponse").WithSourceLocation("simple.proto", 5)

				mainMeth := api.NewTestMethod("DoSimple").
					WithInput(req).
					WithOutput(resp)

				svc := api.NewTestService("SimpleService").
					WithPackage("google.cloud.simple.v1").
					WithSourceLocation("simple.proto", 10).
					WithMethods(mainMeth)
				svc.DefaultHost = "simple.googleapis.com"

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.simple.v1")
				model.Name = "google-cloud-simple"

				cfg := &config.Library{
					Name: "google-cloud-simple",
				}
				return model, cfg
			},
			clientPath:      "google/cloud/simple_v1/services/simple_service/client.py",
			asyncClientPath: "google/cloud/simple_v1/services/simple_service/async_client.py",
			unwantedClientTokens: []string{
				"def get_location(",
				"def list_locations(",
				"def get_operation(",
				"def list_operations(",
				"def cancel_operation(",
				"def delete_operation(",
				"def wait_operation(",
				"from google.cloud.location import locations_pb2",
				"from google.longrunning import operations_pb2",
			},
			unwantedAsyncTokens: []string{
				"async def get_location(",
				"async def list_locations(",
				"async def get_operation(",
				"async def list_operations(",
				"async def cancel_operation(",
				"async def delete_operation(",
				"async def wait_operation(",
				"from google.cloud.location import locations_pb2",
				"from google.longrunning import operations_pb2",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			model, cfg := test.setup()
			if err := Generate(t.Context(), model, outDir, cfg); err != nil {
				t.Fatal(err)
			}

			if test.clientPath != "" {
				content, err := os.ReadFile(filepath.Join(outDir, test.clientPath))
				if err != nil {
					t.Fatal(err)
				}
				clientSrc := string(content)
				for _, token := range test.wantClientTokens {
					if !strings.Contains(clientSrc, token) {
						t.Errorf("Generate() output file %s missing expected token %q", test.clientPath, token)
					}
				}
				for _, token := range test.unwantedClientTokens {
					if strings.Contains(clientSrc, token) {
						t.Errorf("Generate() output file %s contains unexpected token %q", test.clientPath, token)
					}
				}
			}

			if test.asyncClientPath != "" {
				content, err := os.ReadFile(filepath.Join(outDir, test.asyncClientPath))
				if err != nil {
					t.Fatal(err)
				}
				asyncSrc := string(content)
				for _, token := range test.wantAsyncTokens {
					if !strings.Contains(asyncSrc, token) {
						t.Errorf("Generate() output file %s missing expected token %q", test.asyncClientPath, token)
					}
				}
				for _, token := range test.unwantedAsyncTokens {
					if strings.Contains(asyncSrc, token) {
						t.Errorf("Generate() output file %s contains unexpected token %q", test.asyncClientPath, token)
					}
				}
			}

			if test.restTransportPath != "" {
				content, err := os.ReadFile(filepath.Join(outDir, test.restTransportPath))
				if err != nil {
					t.Fatal(err)
				}
				restSrc := string(content)
				for _, token := range test.wantRestTokens {
					if !strings.Contains(restSrc, token) {
						t.Errorf("Generate() output file %s missing expected token %q", test.restTransportPath, token)
					}
				}
			}
		})
	}
}
