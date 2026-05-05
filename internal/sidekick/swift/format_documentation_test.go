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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestFormatDocumentation(t *testing.T) {
	codec := newTestCodec(t, api.NewTestAPI(nil, nil, nil), nil)

	for _, test := range []struct {
		name string
		doc  string
		want []string
	}{
		{
			name: "empty",
			doc:  "",
			want: nil,
		},
		{
			name: "single line",
			doc:  "Hello world",
			want: []string{"Hello world"},
		},
		{
			name: "multiple lines",
			doc:  "Line 1\nLine 2",
			want: []string{"Line 1", "Line 2"},
		},
		{
			name: "trailing newline",
			doc:  "Line 1\n",
			want: []string{"Line 1", ""},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codec.formatDocumentation(test.doc, []string{})
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("formatDocumentation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatDocumentationWithLinks(t *testing.T) {
	someMessage := &api.Message{
		Name:    "SomeMessage",
		ID:      ".test.v1.SomeMessage",
		Package: "test.v1",
	}
	model := api.NewTestAPI([]*api.Message{someMessage}, []*api.Enum{}, []*api.Service{})
	c := newTestCodec(t, model, nil)

	input := `Refer to [SomeMessage][] for details.`
	want := []string{
		"Refer to [SomeMessage][] for details.",
		"",
		"[SomeMessage]: <doc:SomeMessage>",
	}

	got, err := c.formatDocumentation(input, []string{"test.v1"})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("formatDocumentation() mismatch (-want +got):\n%s", diff)
	}
}
