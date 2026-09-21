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

func TestBuildSampleCode(t *testing.T) {
	for _, test := range []struct {
		name             string
		setup            func() (*codec, *api.Method)
		wantPreInitLines []string
		wantArgs         []*sampleRequestArg
	}{
		{
			name: "primitive required fields and oneof",
			setup: func() (*codec, *api.Method) {
				nameField := api.NewTestField("name").
					WithType(api.TypezString).
					WithBehavior(api.FieldBehaviorRequired)
				intField := api.NewTestField("count").
					WithType(api.TypezInt32).
					WithBehavior(api.FieldBehaviorRequired)
				blobField := api.NewTestField("data").
					WithType(api.TypezBytes)
				oneOf := api.NewTestOneOf("payload").
					WithFields(blobField)

				req := api.NewTestMessage("CreateRequest").
					WithFields(nameField, intField, blobField).
					WithOneOfs(oneOf)
				m := api.NewTestMethod("Create").
					WithInput(req)
				model := api.NewTestAPI([]*api.Message{req}, nil, nil)
				return newTestCodec(t, model, nil), m
			},
			wantPreInitLines: nil,
			wantArgs: []*sampleRequestArg{
				{
					Name:  "data",
					Value: "b'data_blob'",
				},
				{
					Name:  "name",
					Value: `"name_value"`,
				},
				{
					Name:  "count",
					Value: sampleIntegerFieldValue,
				},
			},
		},
		{
			name: "enum field in sample",
			setup: func() (*codec, *api.Method) {
				val1 := api.NewTestEnumValue("STATE_UNSPECIFIED", 0)
				val2 := api.NewTestEnumValue("ACTIVE", 1)
				stateEnum := api.NewTestEnum("State").WithValues(val1, val2)
				enumField := api.NewTestField("state").
					WithType(api.TypezEnum)
				enumField.EnumType = stateEnum
				enumField.Behavior = []api.FieldBehavior{api.FieldBehaviorRequired}

				req := api.NewTestMessage("SetStateRequest").WithFields(enumField)
				m := api.NewTestMethod("SetState").
					WithInput(req)
				model := api.NewTestAPI([]*api.Message{req}, []*api.Enum{stateEnum}, nil)
				return newTestCodec(t, model, nil), m
			},
			wantPreInitLines: nil,
			wantArgs: []*sampleRequestArg{
				{
					Name:  "state",
					Value: `"ACTIVE"`,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, m := test.setup()
			gotPreInit, gotArgs := c.buildSampleCode(m, "v1")
			if diff := cmp.Diff(test.wantPreInitLines, gotPreInit); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantArgs, gotArgs); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSampleFieldValue(t *testing.T) {
	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name:  "string name",
			field: api.NewTestField("name").WithType(api.TypezString),
			want:  `"name_value"`,
		},
		{
			name:  "repeated string",
			field: api.NewTestField("tags").WithType(api.TypezString).WithRepeated(),
			want:  "['tags_value1', 'tags_value2']",
		},
		{
			name:  "bytes",
			field: api.NewTestField("content").WithType(api.TypezBytes),
			want:  "b'content_blob'",
		},
		{
			name:  "bool",
			field: api.NewTestField("enabled").WithType(api.TypezBool),
			want:  "True",
		},
		{
			name:  "int",
			field: api.NewTestField("size").WithType(api.TypezInt64),
			want:  sampleIntegerFieldValue,
		},
		{
			name:  "float",
			field: api.NewTestField("score").WithType(api.TypezFloat),
			want:  sampleFloatFieldValue,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := newTestCodec(t, api.NewTestAPI(nil, nil, nil), nil)
			got := c.sampleFieldValue(test.field)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
