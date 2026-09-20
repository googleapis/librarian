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

func TestAnnotateOneOf(t *testing.T) {
	for _, test := range []struct {
		name  string
		oneOf *api.OneOf
		want  *oneofAnnotations
	}{
		{
			name: "basic oneof",
			oneOf: api.NewTestOneOf("payload").
				WithFields(
					api.NewTestField("data").WithType(api.TypezBytes),
					api.NewTestField("data_crc32c").WithType(api.TypezInt64),
				),
			want: &oneofAnnotations{
				Name: "payload",
				Fields: []*fieldAnnotations{
					{Name: "data", IsOneOf: true, OneOfName: "payload", DocIsOneOf: true, DocOneOfName: "payload", IsPrimitive: true},
					{Name: "data_crc32c", IsOneOf: true, OneOfName: "payload", DocIsOneOf: true, DocOneOfName: "payload", IsPrimitive: true},
				},
			},
		},
		{
			name: "oneof with documentation",
			oneOf: func() *api.OneOf {
				o := api.NewTestOneOf("target").
					WithFields(api.NewTestField("destination").WithType(api.TypezString))
				o.Documentation = "The target destination."
				return o
			}(),
			want: &oneofAnnotations{
				Name:     "target",
				DocLines: []string{"The target destination."},
				Fields: []*fieldAnnotations{
					{Name: "destination", IsOneOf: true, OneOfName: "target", DocIsOneOf: true, DocOneOfName: "target", IsPrimitive: true},
				},
			},
		},
		{
			name: "oneof with python keyword escaped",
			oneOf: api.NewTestOneOf("import").
				WithFields(api.NewTestField("item").WithType(api.TypezString)),
			want: &oneofAnnotations{
				Name: "import_",
				Fields: []*fieldAnnotations{
					{Name: "item", IsOneOf: true, OneOfName: "import_", DocIsOneOf: true, DocOneOfName: "import_", IsPrimitive: true},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := api.NewTestMessage("SecretPayload").
				WithOneOfs(test.oneOf).
				WithFields(test.oneOf.Fields...)
			model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
			c := newTestCodec(t, model, nil)
			if err := c.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann, ok := test.oneOf.Codec.(*oneofAnnotations)
			if !ok {
				t.Fatalf("got %T, want *oneofAnnotations", test.oneOf.Codec)
			}
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(oneofAnnotations{}, "Message", "OneOf"),
				cmpopts.IgnoreFields(fieldAnnotations{}, "Message", "Field", "TypeName", "ProtoType", "TypeHint", "SphinxType", "TypeRef", "IsMessage", "IsEnum", "IsOptional", "DocLines"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
