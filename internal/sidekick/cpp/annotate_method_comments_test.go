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

package cpp

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMethod_GetOperationComments(t *testing.T) {
	nameField := api.NewTestField("name").WithType(api.TypezString)
	reqMsg := api.NewTestMessage("GetOperationRequest").WithPackage("google.longrunning").WithFields(nameField)
	respMsg := api.NewTestMessage("Operation").WithPackage("google.longrunning")

	method := api.NewTestMethod("GetOperation").
		WithInput(reqMsg).
		WithOutput(respMsg)
	method.SourceServiceID = "google.longrunning.Operations"
	method.Documentation = "Provides the [Operations][google.longrunning.Operations] service functionality in this service."

	paramComment := formatParameterComment(reqMsg, nameField)
	wantParamComment := "  /// @param name  The name of the operation resource.\n"
	if diff := cmp.Diff(wantParamComment, paramComment); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	docComment := formatMethodDoxygenComments(method, "", nil, "", false, nil, false, false)
	wantSubstr := "Gets the latest state of a long-running operation."
	if !strings.Contains(docComment, wantSubstr) {
		t.Errorf("formatMethodDoxygenComments expected to contain %q, got %s", wantSubstr, docComment)
	}
}

func TestFormatMethodDoxygenComments_IAMAndCustom(t *testing.T) {
	for _, test := range []struct {
		name       string
		method     *api.Method
		wantSubstr string
	}{
		{
			name: "IAM mixin SetIamPolicy fallback comments",
			method: func() *api.Method {
				m := api.NewTestMethod("SetIamPolicy")
				m.SourceServiceID = "google.iam.v1.IAMPolicy"
				m.InputTypeID = ".google.iam.v1.SetIamPolicyRequest"
				m.OutputTypeID = ".google.iam.v1.Policy"
				return m
			}(),
			wantSubstr: "Sets the access control policy on the specified resource. Replaces any",
		},
		{
			name: "IAM mixin SetIamPolicy fallback comments without leading dot in InputTypeID",
			method: func() *api.Method {
				m := api.NewTestMethod("SetIamPolicy")
				m.SourceServiceID = "google.iam.v1.IAMPolicy"
				m.InputTypeID = "google.iam.v1.SetIamPolicyRequest"
				m.OutputTypeID = "google.iam.v1.Policy"
				return m
			}(),
			wantSubstr: "Sets the access control policy on the specified resource. Replaces any",
		},
		{
			name: "IAM mixin SetIamPolicy fallback comments replaces generic mixin documentation",
			method: func() *api.Method {
				m := api.NewTestMethod("SetIamPolicy")
				m.SourceServiceID = "google.iam.v1.IAMPolicy"
				m.InputTypeID = ".google.iam.v1.SetIamPolicyRequest"
				m.OutputTypeID = ".google.iam.v1.Policy"
				m.Documentation = "Provides the [IAMPolicy][google.iam.v1.IAMPolicy] service functionality in this service."
				return m
			}(),
			wantSubstr: "Sets the access control policy on the specified resource. Replaces any",
		},
		{
			name: "IAM mixin GetIamPolicy fallback comments",
			method: func() *api.Method {
				m := api.NewTestMethod("GetIamPolicy")
				m.SourceServiceID = "google.iam.v1.IAMPolicy"
				m.InputTypeID = ".google.iam.v1.GetIamPolicyRequest"
				m.OutputTypeID = ".google.iam.v1.Policy"
				return m
			}(),
			wantSubstr: "Gets the access control policy for a resource.",
		},
		{
			name: "IAM mixin GetIamPolicy fallback comments replaces generic mixin documentation",
			method: func() *api.Method {
				m := api.NewTestMethod("GetIamPolicy")
				m.SourceServiceID = "google.iam.v1.IAMPolicy"
				m.InputTypeID = ".google.iam.v1.GetIamPolicyRequest"
				m.OutputTypeID = ".google.iam.v1.Policy"
				m.Documentation = "Provides the [IAMPolicy][google.iam.v1.IAMPolicy] service functionality in this service."
				return m
			}(),
			wantSubstr: "Gets the access control policy for a resource.",
		},
		{
			name: "IAM mixin TestIamPermissions fallback comments",
			method: func() *api.Method {
				m := api.NewTestMethod("TestIamPermissions")
				m.SourceServiceID = "google.iam.v1.IAMPolicy"
				m.InputTypeID = ".google.iam.v1.TestIamPermissionsRequest"
				m.OutputTypeID = ".google.iam.v1.TestIamPermissionsResponse"
				return m
			}(),
			wantSubstr: "Returns permissions that a caller has on the specified resource.",
		},
		{
			name: "IAM mixin TestIamPermissions fallback comments replaces generic mixin documentation",
			method: func() *api.Method {
				m := api.NewTestMethod("TestIamPermissions")
				m.SourceServiceID = "google.iam.v1.IAMPolicy"
				m.InputTypeID = ".google.iam.v1.TestIamPermissionsRequest"
				m.OutputTypeID = ".google.iam.v1.TestIamPermissionsResponse"
				m.Documentation = "Provides the [IAMPolicy][google.iam.v1.IAMPolicy] service functionality in this service."
				return m
			}(),
			wantSubstr: "Returns permissions that a caller has on the specified resource.",
		},
		{
			name: "non-IAM method preserves custom comments",
			method: func() *api.Method {
				m := api.NewTestMethod("CreateSecret")
				m.SourceServiceID = "google.cloud.secretmanager.v1.SecretManagerService"
				m.InputTypeID = ".google.cloud.secretmanager.v1.CreateSecretRequest"
				m.OutputTypeID = ".google.cloud.secretmanager.v1.Secret"
				m.Documentation = "Creates a new Secret containing no SecretVersions."
				return m
			}(),
			wantSubstr: "Creates a new Secret containing no SecretVersions.",
		},
		{
			name: "method named SetIamPolicy without IAM prefix preserves custom comments",
			method: func() *api.Method {
				m := api.NewTestMethod("SetIamPolicy")
				m.SourceServiceID = "custom.Service"
				m.InputTypeID = ".custom.SetIamPolicyRequest"
				m.OutputTypeID = ".custom.Policy"
				m.Documentation = "Custom policy setter documentation."
				return m
			}(),
			wantSubstr: "Custom policy setter documentation.",
		},
		{
			name: "method named GetIamPolicy without IAM prefix preserves custom comments",
			method: func() *api.Method {
				m := api.NewTestMethod("GetIamPolicy")
				m.SourceServiceID = "custom.Service"
				m.InputTypeID = ".custom.GetIamPolicyRequest"
				m.OutputTypeID = ".custom.Policy"
				m.Documentation = "Custom get policy documentation."
				return m
			}(),
			wantSubstr: "Custom get policy documentation.",
		},
		{
			name: "method named TestIamPermissions without IAM prefix preserves custom comments",
			method: func() *api.Method {
				m := api.NewTestMethod("TestIamPermissions")
				m.SourceServiceID = "custom.Service"
				m.InputTypeID = ".custom.TestIamPermissionsRequest"
				m.OutputTypeID = ".custom.TestIamPermissionsResponse"
				m.Documentation = "Custom test permissions documentation."
				return m
			}(),
			wantSubstr: "Custom test permissions documentation.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatMethodDoxygenComments(test.method, "", nil, "", false, nil, false, false)
			if !strings.Contains(got, test.wantSubstr) {
				t.Errorf("expected comments to contain %q, got:\n%s", test.wantSubstr, got)
			}
		})
	}
}

