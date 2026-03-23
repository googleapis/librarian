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

package java

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFill(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "fill output from name",
			lib: &config.Library{
				Name: "secretmanager",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
			},
		},
		{
			name: "do not overwrite output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Fill(test.lib)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTidy(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "tidy default output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "",
			},
		},
		{
			name: "do not tidy custom output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Tidy(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
