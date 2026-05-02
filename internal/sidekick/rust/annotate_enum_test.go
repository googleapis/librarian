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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestEnumAnnotations(t *testing.T) {
	// Verify we can handle values that are not in SCREAMING_SNAKE_CASE style.
	v0 := &api.EnumValue{
		Name:          "week5",
		ID:            ".test.v1.TestEnum.week5",
		Documentation: "week5 is also documented.",
		Number:        2,
	}
	v1 := &api.EnumValue{
		Name:          "MULTI_WORD_VALUE",
		ID:            ".test.v1.TestEnum.MULTI_WORD_VALUES",
		Documentation: "MULTI_WORD_VALUE is also documented.",
		Number:        1,
	}
	v2 := &api.EnumValue{
		Name:          "VALUE",
		ID:            ".test.v1.TestEnum.VALUE",
		Documentation: "VALUE is also documented.",
		Number:        0,
	}
	v3 := &api.EnumValue{
		Name:   "TEST_ENUM_V3",
		ID:     ".test.v1.TestEnum.TEST_ENUM_V3",
		Number: 3,
	}
	v4 := &api.EnumValue{
		Name:   "TEST_ENUM_2025",
		ID:     ".test.v1.TestEnum.TEST_ENUM_2025",
		Number: 4,
	}
	enum := &api.Enum{
		Name:          "TestEnum",
		ID:            ".test.v1.TestEnum",
		Package:       "test.v1",
		Documentation: "The enum is documented.",
		Values:        []*api.EnumValue{v0, v1, v2, v3, v4},
	}

	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	annotateModel(model, codec)

	wantEnumCodec := &enumAnnotation{
		Name:           "TestEnum",
		ModuleName:     "test_enum",
		QualifiedName:  "crate::model::TestEnum",
		RelativeName:   "TestEnum",
		DocLines:       []string{"/// The enum is documented."},
		UniqueNames:    []*api.EnumValue{v0, v1, v2, v3, v4},
		NameInExamples: "google_cloud_test_v1::model::TestEnum",
	}
	if diff := cmp.Diff(wantEnumCodec, enum.Codec, cmpopts.IgnoreFields(api.EnumValue{}, "Codec", "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantEnumValueCodec := &enumValueAnnotation{
		Name:        "WEEK_5",
		VariantName: "Week5",
		EnumType:    "TestEnum",
		DocLines:    []string{"/// week5 is also documented."},
	}
	if diff := cmp.Diff(wantEnumValueCodec, v0.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantEnumValueCodec = &enumValueAnnotation{
		Name:        "MULTI_WORD_VALUE",
		VariantName: "MultiWordValue",
		EnumType:    "TestEnum",
		DocLines:    []string{"/// MULTI_WORD_VALUE is also documented."},
	}
	if diff := cmp.Diff(wantEnumValueCodec, v1.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantEnumValueCodec = &enumValueAnnotation{
		Name:        "VALUE",
		VariantName: "Value",
		EnumType:    "TestEnum",
		DocLines:    []string{"/// VALUE is also documented."},
	}
	if diff := cmp.Diff(wantEnumValueCodec, v2.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantEnumValueCodec = &enumValueAnnotation{
		Name:        "TEST_ENUM_V3",
		VariantName: "V3",
		EnumType:    "TestEnum",
	}
	if diff := cmp.Diff(wantEnumValueCodec, v3.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantEnumValueCodec = &enumValueAnnotation{
		Name:        "TEST_ENUM_2025",
		VariantName: "TestEnum2025",
		EnumType:    "TestEnum",
	}
	if diff := cmp.Diff(wantEnumValueCodec, v4.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDuplicateEnumValueAnnotations(t *testing.T) {
	// Verify we can handle values that are not in SCREAMING_SNAKE_CASE style.
	v0 := &api.EnumValue{
		Name:   "full",
		ID:     ".test.v1.TestEnum.full",
		Number: 1,
	}
	v1 := &api.EnumValue{
		Name:   "FULL",
		ID:     ".test.v1.TestEnum.FULL",
		Number: 1,
	}
	v2 := &api.EnumValue{
		Name:   "partial",
		ID:     ".test.v1.TestEnum.partial",
		Number: 2,
	}
	// This does not happen in practice, but we want to verify the code can
	// handle it if it ever does.
	v3 := &api.EnumValue{
		Name:   "PARTIAL",
		ID:     ".test.v1.TestEnum.PARTIAL",
		Number: 3,
	}
	enum := &api.Enum{
		Name:    "TestEnum",
		ID:      ".test.v1.TestEnum",
		Package: "test.v1",
		Values:  []*api.EnumValue{v0, v1, v2, v3},
	}

	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	annotateModel(model, codec)

	want := &enumAnnotation{
		Name:           "TestEnum",
		ModuleName:     "test_enum",
		QualifiedName:  "crate::model::TestEnum",
		RelativeName:   "TestEnum",
		UniqueNames:    []*api.EnumValue{v0, v2},
		NameInExamples: "google_cloud_test_v1::model::TestEnum",
	}

	if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(api.EnumValue{}, "Codec", "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
