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

	docComment := formatMethodDoxygenComments(method, "", nil, "", false, nil, false, false, false)
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
			got := formatMethodDoxygenComments(test.method, "", nil, "", false, nil, false, false, false)
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

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithDefinitionLocation(svc.ID, "google/cloud/secretmanager/v1/service.proto", 20).
					WithDefinitionLocation("google.cloud.secretmanager.v1.Secret", "google/cloud/secretmanager/v1/resources.proto", 39).
					WithDefinitionLocation("google.cloud.secretmanager.v1.SecretPayload", "google/cloud/secretmanager/v1/resources.proto", 45).
					WithDefinitionLocation("google.cloud.kms.v1.CryptoKey", "google/cloud/kms/v1/resources.proto", 50)
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

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithDefinitionLocation(svc.ID, "google/cloud/secretmanager/v1/service.proto", 20).
					WithDefinitionLocation("google.cloud.secretmanager.v1.SameFileHelper", "google/cloud/secretmanager/v1/service.proto", 100).
					WithDefinitionLocation("google.cloud.kms.v1.CryptoKey", "google/cloud/kms/v1/resources.proto", 50)
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

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, nil).
					WithDefinitionLocation("google.cloud.secretmanager.v1.SameFileHelper", "google/cloud/secretmanager/v1/service.proto", 100).
					WithDefinitionLocation("google.cloud.kms.v1.CryptoKey", "google/cloud/kms/v1/resources.proto", 50)
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

				// svc.ID is intentionally omitted from DefinitionLocations
				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithDefinitionLocation("google.cloud.secretmanager.v1.SameFileHelper", "google/cloud/secretmanager/v1/service.proto", 100).
					WithDefinitionLocation("google.cloud.kms.v1.CryptoKey", "google/cloud/kms/v1/resources.proto", 50)
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

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, nil).
					WithDefinitionLocation("google.cloud.secretmanager.v1.SameFileHelper", "google/cloud/secretmanager/v1/service.proto", 100).
					WithDefinitionLocation("google.cloud.kms.v1.CryptoKey", "google/cloud/kms/v1/resources.proto", 50)
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

				model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
					WithDefinitionLocation(svc.ID, "google/cloud/secretmanager/v1/service.proto", 20).
					WithDefinitionLocation("google.cloud.secretmanager.v1.Secret", "google/cloud/secretmanager/v1/resources.proto", 39).
					WithDefinitionLocation("google.cloud.secretmanager.v1.SecretPayload", "google/cloud/secretmanager/v1/resources.proto", 45).
					WithDefinitionLocation("google.cloud.kms.v1.CryptoKey", "google/cloud/kms/v1/resources.proto", 50)
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
			got := formatMethodDoxygenComments(method, test.paramDoc, model, test.sourceFile, false, nil, false, false, false)
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

func TestFormatMethodDoxygenComments_MapPagination(t *testing.T) {
	itemMsg := api.NewTestMessage("Item").WithPackage("google.cloud.compute.v1")
	mapEntry := api.NewTestMessage("ItemsScopedList").WithPackage("google.cloud.compute.v1").WithFields(
		api.NewTestField("key").WithType(api.TypezString),
		api.NewTestField("value").WithMessageType(itemMsg),
	)
	mapField := api.NewTestField("items").WithMessageType(mapEntry).WithMap()

	req := api.NewTestMessage("AggregatedListRequest").WithPackage("google.cloud.compute.v1")
	resp := api.NewTestMessage("AggregatedListResponse").WithPackage("google.cloud.compute.v1").WithFields(mapField)
	method := api.NewTestMethod("AggregatedList").
		WithInput(req).
		WithOutput(resp)
	svc := api.NewTestService("Instances").
		WithPackage("google.cloud.compute.v1").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{req, resp, mapEntry, itemMsg}, nil, []*api.Service{svc}).
		WithDefinitionLocation("google.cloud.compute.v1.Item", "google/cloud/compute/v1/compute.proto", 100)

	got := formatMethodDoxygenComments(method, "", model, "google/cloud/compute/v1/instances.proto", true, mapField, false, false, false)

	wantContains := []string{
		"contains elements of type\n  ///     [google.cloud.compute.v1.Item]",
		"[google.cloud.compute.v1.Item]: @cloud_cpp_reference_link{google/cloud/compute/v1/compute.proto#L100}",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("expected comments to contain %q, got:\n%s", want, got)
		}
	}
}

