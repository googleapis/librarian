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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMessage(t *testing.T) {
	for _, test := range []struct {
		name    string
		message *api.Message
		want    *messageAnnotations
	}{
		{
			name:    "simple message",
			message: api.NewTestMessage("Item"),
			want: &messageAnnotations{
				Name: "ItemSample",
			},
		},
		{
			name: "nested types",
			message: api.NewTestMessage("Item").
				WithMessages(api.NewTestMessage("NestedItem")).
				WithEnums(api.NewTestEnum("NestedEnum")),
			want: &messageAnnotations{
				Name: "ItemSample",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{test.message}, nil, nil)
			c, err := newCodec()
			if err != nil {
				t.Fatal(err)
			}
			if err := c.annotateModel(model); err != nil {
				t.Fatal(err)
			}
			got, ok := test.message.Codec.(*messageAnnotations)
			if !ok {
				t.Fatalf("expected *messageAnnotations, got %T", test.message.Codec)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			for _, child := range test.message.Messages {
				if _, ok := child.Codec.(*messageAnnotations); !ok {
					t.Errorf("expected nested message codec *messageAnnotations, got %T", child.Codec)
				}
			}
			for _, e := range test.message.Enums {
				if _, ok := e.Codec.(*enumAnnotations); !ok {
					t.Errorf("expected nested enum codec *enumAnnotations, got %T", e.Codec)
				}
			}
		})
	}
}
