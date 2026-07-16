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
				FieldType:     "Swift.String",
				BaseFieldType: "Swift.String",
			},
		},
		{
			name:     "optional",
			optional: true,
			repeated: false,
			want: &fieldAnnotations{
				FieldType:     "Swift.String?",
				BaseFieldType: "Swift.String",
				Decoding:      DecodingOptional,
			},
		},
		{
			name:     "repeated",
			optional: false,
			repeated: true,
			want: &fieldAnnotations{
				FieldType:     "[Swift.String]",
				BaseFieldType: "Swift.String",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := &api.Field{
				Name:          "secret_payload",
				Documentation: "The secret version payload.",
				ID:            ".test.SecretVersion.secret_payload",
				Typez:         api.TypezString,
				Optional:      test.optional,
				Repeated:      test.repeated,
			}
			msg := &api.Message{
				Name:    "Secret",
				ID:      ".test.SecretVersion",
				Package: "test",
				Fields:  []*api.Field{field},
			}
			field.Parent = msg
			model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, field.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "Name", "DocLines", "Model", "ToGapicStatement", "ToProtoStatement")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateField_Discovery(t *testing.T) {
	mapMessage := &api.Message{
		Name:  "map<string, bytes>",
		ID:    "$map<string, bytes>",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", JSONName: "key", Typez: api.TypezString},
			{Name: "value", JSONName: "value", Typez: api.TypezBytes},
		},
	}

	for _, test := range []struct {
		name  string
		input *api.Field
		want  *fieldAnnotations
	}{
		{
			name: "regular",
			input: &api.Field{
				Name:  "name",
				ID:    ".test.Message.name",
				Typez: api.TypezBytes,
			},
			want: &fieldAnnotations{
				FieldType:     "Foundation.Data",
				BaseFieldType: "Foundation.Data",
				UrlSafeValue:  true,
			},
		},
		{
			name: "regular string",
			input: &api.Field{
				Name:  "name",
				ID:    ".test.Message.name",
				Typez: api.TypezString,
			},
			want: &fieldAnnotations{
				FieldType:     "Swift.String",
				BaseFieldType: "Swift.String",
			},
		},
		{
			name: "optional",
			input: &api.Field{
				Name:     "name",
				ID:       ".test.Message.name",
				Optional: true,
				Typez:    api.TypezBytes,
			},
			want: &fieldAnnotations{
				FieldType:     "Foundation.Data?",
				BaseFieldType: "Foundation.Data",
				UrlSafeValue:  true,
				Decoding:      DecodingOptional,
			},
		},
		{
			name: "repeated",
			input: &api.Field{
				Name:     "name",
				ID:       ".test.Message.name",
				Repeated: true,
				Typez:    api.TypezBytes,
			},
			want: &fieldAnnotations{
				FieldType:     "[Foundation.Data]",
				BaseFieldType: "Foundation.Data",
				UrlSafeValue:  true,
			},
		},
		{
			name: "map",
			input: &api.Field{
				Name:    "name",
				ID:      ".test.Message.name",
				Typez:   api.TypezMessage,
				TypezID: mapMessage.ID,
				Map:     true,
			},
			want: &fieldAnnotations{
				FieldType:     "[Swift.String: Foundation.Data]",
				BaseFieldType: "[Swift.String: Foundation.Data]",
				KeyType:       "Swift.String",
				ValueType:     "Foundation.Data",
				UrlSafeValue:  true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := &api.Message{
				Name:    "Message",
				ID:      ".test.Message",
				Package: "test",
				Fields:  []*api.Field{test.input},
			}
			test.input.Parent = msg
			model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
			model.AddMessage(mapMessage)
			codec := newTestCodec(t, model, map[string]string{})
			codec.UrlSafeForBytes = true
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, test.input.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "Name", "DocLines", "Model", "ToGapicStatement", "ToProtoStatement")); diff != "" {
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
			field := &api.Field{
				Name:          "test_field",
				ID:            ".test.TestMessage.test_field",
				Typez:         test.typez,
				Documentation: "Test documentation.",
			}
			msg := &api.Message{
				Name:    "TestMessage",
				ID:      ".test.TestMessage",
				Package: "test",
				Fields:  []*api.Field{field},
			}
			field.Parent = msg
			model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			want := &fieldAnnotations{
				Name:          "testField",
				FieldType:     test.wantType,
				BaseFieldType: test.wantType,
				DocLines:      []string{"Test documentation."},
				Model:         model.Codec.(*modelAnnotations),
			}
			if diff := cmp.Diff(want, field.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "ToGapicStatement", "ToProtoStatement")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateField_PackageName(t *testing.T) {
	referencedMsg := &api.Message{
		Name:    "SomeMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.SomeMessage",
	}
	field := &api.Field{
		Name:          "external_message",
		Documentation: "The external message.",
		ID:            ".test.SecretVersion.external_message",
		Typez:         api.TypezMessage,
		TypezID:       referencedMsg.ID,
	}
	msg := &api.Message{
		Name:    "Secret",
		ID:      ".test.SecretVersion",
		Package: "test",
		Fields:  []*api.Field{field},
	}
	field.Parent = msg
	model := api.NewTestAPI([]*api.Message{msg, referencedMsg}, nil, nil)
	model.PackageName = "test"
	codec := newTestCodec(t, model, map[string]string{})
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
		Name:          "externalMessage",
		FieldType:     "GoogleCloudExternalV1.SomeMessage",
		BaseFieldType: "GoogleCloudExternalV1.SomeMessage",
		PackageName:   "google.cloud.external.v1",
		DocLines:      []string{"The external message."},
		Model:         model.Codec.(*modelAnnotations),
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(fieldAnnotations{}, "ToGapicStatement", "ToProtoStatement")); diff != "" {
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
				FieldType:     "GoogleCloudWkt.Recursive<Node>?",
				BaseFieldType: "GoogleCloudWkt.Recursive<Node>",
				Recursive:     true,
				Decoding:      DecodingOptional,
			},
		},
		{
			name:     "repeated recursive",
			optional: false,
			repeated: true,
			isOneOf:  false,
			want: &fieldAnnotations{
				FieldType:     "[Node]",
				BaseFieldType: "Node",
				Recursive:     false,
			},
		},
		{
			name:          "oneof recursive",
			optional:      false,
			repeated:      false,
			isOneOf:       true,
			oneofProperty: "alternatives",
			want: &fieldAnnotations{
				FieldType:     "Node",
				BaseFieldType: "Node",
				Recursive:     false,
				OneOfChecker:  "alternativesCheckAndSet",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := &api.Field{
				Name:          "child_node",
				ID:            ".test.Node.child_node",
				Typez:         api.TypezMessage,
				TypezID:       ".test.Node",
				Documentation: "Recursive link.",
				Optional:      test.optional,
				Repeated:      test.repeated,
				IsOneOf:       test.isOneOf,
				Recursive:     true,
			}
			msg := &api.Message{
				Name:    "Node",
				ID:      ".test.Node",
				Package: "test",
				Fields:  []*api.Field{field},
			}
			field.Parent = msg
			field.MessageType = msg

			if test.isOneOf {
				oneof := &api.OneOf{
					Name:   test.oneofProperty,
					Fields: []*api.Field{field},
				}
				field.Group = oneof
				msg.OneOfs = []*api.OneOf{oneof}
			}

			model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, field.Codec, cmpopts.IgnoreFields(fieldAnnotations{}, "Name", "DocLines", "PackageName", "Model", "ToGapicStatement", "ToProtoStatement")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
