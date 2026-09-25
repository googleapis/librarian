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
	v0 := api.NewTestEnumValue("week5", 2).WithDocumentation("week5 is also documented.")
	v1 := api.NewTestEnumValue("MULTI_WORD_VALUE", 1).WithDocumentation("MULTI_WORD_VALUE is also documented.")
	v2 := api.NewTestEnumValue("VALUE", 0).WithDocumentation("VALUE is also documented.")
	v3 := api.NewTestEnumValue("TEST_ENUM_V3", 3)
	v4 := api.NewTestEnumValue("TEST_ENUM_2025", 4)
	enum := api.NewTestEnum("TestEnum").
		WithPackage("test.v1").
		WithDocumentation("The enum is documented.").
		WithValues(v0, v1, v2, v3, v4)

	model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	annotateModel(model, codec)

	wantEnumCodec := &enumAnnotation{
		Name:              "TestEnum",
		ModuleName:        "test_enum",
		QualifiedName:     "crate::model::TestEnum",
		RelativeName:      "TestEnum",
		ProstRelativeName: "TestEnum",
		DocLines:          []string{"/// The enum is documented."},
		UniqueNames:       []*api.EnumValue{v0, v1, v2, v3, v4},
		NameInExamples:    "google_cloud_test_v1::model::TestEnum",
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
	v0 := api.NewTestEnumValue("full", 1)
	v1 := api.NewTestEnumValue("FULL", 1)
	v2 := api.NewTestEnumValue("partial", 2)
	// This does not happen in practice, but we want to verify the code can
	// handle it if it ever does.
	v3 := api.NewTestEnumValue("PARTIAL", 3)
	enum := api.NewTestEnum("TestEnum").
		WithPackage("test.v1").
		WithValues(v0, v1, v2, v3)

	model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	annotateModel(model, codec)

	want := &enumAnnotation{
		Name:              "TestEnum",
		ModuleName:        "test_enum",
		QualifiedName:     "crate::model::TestEnum",
		RelativeName:      "TestEnum",
		ProstRelativeName: "TestEnum",
		UniqueNames:       []*api.EnumValue{v0, v2},
		NameInExamples:    "google_cloud_test_v1::model::TestEnum",
	}

	if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(api.EnumValue{}, "Codec", "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestNestedEnumAnnotations(t *testing.T) {
	v0 := api.NewTestEnumValue("IP_MODE_UNSPECIFIED", 0)
	v1 := api.NewTestEnumValue("DYNAMIC_IP", 1)
	enum := api.NewTestEnum("IPMode").
		WithValues(v0, v1)
	parent := api.NewTestMessage("VertexAISearch").
		WithPackage("test.v1").
		WithEnums(enum)

	model := api.NewTestAPI([]*api.Message{parent}, nil, nil)
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "test.v1", map[string]string{})
	annotateModel(model, codec)

	want := &enumAnnotation{
		Name:              "IPMode",
		ModuleName:        "ip_mode",
		QualifiedName:     "crate::model::vertex_ai_search::IPMode",
		RelativeName:      "vertex_ai_search::IPMode",
		ProstRelativeName: "vertex_ai_search::IpMode",
		UniqueNames:       []*api.EnumValue{v0, v1},
		NameInExamples:    "google_cloud_test_v1::model::vertex_ai_search::IPMode",
	}

	if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(api.EnumValue{}, "Codec", "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDiscoveryEnumAnnotations(t *testing.T) {
	v0 := api.NewTestEnumValue("CREATING", 0)
	v1 := api.NewTestEnumValue("READY", 1)
	enum := api.NewTestEnum("Status").
		WithPackage("test.v1").
		WithValues(v0, v1)

	model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecDiscovery, "", map[string]string{})
	annotateModel(model, codec)

	want := &enumAnnotation{
		Name:              "Status",
		ModuleName:        "status",
		QualifiedName:     "crate::model::Status",
		RelativeName:      "Status",
		ProstRelativeName: "Status",
		UniqueNames:       []*api.EnumValue{v0, v1},
		SerializeAsString: true,
		NameInExamples:    "google_cloud_test_v1::model::Status",
	}

	if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(api.EnumValue{}, "Codec", "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
