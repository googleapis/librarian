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

func TestAnnotateField(t *testing.T) {
	for _, test := range []struct {
		name         string
		optional     bool
		repeated     bool
		wantType     string
		wantBaseType string
	}{
		{
			name:         "regular",
			optional:     false,
			repeated:     false,
			wantType:     "Swift.String",
			wantBaseType: "Swift.String",
		},
		{
			name:         "optional",
			optional:     true,
			repeated:     false,
			wantType:     "Swift.String?",
			wantBaseType: "Swift.String",
		},
		{
			name:         "repeated",
			optional:     false,
			repeated:     true,
			wantType:     "[Swift.String]",
			wantBaseType: "Swift.String",
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
			want := &fieldAnnotations{
				Name:            "secretPayload",
				DocLines:        []string{"The secret version payload."},
				FieldType:       test.wantType,
				BaseFieldType:   test.wantBaseType,
				InitializerType: test.wantType,
			}

			if diff := cmp.Diff(want, field.Codec); diff != "" {
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
				Name:            "testField",
				FieldType:       test.wantType,
				BaseFieldType:   test.wantType,
				DocLines:        []string{"Test documentation."},
				InitializerType: test.wantType,
			}
			if diff := cmp.Diff(want, field.Codec); diff != "" {
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
		Name:            "externalMessage",
		FieldType:       "GoogleCloudExternalV1.SomeMessage",
		BaseFieldType:   "GoogleCloudExternalV1.SomeMessage",
		PackageName:     "google.cloud.external.v1",
		DocLines:        []string{"The external message."},
		InitializerType: "GoogleCloudExternalV1.SomeMessage",
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
		wantType      string
		wantBaseType  string
		wantRecursive bool
		wantInitType  string
		oneofProperty string
	}{
		{
			name:          "singular optional recursive",
			optional:      true,
			repeated:      false,
			isOneOf:       false,
			wantType:      "GoogleCloudWkt.Recursive<Node>?",
			wantBaseType:  "GoogleCloudWkt.Recursive<Node>",
			wantRecursive: true,
			wantInitType:  "Node?",
		},
		{
			name:          "repeated recursive",
			optional:      false,
			repeated:      true,
			isOneOf:       false,
			wantType:      "[Node]",
			wantBaseType:  "Node",
			wantRecursive: false,
			wantInitType:  "[Node]",
		},
		{
			name:          "oneof recursive",
			optional:      false,
			repeated:      false,
			isOneOf:       true,
			wantType:      "Node",
			wantBaseType:  "Node",
			wantRecursive: false,
			wantInitType:  "Node",
			oneofProperty: "alternatives",
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

			want := &fieldAnnotations{
				Name:            "childNode",
				DocLines:        []string{"Recursive link."},
				FieldType:       test.wantType,
				BaseFieldType:   test.wantBaseType,
				PackageName:     "test",
				Recursive:       test.wantRecursive,
				InitializerType: test.wantInitType,
			}
			if test.isOneOf {
				want.OneOfChecker = test.oneofProperty + "CheckAndSet"
			}

			if diff := cmp.Diff(want, field.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
