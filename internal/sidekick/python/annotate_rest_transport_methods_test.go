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

func TestIsProtoPlusType(t *testing.T) {
	for _, test := range []struct {
		name       string
		typeID     string
		servicePkg string
		want       bool
	}{
		{
			name:       "iam policy",
			typeID:     ".google.iam.v1.Policy",
			servicePkg: "google.cloud.secretmanager.v1",
			want:       false,
		},
		{
			name:       "longrunning operation",
			typeID:     "google.longrunning.Operation",
			servicePkg: "google.cloud.secretmanager.v1",
			want:       false,
		},
		{
			name:       "protobuf empty",
			typeID:     ".google.protobuf.Empty",
			servicePkg: "google.cloud.secretmanager.v1",
			want:       false,
		},
		{
			name:       "rpc status",
			typeID:     ".google.rpc.Status",
			servicePkg: "google.cloud.secretmanager.v1",
			want:       false,
		},
		{
			name:       "matching service package type",
			typeID:     ".google.cloud.secretmanager.v1.Secret",
			servicePkg: "google.cloud.secretmanager.v1",
			want:       true,
		},
		{
			name:       "non-matching package type",
			typeID:     ".google.cloud.other.v1.Message",
			servicePkg: "google.cloud.secretmanager.v1",
			want:       false,
		},
		{
			name:       "empty service package",
			typeID:     ".google.cloud.secretmanager.v1.Secret",
			servicePkg: "",
			want:       false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isProtoPlusType(test.typeID, test.servicePkg)
			if got != test.want {
				t.Errorf("isProtoPlusType(%q, %q) = %v, want %v", test.typeID, test.servicePkg, got, test.want)
			}
		})
	}
}

func TestBuildRestBaseMethods(t *testing.T) {
	nativeMethod := api.NewTestMethod("GetSecret")
	nativeMethod.Codec = &methodAnnotations{
		RestMethod: &restMethodAnnotation{
			HasRequiredFields: true,
			RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{
				{Key: "name", Value: "''"},
			},
			HTTPOptions: []*httpOptionAnnotation{
				{Method: "GET", URI: "/v1/{name=projects/*/secrets/*}"},
			},
		},
	}

	mixinMethod := api.NewTestMethod("GetLocation")
	mixinMethod.SourceServiceID = ".google.cloud.location.Locations"
	mixinMethod.Codec = &methodAnnotations{
		RestMethod: &restMethodAnnotation{
			HTTPOptions: []*httpOptionAnnotation{
				{Method: "GET", URI: "/v1/{name=projects/*/locations/*}"},
			},
		},
	}

	baseMethods := buildRestBaseMethods([]*api.Method{nativeMethod}, []*api.Method{mixinMethod})
	if len(baseMethods) != 2 {
		t.Fatalf("buildRestBaseMethods returned %d methods, want 2", len(baseMethods))
	}

	if baseMethods[0].BaseClassName != "_BaseGetSecret" || baseMethods[0].IsMixin {
		t.Errorf("native base method mismatch: %+v", baseMethods[0])
	}
	if !baseMethods[0].HasRequiredFields || len(baseMethods[0].RequiredFieldsDefaultValues) != 1 {
		t.Errorf("native base method required fields mismatch: %+v", baseMethods[0])
	}

	if baseMethods[1].BaseClassName != "_BaseGetLocation" || !baseMethods[1].IsMixin {
		t.Errorf("mixin base method mismatch: %+v", baseMethods[1])
	}
}

