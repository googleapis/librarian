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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMessage(t *testing.T) {
	for _, test := range []struct {
		name    string
		message *api.Message
		want    *messageAnnotations
	}{
		{
			name: "simple",
			message: &api.Message{
				Name:          "Secret",
				Documentation: "A secret message.\nWith two lines.",
				ID:            ".test.Secret",
				Package:       "test",
				Fields: []*api.Field{
					{Name: "secret_key", JSONName: "secretKey", Typez: api.TypezString},
				},
			},
			want: &messageAnnotations{
				Name:                "Secret",
				DocLines:            []string{"A secret message.", "With two lines."},
				TypeURL:             "type.googleapis.com/test.Secret",
				CustomSerialization: false,
			},
		},
		{
			name: "escaped name",
			message: &api.Message{
				Name:          "Protocol",
				Documentation: "A message named Protocol.",
				ID:            ".test.Protocol",
				Package:       "test",
			},
			want: &messageAnnotations{
				Name:                "Protocol_",
				DocLines:            []string{"A message named Protocol."},
				TypeURL:             "type.googleapis.com/test.Protocol",
				CustomSerialization: false,
			},
		},
		{
			name: "with oneof",
			message: &api.Message{
				Name:    "WithOneof",
				ID:      ".test.WithOneof",
				Package: "test",
				OneOfs:  []*api.OneOf{{Name: "choice"}},
			},
			want: &messageAnnotations{
				Name:                "WithOneof",
				TypeURL:             "type.googleapis.com/test.WithOneof",
				CustomSerialization: true,
			},
		},
		{
			name: "with custom json name",
			message: &api.Message{
				Name:    "WithCustomJSON",
				ID:      ".test.WithCustomJSON",
				Package: "test",
				Fields: []*api.Field{
					{Name: "secret_key", JSONName: "specialKey", Typez: api.TypezString},
				},
			},
			want: &messageAnnotations{
				Name:                "WithCustomJSON",
				TypeURL:             "type.googleapis.com/test.WithCustomJSON",
				CustomSerialization: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{test.message}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, test.message.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "Model")); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
