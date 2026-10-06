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
	resource := &Resource{
		Type:     "google.cloud.secretmanager.v1/Secret",
		Singular: "secret",
	}
	resourceWithoutSingular := &Resource{
		Type: "google.cloud.secretmanager.v1/SecretWithoutSingular",
	}
	resourceNameField := &Field{
		Name: "name",
		ResourceReference: &ResourceReference{
			Type: resource.Type,
		},
	}
	resourceOtherNameField := &Field{
		Name: "other_name",
		ResourceReference: &ResourceReference{
			Type: resource.Type,
		},
	}
	resourceNameNoSingularField := &Field{
		Name: "name",
		ResourceReference: &ResourceReference{
			Type: resourceWithoutSingular.Type,
		},
	}
	resourceOtherNameNoSingularField := &Field{
		Name: "other_name",
		ResourceReference: &ResourceReference{
			Type: resourceWithoutSingular.Type,
		},
	}
	nonExistentResourceField := &Field{
		Name: "name",
		ResourceReference: &ResourceReference{
			Type: "nonexistent.googleapis.com/NonExistent",
		},
	}

	wildcardResourceField := &Field{
		Name: "name",
		ResourceReference: &ResourceReference{
			Type: "*",
		},
	}

	model := &API{
		ResourceDefinitions: []*Resource{resource, resourceWithoutSingular},
		resourceByType: map[string]*Resource{
			resource.Type:                resource,
			resourceWithoutSingular.Type: resourceWithoutSingular,
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

func TestIsAIPStandard(t *testing.T) {
	f := newAIPTestFixture()

	// Setup for a valid Get operation
	validGetMethod := &Method{
		Name:       "GetSecret",
		InputType:  &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
		OutputType: &Message{Resource: f.resource},
		Model:      f.model,
	}

	validDeleteMethod := &Method{
		Name:         "DeleteSecret",
		InputType:    &Message{Name: "DeleteSecretRequest", Fields: []*Field{f.resourceNameField}},
		ReturnsEmpty: true,
		Model:        f.model,
	}

	validUndeleteMethod := &Method{
		Name:      "UndeleteSecret",
		InputType: &Message{Name: "UndeleteSecretRequest", Fields: []*Field{f.resourceNameField}},
		OutputType: &Message{
			Resource: f.resource,
		},
		Model: f.model,
	}

	// Setup for an invalid Get operation (e.g., wrong name)
	invalidGetMethod := &Method{
		Name:       "ListSecrets", // Not a Get method
		InputType:  &Message{Name: "ListSecretsRequest"},
		OutputType: &Message{Resource: f.resource},
		Model:      f.model,
	}

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

	// Helper to create an output message since Get needs it
	output := &Message{
		Resource: f.resource,
	}

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid get operation with wildcard resource reference",
			method: &Method{
				Name:       "GetSecret",
				InputType:  &Message{Name: "GetSecretRequest", Fields: []*Field{f.wildcardResourceField}},
				OutputType: output,
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.wildcardResourceField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid get operation",
			method: &Method{
				Name:       "GetSecret",
				InputType:  &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType: output,
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid get operation with missing singular name on resource",
			method: &Method{
				Name:      "GetSecret",
				InputType: &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameNoSingularField}},
				OutputType: &Message{
					Resource: f.resourceWithoutSingular,
				},
				Model: f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameNoSingularField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "method name is incorrect",
			method: &Method{
				Name:       "Get",
				InputType:  &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType: output,
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "request type name is incorrect",
			method: &Method{
				Name:       "GetSecret",
				InputType:  &Message{Name: "GetRequest", Fields: []*Field{f.resourceNameField}},
				OutputType: output,
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "returns empty",
			method: &Method{
				Name:         "GetSecret",
				InputType:    &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType:   output,
				ReturnsEmpty: true,
				Model:        f.model,
			},
			want: nil,
		},
		{
			name: "output is not a resource",
			method: &Method{
				Name:      "GetSecret",
				InputType: &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType: &Message{
					Resource: nil,
				},
				Model: f.model,
			},
			want: nil,
		},
		{
			name: "request does not contain resource name field",
			method: &Method{
				Name:       "GetSecret",
				InputType:  &Message{Name: "GetSecretRequest"},
				OutputType: output,
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "pagination method is not a standard get operation",
			method: &Method{
				Name:       "GetSecret",
				InputType:  &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType: output,
				Pagination: &Field{},
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "client streaming method is not a standard get operation",
			method: &Method{
				Name:                "GetSecret",
				InputType:           &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType:          output,
				ClientSideStreaming: true,
				Model:               f.model,
			},
			want: nil,
		},
		{
			name: "server streaming method is not a standard get operation",
			method: &Method{
				Name:                "GetSecret",
				InputType:           &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType:          output,
				ServerSideStreaming: true,
				Model:               f.model,
			},
			want: nil,
		},
		{
			name: "LRO method is not a standard get operation",
			method: &Method{
				Name:          "GetSecret",
				InputType:     &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType:    output,
				OperationInfo: &OperationInfo{},
				Model:         f.model,
			},
			want: nil,
		},
		{
			name: "Discovery LRO method is not a standard get operation",
			method: &Method{
				Name:         "GetSecret",
				InputType:    &Message{Name: "GetSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType:   output,
				DiscoveryLro: &DiscoveryLro{},
				Model:        f.model,
			},
			want: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got); diff != "" {
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
			method: &Method{
				Name:         "DeleteSecret",
				InputType:    &Message{Name: "DeleteSecretRequest", Fields: []*Field{f.wildcardResourceField}},
				ReturnsEmpty: true,
				Model:        f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.wildcardResourceField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid simple delete",
			method: &Method{
				Name:         "DeleteSecret",
				InputType:    &Message{Name: "DeleteSecretRequest", Fields: []*Field{f.resourceNameField}},
				ReturnsEmpty: true,
				Model:        f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid simple delete with missing singular name on resource",
			method: &Method{
				Name:         "DeleteSecret",
				InputType:    &Message{Name: "DeleteSecretRequest", Fields: []*Field{f.resourceNameNoSingularField}},
				ReturnsEmpty: true,
				Model:        f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameNoSingularField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid lro delete",
			method: &Method{
				Name:          "DeleteSecret",
				InputType:     &Message{Name: "DeleteSecretRequest", Fields: []*Field{f.resourceNameField}},
				OperationInfo: &OperationInfo{},
				Model:         f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid delete with other name matching singular",
			method: &Method{
				Name:         "DeleteSecret",
				InputType:    &Message{Name: "DeleteSecretRequest", Fields: []*Field{f.resourceOtherNameField}},
				ReturnsEmpty: true,
				Model:        f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceOtherNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "incorrect method name",
			method: &Method{
				Name:      "RemoveSecret",
				InputType: &Message{Name: "DeleteSecretRequest", Fields: []*Field{f.resourceNameField}},
				Model:     f.model,
			},
			want: nil,
		},
		{
			name: "incorrect request name",
			method: &Method{
				Name:      "DeleteSecret",
				InputType: &Message{Name: "RemoveSecretRequest", Fields: []*Field{f.resourceNameField}},
				Model:     f.model,
			},
			want: nil,
		},
		{
			name: "resource not found in ResourceByType map",
			method: &Method{
				Name: "DeleteSecret",
				InputType: &Message{
					Name: "DeleteSecretRequest",
					Fields: []*Field{
						f.nonExistentResourceField,
					},
				},
				Model: f.model, // model's ResourceByType does not contain the nonexistent resource
			},
			want: nil,
		},
		{
			name: "invalid delete with no matching field",
			method: &Method{
				Name: "DeleteSecret",
				InputType: &Message{
					Name: "DeleteSecretRequest",
					Fields: []*Field{
						f.nonExistentResourceField,
						f.resourceOtherNameNoSingularField,
					},
				},
				Model: f.model,
			},
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got); diff != "" {
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
			method: &Method{
				Name:      "UndeleteSecret",
				InputType: &Message{Name: "UndeleteSecretRequest", Fields: []*Field{f.wildcardResourceField}},
				OutputType: &Message{
					Resource: f.resource,
				},
				Model: f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.wildcardResourceField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid simple undelete",
			method: &Method{
				Name:      "UndeleteSecret",
				InputType: &Message{Name: "UndeleteSecretRequest", Fields: []*Field{f.resourceNameField}},
				OutputType: &Message{
					Resource: f.resource,
				},
				Model: f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid simple undelete with missing singular name on resource",
			method: &Method{
				Name:      "UndeleteSecret",
				InputType: &Message{Name: "UndeleteSecretRequest", Fields: []*Field{f.resourceNameNoSingularField}},
				OutputType: &Message{
					Resource: f.resourceWithoutSingular,
				},
				Model: f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameNoSingularField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid lro undelete",
			method: &Method{
				Name:          "UndeleteSecret",
				InputType:     &Message{Name: "UndeleteSecretRequest", Fields: []*Field{f.resourceNameField}},
				OperationInfo: &OperationInfo{},
				Model:         f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid undelete with other name matching singular",
			method: &Method{
				Name:      "UndeleteSecret",
				InputType: &Message{Name: "UndeleteSecretRequest", Fields: []*Field{f.resourceOtherNameField}},
				OutputType: &Message{
					Resource: f.resource,
				},
				Model: f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceOtherNameField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "incorrect method name",
			method: &Method{
				Name:      "RestoreSecret",
				InputType: &Message{Name: "UndeleteSecretRequest", Fields: []*Field{f.resourceNameField}},
				Model:     f.model,
			},
			want: nil,
		},
		{
			name: "incorrect request name",
			method: &Method{
				Name:      "UndeleteSecret",
				InputType: &Message{Name: "RestoreSecretRequest", Fields: []*Field{f.resourceNameField}},
				Model:     f.model,
			},
			want: nil,
		},
		{
			name: "resource not found in ResourceByType map",
			method: &Method{
				Name: "UndeleteSecret",
				InputType: &Message{
					Name: "UndeleteSecretRequest",
					Fields: []*Field{
						f.nonExistentResourceField,
					},
				},
				Model: f.model, // model's ResourceByType does not contain the nonexistent resource
			},
			want: nil,
		},
		{
			name: "invalid undelete with no matching field",
			method: &Method{
				Name: "UndeleteSecret",
				InputType: &Message{
					Name: "UndeleteSecretRequest",
					Fields: []*Field{
						f.nonExistentResourceField,
						f.resourceOtherNameNoSingularField,
					},
				},
				Model: f.model,
			},
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardCreateInfo(t *testing.T) {
	f := newAIPTestFixture()

	// Setup for Create
	secretMessage := &Message{ID: "secret_message_id"}
	f.resource.Self = secretMessage

	parentField := &Field{
		Name:              "parent",
		ResourceReference: &ResourceReference{ChildType: f.resource.Type},
		Typez:             TypezString,
	}
	resourceField := &Field{
		Name:    "secret",
		Typez:   TypezMessage,
		TypezID: secretMessage.ID,
	}
	idField := &Field{
		Name:  "secret_id",
		Typez: TypezString,
	}

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid create operation",
			method: &Method{
				Name: "CreateSecret",
				InputType: &Message{
					Name: "CreateSecretRequest",
					Fields: []*Field{
						parentField,
						idField,
						resourceField,
					},
				},
				OutputType: &Message{Resource: f.resource},
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     parentField,
				IsRequestResourceName: true,
				ResourceIDField:       idField,
				MessageField:          resourceField,
			},
		},
		{
			name: "valid create operation without id",
			method: &Method{
				Name: "CreateSecret",
				InputType: &Message{
					Name: "CreateSecretRequest",
					Fields: []*Field{
						parentField,
						resourceField,
					},
				},
				OutputType: &Message{Resource: f.resource},
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     parentField,
				IsRequestResourceName: true,
				MessageField:          resourceField,
			},
		},
		{
			name: "invalid create operation (wrong name)",
			method: &Method{
				Name: "MakeSecret",
				InputType: &Message{
					Name: "CreateSecretRequest",
					Fields: []*Field{
						parentField,
						resourceField,
					},
				},
				OutputType: &Message{Resource: f.resource},
				Model:      f.model,
			},
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardUpdateInfo(t *testing.T) {
	f := newAIPTestFixture()

	// Setup for Update
	secretMessage := &Message{
		ID: "secret_message_id",
		Fields: []*Field{
			f.resourceNameField,
		},
	}
	f.resource.Self = secretMessage

	resourceField := &Field{
		Name:        "secret",
		Typez:       TypezMessage,
		TypezID:     secretMessage.ID,
		MessageType: secretMessage,
	}
	updateMaskField := &Field{
		Name:    "update_mask",
		TypezID: ".google.protobuf.FieldMask",
	}

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid update operation",
			method: &Method{
				Name: "UpdateSecret",
				InputType: &Message{
					Name: "UpdateSecretRequest",
					Fields: []*Field{
						resourceField,
						updateMaskField,
					},
				},
				OutputType: &Message{Resource: f.resource},
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				MessageField:          resourceField,
				IsMessageResourceName: true,
				UpdateMaskField:       updateMaskField,
			},
		},
		{
			name: "valid update operation without mask",
			method: &Method{
				Name: "UpdateSecret",
				InputType: &Message{
					Name: "UpdateSecretRequest",
					Fields: []*Field{
						resourceField,
					},
				},
				OutputType: &Message{Resource: f.resource},
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     f.resourceNameField,
				MessageField:          resourceField,
				IsMessageResourceName: true,
			},
		},
		{
			name: "invalid update operation (wrong name)",
			method: &Method{
				Name: "ModifySecret",
				InputType: &Message{
					Name: "UpdateSecretRequest",
					Fields: []*Field{
						resourceField,
					},
				},
				OutputType: &Message{Resource: f.resource},
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "invalid update operation (wrong request name)",
			method: &Method{
				Name: "UpdateSecret",
				InputType: &Message{
					Name: "ModifySecretRequest",
					Fields: []*Field{
						resourceField,
					},
				},
				OutputType: &Message{Resource: f.resource},
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "invalid update operation (missing resource)",
			method: &Method{
				Name: "UpdateSecret",
				InputType: &Message{
					Name: "UpdateSecretRequest",
					Fields: []*Field{
						updateMaskField,
					},
				},
				OutputType: &Message{Resource: f.resource},
				Model:      f.model,
			},
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got); diff != "" {
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

	secretResource := &Resource{Type: resourceType, Plural: "Secrets"}
	secretMessage := &Message{Resource: secretResource}
	parentField := &Field{
		Name:              "parent",
		ResourceReference: &ResourceReference{ChildType: resourceType},
	}
	otherParentField := &Field{
		Name:              "database", // Not named "parent"
		ResourceReference: &ResourceReference{ChildType: resourceType},
	}
	plainParentField := &Field{Name: "parent"} // No child_type

	pageableItem := &Field{Name: "secrets", MessageType: secretMessage}
	paginationInfo := &PaginationInfo{PageableItem: pageableItem}
	listOutput := &Message{
		Name:       "ListSecretsResponse",
		Pagination: paginationInfo,
	}

	testCases := []struct {
		name   string
		method *Method
		want   *SampleInfo
	}{
		{
			name: "valid list operation with parent field match by child_type",
			method: &Method{
				Name:       "ListSecrets",
				InputType:  &Message{Name: "ListSecretsRequest", Fields: []*Field{parentField}},
				OutputType: listOutput,
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     parentField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid list operation with other field match by child_type",
			method: &Method{
				Name:       "ListSecrets",
				InputType:  &Message{Name: "ListSecretsRequest", Fields: []*Field{otherParentField}},
				OutputType: listOutput,
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     otherParentField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "valid list operation with parent field name match (fallback)",
			method: &Method{
				Name:       "ListSecrets",
				InputType:  &Message{Name: "ListSecretsRequest", Fields: []*Field{plainParentField}},
				OutputType: listOutput,
				Model:      f.model,
			},
			want: &SampleInfo{
				ResourceNameField:     plainParentField,
				IsRequestResourceName: true,
			},
		},
		{
			name: "list operation missing parent field",
			method: &Method{
				Name:       "ListSecrets",
				InputType:  &Message{Name: "ListSecretsRequest", Fields: []*Field{}},
				OutputType: listOutput,
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "method name does not start with List",
			method: &Method{
				Name:       "EnumerateSecrets",
				InputType:  &Message{Name: "ListSecretsRequest", Fields: []*Field{parentField}},
				OutputType: listOutput,
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "input type name mismatch",
			method: &Method{
				Name:       "ListSecrets",
				InputType:  &Message{Name: "EnumerateSecretsRequest", Fields: []*Field{parentField}},
				OutputType: listOutput,
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "output type name mismatch",
			method: &Method{
				Name:       "ListSecrets",
				InputType:  &Message{Name: "ListSecretsRequest", Fields: []*Field{parentField}},
				OutputType: &Message{Name: "EnumerateSecretsResponse", Pagination: paginationInfo},
				Model:      f.model,
			},
			want: nil,
		},
		{
			name: "not a list operation (no pagination)",
			method: &Method{
				Name:       "ListSecrets",
				InputType:  &Message{Name: "ListSecretsRequest", Fields: []*Field{parentField}},
				OutputType: &Message{Name: "ListSecretsResponse"}, // No pagination
				Model:      f.model,
			},
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichMethodSamples(tc.method)
			got := tc.method.SampleInfo
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
