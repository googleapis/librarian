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
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService(t *testing.T) {
	req := api.NewTestMessage("GetSecretRequest")
	resp := api.NewTestMessage("Secret")

	currentYear := fmt.Sprintf("%04d", time.Now().Year())

	for _, test := range []struct {
		name string
		svc  *api.Service
		want *serviceAnnotations
	}{
		{
			name: "basic service",
			svc: func() *api.Service {
				s := api.NewTestService("SecretManagerService").
					WithMethods(
						api.NewTestMethod("GetSecret").WithInput(req).WithOutput(resp),
					)
				s.Documentation = "Secret Manager Service API."
				return s
			}(),
			want: &serviceAnnotations{
				Name:            "SecretManagerService",
				ProtoName:       "SecretManagerService",
				ClientName:      "SecretManagerServiceClient",
				AsyncClientName: "SecretManagerServiceAsyncClient",
				DirectoryName:   "secret_manager_service",
				CopyrightYear:   currentYear,
				DocLines:        []string{"Secret Manager Service API."},
				Methods: []*methodAnnotations{
					{
						Name:           "get_secret",
						ProtoName:      "GetSecret",
						InputTypeName:  "GetSecretRequest",
						OutputTypeName: "Secret",
					},
				},
			},
		},
		{
			name: "service ending with Client",
			svc:  api.NewTestService("EchoClient"),
			want: &serviceAnnotations{
				Name:            "EchoClient",
				ProtoName:       "EchoClient",
				ClientName:      "EchoClient",
				AsyncClientName: "EchoAsyncClient",
				DirectoryName:   "echo_client",
				CopyrightYear:   currentYear,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{test.svc})
			c := newTestCodec(t, model, nil)
			if err := c.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann, ok := test.svc.Codec.(*serviceAnnotations)
			if !ok {
				t.Fatalf("got %T, want *serviceAnnotations", test.svc.Codec)
			}
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(serviceAnnotations{}, "Model", "Service", "Transport", "Client"),
				cmpopts.IgnoreFields(methodAnnotations{}, "Service", "Method"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateService_PagedMethodAttachment(t *testing.T) {
	secretMsg := api.NewTestMessage("Secret").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/secretmanager.proto", 10)
	secretsField := api.NewTestField("secrets").
		WithMessageType(secretMsg).
		WithRepeated()
	nextPageTokenField := api.NewTestField("next_page_token").
		WithType(api.TypezString)
	respMsg := api.NewTestMessage("ListSecretsResponse").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/secretmanager.proto", 20).
		WithFields(secretsField, nextPageTokenField).
		WithPagination(nextPageTokenField, secretsField)
	pageTokenField := api.NewTestField("page_token").
		WithType(api.TypezString)
	reqMsg := api.NewTestMessage("ListSecretsRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/secretmanager.proto", 30).
		WithFields(pageTokenField)
	pagedMethod := api.NewTestMethod("ListSecrets").
		WithInput(reqMsg).
		WithOutput(respMsg).
		WithPagination(pageTokenField)

	getSecretReq := api.NewTestMessage("GetSecretRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/secretmanager.proto", 40)
	unaryMethod := api.NewTestMethod("GetSecret").
		WithInput(getSecretReq).
		WithOutput(secretMsg)

	svc := api.NewTestService("SecretManagerService").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/secretmanager.proto", 50).
		WithMethods(pagedMethod, unaryMethod)

	model := api.NewTestAPI([]*api.Message{secretMsg, reqMsg, respMsg, getSecretReq}, nil, []*api.Service{svc}).
		WithPackageName("google.cloud.secretmanager.v1")
	c := newTestCodec(t, model, nil)

	modelAnn := &modelAnnotations{CopyrightYear: "2026"}
	if err := c.annotateService(svc, modelAnn); err != nil {
		t.Fatal(err)
	}

	sAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok {
		t.Fatalf("svc.Codec is %T, want *serviceAnnotations", svc.Codec)
	}
	if !sAnn.HasPagers {
		t.Errorf("sAnn.HasPagers = false, want true")
	}
	if len(sAnn.Pagers) != 1 {
		t.Errorf("len(sAnn.Pagers) = %d, want 1", len(sAnn.Pagers))
	}
	if len(sAnn.PagerTypeImports) == 0 {
		t.Errorf("len(sAnn.PagerTypeImports) = 0, want > 0")
	}

	pagedAnn, ok := pagedMethod.Codec.(*methodAnnotations)
	if !ok {
		t.Fatalf("pagedMethod.Codec is %T, want *methodAnnotations", pagedMethod.Codec)
	}
	if !pagedAnn.IsPaged {
		t.Errorf("pagedMethod.Codec.IsPaged = false, want true")
	}
	if pagedAnn.Pager == nil {
		t.Errorf("pagedMethod.Codec.Pager = nil, want non-nil")
	}

	unaryAnn, ok := unaryMethod.Codec.(*methodAnnotations)
	if !ok {
		t.Fatalf("unaryMethod.Codec is %T, want *methodAnnotations", unaryMethod.Codec)
	}
	if unaryAnn.IsPaged {
		t.Errorf("unaryMethod.Codec.IsPaged = true, want false")
	}
	if unaryAnn.Pager != nil {
		t.Errorf("unaryMethod.Codec.Pager = %v, want nil", unaryAnn.Pager)
	}
}
