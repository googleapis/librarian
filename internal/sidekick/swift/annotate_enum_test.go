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
		values        []*api.EnumValue
		wantName      string
		wantDocs      []string
		wantDefault   string
	}{
		{
			name:          "basic enum",
			enumName:      "Color",
			documentation: "A color enum.\nWith two lines.",
			values: []*api.EnumValue{
				{Name: "COLOR_UNSPECIFIED", Number: 0},
				{Name: "COLOR_RED", Number: 1},
			},
			wantName:    "Color",
			wantDocs:    []string{"A color enum.", "With two lines."},
			wantDefault: "unspecified",
		},
		{
			name:          "escaped name",
			enumName:      "Protocol",
			documentation: "An enum named Protocol.",
			values: []*api.EnumValue{
				{Name: "PROTOCOL_UNSPECIFIED", Number: 0},
			},
			wantName:    "Protocol_",
			wantDocs:    []string{"An enum named Protocol."},
			wantDefault: "unspecified",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			enum := &api.Enum{
				Name:               test.enumName,
				Documentation:      test.documentation,
				ID:                 ".test." + test.enumName,
				Package:            "test",
				Values:             test.values,
				UniqueNumberValues: test.values,
			}
			for _, ev := range enum.Values {
				ev.Parent = enum
			}
			model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if _, err := annotateModel(codec); err != nil {
				t.Fatal(err)
			}
			want := &enumAnnotations{
				Name:            test.wantName,
				DocLines:        test.wantDocs,
				DefaultCaseName: test.wantDefault,
			}

			if diff := cmp.Diff(want, enum.Codec, cmpopts.IgnoreFields(enumAnnotations{}, "BoilerPlate", "CopyrightYear")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateEnum_Error(t *testing.T) {
	enum := &api.Enum{
		Name:    "Empty",
		ID:      ".test.Empty",
		Package: "test",
	}
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{})

	_, err := annotateModel(codec)
	if err == nil {
		t.Errorf("annotateModel() expected error for enum with no values, got nil")
	}
}
