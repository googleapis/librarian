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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type aipTestFixture struct {
	resource                         *Resource
	resourceWithoutSingular          *Resource
	resourceNameField                *Field
	resourceOtherNameField           *Field
	resourceNameNoSingularField      *Field
	resourceOtherNameNoSingularField *Field
	nonExistentResourceField         *Field
	wildcardResourceField            *Field
	model                            *API
}

func newAIPTestFixture() *aipTestFixture {
	resource := NewTestResource("google.cloud.secretmanager.v1/Secret").WithSingular("secret")
	resourceWithoutSingular := NewTestResource("google.cloud.secretmanager.v1/SecretWithoutSingular")
	resourceNameField := NewTestField("name").WithResourceReference(resource.Type())
	resourceOtherNameField := NewTestField("other_name").WithResourceReference(resource.Type())
	resourceNameNoSingularField := NewTestField("name").WithResourceReference(resourceWithoutSingular.Type())
	resourceOtherNameNoSingularField := NewTestField("other_name").WithResourceReference(resourceWithoutSingular.Type())
	nonExistentResourceField := NewTestField("name").WithResourceReference("nonexistent.googleapis.com/NonExistent")
	wildcardResourceField := NewTestField("name").WithResourceReference("*")

	model := &API{
		ResourceDefinitions: []*Resource{resource, resourceWithoutSingular},
		resourceByType: map[string]*Resource{
			resource.Type():                resource,
			resourceWithoutSingular.Type(): resourceWithoutSingular,
		},
	}

	return &aipTestFixture{
		resource:                         resource,
		resourceWithoutSingular:          resourceWithoutSingular,
		resourceNameField:                resourceNameField,
		resourceOtherNameField:           resourceOtherNameField,
		resourceNameNoSingularField:      resourceNameNoSingularField,
		resourceOtherNameNoSingularField: resourceOtherNameNoSingularField,
		nonExistentResourceField:         nonExistentResourceField,
		wildcardResourceField:            wildcardResourceField,
		model:                            model,
	}
}

func newAIPTestMethod(name string, input, output *Message, model *API) *Method {
	m := NewTestMethod(name).WithInput(input).WithOutput(output)
	m.Model = model
	return m
}

func newAIPTestDeleteMethod(input *Message, model *API) *Method {
	m := NewTestMethod("DeleteSecret").WithInput(input).ReturnEmpty()
	m.Model = model
	return m
}

