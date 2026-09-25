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
		{"double", api.TypezDouble, "Swift.Double", false},
		{"float", api.TypezFloat, "Swift.Float", false},
		{"int64", api.TypezInt64, "Swift.Int64", false},
		{"uint64", api.TypezUint64, "Swift.UInt64", false},
		{"int32", api.TypezInt32, "Swift.Int32", false},
		{"fixed64", api.TypezFixed64, "Swift.UInt64", false},
		{"fixed32", api.TypezFixed32, "Swift.UInt32", false},
		{"bool", api.TypezBool, "Swift.Bool", false},
		{"string", api.TypezString, "Swift.String", false},
		{"bytes", api.TypezBytes, "Foundation.Data", false},
		{"uint32", api.TypezUint32, "Swift.UInt32", false},
		{"sfixed32", api.TypezSfixed32, "Swift.Int32", false},
		{"sfixed64", api.TypezSfixed64, "Swift.Int64", false},
		{"sint32", api.TypezSint32, "Swift.Int32", false},
		{"sint64", api.TypezSint64, "Swift.Int64", false},
		{"default undefined", api.TypezUndefined, "", true},
		{"default message", api.TypezMessage, "", true},
		{"default enum", api.TypezEnum, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := api.NewTestField("field").WithType(test.typez)
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
	outer := api.NewTestMessage("OuterMessage").WithPackage("google.cloud.test.v1")
	nested := api.NewTestMessage("NestedMessage").
		WithPackage("google.cloud.test.v1").
		WithID(".google.cloud.test.v1.OuterMessage.NestedMessage")
	simple := api.NewTestMessage("SimpleMessage").WithPackage("google.cloud.test.v1")

	model := api.NewTestAPI([]*api.Message{outer, simple, nested}, nil, nil)
	c := newTestCodec(t, model, nil)

	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name: "simple message",
			field: api.NewTestField("field1").
				WithMessageType(simple),
			want: "SimpleMessage",
		},
		{
			name: "nested message",
			field: api.NewTestField("field2").
				WithMessageType(nested),
			want: "OuterMessage.NestedMessage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.fieldTypeBase(test.field)
			if err != nil {
				t.Fatal(err)
			}
			want := &fieldTypeNames{Base: test.want}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldTypeName_BaseEnum(t *testing.T) {
	nested := api.NewTestEnum("NestedEnum")
	outer := api.NewTestMessage("OuterMessage").
		WithPackage("google.cloud.test.v1").
		WithEnums(nested)
	simple := api.NewTestEnum("SimpleEnum").WithPackage("google.cloud.test.v1")

	model := api.NewTestAPI([]*api.Message{outer}, []*api.Enum{simple}, nil)
	c := newTestCodec(t, model, nil)

	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name: "simple enum",
			field: api.NewTestField("field1").
				WithEnumType(simple),
			want: "SimpleEnum",
		},
		{
			name: "nested enum",
			field: api.NewTestField("field2").
				WithEnumType(nested),
			want: "OuterMessage.NestedEnum",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.fieldTypeBase(test.field)
			if err != nil {
				t.Fatal(err)
			}
			want := &fieldTypeNames{Base: test.want}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldTypeName_Optional(t *testing.T) {
	secret := api.NewTestMessage("Secret").WithPackage("google.cloud.test.v1")

	model := api.NewTestAPI([]*api.Message{secret}, nil, nil)
	c := newTestCodec(t, model, nil)

	for _, test := range []struct {
		name  string
		field *api.Field
		want  *fieldTypeNames
	}{
		{
			name: "optional message Secret",
			field: api.NewTestField("field1").
				WithMessageType(secret).
				WithOptional(),
			want: &fieldTypeNames{
				Base: "Secret",
				Full: "Secret?",
			},
		},
		{
			name: "optional string",
			field: api.NewTestField("field5").
				WithType(api.TypezString).
				WithOptional(),
			want: &fieldTypeNames{
				Base: "Swift.String",
				Full: "Swift.String?",
			},
		},
		{
			name: "optional bytes",
			field: api.NewTestField("field7").
				WithType(api.TypezBytes).
				WithOptional(),
			want: &fieldTypeNames{
				Base: "Foundation.Data",
				Full: "Foundation.Data?",
			},
		},
		{
			name: "optional int32",
			field: api.NewTestField("field9").
				WithType(api.TypezInt32).
				WithOptional(),
			want: &fieldTypeNames{
				Base: "Swift.Int32",
				Full: "Swift.Int32?",
			},
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
	secret := api.NewTestMessage("Secret").WithPackage("google.cloud.test.v1")

	model := api.NewTestAPI([]*api.Message{secret}, nil, nil)
	c := newTestCodec(t, model, nil)

	for _, test := range []struct {
		name  string
		field *api.Field
		want  *fieldTypeNames
	}{
		{
			name: "repeated message Secret",
			field: api.NewTestField("field2").
				WithMessageType(secret).
				WithRepeated(),
			want: &fieldTypeNames{
				Base: "Secret",
				Full: "[Secret]",
			},
		},
		{
			name: "repeated string",
			field: api.NewTestField("field6").
				WithType(api.TypezString).
				WithRepeated(),
			want: &fieldTypeNames{
				Base: "Swift.String",
				Full: "[Swift.String]",
			},
		},
		{
			name: "repeated bytes",
			field: api.NewTestField("field8").
				WithType(api.TypezBytes).
				WithRepeated(),
			want: &fieldTypeNames{
				Base: "Foundation.Data",
				Full: "[Foundation.Data]",
			},
		},
		{
			name: "repeated int32",
			field: api.NewTestField("field10").
				WithType(api.TypezInt32).
				WithRepeated(),
			want: &fieldTypeNames{
				Base: "Swift.Int32",
				Full: "[Swift.Int32]",
			},
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
	mapEntry := api.NewTestMapMessage("SingularMapEntry", api.TypezString, api.TypezInt32).
		WithPackage("google.cloud.test.v1").
		WithID(".google.cloud.test.v1.WithMap.SingularMapEntry")

	model := api.NewTestAPI([]*api.Message{mapEntry}, nil, nil)
	c := newTestCodec(t, model, nil)

	field := api.NewTestField("field1").
		WithMessageType(mapEntry)

	got, err := c.fieldTypeName(field)
	if err != nil {
		t.Fatal(err)
	}
	want := &fieldTypeNames{
		Full:  "[Swift.String: Swift.Int32]",
		Base:  "[Swift.String: Swift.Int32]",
		Key:   "Swift.String",
		Value: "Swift.Int32",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFieldTypeName_ExternalMessage(t *testing.T) {
	externalMessage := api.NewTestMessage("ExternalMessage").
		WithPackage("google.cloud.external.v1")

	model := api.NewTestAPI([]*api.Message{externalMessage}, nil, nil).
		WithPackageName("google.cloud.test.v1")
	c := newTestCodec(t, model, nil)
	ann := &modelAnnotations{DependsOn: map[string]*Dependency{}}
	c.Model.Codec = ann
	c.withExtraDependencies(t, []config.SwiftDependency{
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "ExternalPackage",
		},
		{
			ApiPackage: "google.cloud.unused.v1",
			Name:       "UnusedPackage",
		},
	})

	got, err := c.messageTypeName(externalMessage)
	if err != nil {
		t.Fatal(err)
	}
	want := "ExternalPackage.ExternalMessage"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFieldTypeName_ExternalEnum(t *testing.T) {
	externalEnum := api.NewTestEnum("ExternalEnum").
		WithPackage("google.cloud.external.v1")

	model := api.NewTestAPI(nil, []*api.Enum{externalEnum}, nil).
		WithPackageName("google.cloud.test.v1")
	c := newTestCodec(t, model, nil)
	ann := &modelAnnotations{DependsOn: map[string]*Dependency{}}
	c.Model.Codec = ann
	c.withExtraDependencies(t, []config.SwiftDependency{
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "ExternalPackage",
		},
		{
			ApiPackage: "google.cloud.unused.v1",
			Name:       "UnusedPackage",
		},
	})

	got, err := c.enumTypeName(externalEnum)
	if err != nil {
		t.Fatal(err)
	}
	want := "ExternalPackage.ExternalEnum"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFieldTypeName_ExternalNestedMessage(t *testing.T) {
	externalOuter := api.NewTestMessage("OuterMessage").
		WithPackage("google.cloud.external.v1")
	externalNested := api.NewTestMessage("NestedMessage").
		WithPackage("google.cloud.external.v1").
		WithID(".google.cloud.external.v1.OuterMessage.NestedMessage")

	model := api.NewTestAPI([]*api.Message{externalOuter, externalNested}, nil, nil).
		WithPackageName("google.cloud.test.v1")
	c := newTestCodec(t, model, nil)
	c.withExtraDependencies(t, []config.SwiftDependency{
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "ExternalPackage",
		},
	})

	got, err := c.messageTypeName(externalNested)
	if err != nil {
		t.Fatal(err)
	}
	want := "ExternalPackage.OuterMessage.NestedMessage"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFullyQualifiedMessageTypeName(t *testing.T) {
	msg := api.NewTestMessage("TestMessage").
		WithPackage("google.cloud.test.v1")
	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)

	t.Run("standalone library with LibraryName", func(t *testing.T) {
		c := newTestCodec(t, model, nil)
		c.LibraryName = "TestLibrary"
		c.Module = false

		got, err := c.fullyQualifiedMessageTypeName(msg)
		if err != nil {
			t.Fatal(err)
		}
		want := "TestLibrary.TestMessage"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("module with empty LibraryName", func(t *testing.T) {
		c := newTestCodec(t, model, nil)
		c.LibraryName = ""
		c.Module = true

		got, err := c.fullyQualifiedMessageTypeName(msg)
		if err != nil {
			t.Fatal(err)
		}
		want := "TestMessage"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
