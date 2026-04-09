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

func TestFormatPath(t *testing.T) {
	for _, test := range []struct {
		name     string
		template *api.PathTemplate
		want     string
	}{
		{
			name: "literals only",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithLiteral("operations"),
			want: "/v1/operations",
		},
		{
			name: "with variable",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("name"),
			want: "/v1/\\(request.name)",
		},
		{
			name: "nested variable",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("project", "name"),
			want: "/v1/\\(request.project.name)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatPath(test.template)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
