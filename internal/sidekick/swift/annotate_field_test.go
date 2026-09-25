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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateField(t *testing.T) {
	for _, test := range []struct {
		name     string
		optional bool
		repeated bool
		want     *fieldAnnotations
	}{
		{
			name:     "regular",
			optional: false,
			repeated: false,
			want: &fieldAnnotations{
				FieldType:            "Swift.String",
				BaseFieldType:        "Swift.String",
				ProtoFieldName:       "secretPayload",
				ProtoFieldNamePascal: "SecretPayload",
				PrimitiveFieldType:   "Swift.String",
			},
		},
		{
			name:     "optional",
			optional: true,
			repeated: false,
			want: &fieldAnnotations{
				FieldType:            "Swift.String?",
				BaseFieldType:        "Swift.String",
				Decoding:             DecodingOptional,
				ProtoFieldName:       "secretPayload",
				ProtoFieldNamePascal: "SecretPayload",
				PrimitiveFieldType:   "Swift.String",
			},
		},
		{
			name:     "repeated",
			optional: false,
			repeated: true,
			want: &fieldAnnotations{
				FieldType:            "[Swift.String]",
				BaseFieldType:        "Swift.String",
				ProtoFieldName:       "secretPayload",
				ProtoFieldNamePascal: "SecretPayload",
				PrimitiveFieldType:   "Swift.String",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := api.NewTestField("secret_payload").
				WithType(api.TypezString).
				WithDocumentation("The secret version payload.")
			if test.optional {
				field.WithOptional()
			}
			if test.repeated {
				field.WithRepeated()
			}
			msg := api.NewTestMessage("Secret").
				WithID(".test.SecretVersion").
				WithFields(field)
			model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, field.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "Name", "DocLines", "Model")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateField_Discovery(t *testing.T) {
	mapMessage := api.NewTestMapMessage("map<string, bytes>", api.TypezString, api.TypezBytes).
		WithID("$map<string, bytes>")

	for _, test := range []struct {
		name  string
		input *api.Field
		want  *fieldAnnotations
	}{
		{
			name:  "regular",
			input: api.NewTestField("name").WithType(api.TypezBytes),
			want: &fieldAnnotations{
				FieldType:            "Foundation.Data",
				BaseFieldType:        "Foundation.Data",
				UrlSafeValue:         true,
				ProtoFieldName:       "name",
				ProtoFieldNamePascal: "Name",
				PrimitiveFieldType:   "Foundation.Data",
			},
		},
		{
			name:  "regular string",
			input: api.NewTestField("name").WithType(api.TypezString),
			want: &fieldAnnotations{
				FieldType:            "Swift.String",
				BaseFieldType:        "Swift.String",
				ProtoFieldName:       "name",
				ProtoFieldNamePascal: "Name",
				PrimitiveFieldType:   "Swift.String",
			},
		},
		{
			name:  "optional",
			input: api.NewTestField("name").WithType(api.TypezBytes).WithOptional(),
			want: &fieldAnnotations{
				FieldType:            "Foundation.Data?",
				BaseFieldType:        "Foundation.Data",
				UrlSafeValue:         true,
				Decoding:             DecodingOptional,
				ProtoFieldName:       "name",
				ProtoFieldNamePascal: "Name",
				PrimitiveFieldType:   "Foundation.Data",
			},
		},
		{
			name:  "repeated",
			input: api.NewTestField("name").WithType(api.TypezBytes).WithRepeated(),
			want: &fieldAnnotations{
				FieldType:            "[Foundation.Data]",
				BaseFieldType:        "Foundation.Data",
				UrlSafeValue:         true,
				ProtoFieldName:       "name",
				ProtoFieldNamePascal: "Name",
				PrimitiveFieldType:   "Foundation.Data",
			},
		},
		{
			name:  "map",
			input: api.NewTestField("name").WithMessageType(mapMessage).WithMap(),
			want: &fieldAnnotations{
				FieldType:            "[Swift.String: Foundation.Data]",
				BaseFieldType:        "[Swift.String: Foundation.Data]",
				KeyType:              "Swift.String",
				ValueType:            "Foundation.Data",
				UrlSafeValue:         true,
				ProtoFieldName:       "name",
				ProtoFieldNamePascal: "Name",
				ValueField:           mapMessage.Fields[1],
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := api.NewTestMessage("Message").WithFields(test.input)
			model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
			model.AddMessage(mapMessage)
			codec := newTestCodec(t, model, nil)
			codec.UrlSafeForBytes = true
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, test.input.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "Name", "DocLines", "Model")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateField_TypeNames(t *testing.T) {
	for _, test := range []struct {
		name     string
		typez    api.Typez
		wantType string
	}{
		{"string", api.TypezString, "Swift.String"},
		{"int32", api.TypezInt32, "Swift.Int32"},
		{"bytes", api.TypezBytes, "Foundation.Data"},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := api.NewTestField("test_field").
				WithType(test.typez).
				WithDocumentation("Test documentation.")
			msg := api.NewTestMessage("TestMessage").
				WithFields(field)
			model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			want := &fieldAnnotations{
				Name:                 "testField",
				FieldType:            test.wantType,
				BaseFieldType:        test.wantType,
				DocLines:             []string{"Test documentation."},
				Model:                model.Codec.(*modelAnnotations),
				ProtoFieldName:       "testField",
				ProtoFieldNamePascal: "TestField",
				PrimitiveFieldType:   test.wantType,
			}
			if diff := cmp.Diff(want, field.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateField_PackageName(t *testing.T) {
	referencedMsg := api.NewTestMessage("SomeMessage").
		WithPackage("google.cloud.external.v1")
	field := api.NewTestField("external_message").
		WithMessageType(referencedMsg).
		WithDocumentation("The external message.")
	msg := api.NewTestMessage("Secret").
		WithID(".test.SecretVersion").
		WithFields(field)
	model := api.NewTestAPI([]*api.Message{msg, referencedMsg}, nil, nil).
		WithPackageName("test")
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "GoogleCloudExternalV1",
		},
	})
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}
	got := field.Codec.(*fieldAnnotations)
	want := &fieldAnnotations{
		Name:                 "externalMessage",
		FieldType:            "GoogleCloudExternalV1.SomeMessage",
		BaseFieldType:        "GoogleCloudExternalV1.SomeMessage",
		PackageName:          "google.cloud.external.v1",
		DocLines:             []string{"The external message."},
		Model:                model.Codec.(*modelAnnotations),
		ProtoFieldName:       "externalMessage",
		ProtoFieldNamePascal: "ExternalMessage",
		PrimitiveFieldType:   "GoogleCloudExternalV1.SomeMessage",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateField_Recursive(t *testing.T) {
	for _, test := range []struct {
		name          string
		optional      bool
		repeated      bool
		isOneOf       bool
		oneofProperty string
		want          *fieldAnnotations
	}{
		{
			name:     "singular optional recursive",
			optional: true,
			repeated: false,
			isOneOf:  false,
			want: &fieldAnnotations{
				FieldType:            "GoogleWKT.WKTRecursive<Node>?",
				BaseFieldType:        "GoogleWKT.WKTRecursive<Node>",
				Recursive:            true,
				Decoding:             DecodingOptional,
				ProtoFieldName:       "childNode",
				ProtoFieldNamePascal: "ChildNode",
				PrimitiveFieldType:   "Node",
			},
		},
		{
			name:     "singular non-optional recursive",
			optional: false,
			repeated: false,
			isOneOf:  false,
			want: &fieldAnnotations{
				FieldType:            "GoogleWKT.WKTRecursive<Node>?",
				BaseFieldType:        "GoogleWKT.WKTRecursive<Node>",
				Recursive:            true,
				Decoding:             DecodingOptional,
				ProtoFieldName:       "childNode",
				ProtoFieldNamePascal: "ChildNode",
				PrimitiveFieldType:   "Node",
			},
		},
		{
			name:     "repeated recursive",
			optional: false,
			repeated: true,
			isOneOf:  false,
			want: &fieldAnnotations{
				FieldType:            "[Node]",
				BaseFieldType:        "Node",
				Recursive:            false,
				ProtoFieldName:       "childNode",
				ProtoFieldNamePascal: "ChildNode",
				PrimitiveFieldType:   "Node",
			},
		},
		{
			name:          "oneof recursive",
			optional:      false,
			repeated:      false,
			isOneOf:       true,
			oneofProperty: "alternatives",
			want: &fieldAnnotations{
				FieldType:            "Node",
				BaseFieldType:        "Node",
				Recursive:            false,
				OneOfChecker:         "alternativesCheckAndSet",
				ProtoFieldName:       "childNode",
				ProtoFieldNamePascal: "ChildNode",
				PrimitiveFieldType:   "Node",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := api.NewTestMessage("Node")
			field := api.NewTestField("child_node").
				WithMessageType(msg).
				WithDocumentation("Recursive link.").
				WithRecursive()
			if test.optional {
				field.WithOptional()
			}
			if test.repeated {
				field.WithRepeated()
			}

			if test.isOneOf {
				oneof := api.NewTestOneOf(test.oneofProperty).WithFields(field)
				msg.WithOneOfs(oneof)
			} else {
				msg.WithFields(field)
			}

			model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, field.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "Name", "DocLines", "PackageName", "Model")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
