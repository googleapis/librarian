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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestFindFieldInMessage(t *testing.T) {
	for _, test := range []struct {
		name      string
		msg       *api.Message
		fieldName string
		wantFound bool
	}{
		{
			name:      "nil message",
			msg:       nil,
			fieldName: "foo",
			wantFound: false,
		},
		{
			name:      "field not found",
			msg:       api.NewTestMessage("Msg").WithFields(api.NewTestField("bar")),
			fieldName: "foo",
			wantFound: false,
		},
		{
			name:      "field found",
			msg:       api.NewTestMessage("Msg").WithFields(api.NewTestField("foo")),
			fieldName: "foo",
			wantFound: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := findFieldInMessage(test.msg, test.fieldName)
			if (got != nil) != test.wantFound {
				t.Errorf("findFieldInMessage() got %v, want found=%v", got, test.wantFound)
			}
		})
	}
}

func TestResolveFieldTypeHintAndSphinx(t *testing.T) {
	targetMsg := api.NewTestMessage("Target").
		WithPackage("google.cloud.example.v1").
		WithSourceLocation("google/cloud/example/v1/target.proto", 1)

	extMsg := api.NewTestMessage("Duration").
		WithPackage("google.protobuf").
		WithSourceLocation("google/protobuf/duration.proto", 1)

	targetEnum := api.NewTestEnum("Status").
		WithPackage("google.cloud.example.v1").
		WithSourceLocation("google/cloud/example/v1/target.proto", 10)

	model := api.NewTestAPI([]*api.Message{targetMsg, extMsg}, []*api.Enum{targetEnum}, nil).
		WithPackageName("google.cloud.example.v1")

	statusField := api.NewTestField("status").WithType(api.TypezEnum)
	statusField.EnumType = targetEnum

	for _, test := range []struct {
		name       string
		field      *api.Field
		wantHint   string
		wantSphinx string
	}{
		{
			name:       "nil field",
			field:      nil,
			wantHint:   "str",
			wantSphinx: "str",
		},
		{
			name:       "same package message",
			field:      api.NewTestField("target").WithType(api.TypezMessage).WithMessageType(targetMsg),
			wantHint:   "target.Target",
			wantSphinx: "google.cloud.example_v1.types.Target",
		},
		{
			name:       "external package message",
			field:      api.NewTestField("duration").WithType(api.TypezMessage).WithMessageType(extMsg),
			wantHint:   "duration_pb2.Duration",
			wantSphinx: "google.protobuf.duration_pb2.Duration",
		},
		{
			name:       "enum field",
			field:      statusField,
			wantHint:   "target.Status",
			wantSphinx: "google.cloud.example_v1.types.Status",
		},
		{
			name:       "repeated string",
			field:      api.NewTestField("names").WithType(api.TypezString).WithRepeated(),
			wantHint:   "MutableSequence[str]",
			wantSphinx: "MutableSequence[str]",
		},
		{
			name:       "bytes field",
			field:      api.NewTestField("raw").WithType(api.TypezBytes),
			wantHint:   "bytes",
			wantSphinx: "bytes",
		},
		{
			name:       "bool field",
			field:      api.NewTestField("flag").WithType(api.TypezBool),
			wantHint:   "bool",
			wantSphinx: "bool",
		},
		{
			name:       "int field",
			field:      api.NewTestField("count").WithType(api.TypezInt64),
			wantHint:   "int",
			wantSphinx: "int",
		},
		{
			name:       "float field",
			field:      api.NewTestField("ratio").WithType(api.TypezFloat),
			wantHint:   "float",
			wantSphinx: "float",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := newTestCodec(t, model, nil)
			gotHint, gotSphinx := c.resolveFieldTypeHintAndSphinx(test.field, "google.cloud.example_v1")
			if diff := cmp.Diff(test.wantHint, gotHint); diff != "" {
				t.Errorf("hint mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantSphinx, gotSphinx); diff != "" {
				t.Errorf("sphinx mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveMessageStem(t *testing.T) {
	for _, test := range []struct {
		name string
		msg  *api.Message
		want string
	}{
		{
			name: "nil message",
			msg:  nil,
			want: "common",
		},
		{
			name: "valid message",
			msg: api.NewTestMessage("Foo").
				WithSourceLocation("google/cloud/test/v1/my_resource.proto", 1),
			want: "my_resource",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := newTestCodec(t, api.NewTestAPI(nil, nil, nil), nil)
			got := c.resolveMessageStem(test.msg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveMessageType(t *testing.T) {
	msg := api.NewTestMessage("Foo").WithID(".google.cloud.test.v1.Foo")
	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)

	for _, test := range []struct {
		name      string
		id        string
		wantFound bool
	}{
		{
			name:      "empty id",
			id:        "",
			wantFound: false,
		},
		{
			name:      "exact id with leading dot",
			id:        ".google.cloud.test.v1.Foo",
			wantFound: true,
		},
		{
			name:      "id without leading dot",
			id:        "google.cloud.test.v1.Foo",
			wantFound: true,
		},
		{
			name:      "unknown id",
			id:        ".google.cloud.test.v1.NonExistent",
			wantFound: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := newTestCodec(t, model, nil)
			got := c.resolveMessageType(test.id)
			if (got != nil) != test.wantFound {
				t.Errorf("resolveMessageType(%q) got %v, want found=%v", test.id, got, test.wantFound)
			}
		})
	}
}
