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

func TestAnnotatePagers(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func() (*codec, *api.Service)
		want  *pagersAnnotation
	}{
		{
			name: "nil service",
			setup: func() (*codec, *api.Service) {
				c := newTestCodec(t, api.NewTestAPI(nil, nil, nil), nil)
				return c, nil
			},
			want: &pagersAnnotation{
				HasPagers: false,
			},
		},
		{
			name: "service with no paged methods",
			setup: func() (*codec, *api.Service) {
				req := api.NewTestMessage("GetSecretRequest")
				resp := api.NewTestMessage("Secret")
				svc := api.NewTestService("SecretManagerService").
					WithMethods(api.NewTestMethod("GetSecret").WithInput(req).WithOutput(resp))
				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.secretmanager.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &pagersAnnotation{
				Pagers:      nil,
				TypeImports: nil,
				HasPagers:   false,
			},
		},
		{
			name: "service with nil method",
			setup: func() (*codec, *api.Service) {
				model := api.NewTestAPI(nil, nil, nil).
					WithPackageName("google.cloud.secretmanager.v1")
				svc := api.NewTestService("SecretManagerService")
				svc.Methods = []*api.Method{nil}
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &pagersAnnotation{
				Pagers:      nil,
				TypeImports: nil,
				HasPagers:   false,
			},
		},
		{
			name: "top-level item type message",
			setup: func() (*codec, *api.Service) {
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
				method := api.NewTestMethod("ListSecrets").
					WithInput(reqMsg).
					WithOutput(respMsg).
					WithPagination(pageTokenField)
				svc := api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/secretmanager.proto", 40).
					WithMethods(method)
				model := api.NewTestAPI([]*api.Message{secretMsg, reqMsg, respMsg}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.secretmanager.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &pagersAnnotation{
				Pagers: []*pagerAnnotations{
					{
						MethodNameSnake:    "list_secrets",
						MethodNamePascal:   "ListSecrets",
						RequestType:        "secretmanager.ListSecretsRequest",
						RequestDocType:     "google.cloud.secretmanager_v1.types.ListSecretsRequest",
						ResponseType:       "secretmanager.ListSecretsResponse",
						ResponseDocType:    "google.cloud.secretmanager_v1.types.ListSecretsResponse",
						ItemType:           "secretmanager.Secret",
						PageTokenField:     "page_token",
						NextPageTokenField: "next_page_token",
						PageableItemField:  "secrets",
					},
				},
				TypeImports: []*pagerTypeImport{
					{
						Package: "google.cloud.secretmanager_v1.types",
						Module:  "secretmanager",
					},
				},
				HasPagers: true,
			},
		},
		{
			name: "nested item type message",
			setup: func() (*codec, *api.Service) {
				parentResp := api.NewTestMessage("AnalyzeOrgPoliciesResponse").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 10)
				nestedResult := api.NewTestMessage("OrgPolicyResult").
					WithPackage("google.cloud.asset.v1.AnalyzeOrgPoliciesResponse").
					WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 20)

				resultsField := api.NewTestField("org_policy_results").
					WithMessageType(nestedResult).
					WithRepeated()
				nextPageTokenField := api.NewTestField("next_page_token").
					WithType(api.TypezString)
				parentResp.WithFields(resultsField, nextPageTokenField).
					WithPagination(nextPageTokenField, resultsField)

				pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
				reqMsg := api.NewTestMessage("AnalyzeOrgPoliciesRequest").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 30).
					WithFields(pageTokenField)
				method := api.NewTestMethod("AnalyzeOrgPolicies").
					WithInput(reqMsg).
					WithOutput(parentResp).
					WithPagination(pageTokenField)
				svc := api.NewTestService("AssetService").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 40).
					WithMethods(method)
				model := api.NewTestAPI([]*api.Message{parentResp, nestedResult, reqMsg}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.asset.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &pagersAnnotation{
				Pagers: []*pagerAnnotations{
					{
						MethodNameSnake:    "analyze_org_policies",
						MethodNamePascal:   "AnalyzeOrgPolicies",
						RequestType:        "asset_service.AnalyzeOrgPoliciesRequest",
						RequestDocType:     "google.cloud.asset_v1.types.AnalyzeOrgPoliciesRequest",
						ResponseType:       "asset_service.AnalyzeOrgPoliciesResponse",
						ResponseDocType:    "google.cloud.asset_v1.types.AnalyzeOrgPoliciesResponse",
						ItemType:           "asset_service.AnalyzeOrgPoliciesResponse.OrgPolicyResult",
						PageTokenField:     "page_token",
						NextPageTokenField: "next_page_token",
						PageableItemField:  "org_policy_results",
					},
				},
				TypeImports: []*pagerTypeImport{
					{
						Package: "google.cloud.asset_v1.types",
						Module:  "asset_service",
					},
				},
				HasPagers: true,
			},
		},
		{
			name: "message from a different proto file",
			setup: func() (*codec, *api.Service) {
				assetMsg := api.NewTestMessage("Asset").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/assets.proto", 1)
				assetsField := api.NewTestField("assets").
					WithMessageType(assetMsg).
					WithRepeated()
				nextPageTokenField := api.NewTestField("next_page_token").
					WithType(api.TypezString)
				respMsg := api.NewTestMessage("ListAssetsResponse").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 10).
					WithFields(assetsField, nextPageTokenField).
					WithPagination(nextPageTokenField, assetsField)
				pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
				reqMsg := api.NewTestMessage("ListAssetsRequest").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 20).
					WithFields(pageTokenField)
				method := api.NewTestMethod("ListAssets").
					WithInput(reqMsg).
					WithOutput(respMsg).
					WithPagination(pageTokenField)
				svc := api.NewTestService("AssetService").
					WithPackage("google.cloud.asset.v1").
					WithSourceLocation("google/cloud/asset/v1/asset_service.proto", 30).
					WithMethods(method)
				model := api.NewTestAPI([]*api.Message{assetMsg, reqMsg, respMsg}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.asset.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &pagersAnnotation{
				Pagers: []*pagerAnnotations{
					{
						MethodNameSnake:    "list_assets",
						MethodNamePascal:   "ListAssets",
						RequestType:        "asset_service.ListAssetsRequest",
						RequestDocType:     "google.cloud.asset_v1.types.ListAssetsRequest",
						ResponseType:       "asset_service.ListAssetsResponse",
						ResponseDocType:    "google.cloud.asset_v1.types.ListAssetsResponse",
						ItemType:           "assets.Asset",
						PageTokenField:     "page_token",
						NextPageTokenField: "next_page_token",
						PageableItemField:  "assets",
					},
				},
				TypeImports: []*pagerTypeImport{
					{
						Package: "google.cloud.asset_v1.types",
						Module:  "asset_service",
					},
					{
						Package: "google.cloud.asset_v1.types",
						Module:  "assets",
					},
				},
				HasPagers: true,
			},
		},
		{
			name: "primitive item type",
			setup: func() (*codec, *api.Service) {
				namesField := api.NewTestField("names").
					WithType(api.TypezString).
					WithRepeated()
				nextPageTokenField := api.NewTestField("next_page_token").
					WithType(api.TypezString)
				respMsg := api.NewTestMessage("ListNamesResponse").
					WithPackage("google.cloud.foo.v1").
					WithSourceLocation("google/cloud/foo/v1/foo.proto", 10).
					WithFields(namesField, nextPageTokenField).
					WithPagination(nextPageTokenField, namesField)
				pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
				reqMsg := api.NewTestMessage("ListNamesRequest").
					WithPackage("google.cloud.foo.v1").
					WithSourceLocation("google/cloud/foo/v1/foo.proto", 20).
					WithFields(pageTokenField)
				method := api.NewTestMethod("ListNames").
					WithInput(reqMsg).
					WithOutput(respMsg).
					WithPagination(pageTokenField)
				svc := api.NewTestService("FooService").
					WithPackage("google.cloud.foo.v1").
					WithSourceLocation("google/cloud/foo/v1/foo.proto", 30).
					WithMethods(method)
				model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.foo.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &pagersAnnotation{
				Pagers: []*pagerAnnotations{
					{
						MethodNameSnake:    "list_names",
						MethodNamePascal:   "ListNames",
						RequestType:        "foo.ListNamesRequest",
						RequestDocType:     "google.cloud.foo_v1.types.ListNamesRequest",
						ResponseType:       "foo.ListNamesResponse",
						ResponseDocType:    "google.cloud.foo_v1.types.ListNamesResponse",
						ItemType:           "str",
						PageTokenField:     "page_token",
						NextPageTokenField: "next_page_token",
						PageableItemField:  "names",
					},
				},
				TypeImports: []*pagerTypeImport{
					{
						Package: "google.cloud.foo_v1.types",
						Module:  "foo",
					},
				},
				HasPagers: true,
			},
		},
		{
			name: "mixin method exclusion",
			setup: func() (*codec, *api.Service) {
				itemField := api.NewTestField("items").
					WithType(api.TypezString).
					WithRepeated()
				nextPageToken := api.NewTestField("next_page_token").
					WithType(api.TypezString)
				respMsg := api.NewTestMessage("ListOperationsResponse").
					WithFields(itemField, nextPageToken).
					WithPagination(nextPageToken, itemField)
				reqMsg := api.NewTestMessage("ListOperationsRequest")

				opSvc := api.NewTestService("Operations").WithPackage("google.longrunning")
				opMethod := api.NewTestMethod("ListOperations").
					WithInput(reqMsg).
					WithOutput(respMsg)
				opSvc.WithMethods(opMethod)

				mixinMethod := api.NewTestMethod("ListOperations").WithSourceMethod(opMethod)
				svc := api.NewTestService("FooService").
					WithPackage("google.cloud.foo.v1").
					WithMethods(mixinMethod)
				model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc, opSvc}).
					WithPackageName("google.cloud.foo.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &pagersAnnotation{
				Pagers:      nil,
				TypeImports: nil,
				HasPagers:   false,
			},
		},
		{
			name: "enum item type",
			setup: func() (*codec, *api.Service) {
				stateEnum := api.NewTestEnum("SecretVersionState").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/enums.proto", 10)
				statesField := api.NewTestField("states").
					WithType(api.TypezEnum).
					WithTypezID(stateEnum.ID).
					WithRepeated()
				nextPageTokenField := api.NewTestField("next_page_token").
					WithType(api.TypezString)
				respMsg := api.NewTestMessage("ListSecretStatesResponse").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 10).
					WithFields(statesField, nextPageTokenField).
					WithPagination(nextPageTokenField, statesField)
				pageTokenField := api.NewTestField("page_token").
					WithType(api.TypezString)
				reqMsg := api.NewTestMessage("ListSecretStatesRequest").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 20).
					WithFields(pageTokenField)
				method := api.NewTestMethod("ListSecretStates").
					WithInput(reqMsg).
					WithOutput(respMsg).
					WithPagination(pageTokenField)
				svc := api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 30).
					WithMethods(method)
				model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, []*api.Enum{stateEnum}, []*api.Service{svc}).
					WithPackageName("google.cloud.secretmanager.v1")
				if err := api.CrossReference(model); err != nil {
					t.Fatal(err)
				}
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &pagersAnnotation{
				Pagers: []*pagerAnnotations{
					{
						MethodNameSnake:    "list_secret_states",
						MethodNamePascal:   "ListSecretStates",
						RequestType:        "service.ListSecretStatesRequest",
						RequestDocType:     "google.cloud.secretmanager_v1.types.ListSecretStatesRequest",
						ResponseType:       "service.ListSecretStatesResponse",
						ResponseDocType:    "google.cloud.secretmanager_v1.types.ListSecretStatesResponse",
						ItemType:           "enums.SecretVersionState",
						PageTokenField:     "page_token",
						NextPageTokenField: "next_page_token",
						PageableItemField:  "states",
					},
				},
				TypeImports: []*pagerTypeImport{
					{
						Package: "google.cloud.secretmanager_v1.types",
						Module:  "enums",
					},
					{
						Package: "google.cloud.secretmanager_v1.types",
						Module:  "service",
					},
				},
				HasPagers: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, svc := test.setup()
			got := c.annotatePagers(svc)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
