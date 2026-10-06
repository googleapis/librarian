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
	"github.com/google/go-cmp/cmp/cmpopts"
)

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
			got := findBestResourceFieldByType(msg, f.model, targetType)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
			got := findBestResourceFieldBySingular(msg, f.model, targetSingular)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
		{field: &Field{Typez: TypezString}, isString: true},
		{field: &Field{Typez: TypezBytes}, isBytes: true},
		{field: &Field{Typez: TypezBool}, isBool: true},
		{field: &Field{Typez: TypezInt32}, isInt: true},
		{field: &Field{Typez: TypezInt64}, isInt: true},
		{field: &Field{Typez: TypezSint32}, isInt: true},
		{field: &Field{Typez: TypezSint64}, isInt: true},
		{field: &Field{Typez: TypezSfixed32}, isInt: true},
		{field: &Field{Typez: TypezSfixed64}, isInt: true},
		{field: &Field{Typez: TypezUint32}, isUInt: true},
		{field: &Field{Typez: TypezUint64}, isUInt: true},
		{field: &Field{Typez: TypezFixed32}, isUInt: true},
		{field: &Field{Typez: TypezFixed64}, isUInt: true},
		{field: &Field{Typez: TypezFloat}, isFloat: true},
		{field: &Field{Typez: TypezDouble}, isFloat: true},
		{field: &Field{Typez: TypezEnum}, isEnum: true},
		{field: &Field{Typez: TypezMessage}, isObject: true},
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
			Input: (&PathTemplate{}),
			Want:  "",
		},
		{
			Input: (&PathTemplate{}).
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("zones").
				WithVariableNamed("zone"),
			Want: "projects/{project}/zones/{zone}",
		},
		{
			Input: (&PathTemplate{}).
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("global").
				WithLiteral("location"),
			Want: "projects/{project}/global/location",
		},
		{
			Input: (&PathTemplate{}).
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
	typeMatchField := &Field{Typez: TypezMessage, TypezID: messageType.ID, Name: "type_match"}
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
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindResourceIDField(t *testing.T) {
	idField := &Field{Name: "book_id", Typez: TypezString}
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
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindQuickstartMethod(t *testing.T) {
	fooMethod := &Method{Name: "FooMethod"}
	listPolicies := &Method{Name: "ListAccessPolicies", IsAIPStandardList: true, OutputType: &Message{Resource: &Resource{Singular: "accesspolicy"}}}
	getPolicy := &Method{Name: "GetAccessPolicy", IsAIPStandardGet: true}
	createPolicy := &Method{Name: "CreateAccessPolicy", IsAIPStandardCreate: true}
	deletePolicy := &Method{Name: "DeleteAccessPolicy", IsAIPStandardDelete: true}
	updatePolicy := &Method{Name: "UpdateAccessPolicy", IsAIPStandardUpdate: true}
	listOther := &Method{Name: "ListOtherThings", IsAIPStandardList: true, OutputType: &Message{Resource: &Resource{Singular: "otherthing"}}}

	testCases := []struct {
		name    string
		service *Service
		want    *Method
	}{
		{
			name:    "empty service",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{}},
			want:    nil,
		},
		{
			name:    "fallback to simple method when no standard methods exist",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{fooMethod}},
			want:    fooMethod,
		},
		{
			name:    "prefer non-deprecated simple method",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{{Name: "DeprecatedList", Deprecated: true}, fooMethod}},
			want:    fooMethod,
		},
		{
			name:    "only get method",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{getPolicy}},
			want:    getPolicy,
		},
		{
			name:    "prioritizes list over get",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{getPolicy, listOther}},
			want:    listOther,
		},
		{
			name:    "prioritizes create over delete",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{deletePolicy, createPolicy}},
			want:    createPolicy,
		},
		{
			name:    "prioritizes delete over update",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{updatePolicy, deletePolicy}},
			want:    deletePolicy,
		},
		{
			name:    "tie-breaking on name matching (ListAccessPolicies vs ListOtherThings for AccessPolicyService)",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{listOther, listPolicies}},
			want:    listPolicies,
		},
		{
			name:    "tie-breaking fallback to resource singular/plural",
			service: &Service{Name: "AccessPolicyService", Methods: []*Method{listOther, {Name: "ListPolicies", IsAIPStandardList: true, OutputType: &Message{Resource: &Resource{Singular: "accesspolicy"}}}}},
			want:    &Method{Name: "ListPolicies", IsAIPStandardList: true, OutputType: &Message{Resource: &Resource{Singular: "accesspolicy"}}}, // matches singular 'accesspolicy'
		},
		{
			name:    "fallback to server streaming method when no simple methods exist",
			service: &Service{Name: "EchoService", Methods: []*Method{{Name: "Expand", ServerSideStreaming: true}}},
			want:    &Method{Name: "Expand", ServerSideStreaming: true},
		},
		{
			name:    "fallback to bidi streaming method when no unary or server streaming methods exist",
			service: &Service{Name: "EchoService", Methods: []*Method{{Name: "Chat", ClientSideStreaming: true, ServerSideStreaming: true}}},
			want:    &Method{Name: "Chat", ClientSideStreaming: true, ServerSideStreaming: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findQuickstartMethod(tc.service)
			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(Method{}, "Service", "Model")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindQuickstartService(t *testing.T) {
	fooMethod := &Method{Name: "FooMethod"}
	serviceA := &Service{Name: "ServiceA", Methods: []*Method{fooMethod}}
	serviceB := &Service{Name: "SecretManagerService", Methods: []*Method{fooMethod}}
	deprecatedService := &Service{Name: "SecretManagerService", Deprecated: true, Methods: []*Method{fooMethod}}

	testCases := []struct {
		name string
		api  *API
		want *Service
	}{
		{
			name: "no services",
			api:  &API{Name: "secretmanager", Services: nil},
			want: nil,
		},
		{
			name: "one service",
			api:  &API{Name: "secretmanager", Services: []*Service{serviceA}},
			want: serviceA,
		},
		{
			name: "match service name to api name",
			api:  &API{Name: "secretmanager", Services: []*Service{serviceA, serviceB}},
			want: serviceB,
		},
		{
			name: "no match defaults to first",
			api:  &API{Name: "otherapi", Services: []*Service{serviceA, serviceB}},
			want: serviceA,
		},
		{
			name: "prefer non-deprecated service",
			api:  &API{Name: "secretmanager", Services: []*Service{deprecatedService, serviceA}},
			want: serviceA,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findQuickstartService(tc.api)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
