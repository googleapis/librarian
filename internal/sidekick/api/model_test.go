// Copyright 2025 Google LLC
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

func TestRoutingCombosSimpleOr(t *testing.T) {
	v1 := &RoutingInfoVariant{
		FieldPath: []string{"v1"},
	}
	v2 := &RoutingInfoVariant{
		FieldPath: []string{"v2"},
	}
	info := &RoutingInfo{
		Name:     "key",
		Variants: []*RoutingInfoVariant{v1, v2},
	}
	method := Method{
		Routing: []*RoutingInfo{info},
	}

	want := []*RoutingInfoCombo{
		{
			Items: []*RoutingInfoComboItem{
				{
					Name:    "key",
					Variant: v1,
				},
			},
		},
		{
			Items: []*RoutingInfoComboItem{
				{
					Name:    "key",
					Variant: v2,
				},
			},
		},
	}
	if diff := cmp.Diff(want, method.RoutingCombos()); diff != "" {
		t.Errorf("Incorrect routing combos (-want, +got):\n%s", diff)
	}
}

func TestRoutingCombosSimpleAnd(t *testing.T) {
	v1 := &RoutingInfoVariant{
		FieldPath: []string{"v1"},
	}
	i1 := &RoutingInfo{
		Name:     "key1",
		Variants: []*RoutingInfoVariant{v1},
	}
	v2 := &RoutingInfoVariant{
		FieldPath: []string{"v2"},
	}
	i2 := &RoutingInfo{
		Name:     "key2",
		Variants: []*RoutingInfoVariant{v2},
	}
	method := Method{
		Routing: []*RoutingInfo{i1, i2},
	}

	want := []*RoutingInfoCombo{
		{
			Items: []*RoutingInfoComboItem{
				{
					Name:    "key1",
					Variant: v1,
				},
				{
					Name:    "key2",
					Variant: v2,
				},
			},
		},
	}
	if diff := cmp.Diff(want, method.RoutingCombos()); diff != "" {
		t.Errorf("Incorrect routing combos (-want, +got):\n%s", diff)
	}
}

func TestRoutingCombosFull(t *testing.T) {
	va1 := &RoutingInfoVariant{
		FieldPath: []string{"va1"},
	}
	va2 := &RoutingInfoVariant{
		FieldPath: []string{"va2"},
	}
	va3 := &RoutingInfoVariant{
		FieldPath: []string{"va3"},
	}
	a := &RoutingInfo{
		Name:     "a",
		Variants: []*RoutingInfoVariant{va1, va2, va3},
	}

	vb1 := &RoutingInfoVariant{
		FieldPath: []string{"vb1"},
	}
	vb2 := &RoutingInfoVariant{
		FieldPath: []string{"vb2"},
	}
	b := &RoutingInfo{
		Name:     "b",
		Variants: []*RoutingInfoVariant{vb1, vb2},
	}

	vc1 := &RoutingInfoVariant{
		FieldPath: []string{"vc1"},
	}
	c := &RoutingInfo{
		Name:     "c",
		Variants: []*RoutingInfoVariant{vc1},
	}

	method := Method{
		Routing: []*RoutingInfo{a, b, c},
	}

	make_combo := func(va *RoutingInfoVariant, vb *RoutingInfoVariant, vc *RoutingInfoVariant) *RoutingInfoCombo {
		return &RoutingInfoCombo{
			Items: []*RoutingInfoComboItem{
				{
					Name:    "a",
					Variant: va,
				},
				{
					Name:    "b",
					Variant: vb,
				},
				{
					Name:    "c",
					Variant: vc,
				},
			},
		}
	}
	want := []*RoutingInfoCombo{
		make_combo(va1, vb1, vc1),
		make_combo(va1, vb2, vc1),
		make_combo(va2, vb1, vc1),
		make_combo(va2, vb2, vc1),
		make_combo(va3, vb1, vc1),
		make_combo(va3, vb2, vc1),
	}
	if diff := cmp.Diff(want, method.RoutingCombos()); diff != "" {
		t.Errorf("Incorrect routing combos (-want, +got):\n%s", diff)
	}
}

