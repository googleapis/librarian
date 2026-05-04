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

package gcloud

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFlag(t *testing.T) {
	for _, test := range []struct {
		name     string
		flagName string
		kind     string
		required bool
		want     Flag
	}{
		{
			name:     "required project",
			flagName: "project",
			kind:     "String",
			required: true,
			want:     Flag{Name: "project", Kind: "String", Required: true, Usage: "The project."},
		},
		{
			name:     "optional page-token",
			flagName: "page-token",
			kind:     "String",
			required: false,
			want:     Flag{Name: "page-token", Kind: "String", Required: false, Usage: "The page token."},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := flag(test.flagName, test.kind, test.required)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPathFlag(t *testing.T) {
	got := pathFlag("location")
	want := Flag{Name: "location", Kind: "String", Required: true, Usage: "The location."}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
