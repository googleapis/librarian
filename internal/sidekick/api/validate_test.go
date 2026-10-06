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

func TestValidate(t *testing.T) {
	model := NewTestAPI(
		[]*Message{NewTestMessage("m1").WithPackage("p1")},
		[]*Enum{NewTestEnum("e1").WithPackage("p1")},
		[]*Service{NewTestService("s1").WithPackage("p1")}).
		WithPackageName("p1")
	if err := Validate(model); err != nil {
		t.Errorf("unexpected error in API validation %q", err)
	}
}

func TestValidateMessageMismatch(t *testing.T) {
	test := NewTestAPI(
		[]*Message{NewTestMessage("m1").WithPackage("p1"), NewTestMessage("m2").WithPackage("p2")},
		[]*Enum{NewTestEnum("e1").WithPackage("p1")},
		[]*Service{NewTestService("s1").WithPackage("p1")}).
		WithPackageName("p1")
	if err := Validate(test); err == nil {
		t.Errorf("expected an error in API validation got=%s", test.PackageName)
	}

	test = NewTestAPI(
		[]*Message{NewTestMessage("m1").WithPackage("p1")},
		[]*Enum{NewTestEnum("e1").WithPackage("p1"), NewTestEnum("e2").WithPackage("p2")},
		[]*Service{NewTestService("s1").WithPackage("p1")}).
		WithPackageName("p1")
	if err := Validate(test); err == nil {
		t.Errorf("expected an error in API validation got=%s", test.PackageName)
	}

	test = NewTestAPI(
		[]*Message{NewTestMessage("m1").WithPackage("p1")},
		[]*Enum{NewTestEnum("e1").WithPackage("p1")},
		[]*Service{NewTestService("s1").WithPackage("p1"), NewTestService("s2").WithPackage("p2")}).
		WithPackageName("p1")
	if err := Validate(test); err == nil {
		t.Errorf("expected an error in API validation got=%s", test.PackageName)
	}
}

func TestValidateMessageMismatchNoPackage(t *testing.T) {
	test := NewTestAPI(
		[]*Message{NewTestMessage("m1").WithPackage("p1"), NewTestMessage("m2").WithPackage("p2")},
		[]*Enum{NewTestEnum("e1").WithPackage("p1")},
		[]*Service{NewTestService("s1").WithPackage("p1")}).
		WithPackageName("")
	if err := Validate(test); err == nil {
		t.Errorf("expected an error in API validation got=%s", test.PackageName)
	}

	test = NewTestAPI(
		[]*Message{NewTestMessage("m1").WithPackage("p1")},
		[]*Enum{NewTestEnum("e1").WithPackage("p1"), NewTestEnum("e2").WithPackage("p2")},
		[]*Service{NewTestService("s1").WithPackage("p1")}).
		WithPackageName("")
	if err := Validate(test); err == nil {
		t.Errorf("expected an error in API validation got=%s", test.PackageName)
	}

	test = NewTestAPI(
		[]*Message{NewTestMessage("m1").WithPackage("p1")},
		[]*Enum{NewTestEnum("e1").WithPackage("p1")},
		[]*Service{NewTestService("s1").WithPackage("p1"), NewTestService("s2").WithPackage("p2")}).
		WithPackageName("")
	if err := Validate(test); err == nil {
		t.Errorf("expected an error in API validation got=%s", test.PackageName)
	}
}

func TestValidateMap(t *testing.T) {
	validMap := NewTestMap("ValidMap", TypezString, TypezInt32)
	model := NewTestAPI(nil, nil, nil).WithPackageName("p1").WithMaps(validMap)
	if err := Validate(model); err != nil {
		t.Fatal(err)
	}
}

func TestValidateMap_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		mapType *Map
		wantErr string
	}{
		{
			name:    "nil key",
			mapType: &Map{ID: ".test.NilKey", Value: NewTestField("value").WithType(TypezString)},
			wantErr: `map ".test.NilKey" must have non-nil key and value fields`,
		},
		{
			name:    "nil value",
			mapType: &Map{ID: ".test.NilValue", Key: NewTestField("key").WithType(TypezString)},
			wantErr: `map ".test.NilValue" must have non-nil key and value fields`,
		},
		{
			name: "key is map type",
			mapType: NewTestMapWithFields(
				"KeyIsMap",
				NewTestField("key").WithType(TypezMap),
				NewTestField("value").WithType(TypezString),
			),
			wantErr: `map ".test.KeyIsMap" key cannot be a map or repeated field`,
		},
		{
			name: "key is map flag",
			mapType: NewTestMapWithFields(
				"KeyIsMapFlag",
				NewTestField("key").WithType(TypezString).WithMap(),
				NewTestField("value").WithType(TypezString),
			),
			wantErr: `map ".test.KeyIsMapFlag" key cannot be a map or repeated field`,
		},
		{
			name: "key is repeated",
			mapType: NewTestMapWithFields(
				"KeyIsRepeated",
				NewTestField("key").WithType(TypezString).WithRepeated(),
				NewTestField("value").WithType(TypezString),
			),
			wantErr: `map ".test.KeyIsRepeated" key cannot be a map or repeated field`,
		},
		{
			name: "value is map type",
			mapType: NewTestMapWithFields(
				"ValueIsMap",
				NewTestField("key").WithType(TypezString),
				NewTestField("value").WithType(TypezMap),
			),
			wantErr: `map ".test.ValueIsMap" value cannot be a map or repeated field`,
		},
		{
			name: "value is map flag",
			mapType: NewTestMapWithFields(
				"ValueIsMapFlag",
				NewTestField("key").WithType(TypezString),
				NewTestField("value").WithType(TypezString).WithMap(),
			),
			wantErr: `map ".test.ValueIsMapFlag" value cannot be a map or repeated field`,
		},
		{
			name: "value is repeated",
			mapType: NewTestMapWithFields(
				"ValueIsRepeated",
				NewTestField("key").WithType(TypezString),
				NewTestField("value").WithType(TypezString).WithRepeated(),
			),
			wantErr: `map ".test.ValueIsRepeated" value cannot be a map or repeated field`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := NewTestAPI(nil, nil, nil).WithPackageName("p1").WithMaps(test.mapType)
			err := Validate(model)
			if err == nil {
				t.Fatalf("expected error %q, got nil", test.wantErr)
			}
			if diff := cmp.Diff(test.wantErr, err.Error()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
