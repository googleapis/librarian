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

package cpp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateModel(t *testing.T) {
	for _, test := range []struct {
		name  string
		lib   *config.CppLibrary
		model *api.API
		want  *modelAnnotations
	}{
		{
			name:  "annotates empty model default year",
			lib:   nil,
			model: api.NewTestAPI(nil, nil, nil),
			want: &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			},
		},
		{
			name: "annotates model with custom copyright year",
			lib: &config.CppLibrary{
				InitialCopyrightYear: "2022",
			},
			model: api.NewTestAPI(
				[]*api.Message{api.NewTestMessage("Item")},
				nil,
				[]*api.Service{api.NewTestService("ItemsService")},
			),
			want: &modelAnnotations{
				CopyrightYear: "2022",
				BoilerPlate:   license.HeaderBulk(),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := newCodec(test.lib)
			if err := c.annotateModel(test.model); err != nil {
				t.Fatal(err)
			}
			got, ok := test.model.Codec.(*modelAnnotations)
			if !ok {
				t.Fatalf("expected *modelAnnotations, got %T", test.model.Codec)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