func TestIsAIPStandard(t *testing.T) {
	f := newAIPTestFixture()

	output := NewTestMessage("Response").WithResource(f.resource)

	validGetMethod := newAIPTestMethod(
		"GetSecret",
		NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
		output,
		f.model,
	)

	validDeleteMethod := newAIPTestDeleteMethod(
		NewTestMessage("DeleteSecretRequest").WithFields(f.resourceNameField),
		f.model,
	)

	validUndeleteMethod := newAIPTestMethod(
		"UndeleteSecret",
		NewTestMessage("UndeleteSecretRequest").WithFields(f.resourceNameField),
		output,
		f.model,
	)

	invalidGetMethod := newAIPTestMethod(
		"ListSecrets",
		NewTestMessage("ListSecretsRequest"),
		output,
		f.model,
	)

	testCases := []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "standard get method returns true",
			method: validGetMethod,
			want:   true,
		},
		{
			name:   "standard delete method returns true",
			method: validDeleteMethod,
			want:   true,
		},
		{
			name:   "standard undelete method returns true",
			method: validUndeleteMethod,
			want:   true,
		},
		{
			name:   "non-standard method returns false",
			method: invalidGetMethod,
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			if got := tc.method.IsAIPStandard; got != tc.want {
				t.Errorf("IsAIPStandard() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAIPStandardGetInfo(t *testing.T) {
	f := newAIPTestFixture()

	output := NewTestMessage("Response").WithResource(f.resource)

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid get operation with wildcard resource reference",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.wildcardResourceField),
				output,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.wildcardResourceField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid get operation",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid get operation with missing singular name on resource",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameNoSingularField),
				NewTestMessage("Response").WithResource(f.resourceWithoutSingular),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameNoSingularField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "method name is incorrect",
			method: newAIPTestMethod(
				"Get",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			),
			want: nil,
		},
		{
			name: "request type name is incorrect",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			),
			want: nil,
		},
		{
			name: "returns empty",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			).ReturnEmpty(),
			want: nil,
		},
		{
			name: "output is not a resource",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				NewTestMessage("Response"),
				f.model,
			),
			want: nil,
		},
		{
			name: "request does not contain resource name field",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest"),
				output,
				f.model,
			),
			want: nil,
		},
		{
			name: "pagination method is not a standard get operation",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			).WithPagination(NewTestField("page")),
			want: nil,
		},
		{
			name: "client streaming method is not a standard get operation",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			).WithClientSideStreaming(),
			want: nil,
		},
		{
			name: "server streaming method is not a standard get operation",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			).WithServerSideStreaming(),
			want: nil,
		},
		{
			name: "LRO method is not a standard get operation",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			).WithOperationInfo(NewTestOperationInfo("", "")),
			want: nil,
		},
		{
			name: "Discovery LRO method is not a standard get operation",
			method: newAIPTestMethod(
				"GetSecret",
				NewTestMessage("GetSecretRequest").WithFields(f.resourceNameField),
				output,
				f.model,
			).WithDiscoveryLro(NewTestDiscoveryLro()),
			want: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardDeleteInfo(t *testing.T) {
	f := newAIPTestFixture()

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid simple delete with wildcard resource reference",
			method: newAIPTestDeleteMethod(
				NewTestMessage("DeleteSecretRequest").WithFields(f.wildcardResourceField),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.wildcardResourceField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid simple delete",
			method: newAIPTestDeleteMethod(
				NewTestMessage("DeleteSecretRequest").WithFields(f.resourceNameField),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid simple delete with missing singular name on resource",
			method: newAIPTestDeleteMethod(
				NewTestMessage("DeleteSecretRequest").WithFields(f.resourceNameNoSingularField),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameNoSingularField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid lro delete",
			method: newAIPTestMethod(
				"DeleteSecret",
				NewTestMessage("DeleteSecretRequest").WithFields(f.resourceNameField),
				nil,
				f.model,
			).WithOperationInfo(NewTestOperationInfo("", "")),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid delete with other name matching singular",
			method: newAIPTestDeleteMethod(
				NewTestMessage("DeleteSecretRequest").WithFields(f.resourceOtherNameField),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceOtherNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "incorrect method name",
			method: newAIPTestMethod(
				"RemoveSecret",
				NewTestMessage("DeleteSecretRequest").WithFields(f.resourceNameField),
				nil,
				f.model,
			),
			want: nil,
		},
		{
			name: "incorrect request name",
			method: newAIPTestMethod(
				"DeleteSecret",
				NewTestMessage("RemoveSecretRequest").WithFields(f.resourceNameField),
				nil,
				f.model,
			),
			want: nil,
		},
		{
			name: "resource not found in ResourceByType map",
			method: newAIPTestMethod(
				"DeleteSecret",
				NewTestMessage("DeleteSecretRequest").WithFields(f.nonExistentResourceField),
				nil,
				f.model,
			),
			want: nil,
		},
		{
			name: "invalid delete with no matching field",
			method: newAIPTestMethod(
				"DeleteSecret",
				NewTestMessage("DeleteSecretRequest").WithFields(
					f.nonExistentResourceField,
					f.resourceOtherNameNoSingularField,
				),
				nil,
				f.model,
			),
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardUndeleteInfo(t *testing.T) {
	f := newAIPTestFixture()

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid simple undelete with wildcard resource reference",
			method: newAIPTestMethod(
				"UndeleteSecret",
				NewTestMessage("UndeleteSecretRequest").WithFields(f.wildcardResourceField),
				NewTestMessage("UndeleteSecretResponse").WithResource(f.resource),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.wildcardResourceField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid simple undelete",
			method: newAIPTestMethod(
				"UndeleteSecret",
				NewTestMessage("UndeleteSecretRequest").WithFields(f.resourceNameField),
				NewTestMessage("UndeleteSecretResponse").WithResource(f.resource),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid simple undelete with missing singular name on resource",
			method: newAIPTestMethod(
				"UndeleteSecret",
				NewTestMessage("UndeleteSecretRequest").WithFields(f.resourceNameNoSingularField),
				NewTestMessage("UndeleteSecretResponse").WithResource(f.resourceWithoutSingular),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameNoSingularField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid lro undelete",
			method: func() *Method {
				m := NewTestMethod("UndeleteSecret").
					WithInput(NewTestMessage("UndeleteSecretRequest").WithFields(f.resourceNameField)).
					WithOperationInfo(NewTestOperationInfo("", ""))
				m.Model = f.model
				return m
			}(),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid undelete with other name matching singular",
			method: newAIPTestMethod(
				"UndeleteSecret",
				NewTestMessage("UndeleteSecretRequest").WithFields(f.resourceOtherNameField),
				NewTestMessage("UndeleteSecretResponse").WithResource(f.resource),
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceOtherNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "incorrect method name",
			method: func() *Method {
				m := NewTestMethod("RestoreSecret").
					WithInput(NewTestMessage("UndeleteSecretRequest").WithFields(f.resourceNameField))
				m.Model = f.model
				return m
			}(),
			want: nil,
		},
		{
			name: "incorrect request name",
			method: func() *Method {
				m := NewTestMethod("UndeleteSecret").
					WithInput(NewTestMessage("RestoreSecretRequest").WithFields(f.resourceNameField))
				m.Model = f.model
				return m
			}(),
			want: nil,
		},
		{
			name: "resource not found in ResourceByType map",
			method: func() *Method {
				m := NewTestMethod("UndeleteSecret").
					WithInput(NewTestMessage("UndeleteSecretRequest").WithFields(f.nonExistentResourceField))
				m.Model = f.model
				return m
			}(),
			want: nil,
		},
		{
			name: "invalid undelete with no matching field",
			method: func() *Method {
				m := NewTestMethod("UndeleteSecret").
					WithInput(NewTestMessage("UndeleteSecretRequest").WithFields(
						f.nonExistentResourceField,
						f.resourceOtherNameNoSingularField,
					))
				m.Model = f.model
				return m
			}(),
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardCreateInfo(t *testing.T) {
	f := newAIPTestFixture()

	// Setup for Create
	secretMessage := NewTestMessage("secret_message_id")

	parentField := NewTestField("parent").
		WithType(TypezString).
		WithChildTypeReference(f.resource.Type())
	resourceField := NewTestField("secret").
		WithMessageType(secretMessage)
	idField := NewTestField("secret_id").
		WithType(TypezString)
	output := NewTestMessage("Response").WithResource(f.resource)
	f.resource.self = secretMessage

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid create operation",
			method: newAIPTestMethod(
				"CreateSecret",
				NewTestMessage("CreateSecretRequest").WithFields(parentField, idField, resourceField),
				output,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     parentField,
				IsRequestResourceName: true,
				ResourceIDField:       idField,
				MessageField:          resourceField,
			},
		},
		{
			name: "valid create operation without id",
			method: newAIPTestMethod(
				"CreateSecret",
				NewTestMessage("CreateSecretRequest").WithFields(parentField, resourceField),
				output,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     parentField,
				IsRequestResourceName: true,
				MessageField:          resourceField,
			},
		},
		{
			name: "invalid create operation (wrong name)",
			method: newAIPTestMethod(
				"MakeSecret",
				NewTestMessage("CreateSecretRequest").WithFields(parentField, resourceField),
				output,
				f.model,
			),
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardUpdateInfo(t *testing.T) {
	f := newAIPTestFixture()

	// Setup for Update
	secretMessage := NewTestMessage("secret_message_id").
		WithID("secret_message_id").
		WithFields(f.resourceNameField)

	resourceField := NewTestField("secret").
		WithMessageType(secretMessage)
	updateMaskField := NewTestField("update_mask").
		WithTypezID(".google.protobuf.FieldMask")
	output := NewTestMessage("Response").WithResource(f.resource)
	f.resource.self = secretMessage

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid update operation",
			method: newAIPTestMethod(
				"UpdateSecret",
				NewTestMessage("UpdateSecretRequest").WithFields(resourceField, updateMaskField),
				output,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				MessageField:          resourceField,
				IsMessageResourceName: true,
				UpdateMaskField:       updateMaskField,
			},
		},
		{
			name: "valid update operation without mask",
			method: newAIPTestMethod(
				"UpdateSecret",
				NewTestMessage("UpdateSecretRequest").WithFields(resourceField),
				output,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				MessageField:          resourceField,
				IsMessageResourceName: true,
			},
		},
		{
			name: "invalid update operation (wrong name)",
			method: newAIPTestMethod(
				"ModifySecret",
				NewTestMessage("UpdateSecretRequest").WithFields(resourceField),
				output,
				f.model,
			),
			want: nil,
		},
		{
			name: "invalid update operation (wrong request name)",
			method: newAIPTestMethod(
				"UpdateSecret",
				NewTestMessage("ModifySecretRequest").WithFields(resourceField),
				output,
				f.model,
			),
			want: nil,
		},
		{
			name: "invalid update operation (missing resource)",
			method: newAIPTestMethod(
				"UpdateSecret",
				NewTestMessage("UpdateSecretRequest").WithFields(updateMaskField),
				output,
				f.model,
			),
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardListInfo(t *testing.T) {
	f := newAIPTestFixture()

	// Create a resource type for the list items
	resourceType := "google.cloud.secretmanager.v1/Secret"
	// Ensure the parent field points to this resource type in child_type?
	// No, parent field child_type matches the listed resource type.

	secretResource := NewTestResource(resourceType).WithPlural("Secrets")
	secretMessage := NewTestMessage("Secret").WithResource(secretResource)
	parentField := NewTestField("parent").
		WithChildTypeReference(resourceType)
	otherParentField := NewTestField("database").
		WithChildTypeReference(resourceType) // Not named "parent"
	plainParentField := NewTestField("parent") // No child_type

	pageableItem := NewTestField("secrets").WithMessageType(secretMessage)
	listOutput := NewTestMessage("ListSecretsResponse").WithPagination(nil, pageableItem)

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid list operation with parent field match by child_type",
			method: newAIPTestMethod(
				"ListSecrets",
				NewTestMessage("ListSecretsRequest").WithFields(parentField),
				listOutput,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     parentField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid list operation with other field match by child_type",
			method: newAIPTestMethod(
				"ListSecrets",
				NewTestMessage("ListSecretsRequest").WithFields(otherParentField),
				listOutput,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     otherParentField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid list operation with parent field name match (fallback)",
			method: newAIPTestMethod(
				"ListSecrets",
				NewTestMessage("ListSecretsRequest").WithFields(plainParentField),
				listOutput,
				f.model,
			),
			want: &SampleInfo{
				ResourceNameField:     plainParentField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "list operation missing parent field",
			method: newAIPTestMethod(
				"ListSecrets",
				NewTestMessage("ListSecretsRequest"),
				listOutput,
				f.model,
			),
			want: nil,
		},
		{
			name: "method name does not start with List",
			method: newAIPTestMethod(
				"EnumerateSecrets",
				NewTestMessage("ListSecretsRequest").WithFields(parentField),
				listOutput,
				f.model,
			),
			want: nil,
		},
		{
			name: "input type name mismatch",
			method: newAIPTestMethod(
				"ListSecrets",
				NewTestMessage("EnumerateSecretsRequest").WithFields(parentField),
				listOutput,
				f.model,
			),
			want: nil,
		},
		{
			name: "output type name mismatch",
			method: func() *Method {
				out := NewTestMessage("EnumerateSecretsResponse").WithPagination(nil, pageableItem)
				return newAIPTestMethod(
					"ListSecrets",
					NewTestMessage("ListSecretsRequest").WithFields(parentField),
					out,
					f.model,
				)
			}(),
			want: nil,
		},
		{
			name: "not a list operation (no pagination)",
			method: newAIPTestMethod(
				"ListSecrets",
				NewTestMessage("ListSecretsRequest").WithFields(parentField),
				NewTestMessage("ListSecretsResponse"),
				f.model,
			),
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