func TestRoutingInfoVarianFieldName(t *testing.T) {
	variant := &RoutingInfoVariant{
		FieldPath: []string{"request", "b", "c"},
	}
	got := variant.FieldName()
	want := "request.b.c"
	if got != want {
		t.Errorf("mismatch in FieldName got=%q, want=%q", got, want)
	}
}

func TestRoutingInfoVariantTemplateAsString(t *testing.T) {
	variant := &RoutingInfoVariant{
		Prefix: RoutingPathSpec{
			Segments: []string{"a", "b", "c"},
		},
		Matching: RoutingPathSpec{
			Segments: []string{"d", "*"},
		},
		Suffix: RoutingPathSpec{
			Segments: []string{"e", "**"},
		},
	}
	got := variant.TemplateAsString()
	want := "a/b/c/d/*/e/**"
	if got != want {
		t.Errorf("mismatch in TemplateAsString got=%q, want=%q", got, want)
	}
}

func TestPathTemplateBuilder(t *testing.T) {
	got := NewPathTemplate().
		WithLiteral("v1").
		WithVariable(NewPathVariable("parent", "child").
			WithLiteral("projects").
			WithMatch().
			WithAllowReserved().
			WithLiteral("locations").
			WithMatchRecursive()).
		WithVariableNamed("v2", "field").
		WithVerb("verb")
	name := "v1"
	verb := "verb"
	want := &PathTemplate{
		Segments: []PathSegment{
			{
				Literal: &name,
			},
			{
				Variable: &PathVariable{
					FieldPath:     []string{"parent", "child"},
					Segments:      []string{"projects", "*", "locations", "**"},
					AllowReserved: true,
				},
			},
			{
				Variable: &PathVariable{
					FieldPath: []string{"v2", "field"},
					Segments:  []string{"*"},
				},
			},
		},
		Verb: &verb,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("bad builder result (-want, +got):\n%s", diff)
	}
}

func TestPathBindingHeuristic(t *testing.T) {
	heuristic := &ResourceNameHeuristic{
		FieldPaths: [][]string{{"project"}, {"zone"}, {"instance"}},
		PathTemplate: NewPathTemplate().
			WithLiteral("projects").WithVariableNamed("project").
			WithLiteral("zones").WithVariableNamed("zone").
			WithLiteral("instances").WithVariableNamed("instance"),
	}
	binding := &PathBinding{
		Verb:                  "GET",
		ResourceNameHeuristic: heuristic,
	}

	if binding.Verb != "GET" {
		t.Errorf("expected GET, got %s", binding.Verb)
	}

	if binding.ResourceNameHeuristic.FieldPaths[0][0] != "project" {
		t.Errorf("expected project, got %s", binding.ResourceNameHeuristic.FieldPaths[0][0])
	}
}

func TestIsSimpleMethod(t *testing.T) {
	somePagination := &Field{}
	someOperationInfo := &OperationInfo{}
	someDiscoverLro := &DiscoveryLro{}
	testCases := []struct {
		name     string
		method   *Method
		isSimple bool
	}{
		{
			name:     "simple method",
			method:   &Method{},
			isSimple: true,
		},
		{
			name:     "pagination method",
			method:   &Method{Pagination: somePagination},
			isSimple: false,
		},
		{
			name:     "client streaming method",
			method:   &Method{ClientSideStreaming: true},
			isSimple: false,
		},
		{
			name:     "server streaming method",
			method:   &Method{ServerSideStreaming: true},
			isSimple: false,
		},
		{
			name:     "LRO method",
			method:   &Method{OperationInfo: someOperationInfo},
			isSimple: false,
		},
		{
			name:     "Discovery LRO method",
			method:   &Method{DiscoveryLro: someDiscoverLro},
			isSimple: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.method.IsSimple(); got != tc.isSimple {
				t.Errorf("IsSimple() = %v, want %v", got, tc.isSimple)
			}
		})
	}
}

