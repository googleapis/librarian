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

func TestAnnotateEnumValue(t *testing.T) {
	for _, test := range []struct {
		name string
		ev   *api.EnumValue
		want *enumValueAnnotations
	}{
		{
			name: "basic enum value",
			ev:   api.NewTestEnumValue("STATE_UNSPECIFIED", 0),
			want: &enumValueAnnotations{
				Name:   "STATE_UNSPECIFIED",
				Number: 0,
			},
		},
		{
			name: "with documentation",
			ev:   api.NewTestEnumValue("ACTIVE", 1).WithDocumentation("The resource is active."),
			want: &enumValueAnnotations{
				Name:     "ACTIVE",
				Number:   1,
				DocLines: []string{"The resource is active."},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enum := api.NewTestEnum("State").WithValues(test.ev)
			model := api.NewTestAPI(nil, []*api.Enum{enum}, nil)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann, ok := test.ev.Codec.(*enumValueAnnotations)
			if !ok {
				t.Fatalf("got %T, want *enumValueAnnotations", test.ev.Codec)
			}
			if diff := cmp.Diff(test.want, ann, cmpopts.IgnoreFields(enumValueAnnotations{}, "Enum", "Value")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
