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
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestMessageAnnotations(t *testing.T) {
	message := &api.Message{
		Name:          "TestMessage",
		Package:       "test.v1",
		ID:            ".test.v1.TestMessage",
		Documentation: "A test message.",
	}
	nested := &api.Message{
		Name:          "NestedMessage",
		Package:       "test.v1",
		ID:            ".test.v1.TestMessage.NestedMessage",
		Documentation: "A nested message.",
		Parent:        message,
	}
	message.Messages = []*api.Message{nested}

	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "test.v1", map[string]string{})
	annotateModel(model, codec)
	want := &messageAnnotation{
		Name:              "TestMessage",
		ModuleName:        "test_message",
		QualifiedName:     "crate::model::TestMessage",
		RelativeName:      "TestMessage",
		NameInExamples:    "google_cloud_test_v1::model::TestMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage",
		DocLines:          []string{"/// A test message."},
		HasNestedTypes:    true,
		BasicFields:       []*api.Field{},
	}
	if diff := cmp.Diff(want, message.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	want = &messageAnnotation{
		Name:              "NestedMessage",
		ModuleName:        "nested_message",
		QualifiedName:     "crate::model::test_message::NestedMessage",
		RelativeName:      "test_message::NestedMessage",
		NameInExamples:    "google_cloud_test_v1::model::test_message::NestedMessage",
		PackageModuleName: "test::v1",
		SourceFQN:         "test.v1.TestMessage.NestedMessage",
		DocLines:          []string{"/// A nested message."},
		HasNestedTypes:    false,
		BasicFields:       []*api.Field{},
	}
	if diff := cmp.Diff(want, nested.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSetterSampleAnnotations(t *testing.T) {
	enum := &api.Enum{
		Name:    "TestEnum",
		ID:      ".test.v1.TestEnum",
		Package: "test.v1",
	}
	message := &api.Message{
		Name:    "TestMessage",
		ID:      ".test.v1.TestMessage",
		Package: "test.v1",
		Fields: []*api.Field{
			{
				Name:    "enum_field",
				ID:      ".test.v1.TestMessage.enum_field",
				Typez:   api.TypezEnum,
				TypezID: ".test.v1.TestEnum",
			},
			{
				Name:    "message_field",
				ID:      ".test.v1.TestMessage.message_field",
				Typez:   api.TypezMessage,
				TypezID: ".test.v1.TestMessage",
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{enum}, []*api.Service{})
	api.CrossReference(model)
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"generate-setter-samples": "true",
	})
	annotateModel(model, codec)

	if message.Codec.(*messageAnnotation).NameInExamples != "google_cloud_test_v1::model::TestMessage" {
		t.Errorf("mismatch in message NameInExamples: got %q", message.Codec.(*messageAnnotation).NameInExamples)
	}
	if enum.Codec.(*enumAnnotation).NameInExamples != "google_cloud_test_v1::model::TestEnum" {
		t.Errorf("mismatch in enum NameInExamples: got %q", enum.Codec.(*enumAnnotation).NameInExamples)
	}

	enumField := message.Fields[0]
	if enumField.Parent != message {
		t.Errorf("mismatch in enum_field.Parent")
	}
	if enumField.EnumType != enum {
		t.Errorf("mismatch in enum_field.EnumType")
	}

	messageField := message.Fields[1]
	if messageField.Parent != message {
		t.Errorf("mismatch in message_field.Parent")
	}
	if messageField.MessageType != message {
		t.Errorf("mismatch in message_field.MessageType")
	}
}

func TestInternalMessageOverrides(t *testing.T) {
	public := &api.Message{
		Name: "Public",
		ID:   ".test.Public",
	}
	private1 := &api.Message{
		Name: "Private1",
		ID:   ".test.Private1",
	}
	private2 := &api.Message{
		Name: "Private2",
		ID:   ".test.Private2",
	}
	model := api.NewTestAPI([]*api.Message{public, private1, private2},
		[]*api.Enum{},
		[]*api.Service{})
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"internal-types": ".test.Private1,.test.Private2",
	})
	annotateModel(model, codec)

	if public.Codec.(*messageAnnotation).Internal {
		t.Errorf("Public method should not be flagged as internal")
	}
	if !private1.Codec.(*messageAnnotation).Internal {
		t.Errorf("Private method should not be flagged as internal")
	}
	if !private2.Codec.(*messageAnnotation).Internal {
		t.Errorf("Private method should not be flagged as internal")
	}
}
