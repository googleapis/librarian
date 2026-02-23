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

package serviceconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractVersion(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		path string
		want string
	}{
		{"google/cloud/secretmanager/v1", "v1"},
		{"google/cloud/secretmanager/v1beta2", "v1beta2"},
		{"google/cloud/v2/secretmanager", "v2"},
		{"google/cloud/secretmanager", ""},
		{"google/ai/generativelanguage/v1alpha", "v1alpha"},
	} {
		t.Run(test.path, func(t *testing.T) {
			got := ExtractVersion(test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("ExtractVersion(%q) returned diff (-want +got): %s", test.path, diff)
			}
		})
	}
}

func TestIsVersion(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		s    string
		want bool
	}{
		{"v1", true},
		{"v1beta1", true},
		{"v2alpha", true},
		{"apiv1", false},
		{"v1-py", false},
		{"", false},
	} {
		t.Run(test.s, func(t *testing.T) {
			got := IsVersion(test.s)
			if got != test.want {
				t.Errorf("IsVersion(%q) = %v, want %v", test.s, got, test.want)
			}
		})
	}
}