func TestFormatMethodDoxygenComments_ComputeLRO(t *testing.T) {
	op := api.NewTestMessage("Operation").WithPackage("google.cloud.cpp.compute.v1")
	req := api.NewTestMessage("InsertRequest").WithPackage("google.cloud.compute.v1")
	method := api.NewTestMethod("Insert").
		WithInput(req).
		WithOutput(op).
		WithOperationService("RegionOperations")
	svc := api.NewTestService("RegionOperationsService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{req, op}, nil, []*api.Service{svc}).
		WithDefinitionLocation("google.cloud.compute.v1.InsertRequest", "google/cloud/compute/v1/operations.proto", 10).
		WithDefinitionLocation("google.cloud.cpp.compute.v1.Operation", "google/cloud/compute/v1/operations.proto", 42)

	got := formatMethodDoxygenComments(method, "", model, "google/cloud/compute/v1/operations.proto", false, nil, true, false, true)

	wantContains := []string{
		"[Long Running Operation]: http://cloud/compute/docs/api/how-tos/api-requests-responses#handling_api_responses",
		"For this RPC the result is a\n  ///     [google.cloud.cpp.compute.v1.Operation] proto message.",
		"[google.cloud.compute.v1.InsertRequest]: @cloud_cpp_reference_link{google/cloud/compute/v1/operations.proto#L10}",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("expected comments to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "https://google.aip.dev/151") {
		t.Errorf("expected comments to NOT contain https://google.aip.dev/151 for Compute LRO, got:\n%s", got)
	}
	if strings.Contains(got, "google.cloud.cpp.compute.v1.Operation]:") {
		t.Errorf("expected comments to NOT contain Operation reference link for Compute LRO, got:\n%s", got)
	}
}

func TestFormatMethodDoxygenComments_ComputeLRO_WithoutOperationService(t *testing.T) {
	op := api.NewTestMessage("Operation").WithPackage("google.cloud.cpp.compute.v1")
	req := api.NewTestMessage("InsertRequest").WithPackage("google.cloud.compute.v1")
	// Method without OperationService set on api.Method, but isComputeLRO is true (e.g. from !GenerateGrpcTransport)
	method := api.NewTestMethod("Insert").
		WithInput(req).
		WithOutput(op)
	svc := api.NewTestService("ComputeService").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{req, op}, nil, []*api.Service{svc}).
		WithDefinitionLocation("google.cloud.compute.v1.InsertRequest", "google/cloud/compute/v1/operations.proto", 10).
		WithDefinitionLocation("google.cloud.cpp.compute.v1.Operation", "google/cloud/compute/v1/operations.proto", 42)

	got := formatMethodDoxygenComments(method, "", model, "google/cloud/compute/v1/operations.proto", false, nil, true, false, true)

	wantContains := []string{
		"[Long Running Operation]: http://cloud/compute/docs/api/how-tos/api-requests-responses#handling_api_responses",
		"For this RPC the result is a\n  ///     [google.cloud.cpp.compute.v1.Operation] proto message.",
		"[google.cloud.compute.v1.InsertRequest]: @cloud_cpp_reference_link{google/cloud/compute/v1/operations.proto#L10}",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("expected comments to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "https://google.aip.dev/151") {
		t.Errorf("expected comments to NOT contain https://google.aip.dev/151 for Compute LRO, got:\n%s", got)
	}
	if strings.Contains(got, "google.cloud.cpp.compute.v1.Operation]:") {
		t.Errorf("expected comments to NOT contain Operation reference link for Compute LRO, got:\n%s", got)
	}
}
