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
	targetType := f.resource.Type()

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
			msg := NewTestMessage("Message").WithFields(tc.fields...)
			got := findBestResourceFieldByType(msg, f.model, targetType)
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindBestResourceFieldBySingular(t *testing.T) {
	f := newAIPTestFixture()
	targetSingular := f.resource.Singular()

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
			msg := NewTestMessage("Message").WithFields(tc.fields...)
			got := findBestResourceFieldBySingular(msg, f.model, targetSingular)
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindBestParentFieldByType(t *testing.T) {
	childType := "google.cloud.secretmanager.v1/SecretVersion"

	parentField := NewTestField("parent").WithChildTypeReference(childType)
	parentFieldByName := NewTestField("parent")
	parentFieldByChildType := NewTestField("database").WithChildTypeReference(childType)
	wrongField := NewTestField("wrong")

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
			msg := NewTestMessage("Message").WithFields(tc.fields...)
			got := findBestParentFieldByType(msg, childType)
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
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
		{field: NewTestField("f").WithType(TypezString), isString: true},
		{field: NewTestField("f").WithType(TypezBytes), isBytes: true},
		{field: NewTestField("f").WithType(TypezBool), isBool: true},
		{field: NewTestField("f").WithType(TypezInt32), isInt: true},
		{field: NewTestField("f").WithType(TypezInt64), isInt: true},
		{field: NewTestField("f").WithType(TypezSint32), isInt: true},
		{field: NewTestField("f").WithType(TypezSint64), isInt: true},
		{field: NewTestField("f").WithType(TypezSfixed32), isInt: true},
		{field: NewTestField("f").WithType(TypezSfixed64), isInt: true},
		{field: NewTestField("f").WithType(TypezUint32), isUInt: true},
		{field: NewTestField("f").WithType(TypezUint64), isUInt: true},
		{field: NewTestField("f").WithType(TypezFixed32), isUInt: true},
		{field: NewTestField("f").WithType(TypezFixed64), isUInt: true},
		{field: NewTestField("f").WithType(TypezFloat), isFloat: true},
		{field: NewTestField("f").WithType(TypezDouble), isFloat: true},
		{field: NewTestField("f").WithType(TypezEnum), isEnum: true},
		{field: NewTestField("f").WithType(TypezMessage), isObject: true},
	}
	for _, tc := range testCases {
		if tc.field.IsString() != tc.isString {
			t.Errorf("IsString() for %v should be %v", tc.field.Typez(), tc.isString)
		}
		if tc.field.IsBytes() != tc.isBytes {
			t.Errorf("IsBytes() for %v should be %v", tc.field.Typez(), tc.isBytes)
		}
		if tc.field.IsBool() != tc.isBool {
			t.Errorf("IsBool() for %v should be %v", tc.field.Typez(), tc.isBool)
		}
		if tc.field.IsLikeInt() != tc.isInt {
			t.Errorf("IsLikeInt() for %v should be %v", tc.field.Typez(), tc.isInt)
		}
		if tc.field.IsLikeUInt() != tc.isUInt {
			t.Errorf("IsLikeUInt() for %v should be %v", tc.field.Typez(), tc.isUInt)
		}
		if tc.field.IsLikeFloat() != tc.isFloat {
			t.Errorf("IsLikeFloat() for %v should be %v", tc.field.Typez(), tc.isFloat)
		}
		if tc.field.IsEnum() != tc.isEnum {
			t.Errorf("IsEnum() for %v should be %v", tc.field.Typez(), tc.isEnum)
		}
		if tc.field.IsObject() != tc.isObject {
			t.Errorf("IsObject() for %v should be %v", tc.field.Typez(), tc.isObject)
		}
	}
}