func TestFormatMethodDoxygenComments_ReferenceScoping(t *testing.T) {
	for _, test := range []struct {
		name         string
		doc          string
		paramDoc     string
		sourceFile   string
		setup        func() (*api.Method, *api.API)
		wantContains []string
		wantExcludes []string
	}{
		{
			name: "symbols in service dependency files are kept and unimported cross-service symbols filtered out",
			doc:  "Creates a [SecretPayload][google.cloud.secretmanager.v1.SecretPayload] and links [CryptoKey][google.cloud.kms.v1.CryptoKey].",
			setup: func() (*api.Method, *api.API) {
				req := api.NewTestMessage("CreateSecretRequest").WithPackage("google.cloud.secretmanager.v1")
				resp := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
				method := api.NewTestMethod("CreateSecret").WithInput(req).WithOutput(resp)
				svc := api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithMethods(method)
				method.Service = svc

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
				model.AddDefinitionLocation(svc.ID, api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     20,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.Secret", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     39,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SecretPayload", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     45,
				})
				model.AddDefinitionLocation("google.cloud.kms.v1.CryptoKey", api.SourceLocation{
					Filename: "google/cloud/kms/v1/resources.proto",
					Line:     50,
				})
				return method, model
			},
			wantContains: []string{
				"[google.cloud.secretmanager.v1.SecretPayload]: @googleapis_reference_link{google/cloud/secretmanager/v1/resources.proto#L45}",
			},
			wantExcludes: []string{
				"[google.cloud.kms.v1.CryptoKey]:",
			},
		},
		{
			name: "symbols in same file as service definition are kept",
			doc:  "References [SameFile][google.cloud.secretmanager.v1.SameFileHelper] and [CryptoKey][google.cloud.kms.v1.CryptoKey].",
			setup: func() (*api.Method, *api.API) {
				req := api.NewTestMessage("CreateSecretRequest").WithPackage("google.cloud.secretmanager.v1")
				resp := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
				method := api.NewTestMethod("CreateSecret").WithInput(req).WithOutput(resp)
				svc := api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithMethods(method)
				method.Service = svc

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
				model.AddDefinitionLocation(svc.ID, api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     20,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SameFileHelper", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     100,
				})
				model.AddDefinitionLocation("google.cloud.kms.v1.CryptoKey", api.SourceLocation{
					Filename: "google/cloud/kms/v1/resources.proto",
					Line:     50,
				})
				return method, model
			},
			wantContains: []string{
				"[google.cloud.secretmanager.v1.SameFileHelper]: @googleapis_reference_link{google/cloud/secretmanager/v1/service.proto#L100}",
			},
			wantExcludes: []string{
				"[google.cloud.kms.v1.CryptoKey]:",
			},
		},
		{
			name:       "fallback to sourceFile when method has no service definition location",
			doc:        "References [SameFile][google.cloud.secretmanager.v1.SameFileHelper] and [CryptoKey][google.cloud.kms.v1.CryptoKey].",
			sourceFile: "google/cloud/secretmanager/v1/service.proto",
			setup: func() (*api.Method, *api.API) {
				req := api.NewTestMessage("CreateSecretRequest").WithPackage("google.cloud.secretmanager.v1")
				resp := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
				method := api.NewTestMethod("CreateSecret").WithInput(req).WithOutput(resp)

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, nil)
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SameFileHelper", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     100,
				})
				model.AddDefinitionLocation("google.cloud.kms.v1.CryptoKey", api.SourceLocation{
					Filename: "google/cloud/kms/v1/resources.proto",
					Line:     50,
				})
				return method, model
			},
			wantContains: []string{
				"[google.cloud.secretmanager.v1.SameFileHelper]: @googleapis_reference_link{google/cloud/secretmanager/v1/service.proto#L100}",
			},
			wantExcludes: []string{
				"[google.cloud.kms.v1.CryptoKey]:",
			},
		},
		{
			name:       "fallback to sourceFile when method service has no definition location in model",
			doc:        "References [SameFile][google.cloud.secretmanager.v1.SameFileHelper] and [CryptoKey][google.cloud.kms.v1.CryptoKey].",
			sourceFile: "google/cloud/secretmanager/v1/service.proto",
			setup: func() (*api.Method, *api.API) {
				req := api.NewTestMessage("CreateSecretRequest").WithPackage("google.cloud.secretmanager.v1")
				resp := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
				method := api.NewTestMethod("CreateSecret").WithInput(req).WithOutput(resp)
				svc := api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithMethods(method)
				method.Service = svc

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
				// svc.ID is intentionally omitted from DefinitionLocations
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SameFileHelper", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     100,
				})
				model.AddDefinitionLocation("google.cloud.kms.v1.CryptoKey", api.SourceLocation{
					Filename: "google/cloud/kms/v1/resources.proto",
					Line:     50,
				})
				return method, model
			},
			wantContains: []string{
				"[google.cloud.secretmanager.v1.SameFileHelper]: @googleapis_reference_link{google/cloud/secretmanager/v1/service.proto#L100}",
			},
			wantExcludes: []string{
				"[google.cloud.kms.v1.CryptoKey]:",
			},
		},
		{
			name:       "all symbols kept when neither service location nor sourceFile is available",
			doc:        "References [SameFile][google.cloud.secretmanager.v1.SameFileHelper] and [CryptoKey][google.cloud.kms.v1.CryptoKey].",
			sourceFile: "",
			setup: func() (*api.Method, *api.API) {
				req := api.NewTestMessage("CreateSecretRequest").WithPackage("google.cloud.secretmanager.v1")
				resp := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
				method := api.NewTestMethod("CreateSecret").WithInput(req).WithOutput(resp)

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, nil)
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SameFileHelper", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     100,
				})
				model.AddDefinitionLocation("google.cloud.kms.v1.CryptoKey", api.SourceLocation{
					Filename: "google/cloud/kms/v1/resources.proto",
					Line:     50,
				})
				return method, model
			},
			wantContains: []string{
				"[google.cloud.kms.v1.CryptoKey]: @googleapis_reference_link{google/cloud/kms/v1/resources.proto#L50}",
				"[google.cloud.secretmanager.v1.SameFileHelper]: @googleapis_reference_link{google/cloud/secretmanager/v1/service.proto#L100}",
			},
			wantExcludes: nil,
		},
		{
			name:     "references in parameter comments are also scoped",
			doc:      "Simple method doc.",
			paramDoc: "  /// @param req Uses [SecretPayload][google.cloud.secretmanager.v1.SecretPayload] and [CryptoKey][google.cloud.kms.v1.CryptoKey].\n",
			setup: func() (*api.Method, *api.API) {
				req := api.NewTestMessage("CreateSecretRequest").WithPackage("google.cloud.secretmanager.v1")
				resp := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
				method := api.NewTestMethod("CreateSecret").WithInput(req).WithOutput(resp)
				svc := api.NewTestService("SecretManagerService").
					WithPackage("google.cloud.secretmanager.v1").
					WithMethods(method)
				method.Service = svc

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
				model.AddDefinitionLocation(svc.ID, api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/service.proto",
					Line:     20,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.Secret", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     39,
				})
				model.AddDefinitionLocation("google.cloud.secretmanager.v1.SecretPayload", api.SourceLocation{
					Filename: "google/cloud/secretmanager/v1/resources.proto",
					Line:     45,
				})
				model.AddDefinitionLocation("google.cloud.kms.v1.CryptoKey", api.SourceLocation{
					Filename: "google/cloud/kms/v1/resources.proto",
					Line:     50,
				})
				return method, model
			},
			wantContains: []string{
				"[google.cloud.secretmanager.v1.SecretPayload]: @googleapis_reference_link{google/cloud/secretmanager/v1/resources.proto#L45}",
			},
			wantExcludes: []string{
				"[google.cloud.kms.v1.CryptoKey]:",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			method, model := test.setup()
			method.Documentation = test.doc
			got := formatMethodDoxygenComments(method, test.paramDoc, model, test.sourceFile, false, nil, false, false)
			for _, want := range test.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("expected comments to contain %q, got:\n%s", want, got)
				}
			}
			for _, exclude := range test.wantExcludes {
				if strings.Contains(got, exclude) {
					t.Errorf("expected comments to NOT contain %q, got:\n%s", exclude, got)
				}
			}
		})
	}
}
