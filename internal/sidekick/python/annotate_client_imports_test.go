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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestCollectClientImports(t *testing.T) {
	for _, test := range []struct {
		name            string
		setup           func() (*codec, *api.Service)
		wantTypeImports []*clientTypeImport
		wantExtImports  []*clientExternalImport
	}{
		{
			name: "service with same-package and external types",
			setup: func() (*codec, *api.Service) {
				req := api.NewTestMessage("GetThingRequest").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/resources.proto", 1)
				resp := api.NewTestMessage("Thing").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/resources.proto", 10)
				extMsg := api.NewTestMessage("ExternalConfig").
					WithPackage("google.cloud.other.v1").
					WithSourceLocation("google/cloud/other/v1/config.proto", 5)

				field := api.NewTestField("config").
					WithType(api.TypezMessage).
					WithMessageType(extMsg)
				resp.Fields = []*api.Field{field}

				method := api.NewTestMethod("GetThing").
					WithInput(req).
					WithOutput(resp)
				svc := api.NewTestService("ExampleService").
					WithPackage("google.cloud.example.v1").
					WithMethods(method)

				model := api.NewTestAPI([]*api.Message{req, resp, extMsg}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.example.v1")
				return newTestCodec(t, model, nil), svc
			},
			wantTypeImports: []*clientTypeImport{
				{Stem: "resources"},
			},
			wantExtImports: []*clientExternalImport{
				{
					Module: "google.cloud.other.v1.config_pb2",
					Alias:  "config_pb2",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, svc := test.setup()
			gotTypes, gotExt := c.collectClientImports(svc)
			if diff := cmp.Diff(test.wantTypeImports, gotTypes); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantExtImports, gotExt); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
