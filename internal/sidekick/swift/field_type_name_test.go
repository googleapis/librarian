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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestScalarFieldTypeName(t *testing.T) {
	for _, test := range []struct {
		name    string
		typez   api.Typez
		want    string
		wantErr bool
	}{
		{"double", api.DOUBLE_TYPE, "Double", false},
		{"float", api.FLOAT_TYPE, "Float", false},
		{"int64", api.INT64_TYPE, "Int64", false},
		{"uint64", api.UINT64_TYPE, "UInt64", false},
		{"int32", api.INT32_TYPE, "Int32", false},
		{"fixed64", api.FIXED64_TYPE, "UInt64", false},
		{"fixed32", api.FIXED32_TYPE, "UInt32", false},
		{"bool", api.BOOL_TYPE, "Bool", false},
		{"string", api.STRING_TYPE, "String", false},
		{"bytes", api.BYTES_TYPE, "Data", false},
		{"uint32", api.UINT32_TYPE, "UInt32", false},
		{"sfixed32", api.SFIXED32_TYPE, "Int32", false},
		{"sfixed64", api.SFIXED64_TYPE, "Int64", false},
		{"sint32", api.SINT32_TYPE, "Int32", false},
		{"sint64", api.SINT64_TYPE, "Int64", false},
		{"default undefined", api.UNDEFINED_TYPE, "", true},
		{"default message", api.MESSAGE_TYPE, "", true},
		{"default enum", api.ENUM_TYPE, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := &api.Field{Typez: test.typez, ID: ".test.field"}
			got, err := scalarFieldTypeName(field)
			if test.wantErr {
				if err == nil {
					t.Fatalf("wanted error, got=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldTypeName_BaseMessage(t *testing.T) {
	outer := &api.Message{
		Name:    "OuterMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.OuterMessage",
	}
	nested := &api.Message{
		Name:    "NestedMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.OuterMessage.NestedMessage",
		Parent:  outer,
	}
	outer.Messages = append(outer.Messages, nested)
	simple := &api.Message{
		Name:    "SimpleMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.SimpleMessage",
	}

	model := api.NewTestAPI([]*api.Message{outer, simple}, nil, nil)
	model.State.MessageByID[nested.ID] = nested
	c := newTestCodec(t, model, map[string]string{})

	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name: "simple message",
			field: &api.Field{
				Typez:   api.MESSAGE_TYPE,
				TypezID: ".google.cloud.test.v1.SimpleMessage",
				ID:      ".test.field1",
			},
			want: "SimpleMessage",
		},
		{
			name: "nested message",
			field: &api.Field{
				Typez:   api.MESSAGE_TYPE,
				TypezID: ".google.cloud.test.v1.OuterMessage.NestedMessage",
				ID:      ".test.field2",
			},
			want: "OuterMessage.NestedMessage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.baseFieldTypeName(test.field)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldTypeName_BaseEnum(t *testing.T) {
	outer := &api.Message{
		Name:    "OuterMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.OuterMessage",
	}
	nested := &api.Enum{
		Name:    "NestedEnum",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.OuterMessage.NestedEnum",
		Parent:  outer,
	}
	outer.Enums = append(outer.Enums, nested)
	simple := &api.Enum{
		Name:    "SimpleEnum",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.SimpleEnum",
	}

	model := api.NewTestAPI([]*api.Message{outer}, []*api.Enum{simple}, nil)
	model.State.EnumByID[nested.ID] = nested
	c := newTestCodec(t, model, map[string]string{})

	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name: "simple enum",
			field: &api.Field{
				Typez:   api.ENUM_TYPE,
				TypezID: ".google.cloud.test.v1.SimpleEnum",
				ID:      ".test.field1",
			},
			want: "SimpleEnum",
		},
		{
			name: "nested enum",
			field: &api.Field{
				Typez:   api.ENUM_TYPE,
				TypezID: ".google.cloud.test.v1.OuterMessage.NestedEnum",
				ID:      ".test.field2",
			},
			want: "OuterMessage.NestedEnum",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.baseFieldTypeName(test.field)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldTypeName_Optional(t *testing.T) {
	secret := &api.Message{
		Name:    "Secret",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.Secret",
	}

	model := api.NewTestAPI([]*api.Message{secret}, nil, nil)
	c := newTestCodec(t, model, nil)

	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name: "optional message Secret",
			field: &api.Field{
				Typez:       api.MESSAGE_TYPE,
				TypezID:     ".google.cloud.test.v1.Secret",
				ID:          ".test.field1",
				Optional:    true,
				MessageType: secret,
			},
			want: "Secret?",
		},
		{
			name: "optional string",
			field: &api.Field{
				Typez:    api.STRING_TYPE,
				ID:       ".test.field5",
				Optional: true,
			},
			want: "String?",
		},
		{
			name: "optional bytes",
			field: &api.Field{
				Typez:    api.BYTES_TYPE,
				ID:       ".test.field7",
				Optional: true,
			},
			want: "Data?",
		},
		{
			name: "optional int32",
			field: &api.Field{
				Typez:    api.INT32_TYPE,
				ID:       ".test.field9",
				Optional: true,
			},
			want: "Int32?",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.fieldTypeName(test.field)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldTypeName_Repeated(t *testing.T) {
	secret := &api.Message{
		Name:    "Secret",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.Secret",
	}

	model := api.NewTestAPI([]*api.Message{secret}, nil, nil)
	c := newTestCodec(t, model, nil)

	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name: "repeated message Secret",
			field: &api.Field{
				Typez:       api.MESSAGE_TYPE,
				TypezID:     ".google.cloud.test.v1.Secret",
				ID:          ".test.field2",
				Repeated:    true,
				MessageType: secret,
			},
			want: "[Secret]",
		},
		{
			name: "repeated string",
			field: &api.Field{
				Typez:    api.STRING_TYPE,
				ID:       ".test.field6",
				Repeated: true,
			},
			want: "[String]",
		},
		{
			name: "repeated bytes",
			field: &api.Field{
				Typez:    api.BYTES_TYPE,
				ID:       ".test.field8",
				Repeated: true,
			},
			want: "[Data]",
		},
		{
			name: "repeated int32",
			field: &api.Field{
				Typez:    api.INT32_TYPE,
				ID:       ".test.field10",
				Repeated: true,
			},
			want: "[Int32]",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.fieldTypeName(test.field)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldTypeName_Map(t *testing.T) {
	mapEntry := &api.Message{
		Name:    "SingularMapEntry",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.WithMap.SingularMapEntry",
		IsMap:   true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.STRING_TYPE, ID: ".google.cloud.test.v1.WithMap.SingularMapEntry.key"},
			{Name: "value", Typez: api.INT32_TYPE, ID: ".google.cloud.test.v1.WithMap.SingularMapEntry.value"},
		},
	}

	model := api.NewTestAPI(nil, nil, nil)
	model.PackageName = mapEntry.Package
	model.State.MessageByID[mapEntry.ID] = mapEntry
	c := newTestCodec(t, model, map[string]string{})

	field := &api.Field{
		Typez:   api.MESSAGE_TYPE,
		TypezID: ".google.cloud.test.v1.WithMap.SingularMapEntry",
		ID:      ".test.field1",
	}

	got, err := c.baseFieldTypeName(field)
	if err != nil {
		t.Fatal(err)
	}
	want := "[String: Int32]"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFieldTypeName_ExternalMessage(t *testing.T) {
	externalMessage := &api.Message{
		Name:    "ExternalMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.ExternalMessage",
	}

	model := api.NewTestAPI(nil, nil, nil)
	model.PackageName = "google.cloud.test.v1"
	model.State.MessageByID[externalMessage.ID] = externalMessage
	c := newTestCodec(t, model, nil)
	c.ApiPackages["google.cloud.external.v1"] = &Dependency{
		SwiftDependency: config.SwiftDependency{
			ApiPackage: "google.cloud.external.v1",
			Name:       "ExternalPackage",
		},
	}

	got, err := c.messageTypeName(externalMessage)
	if err != nil {
		t.Fatal(err)
	}
	want := "ExternalPackage.ExternalMessage"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	dep := c.ApiPackages["google.cloud.external.v1"]
	if !dep.Required {
		t.Errorf("expected dependency to be marked as required")
	}
}

func TestFieldTypeName_ExternalEnum(t *testing.T) {
	externalEnum := &api.Enum{
		Name:    "ExternalEnum",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.ExternalEnum",
	}

	model := api.NewTestAPI(nil, nil, nil)
	model.PackageName = "google.cloud.test.v1"
	model.State.EnumByID[externalEnum.ID] = externalEnum
	c := newTestCodec(t, model, nil)
	c.ApiPackages["google.cloud.external.v1"] = &Dependency{
		SwiftDependency: config.SwiftDependency{
			ApiPackage: "google.cloud.external.v1",
			Name:       "ExternalPackage",
		},
	}

	got, err := c.enumTypeName(externalEnum)
	if err != nil {
		t.Fatal(err)
	}
	want := "ExternalPackage.ExternalEnum"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	dep := c.ApiPackages["google.cloud.external.v1"]
	if !dep.Required {
		t.Errorf("expected dependency to be marked as required")
	}
}

func TestFieldTypeName_ExternalNestedMessage(t *testing.T) {
	externalOuter := &api.Message{
		Name:    "OuterMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.OuterMessage",
	}
	externalNested := &api.Message{
		Name:    "NestedMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.OuterMessage.NestedMessage",
		Parent:  externalOuter,
	}
	externalOuter.Messages = append(externalOuter.Messages, externalNested)

	model := api.NewTestAPI(nil, nil, nil)
	model.PackageName = "google.cloud.test.v1"
	model.State.MessageByID[externalNested.ID] = externalNested
	model.State.MessageByID[externalOuter.ID] = externalOuter
	c := newTestCodec(t, model, nil)
	c.ApiPackages["google.cloud.external.v1"] = &Dependency{
		SwiftDependency: config.SwiftDependency{
			ApiPackage: "google.cloud.external.v1",
			Name:       "ExternalPackage",
		},
	}

	got, err := c.messageTypeName(externalNested)
	if err != nil {
		t.Fatal(err)
	}
	want := "ExternalPackage.OuterMessage.NestedMessage"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
