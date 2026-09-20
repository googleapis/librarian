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

func TestAnnotateEnum(t *testing.T) {
	for _, test := range []struct {
		name string
		enum *api.Enum
		want *enumAnnotations
	}{
		{
			name: "basic enum with values",
			enum: api.NewTestEnum("SecretStatus").
				WithDocumentation("Status of secret.").
				WithValues(
					api.NewTestEnumValue("STATUS_UNSPECIFIED", 0),
					api.NewTestEnumValue("ENABLED", 1),
				),
			want: &enumAnnotations{
				Name:            "SecretStatus",
				DocLines:        []string{"Status of secret."},
				FirstDocLine:    "Status of secret.",
				HasDocLines:     true,
				HasMultiLineDoc: false,
				HasValues:       true,
				Values: []*enumValueAnnotations{
					{Name: "STATUS_UNSPECIFIED", Number: 0},
					{Name: "ENABLED", Number: 1},
				},
			},
		},
		{
			name: "snake_case enum name normalized to PascalCase",
			enum: api.NewTestEnum("crypto_key_version_state").
				WithValues(api.NewTestEnumValue("STATE_UNSPECIFIED", 0)),
			want: &enumAnnotations{
				Name:      "CryptoKeyVersionState",
				HasValues: true,
				Values: []*enumValueAnnotations{
					{Name: "STATE_UNSPECIFIED", Number: 0},
				},
			},
		},
		{
			name: "enum with multiline documentation",
			enum: func() *api.Enum {
				e := api.NewTestEnum("Color")
				e.Documentation = "Color of the item.\n\nRepresents RGB colors."
				return e
			}(),
			want: &enumAnnotations{
				Name:              "Color",
				DocLines:          []string{"Color of the item.", "", "Represents RGB colors."},
				FirstDocLine:      "Color of the item.",
				RemainingDocLines: []string{"", "Represents RGB colors."},
				HasDocLines:       true,
				HasMultiLineDoc:   true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI(nil, []*api.Enum{test.enum}, nil)
			c := newTestCodec(t, model, nil)
			if err := c.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann, ok := test.enum.Codec.(*enumAnnotations)
			if !ok {
				t.Fatalf("got %T, want *enumAnnotations", test.enum.Codec)
			}
			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(enumAnnotations{}, "Model", "Enum"),
				cmpopts.IgnoreFields(enumValueAnnotations{}, "Enum", "Value"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