func TestFlatPath(t *testing.T) {
	for _, test := range []struct {
		Input *PathTemplate
		Want  string
	}{
		{
			Input: NewPathTemplateBuilder().Build(),
			Want:  "",
		},
		{
			Input: NewPathTemplateBuilder().
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("zones").
				WithVariableNamed("zone").
				Build(),
			Want: "projects/{project}/zones/{zone}",
		},
		{
			Input: NewPathTemplateBuilder().
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("global").
				WithLiteral("location").
				Build(),
			Want: "projects/{project}/global/location",
		},
		{
			Input: NewPathTemplateBuilder().
				WithLiteral("projects").
				WithVariable(NewPathVariableBuilder("a", "b", "c").WithMatchRecursive().Build()).
				Build(),
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
			field: NewTestField("f"),
			want:  false,
		},
		{
			name:  "non-nil ResourceReference",
			field: NewTestField("f").WithResourceReference("type"),
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
	messageType := NewTestMessage("message_id").WithID("message_id")
	bodyField := NewTestField("body_field")
	typeMatchField := NewTestField("type_match").WithMessageType(messageType)
	otherField := NewTestField("other")

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
			message:  NewTestMessage("Request").WithFields(bodyField, otherField),
			pathInfo: NewPathInfoBuilder().WithBodyFieldPath("body_field").Build(),
			want:     bodyField,
		},
		{
			name:       "match by type and name",
			message:    NewTestMessage("Request").WithFields(typeMatchField, otherField),
			targetType: messageType.ID,
			singular:   "type_match",
			want:       typeMatchField,
		},
		{
			name:       "match by type but wrong name",
			message:    NewTestMessage("Request").WithFields(typeMatchField, otherField),
			targetType: messageType.ID,
			singular:   "wrong_name",
			want:       nil,
		},
		{
			name:       "match by name but wrong type",
			message:    NewTestMessage("Request").WithFields(typeMatchField, otherField),
			targetType: "different_message_id",
			singular:   "type_match",
			want:       nil,
		},
		{
			name:       "path info overrides type match",
			message:    NewTestMessage("Request").WithFields(typeMatchField, bodyField),
			pathInfo:   NewPathInfoBuilder().WithBodyFieldPath("body_field").Build(),
			targetType: messageType.ID,
			singular:   "type_match",
			want:       bodyField,
		},
		{
			name:    "no match",
			message: NewTestMessage("Request").WithFields(otherField),
			want:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findBodyField(tc.message, tc.pathInfo, tc.targetType, tc.singular)
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindResourceIDField(t *testing.T) {
	idField := NewTestField("book_id").WithType(TypezString)
	otherField := NewTestField("other")

	testCases := []struct {
		name     string
		message  *Message
		singular string
		want     *Field
	}{
		{
			name:     "found id field",
			message:  NewTestMessage("Request").WithFields(idField, otherField),
			singular: "book",
			want:     idField,
		},
		{
			name:     "not found",
			message:  NewTestMessage("Request").WithFields(otherField),
			singular: "book",
			want:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findResourceIDField(tc.message, tc.singular)
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindQuickstartMethod(t *testing.T) {
	fooMethod := NewTestMethod("FooMethod")
	listPolicies := NewTestMethod("ListAccessPolicies").
		WithOutput(NewTestMessage("ListAccessPoliciesResponse").WithResource(NewTestResource("accesspolicy").WithSingular("accesspolicy")))
	listPolicies.IsAIPStandardList = true
	getPolicy := NewTestMethod("GetAccessPolicy")
	getPolicy.IsAIPStandardGet = true
	createPolicy := NewTestMethod("CreateAccessPolicy")
	createPolicy.IsAIPStandardCreate = true
	deletePolicy := NewTestMethod("DeleteAccessPolicy")
	deletePolicy.IsAIPStandardDelete = true
	updatePolicy := NewTestMethod("UpdateAccessPolicy")
	updatePolicy.IsAIPStandardUpdate = true
	listOther := NewTestMethod("ListOtherThings").
		WithOutput(NewTestMessage("ListOtherThingsResponse").WithResource(NewTestResource("otherthing").WithSingular("otherthing")))
	listOther.IsAIPStandardList = true

	deprecatedList := NewTestMethod("DeprecatedList").WithDeprecated(true)
	listPoliciesMatch := NewTestMethod("ListPolicies").
		WithOutput(NewTestMessage("ListPoliciesResponse").WithResource(NewTestResource("accesspolicy").WithSingular("accesspolicy")))
	listPoliciesMatch.IsAIPStandardList = true
	expandMethod := NewTestMethod("Expand").WithServerSideStreaming()
	chatMethod := NewTestMethod("Chat").WithBidiStreaming()

	testCases := []struct {
		name    string
		service *Service
		want    *Method
	}{
		{
			name:    "empty service",
			service: NewTestService("AccessPolicyService"),
			want:    nil,
		},
		{
			name:    "fallback to simple method when no standard methods exist",
			service: NewTestService("AccessPolicyService").WithMethods(fooMethod),
			want:    fooMethod,
		},
		{
			name:    "prefer non-deprecated simple method",
			service: NewTestService("AccessPolicyService").WithMethods(deprecatedList, fooMethod),
			want:    fooMethod,
		},
		{
			name:    "only get method",
			service: NewTestService("AccessPolicyService").WithMethods(getPolicy),
			want:    getPolicy,
		},
		{
			name:    "prioritizes list over get",
			service: NewTestService("AccessPolicyService").WithMethods(getPolicy, listOther),
			want:    listOther,
		},
		{
			name:    "prioritizes create over delete",
			service: NewTestService("AccessPolicyService").WithMethods(deletePolicy, createPolicy),
			want:    createPolicy,
		},
		{
			name:    "prioritizes delete over update",
			service: NewTestService("AccessPolicyService").WithMethods(updatePolicy, deletePolicy),
			want:    deletePolicy,
		},
		{
			name:    "tie-breaking on name matching (ListAccessPolicies vs ListOtherThings for AccessPolicyService)",
			service: NewTestService("AccessPolicyService").WithMethods(listOther, listPolicies),
			want:    listPolicies,
		},
		{
			name:    "tie-breaking fallback to resource singular/plural",
			service: NewTestService("AccessPolicyService").WithMethods(listOther, listPoliciesMatch),
			want:    listPoliciesMatch,
		},
		{
			name:    "fallback to server streaming method when no simple methods exist",
			service: NewTestService("EchoService").WithMethods(expandMethod),
			want:    expandMethod,
		},
		{
			name:    "fallback to bidi streaming method when no unary or server streaming methods exist",
			service: NewTestService("EchoService").WithMethods(chatMethod),
			want:    chatMethod,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findQuickstartMethod(tc.service)
			if diff := cmp.Diff(tc.want, got, allowUnexported(), cmpopts.IgnoreFields(Method{}, "Service", "Model")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindQuickstartService(t *testing.T) {
	fooMethod := NewTestMethod("FooMethod")
	serviceA := NewTestService("ServiceA").WithMethods(fooMethod)
	serviceB := NewTestService("SecretManagerService").WithMethods(fooMethod)
	deprecatedService := NewTestService("SecretManagerService").WithDeprecated(true).WithMethods(fooMethod)

	testCases := []struct {
		name string
		api  *API
		want *Service
	}{
		{
			name: "no services",
			api:  NewTestAPI(nil, nil, nil),
			want: nil,
		},
		{
			name: "one service",
			api:  NewTestAPI(nil, nil, []*Service{serviceA}),
			want: serviceA,
		},
		{
			name: "match service name to api name",
			api:  NewTestAPI(nil, nil, []*Service{serviceA, serviceB}).WithName("secretmanager"),
			want: serviceB,
		},
		{
			name: "no match defaults to first",
			api:  NewTestAPI(nil, nil, []*Service{serviceA, serviceB}).WithName("otherapi"),
			want: serviceA,
		},
		{
			name: "prefer non-deprecated service",
			api:  NewTestAPI(nil, nil, []*Service{deprecatedService, serviceA}).WithName("secretmanager"),
			want: serviceA,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findQuickstartService(tc.api)
			if diff := cmp.Diff(tc.want, got, allowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