func TestBuildRestPrimaryMethods(t *testing.T) {
	reqMsg := api.NewTestMessage("GetSecretRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 1)
	respMsg := api.NewTestMessage("Secret").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 1)

	svc := api.NewTestService("SecretManagerService").
		WithPackage("google.cloud.secretmanager.v1")

	m := api.NewTestMethod("GetSecret").WithInput(reqMsg).WithOutput(respMsg)
	m.InputTypeID = ".google.cloud.secretmanager.v1.GetSecretRequest"
	m.OutputTypeID = ".google.cloud.secretmanager.v1.Secret"
	m.Codec = &methodAnnotations{
		RestMethod: &restMethodAnnotation{
			HTTPOptions: []*httpOptionAnnotation{
				{Method: "GET", URI: "/v1/{name=projects/*/secrets/*}", HasBody: false},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc}).
		WithPackageName("google.cloud.secretmanager.v1")
	c := newTestCodec(t, model, nil)

	methods := c.buildRestPrimaryMethods([]*api.Method{m}, svc)
	if len(methods) != 1 {
		t.Fatalf("buildRestPrimaryMethods returned %d methods, want 1", len(methods))
	}

	pm := methods[0]
	if pm.Name != "get_secret" || pm.MethodPascalName != "GetSecret" {
		t.Errorf("method names mismatch: got (%q, %q)", pm.Name, pm.MethodPascalName)
	}
	if !pm.IsInputProtoPlus || !pm.IsOutputProtoPlus {
		t.Errorf("proto-plus flags mismatch: got input=%v, output=%v, want both true", pm.IsInputProtoPlus, pm.IsOutputProtoPlus)
	}
}

func TestBuildRestPrimaryMethods_IAMAndSpecialTypes(t *testing.T) {
	svc := api.NewTestService("SecretManagerService").
		WithPackage("google.cloud.secretmanager.v1")

	iamReq := api.NewTestMessage("GetIamPolicyRequest").WithPackage("google.iam.v1")
	iamResp := api.NewTestMessage("Policy").WithPackage("google.iam.v1")

	mIAM := api.NewTestMethod("GetIamPolicy").WithInput(iamReq).WithOutput(iamResp)
	mIAM.InputTypeID = ".google.iam.v1.GetIamPolicyRequest"
	mIAM.OutputTypeID = ".google.iam.v1.Policy"
	mIAM.Codec = &methodAnnotations{
		RestMethod: &restMethodAnnotation{
			HTTPOptions: []*httpOptionAnnotation{{Method: "GET", URI: "/v1/{resource=*}"}},
		},
	}

	mEmpty := api.NewTestMethod("DeleteSecret")
	mEmpty.ReturnsEmpty = true
	mEmpty.OutputTypeID = api.WktEmptyID
	mEmpty.Codec = &methodAnnotations{
		RestMethod: &restMethodAnnotation{
			HTTPOptions: []*httpOptionAnnotation{{Method: "DELETE", URI: "/v1/{name=*}"}},
		},
	}

	model := api.NewTestAPI(nil, nil, []*api.Service{svc}).
		WithPackageName("google.cloud.secretmanager.v1")
	c := newTestCodec(t, model, nil)

	methods := c.buildRestPrimaryMethods([]*api.Method{mIAM, mEmpty}, svc)
	if len(methods) != 2 {
		t.Fatalf("buildRestPrimaryMethods returned %d methods, want 2", len(methods))
	}

	if methods[0].IsInputProtoPlus || methods[0].IsOutputProtoPlus {
		t.Errorf("IAM method should not be proto-plus: got in=%v, out=%v", methods[0].IsInputProtoPlus, methods[0].IsOutputProtoPlus)
	}
	if methods[0].InputTypeIdent != "iam_policy_pb2.GetIamPolicyRequest" || methods[0].OutputTypeIdent != "policy_pb2.Policy" {
		t.Errorf("IAM method type idents mismatch: got in=%q, out=%q", methods[0].InputTypeIdent, methods[0].OutputTypeIdent)
	}

	if !methods[1].IsVoid || methods[1].OutputTypeIdent != "empty_pb2.Empty" {
		t.Errorf("Empty method mismatch: isVoid=%v, outIdent=%q", methods[1].IsVoid, methods[1].OutputTypeIdent)
	}
}

func TestBuildRestMixinMethodsAndLRO(t *testing.T) {
	locMethod := api.NewTestMethod("GetLocation")
	locMethod.SourceServiceID = ".google.cloud.location.Locations"

	opMethod := api.NewTestMethod("GetOperation")
	opMethod.SourceServiceID = ".google.longrunning.Operations"

	mixins := buildRestMixinMethods([]*api.Method{locMethod, opMethod})
	if len(mixins) != 2 {
		t.Fatalf("buildRestMixinMethods returned %d methods, want 2", len(mixins))
	}

	if diff := cmp.Diff("get_location", mixins[0].Name); diff != "" {
		t.Errorf("mixin[0].Name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("get_operation", mixins[1].Name); diff != "" {
		t.Errorf("mixin[1].Name mismatch (-want +got):\n%s", diff)
	}

	lroOps := buildRestLROOperations([]*api.Method{opMethod})
	if len(lroOps) != 1 {
		t.Fatalf("buildRestLROOperations returned %d ops, want 1", len(lroOps))
	}
	if lroOps[0].Selector != "google.longrunning.Operations.GetOperation" {
		t.Errorf("lroOps[0].Selector = %q, want %q", lroOps[0].Selector, "google.longrunning.Operations.GetOperation")
	}
}
