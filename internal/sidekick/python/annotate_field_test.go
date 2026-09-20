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

func TestAnnotateField(t *testing.T) {
	for _, test := range []struct {
		name  string
		field *api.Field
		want  *fieldAnnotations
	}{
		{
			name: "primitive field",
			field: func() *api.Field {
				f := api.NewTestField("secret_name").WithType(api.TypezString)
				f.Documentation = "The name of the secret."
				return f
			}(),
			want: &fieldAnnotations{
				Name:     "secret_name",
				DocLines: []string{"The name of the secret."},
			},
		},
		{
			name: "python keyword field escaped",
			field: api.NewTestField("from").
				WithType(api.TypezString),
			want: &fieldAnnotations{
				Name: "from_",
			},
		},
		{
			name: "repeated field",
			field: api.NewTestField("aliases").
				WithType(api.TypezString).
				WithRepeated(),
			want: &fieldAnnotations{
				Name:       "aliases",
				IsRepeated: true,
			},
		},
		{
			name: "map field",
			field: api.NewTestField("labels").
				WithMap(),
			want: &fieldAnnotations{
				Name:  "labels",
				IsMap: true,
			},
		},
		{
			name: "message reference field",
			field: api.NewTestField("secret_payload").
				WithMessageType(api.NewTestMessage("SecretPayload").WithPackage("google.cloud.secretmanager.v1")),
			want: &fieldAnnotations{
				Name:     "secret_payload",
				TypeName: ".google.cloud.secretmanager.v1.SecretPayload",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := api.NewTestMessage("Secret").WithFields(test.field)
			model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann, ok := test.field.Codec.(*fieldAnnotations)
			if !ok {
				t.Fatalf("got %T, want *fieldAnnotations", test.field.Codec)
			}
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(fieldAnnotations{}, "Message", "Field"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