func TestIsLRO(t *testing.T) {
	testCases := []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "simple method is not LRO",
			method: &Method{},
			want:   false,
		},
		{
			name:   "LRO method is LRO",
			method: &Method{OperationInfo: &OperationInfo{}},
			want:   true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.method.IsLRO(); got != tc.want {
				t.Errorf("IsLRO() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLongRunningHelpers(t *testing.T) {
	emptyMsg := &Message{ID: ".google.protobuf.Empty"}
	responseMsg := &Message{ID: "some.response.Message"}
	model := &API{
		State: &APIState{
			MessageByID: map[string]*Message{
				emptyMsg.ID:    emptyMsg,
				responseMsg.ID: responseMsg,
			},
		},
	}

	testCases := []struct {
		name         string
		method       *Method
		wantResponse *Message
		wantEmpty    bool
	}{
		{
			name: "LRO with empty response",
			method: &Method{
				OperationInfo: &OperationInfo{ResponseTypeID: emptyMsg.ID},
				Model:         model,
			},
			wantResponse: emptyMsg,
			wantEmpty:    true,
		},
		{
			name: "LRO with non-empty response",
			method: &Method{
				OperationInfo: &OperationInfo{ResponseTypeID: responseMsg.ID},
				Model:         model,
			},
			wantResponse: responseMsg,
			wantEmpty:    false,
		},
		{
			name: "non-LRO method",
			method: &Method{
				Model: model,
			},
			wantResponse: nil,
			wantEmpty:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.method.LongRunningResponseType(); got != tc.wantResponse {
				t.Errorf("LongRunningResponseType() = %v, want %v", got, tc.wantResponse)
			}
			if got := tc.method.LongRunningReturnsEmpty(); got != tc.wantEmpty {
				t.Errorf("LongRunningReturnsEmpty() = %v, want %v", got, tc.wantEmpty)
			}
		})
	}
}

func TestIsList(t *testing.T) {
	testCases := []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "list method",
			method: &Method{OutputType: &Message{Pagination: &PaginationInfo{}}},
			want:   true,
		},
		{
			name:   "simple method",
			method: &Method{},
			want:   false,
		},
		{
			name:   "no output type",
			method: &Method{OutputType: nil},
			want:   false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.method.IsList(); got != tc.want {
				t.Errorf("IsList() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsStreaming(t *testing.T) {
	testCases := []struct {
		name   string
		method *Method
		want   bool
	}{
		{
			name:   "unary method",
			method: &Method{},
			want:   false,
		},
		{
			name:   "client streaming",
			method: &Method{ClientSideStreaming: true},
			want:   true,
		},
		{
			name:   "server streaming",
			method: &Method{ServerSideStreaming: true},
			want:   true,
		},
		{
			name:   "bidi streaming",
			method: &Method{ClientSideStreaming: true, ServerSideStreaming: true},
			want:   true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.method.IsStreaming(); got != tc.want {
				t.Errorf("IsStreaming() = %v, want %v", got, tc.want)
			}
		})
	}
}

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
		State: &APIState{
			ResourceByType: map[string]*Resource{
				resource.Type:                resource,
				resourceWithoutSingular.Type: resourceWithoutSingular,
			},
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
			if got := tc.method.IsAIPStandard(); got != tc.want {
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
		want   *AIPStandardGetInfo
	}{
		{
			name: "valid get operation with wildcard resource reference",
			method: &Method{
				Name:       "GetSecret",
				InputType:  &Message{Name: "GetSecretRequest", Fields: []*Field{f.wildcardResourceField}},
				OutputType: output,
				Model:      f.model,
			},
			want: &AIPStandardGetInfo{
				ResourceNameRequestField: f.wildcardResourceField,
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
			want: &AIPStandardGetInfo{
				ResourceNameRequestField: f.resourceNameField,
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
			want: &AIPStandardGetInfo{
				ResourceNameRequestField: f.resourceNameNoSingularField,
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
			got := tc.method.AIPStandardGetInfo()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AIPStandardGetInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardDeleteInfo(t *testing.T) {
	f := newAIPTestFixture()

	testCases := []struct {
		name   string
		method *Method
		want   *AIPStandardDeleteInfo
	}{
		{
			name: "valid simple delete with wildcard resource reference",
			method: &Method{
				Name:         "DeleteSecret",
				InputType:    &Message{Name: "DeleteSecretRequest", Fields: []*Field{f.wildcardResourceField}},
				ReturnsEmpty: true,
				Model:        f.model,
			},
			want: &AIPStandardDeleteInfo{
				ResourceNameRequestField: f.wildcardResourceField,
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
			want: &AIPStandardDeleteInfo{
				ResourceNameRequestField: f.resourceNameField,
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
			want: &AIPStandardDeleteInfo{
				ResourceNameRequestField: f.resourceNameNoSingularField,
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
			want: &AIPStandardDeleteInfo{
				ResourceNameRequestField: f.resourceNameField,
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
			want: &AIPStandardDeleteInfo{
				ResourceNameRequestField: f.resourceOtherNameField,
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
			got := tc.method.AIPStandardDeleteInfo()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AIPStandardDeleteInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardUndeleteInfo(t *testing.T) {
	f := newAIPTestFixture()

	testCases := []struct {
		name   string
		method *Method
		want   *AIPStandardUndeleteInfo
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
			want: &AIPStandardUndeleteInfo{
				ResourceNameRequestField: f.wildcardResourceField,
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
			want: &AIPStandardUndeleteInfo{
				ResourceNameRequestField: f.resourceNameField,
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
			want: &AIPStandardUndeleteInfo{
				ResourceNameRequestField: f.resourceNameNoSingularField,
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
			want: &AIPStandardUndeleteInfo{
				ResourceNameRequestField: f.resourceNameField,
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
			want: &AIPStandardUndeleteInfo{
				ResourceNameRequestField: f.resourceOtherNameField,
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
			got := tc.method.AIPStandardUndeleteInfo()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AIPStandardUndeleteInfo() mismatch (-want +got):\n%s", diff)
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
		Typez:             STRING_TYPE,
	}
	resourceField := &Field{
		Name:    "secret",
		Typez:   MESSAGE_TYPE,
		TypezID: secretMessage.ID,
	}
	idField := &Field{
		Name:  "secret_id",
		Typez: STRING_TYPE,
	}

	testCases := []struct {
		name   string
		method *Method
		want   *AIPStandardCreateInfo
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
			want: &AIPStandardCreateInfo{
				ParentRequestField:     parentField,
				ResourceIDRequestField: idField,
				ResourceRequestField:   resourceField,
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
			want: &AIPStandardCreateInfo{
				ParentRequestField:   parentField,
				ResourceRequestField: resourceField,
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
			got := tc.method.AIPStandardCreateInfo()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AIPStandardCreateInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAIPStandardUpdateInfo(t *testing.T) {
	f := newAIPTestFixture()

	// Setup for Update
	secretMessage := &Message{ID: "secret_message_id"}
	f.resource.Self = secretMessage

	resourceField := &Field{
		Name:    "secret",
		Typez:   MESSAGE_TYPE,
		TypezID: secretMessage.ID,
	}
	updateMaskField := &Field{
		Name:    "update_mask",
		TypezID: ".google.protobuf.FieldMask",
	}

	testCases := []struct {
		name   string
		method *Method
		want   *AIPStandardUpdateInfo
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
			want: &AIPStandardUpdateInfo{
				ResourceRequestField:   resourceField,
				UpdateMaskRequestField: updateMaskField,
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
			want: &AIPStandardUpdateInfo{
				ResourceRequestField: resourceField,
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
			got := tc.method.AIPStandardUpdateInfo()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AIPStandardUpdateInfo() mismatch (-want +got):\n%s", diff)
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
		want   *AIPStandardListInfo
	}{
		{
			name: "valid list operation with parent field match by child_type",
			method: &Method{
				Name:       "ListSecrets",
				InputType:  &Message{Name: "ListSecretsRequest", Fields: []*Field{parentField}},
				OutputType: listOutput,
				Model:      f.model,
			},
			want: &AIPStandardListInfo{
				ParentRequestField: parentField,
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
			want: &AIPStandardListInfo{
				ParentRequestField: otherParentField,
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
			want: &AIPStandardListInfo{
				ParentRequestField: plainParentField,
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
			got := tc.method.AIPStandardListInfo()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AIPStandardListInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindBestResourceFieldByType(t *testing.T) {
	f := newAIPTestFixture()
	targetType := f.resource.Type

	for _, tc := range []struct {
		name   string
		fields []*Field
		want   *Field
	}{
		{
			name:   "name field with wildcard",
			fields: []*Field{f.wildcardResourceField},
			want:   f.wildcardResourceField,
		},
		{
			name:   "name field with exact match",
			fields: []*Field{f.resourceNameField},
			want:   f.resourceNameField,
		},
		{
			name:   "other field with exact match",
			fields: []*Field{f.resourceOtherNameField},
			want:   f.resourceOtherNameField,
		},
		{
			name:   "name field with exact match wins over other field with exact match",
			fields: []*Field{f.resourceNameField, f.resourceOtherNameField},
			want:   f.resourceNameField,
		},
		{
			name:   "name field with wildcard wins over other field with exact match",
			fields: []*Field{f.wildcardResourceField, f.resourceOtherNameField},
			want:   f.wildcardResourceField,
		},
		{
			name:   "no match",
			fields: []*Field{f.nonExistentResourceField},
			want:   nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := &Message{Fields: tc.fields}
			got := findBestResourceFieldByType(msg, f.model.State.ResourceByType, targetType)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("findBestResourceFieldByType() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindBestResourceFieldBySingular(t *testing.T) {
	f := newAIPTestFixture()
	targetSingular := f.resource.Singular

	for _, tc := range []struct {
		name   string
		fields []*Field
		want   *Field
	}{
		{
			name:   "name field with wildcard",
			fields: []*Field{f.wildcardResourceField},
			want:   f.wildcardResourceField,
		},
		{
			name:   "name field with exact match",
			fields: []*Field{f.resourceNameField},
			want:   f.resourceNameField,
		},
		{
			name:   "name field with empty singular match",
			fields: []*Field{f.resourceNameNoSingularField},
			want:   f.resourceNameNoSingularField,
		},
		{
			name:   "other field with exact match",
			fields: []*Field{f.resourceOtherNameField},
			want:   f.resourceOtherNameField,
		},
		{
			name:   "name field with exact match wins over other field with exact match",
			fields: []*Field{f.resourceNameField, f.resourceOtherNameField},
			want:   f.resourceNameField,
		},
		{
			name:   "name field with wildcard wins over other field with exact match",
			fields: []*Field{f.wildcardResourceField, f.resourceOtherNameField},
			want:   f.wildcardResourceField,
		},
		{
			name:   "no match",
			fields: []*Field{f.nonExistentResourceField},
			want:   nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := &Message{Fields: tc.fields}
			got := findBestResourceFieldBySingular(msg, f.model.State.ResourceByType, targetSingular)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("findBestResourceFieldBySingular() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindBestParentFieldByType(t *testing.T) {
	childType := "google.cloud.secretmanager.v1/SecretVersion"

	parentField := &Field{
		Name:              "parent",
		ResourceReference: &ResourceReference{ChildType: childType},
	}

	parentFieldByName := &Field{
		Name: "parent",
		// No matching child type, just name
	}

	parentFieldByChildType := &Field{
		Name:              "database",
		ResourceReference: &ResourceReference{ChildType: childType},
	}

	wrongField := &Field{Name: "wrong"}

	for _, tc := range []struct {
		name   string
		fields []*Field
		want   *Field
	}{
		{
			name:   "exact match (name + child type)",
			fields: []*Field{parentField},
			want:   parentField,
		},
		{
			name:   "name match only (fallback)",
			fields: []*Field{parentFieldByName},
			want:   parentFieldByName,
		},
		{
			name:   "child type match only",
			fields: []*Field{parentFieldByChildType},
			want:   parentFieldByChildType,
		},
		{
			name:   "name match prefers exact name",
			fields: []*Field{parentFieldByName, parentFieldByChildType},
			want:   parentFieldByName,
		},
		{
			name:   "no match",
			fields: []*Field{wrongField},
			want:   nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := &Message{Fields: tc.fields}
			got := findBestParentFieldByType(msg, childType)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("findBestParentFieldByType() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldTypePredicates(t *testing.T) {
	type TestCase struct {
		field    *Field
		isString bool
		isBytes  bool
		isBool   bool
		isInt    bool
		isUInt   bool
		isFloat  bool
		isEnum   bool
		isObject bool
	}
	testCases := []TestCase{
		{field: &Field{Typez: STRING_TYPE}, isString: true},
		{field: &Field{Typez: BYTES_TYPE}, isBytes: true},
		{field: &Field{Typez: BOOL_TYPE}, isBool: true},
		{field: &Field{Typez: INT32_TYPE}, isInt: true},
		{field: &Field{Typez: INT64_TYPE}, isInt: true},
		{field: &Field{Typez: SINT32_TYPE}, isInt: true},
		{field: &Field{Typez: SINT64_TYPE}, isInt: true},
		{field: &Field{Typez: SFIXED32_TYPE}, isInt: true},
		{field: &Field{Typez: SFIXED64_TYPE}, isInt: true},
		{field: &Field{Typez: UINT32_TYPE}, isUInt: true},
		{field: &Field{Typez: UINT64_TYPE}, isUInt: true},
		{field: &Field{Typez: FIXED32_TYPE}, isUInt: true},
		{field: &Field{Typez: FIXED64_TYPE}, isUInt: true},
		{field: &Field{Typez: FLOAT_TYPE}, isFloat: true},
		{field: &Field{Typez: DOUBLE_TYPE}, isFloat: true},
		{field: &Field{Typez: ENUM_TYPE}, isEnum: true},
		{field: &Field{Typez: MESSAGE_TYPE}, isObject: true},
	}
	for _, tc := range testCases {
		if tc.field.IsString() != tc.isString {
			t.Errorf("IsString() for %v should be %v", tc.field.Typez, tc.isString)
		}
		if tc.field.IsBytes() != tc.isBytes {
			t.Errorf("IsBytes() for %v should be %v", tc.field.Typez, tc.isBytes)
		}
		if tc.field.IsBool() != tc.isBool {
			t.Errorf("IsBool() for %v should be %v", tc.field.Typez, tc.isBool)
		}
		if tc.field.IsLikeInt() != tc.isInt {
			t.Errorf("IsLikeInt() for %v should be %v", tc.field.Typez, tc.isInt)
		}
		if tc.field.IsLikeUInt() != tc.isUInt {
			t.Errorf("IsLikeUInt() for %v should be %v", tc.field.Typez, tc.isUInt)
		}
		if tc.field.IsLikeFloat() != tc.isFloat {
			t.Errorf("IsLikeFloat() for %v should be %v", tc.field.Typez, tc.isFloat)
		}
		if tc.field.IsEnum() != tc.isEnum {
			t.Errorf("IsEnum() for %v should be %v", tc.field.Typez, tc.isEnum)
		}
		if tc.field.IsObject() != tc.isObject {
			t.Errorf("IsObject() for %v should be %v", tc.field.Typez, tc.isObject)
		}
	}
}

func TestFlatPath(t *testing.T) {
	for _, test := range []struct {
		Input *PathTemplate
		Want  string
	}{
		{
			Input: NewPathTemplate(),
			Want:  "",
		},
		{
			Input: NewPathTemplate().
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("zones").
				WithVariableNamed("zone"),
			Want: "projects/{project}/zones/{zone}",
		},
		{
			Input: NewPathTemplate().
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("global").
				WithLiteral("location"),
			Want: "projects/{project}/global/location",
		},
		{
			Input: NewPathTemplate().
				WithLiteral("projects").
				WithVariable(NewPathVariable("a", "b", "c").WithMatchRecursive()),
			Want: "projects/{a.b.c}",
		},
	} {
		got := test.Input.FlatPath()
		if got != test.Want {
			t.Errorf("mismatch want=%q, got=%q", test.Want, got)
		}
	}
}

func TestField_IsResourceReference(t *testing.T) {
	for _, test := range []struct {
		name  string
		field *Field
		want  bool
	}{
		{
			name:  "nil ResourceReference",
			field: &Field{},
			want:  false,
		},
		{
			name:  "non-nil ResourceReference",
			field: &Field{ResourceReference: &ResourceReference{}},
			want:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.field.IsResourceReference()
			if got != test.want {
				t.Errorf("IsResourceReference() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFindBodyField(t *testing.T) {
	messageType := &Message{ID: "message_id"}
	bodyField := &Field{Name: "body_field"}
	typeMatchField := &Field{Typez: MESSAGE_TYPE, TypezID: messageType.ID, Name: "type_match"}
	otherField := &Field{Name: "other"}

	testCases := []struct {
		name       string
		message    *Message
		pathInfo   *PathInfo
		targetType string
		singular   string
		want       *Field
	}{
		{
			name:     "match by path info",
			message:  &Message{Fields: []*Field{bodyField, otherField}},
			pathInfo: &PathInfo{BodyFieldPath: "body_field"},
			want:     bodyField,
		},
		{
			name:       "match by type and name",
			message:    &Message{Fields: []*Field{typeMatchField, otherField}},
			targetType: messageType.ID,
			singular:   "type_match",
			want:       typeMatchField,
		},
		{
			name:       "match by type but wrong name",
			message:    &Message{Fields: []*Field{typeMatchField, otherField}},
			targetType: messageType.ID,
			singular:   "wrong_name",
			want:       nil,
		},
		{
			name:       "match by name but wrong type",
			message:    &Message{Fields: []*Field{typeMatchField, otherField}},
			targetType: "different_message_id",
			singular:   "type_match",
			want:       nil,
		},
		{
			name:       "path info overrides type match",
			message:    &Message{Fields: []*Field{typeMatchField, bodyField}},
			pathInfo:   &PathInfo{BodyFieldPath: "body_field"},
			targetType: messageType.ID,
			singular:   "type_match",
			want:       bodyField,
		},
		{
			name:    "no match",
			message: &Message{Fields: []*Field{otherField}},
			want:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findBodyField(tc.message, tc.pathInfo, tc.targetType, tc.singular)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("findBodyField() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindResourceIDField(t *testing.T) {
	idField := &Field{Name: "book_id", Typez: STRING_TYPE}
	otherField := &Field{Name: "other"}

	testCases := []struct {
		name     string
		message  *Message
		singular string
		want     *Field
	}{
		{
			name:     "found id field",
			message:  &Message{Fields: []*Field{idField, otherField}},
			singular: "book",
			want:     idField,
		},
		{
			name:     "not found",
			message:  &Message{Fields: []*Field{otherField}},
			singular: "book",
			want:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findResourceIDField(tc.message, tc.singular)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("findResourceIDField() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
