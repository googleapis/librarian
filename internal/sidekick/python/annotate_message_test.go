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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMessage(t *testing.T) {
	for _, test := range []struct {
		name string
		msg  *api.Message
		want *messageAnnotations
	}{
		{
			name: "basic message with fields",
			msg: func() *api.Message {
				m := api.NewTestMessage("Secret").
					WithFields(
						api.NewTestField("name").WithType(api.TypezString),
						api.NewTestField("labels").WithMap(),
					)
				m.Documentation = "A secret object."
				return m
			}(),
			want: &messageAnnotations{
				Name:         "Secret",
				DocLines:     []string{"A secret object."},
				FirstDocLine: "A secret object.",
				HasDocLines:  true,
				HasFields:    true,
				Fields: []*fieldAnnotations{
					{Name: "name"},
					{Name: "labels", IsMap: true},
				},
			},
		},
		{
			name: "message with oneofs, nested messages, and nested enums",
			msg: func() *api.Message {
				nestedMsg := api.NewTestMessage("Rotation").WithFields(
					api.NewTestField("next_rotation_time").WithType(api.TypezString),
				)
				nestedEnum := api.NewTestEnum("State").WithValues(
					api.NewTestEnumValue("ACTIVE", 1),
				)
				oneof := api.NewTestOneOf("payload").WithFields(
					api.NewTestField("data").WithType(api.TypezBytes),
				)
				m := api.NewTestMessage("Secret").
					WithFields(
						api.NewTestField("name").WithType(api.TypezString),
					).
					WithOneOfs(oneof).
					WithEnums(nestedEnum)
				m.Messages = []*api.Message{nestedMsg}
				return m
			}(),
			want: &messageAnnotations{
				Name:              "Secret",
				HasFields:         true,
				HasOneOfs:         true,
				HasOneOfLink:      true,
				HasNestedMessages: true,
				HasNestedEnums:    true,
				NestedMessages: []*messageAnnotations{
					{
						Name:      "Rotation",
						HasFields: true,
						Fields: []*fieldAnnotations{
							{Name: "next_rotation_time"},
						},
					},
				},
				NestedEnums: []*enumAnnotations{
					{Name: "State", HasValues: true},
				},
				Fields: []*fieldAnnotations{
					{Name: "name"},
					{Name: "data"},
				},
				OneOfs: []*oneofAnnotations{
					{Name: "payload"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{test.msg}, nil, nil)
			c := newTestCodec(t, model, nil)
			if err := c.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann, ok := test.msg.Codec.(*messageAnnotations)
			if !ok {
				t.Fatalf("got %T, want *messageAnnotations", test.msg.Codec)
			}
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(messageAnnotations{}, "Model", "Message"),
				cmpopts.IgnoreFields(fieldAnnotations{}, "Message", "Field", "TypeName", "ProtoType", "TypeHint", "SphinxType", "TypeRef", "IsPrimitive", "IsMessage", "IsEnum", "IsOptional", "IsOneOf", "OneOfName", "DocIsOneOf", "DocOneOfName", "KeyProtoType", "KeyTypeHint", "ValueProtoType", "ValueTypeHint", "ValueTypeRef"),
				cmpopts.IgnoreFields(oneofAnnotations{}, "Message", "OneOf", "Fields"),
				cmpopts.IgnoreFields(enumAnnotations{}, "Model", "Enum", "Values", "DocLines", "FirstDocLine", "RemainingDocLines"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			for _, nested := range test.msg.Messages {
				if _, ok := nested.Codec.(*messageAnnotations); !ok {
					t.Errorf("nested message %q Codec = %T, want *messageAnnotations", nested.Name, nested.Codec)
				}
			}
			for _, enum := range test.msg.Enums {
				if _, ok := enum.Codec.(*enumAnnotations); !ok {
					t.Errorf("nested enum %q Codec = %T, want *enumAnnotations", enum.Name, enum.Codec)
				}
			}
		})
	}
}

func TestAnnotateMessage_ComputeOperationDoneProperty(t *testing.T) {
	opMsg := api.NewTestMessage("Operation").WithFields(
		api.NewTestField("status").WithType(api.TypezString),
		api.NewTestField("id").WithType(api.TypezUint64),
	)
	opMsg.Package = "google.cloud.compute.v1"
	model := api.NewTestAPI([]*api.Message{opMsg}, nil, nil)
	model.PackageName = "google-cloud-compute"
	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}
	ann, ok := opMsg.Codec.(*messageAnnotations)
	if !ok {
		t.Fatalf("got %T, want *messageAnnotations", opMsg.Codec)
	}
	if !ann.HasExtendedOperationDoneProperty {
		t.Errorf("HasExtendedOperationDoneProperty = false, want true")
	}
	if ann.DoneStatusFieldName != "status" {
		t.Errorf("DoneStatusFieldName = %q, want %q", ann.DoneStatusFieldName, "status")
	}
}
