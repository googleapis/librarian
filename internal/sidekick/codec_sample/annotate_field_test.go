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

package codec_sample

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
			name:  "simple field",
			field: api.NewTestField("name").WithType(api.TypezString),
			want: &fieldAnnotations{
				Name: "nameSample",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := api.NewTestMessage("Item").WithFields(test.field)
			model := api.NewTestAPI([]*api.Message{msg}, nil, nil)
			c := newTestCodec()
			if err := c.annotateModel(model); err != nil {
				t.Fatal(err)
			}
			got, ok := test.field.Codec.(*fieldAnnotations)
			if !ok {
				t.Fatalf("expected *fieldAnnotations, got %T", test.field.Codec)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(fieldAnnotations{}, "Message")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
