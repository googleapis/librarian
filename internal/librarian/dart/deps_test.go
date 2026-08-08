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

package dart

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestSortByDeps(t *testing.T) {
	for _, tc := range []struct {
		name          string
		libraryNames  []string
		deps          map[string][]string
		want          []string
		wantErrSubstr string
	}{
		{
			name:         "empty",
			libraryNames: nil,
			deps:         map[string][]string{},
			want:         nil,
		},
		{
			name:         "single library",
			libraryNames: []string{"a"},
			deps:         map[string][]string{},
			want:         []string{"a"},
		},
		{
			name:         "independent libraries (stable sort)",
			libraryNames: []string{"b", "a", "c"},
			deps:         map[string][]string{},
			want:         []string{"a", "b", "c"},
		},
		{
			name:         "simple chain",
			libraryNames: []string{"a", "b"},
			deps: map[string][]string{
				"a": {"b"},
			},
			want: []string{"b", "a"},
		},
		{
			name:         "simple DAG",
			libraryNames: []string{"a", "b", "c"},
			deps: map[string][]string{
				"a": {"b", "c"},
				"b": {"c"},
			},
			want: []string{"c", "b", "a"},
		},
		{
			name:         "cycle detected",
			libraryNames: []string{"a", "b"},
			deps: map[string][]string{
				"a": {"b"},
				"b": {"a"},
			},
			wantErrSubstr: "cycle detected",
		},
		{
			name:         "self loop",
			libraryNames: []string{"a"},
			deps: map[string][]string{
				"a": {"a"},
			},
			wantErrSubstr: "cycle detected",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			libraryByName := make(map[string]*config.Library)
			for _, name := range tc.libraryNames {
				libraryByName[name] = &config.Library{Name: name}
			}

			got, err := sortByDeps(libraryByName, tc.deps)
			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErrSubstr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("sortByDeps mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
