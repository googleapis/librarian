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

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSimplifyRestNumericEnums(t *testing.T) {
	for _, test := range []struct {
		name             string
		restNumericEnums map[string]bool
		want             map[string]bool
	}{
		{
			name: "all true",
			restNumericEnums: map[string]bool{
				"csharp": true,
				"go":     true,
				"java":   true,
				"nodejs": true,
				"php":    true,
				"python": true,
				"ruby":   true,
			},
			want: map[string]bool{"all": true},
		},
		{
			name: "all false",
			restNumericEnums: map[string]bool{
				"csharp": false,
				"go":     false,
				"java":   false,
				"nodejs": false,
				"php":    false,
				"python": false,
				"ruby":   false,
			},
			want: map[string]bool{},
		},
		{
			name: "all present, different values",
			restNumericEnums: map[string]bool{
				"csharp": false,
				"go":     true,
				"java":   false,
				"nodejs": false,
				"php":    false,
				"python": true,
				"ruby":   false,
			},
			want: map[string]bool{
				"go":     true,
				"python": true,
			},
		},
		{
			name: "some present, different values",
			restNumericEnums: map[string]bool{
				"csharp": false,
				"java":   true,
				"python": true,
				"ruby":   false,
			},
			want: map[string]bool{
				"java":   true,
				"python": true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := simplifyRestNumericEnums(test.restNumericEnums)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
