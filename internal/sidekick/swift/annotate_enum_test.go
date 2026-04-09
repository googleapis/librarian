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

func TestAnnotateEnum(t *testing.T) {
	for _, test := range []struct {
		name          string
		enumName      string
		documentation string
		wantName      string
		wantDocs      []string
	}{
		{
			name:          "basic enum",
			enumName:      "Color",
			documentation: "A color enum.\nWith two lines.",
			wantName:      "Color",
			wantDocs:      []string{"A color enum.", "With two lines."},
		},
		{
			name:          "escaped name",
			enumName:      "Protocol",
			documentation: "An enum named Protocol.",
			wantName:      "Protocol_",
			wantDocs:      []string{"An enum named Protocol."},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enum := &api.Enum{
				Name:          test.enumName,
				Documentation: test.documentation,
				ID:            ".test." + test.enumName,
				Package:       "test",
			}
			model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			want := &enumAnnotations{
				Name:     test.wantName,
				DocLines: test.wantDocs,
			}

			if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(enumAnnotations{}, "BoilerPlate", "CopyrightYear")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
