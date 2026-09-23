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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateClient(t *testing.T) {
	currentYear := fmt.Sprintf("%04d", time.Now().Year())

	for _, test := range []struct {
		name  string
		setup func() (*codec, *api.Service)
		want  *clientAnnotations
	}{
		{
			name: "basic service with unary method",
			setup: func() (*codec, *api.Service) {
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
				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.secretmanager.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &clientAnnotations{
				Name:               "SecretManagerService",
				ClientName:         "SecretManagerServiceClient",
				AsyncClientName:    "SecretManagerServiceAsyncClient",
				TransportClassName: "SecretManagerServiceTransport",
				DirectoryName:      "secret_manager_service",
				ServiceDocTitle:    "secret manager service",
				DocHead:            "Secret Manager Service API.",
				VersionPackage:     "google.cloud.secretmanager_v1",
				PackageImport:      "google.cloud",
				EndpointTemplate:   ".{UNIVERSE_DOMAIN}",
				ServiceFQN:         "google.cloud.secretmanager.v1.SecretManagerService",
				CopyrightYear:      currentYear,
				Methods: []*clientMethodAnnotations{
					{
						Name:              "get_secret",
						DocHead:           "Gets a secret.",
						RequestTypeHint:   "resources.GetSecretRequest",
						RequestSphinxType: "google.cloud.secretmanager_v1.types.GetSecretRequest",
						RequestTypeName:   "GetSecretRequest",
						ReturnType:        "resources.Secret",
						ReturnSphinxType:  "google.cloud.secretmanager_v1.types.Secret",
						PagerClassName:    "GetSecretPager",
						IsSimpleUnary:     true,
					},
				},
			},
		},
		{
			name: "service ending with Client",
			setup: func() (*codec, *api.Service) {
				svc := api.NewTestService("EchoClient").
					WithPackage("google.example.v1").
					WithSourceLocation("google/example/v1/service.proto", 1)
				model := api.NewTestAPI(nil, nil, []*api.Service{svc}).
					WithPackageName("google.example.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &clientAnnotations{
				Name:               "EchoClient",
				ClientName:         "EchoClient",
				AsyncClientName:    "EchoAsyncClient",
				TransportClassName: "EchoClientTransport",
				DirectoryName:      "echo_client",
				ServiceDocTitle:    "echo client",
				VersionPackage:     "google.example._v1",
				PackageImport:      "google.example",
				EndpointTemplate:   ".{UNIVERSE_DOMAIN}",
				ServiceFQN:         "google.example.v1.EchoClient",
				CopyrightYear:      currentYear,
			},
		},
		{
			name: "method with LRO",
			setup: func() (*codec, *api.Service) {
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
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &clientAnnotations{
				Name:               "CloudRedis",
				ClientName:         "CloudRedisClient",
				AsyncClientName:    "CloudRedisAsyncClient",
				TransportClassName: "CloudRedisTransport",
				DirectoryName:      "cloud_redis",
				ServiceDocTitle:    "cloud redis",
				VersionPackage:     "google.cloud.redis_v1",
				PackageImport:      "google.cloud",
				EndpointTemplate:   ".{UNIVERSE_DOMAIN}",
				ServiceFQN:         "google.cloud.redis.v1.CloudRedis",
				CopyrightYear:      currentYear,
				HasOperations:      true,
				HasLRO:             true,
				Methods: []*clientMethodAnnotations{
					{
						Name:                  "create_instance",
						RequestTypeHint:       "cloud_redis.CreateInstanceRequest",
						RequestSphinxType:     "google.cloud.redis_v1.types.CreateInstanceRequest",
						RequestTypeName:       "CreateInstanceRequest",
						IsLRO:                 true,
						PagerClassName:        "CreateInstancePager",
						ReturnType:            "operation.Operation",
						ReturnSphinxType:      "google.api_core.operation.Operation",
						OperationResponseType: "cloud_redis.Instance",
						OperationMetadataType: "cloud_redis.OperationMetadata",
					},
				},
			},
		},
		{
			name: "method with pagination",
			setup: func() (*codec, *api.Service) {
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
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &clientAnnotations{
				Name:               "ExampleService",
				ClientName:         "ExampleServiceClient",
				AsyncClientName:    "ExampleServiceAsyncClient",
				TransportClassName: "ExampleServiceTransport",
				DirectoryName:      "example_service",
				ServiceDocTitle:    "example service",
				VersionPackage:     "google.cloud.example_v1",
				PackageImport:      "google.cloud",
				EndpointTemplate:   ".{UNIVERSE_DOMAIN}",
				ServiceFQN:         "google.cloud.example.v1.ExampleService",
				CopyrightYear:      currentYear,
				HasPagers:          true,
				Methods: []*clientMethodAnnotations{
					{
						Name:              "list_items",
						RequestTypeHint:   "example.ListItemsRequest",
						RequestSphinxType: "google.cloud.example_v1.types.ListItemsRequest",
						RequestTypeName:   "ListItemsRequest",
						IsPaged:           true,
						PagerClassName:    "ListItemsPager",
						ReturnType:        "pagers.ListItemsPager",
						ReturnSphinxType:  "google.cloud.example_v1.services.example_service.pagers.ListItemsPager",
					},
				},
			},
		},
		{
			name: "method with Empty return",
			setup: func() (*codec, *api.Service) {
				req := api.NewTestMessage("DeleteSecretRequest").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 1)
				emptyResp := api.NewTestMessage("Empty").
					WithPackage("google.protobuf")
				emptyResp.ID = ".google.protobuf.Empty"
				method := api.NewTestMethod("DeleteSecret").
					WithInput(req).
					WithOutput(emptyResp)
				method.OutputTypeID = "google.protobuf.Empty"
				svc := api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 1).
					WithMethods(method)
				model := api.NewTestAPI([]*api.Message{req, emptyResp}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.secretmanager.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &clientAnnotations{
				Name:               "SecretManagerService",
				ClientName:         "SecretManagerServiceClient",
				AsyncClientName:    "SecretManagerServiceAsyncClient",
				TransportClassName: "SecretManagerServiceTransport",
				DirectoryName:      "secret_manager_service",
				ServiceDocTitle:    "secret manager service",
				VersionPackage:     "google.cloud.secretmanager_v1",
				PackageImport:      "google.cloud",
				EndpointTemplate:   ".{UNIVERSE_DOMAIN}",
				ServiceFQN:         "google.cloud.secretmanager.v1.SecretManagerService",
				CopyrightYear:      currentYear,
				Methods: []*clientMethodAnnotations{
					{
						Name:              "delete_secret",
						RequestTypeHint:   "resources.DeleteSecretRequest",
						RequestSphinxType: "google.cloud.secretmanager_v1.types.DeleteSecretRequest",
						RequestTypeName:   "DeleteSecretRequest",
						ReturnsEmpty:      true,
						PagerClassName:    "DeleteSecretPager",
						ReturnType:        "None",
					},
				},
			},
		},
		{
			name: "service with flattened params, routing header, and mixins",
			setup: func() (*codec, *api.Service) {
				parentField := api.NewTestField("parent").WithType(api.TypezString)
				req := api.NewTestMessage("ListThingsRequest").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/example.proto", 1).
					WithFields(parentField)
				resp := api.NewTestMessage("ListThingsResponse").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/example.proto", 10)
				method := api.NewTestMethod("ListThings").
					WithInput(req).
					WithOutput(resp)
				method.WithSignatures(&api.MethodSignature{Names: []string{"parent"}})
				method.Routing = []*api.RoutingInfo{
					{
						Name: "parent",
						Variants: []*api.RoutingInfoVariant{
							{FieldPath: []string{"parent"}},
						},
					},
				}
				mixinMethod := api.NewTestMethod("GetLocation")
				mixinMethod.SourceServiceID = "google.cloud.location.Locations"
				svc := api.NewTestService("ExampleService").
					WithPackage("google.cloud.example.v1").
					WithSourceLocation("google/cloud/example/v1/example.proto", 20).
					WithMethods(method, mixinMethod)
				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithPackageName("google.cloud.example.v1")
				c := newTestCodec(t, model, nil)
				return c, svc
			},
			want: &clientAnnotations{
				Name:               "ExampleService",
				ClientName:         "ExampleServiceClient",
				AsyncClientName:    "ExampleServiceAsyncClient",
				TransportClassName: "ExampleServiceTransport",
				DirectoryName:      "example_service",
				ServiceDocTitle:    "example service",
				VersionPackage:     "google.cloud.example_v1",
				PackageImport:      "google.cloud",
				EndpointTemplate:   ".{UNIVERSE_DOMAIN}",
				ServiceFQN:         "google.cloud.example.v1.ExampleService",
				CopyrightYear:      currentYear,
				HasLocationMixin:   true,
				HasGetLocation:     true,
				Methods: []*clientMethodAnnotations{
					{
						Name:                  "list_things",
						RequestTypeHint:       "example.ListThingsRequest",
						RequestSphinxType:     "google.cloud.example_v1.types.ListThingsRequest",
						RequestTypeName:       "ListThingsRequest",
						ReturnType:            "example.ListThingsResponse",
						ReturnSphinxType:      "google.cloud.example_v1.types.ListThingsResponse",
						IsSimpleUnary:         true,
						PagerClassName:        "ListThingsPager",
						HasFlattenedParams:    true,
						FlattenedParamsList:   "parent",
						HasRoutingHeader:      true,
						RoutingHeaderKey:      "parent",
						RoutingHeaderAccessor: "request.parent",
						FlattenedParams: []*flattenedParam{
							{
								Name:       "parent",
								TypeHint:   "str",
								SphinxType: "str",
								DocLines: []string{
									"This corresponds to the ``parent`` field",
									"on the ``request`` instance; if ``request`` is provided, this",
									"should not be set.",
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, svc := test.setup()
			if err := c.annotateModel(); err != nil {
				t.Fatal(err)
			}
			sAnn, ok := svc.Codec.(*serviceAnnotations)
			if !ok {
				t.Fatalf("svc.Codec got %T, want *serviceAnnotations", svc.Codec)
			}
			got := sAnn.Client
			if diff := cmp.Diff(test.want, got,
				cmpopts.IgnoreFields(clientAnnotations{}, "Service", "TypeImports", "ExternalImports", "CustomResourcePaths", "DefaultHost", "DocBody", "RestAsyncIOEnabled", "VersionSegment", "HasMultiLineDoc", "ShowRestBetaPreview", "HasGRPCTransport", "HasRESTTransport"),
				cmpopts.IgnoreFields(clientMethodAnnotations{}, "Method", "DocLines", "DocBody", "HasDocHead", "HasDocBody", "RequestDocLines", "ReturnDocLines", "HasReturnDoc", "HasReturnDocTrailer", "SamplePreInitLines", "HasSamplePreInit", "SampleRequestArgs", "PackageImport", "VersionSegment", "ClientName", "AsyncReturnType", "AsyncReturnSphinxType", "AsyncPagerClassName", "RoutingHeaders"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateClient_Error(t *testing.T) {
	tempDir := t.TempDir()
	pkgDir := filepath.Join(tempDir, "google/example/v1")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corruptedYAML := filepath.Join(pkgDir, "service_v1.yaml")
	if err := os.WriteFile(corruptedYAML, []byte("invalid: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := api.NewTestService("FailingService").
		WithPackage("google.example.v1").
		WithSourceLocation("google/example/v1/service.proto", 1)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc}).
		WithPackageName("google.example.v1")
	lib := &config.Library{
		Roots: []string{tempDir},
	}
	c := newTestCodec(t, model, lib)

	err := c.annotateModel()
	if !errors.Is(err, ErrLoadServiceConfig) {
		t.Errorf("annotateModel() error = %v, want errors.Is %v", err, ErrLoadServiceConfig)
	}
}

func TestAnnotateClient_ShowRestBetaPreview(t *testing.T) {
	for _, test := range []struct {
		name        string
		optArgs     map[string][]string
		wantPreview bool
	}{
		{
			name:        "default_without_opt_args",
			optArgs:     nil,
			wantPreview: true,
		},
		{
			name: "with_rest_numeric_enums",
			optArgs: map[string][]string{
				"google/example/v1": {"rest-numeric-enums"},
			},
			wantPreview: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService("ExampleService").
				WithPackage("google.example.v1")
			model := api.NewTestAPI(nil, nil, []*api.Service{svc}).
				WithPackageName("google.example.v1")
			lib := &config.Library{
				Python: &config.PythonPackage{
					OptArgsByAPI: test.optArgs,
				},
			}
			c := newTestCodec(t, model, lib)
			if err := c.annotateModel(); err != nil {
				t.Fatal(err)
			}
			sAnn, ok := svc.Codec.(*serviceAnnotations)
			if !ok {
				t.Fatalf("svc.Codec got %T, want *serviceAnnotations", svc.Codec)
			}
			if sAnn.Client.ShowRestBetaPreview != test.wantPreview {
				t.Errorf("ShowRestBetaPreview = %v, want %v", sAnn.Client.ShowRestBetaPreview, test.wantPreview)
			}
		})
	}
}

func TestAnnotateClient_IAMPolicyMixin(t *testing.T) {
	setIamPolicy := api.NewTestMethod("SetIamPolicy")
	setIamPolicy.SourceServiceID = ".google.iam.v1.IAMPolicy"
	svc := api.NewTestService("ExampleService").
		WithPackage("google.example.v1").
		WithMethods(setIamPolicy)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc}).
		WithPackageName("google.example.v1")
	c := newTestCodec(t, model, &config.Library{})
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}
	sAnn := svc.Codec.(*serviceAnnotations)
	if !sAnn.Client.HasIAMPolicyMixin || !sAnn.Client.HasSetIamPolicy {
		t.Errorf("got HasIAMPolicyMixin=%v, HasSetIamPolicy=%v, want true, true",
			sAnn.Client.HasIAMPolicyMixin, sAnn.Client.HasSetIamPolicy)
	}
}
