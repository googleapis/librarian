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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLocalName(t *testing.T) {
	for _, test := range []struct {
		name string
		dep  SwiftDependency
		want string
	}{
		{
			name: "path simple",
			dep:  SwiftDependency{Path: "packages/auth"},
			want: "auth",
		},
		{
			name: "path nested",
			dep:  SwiftDependency{Path: "generated/google-cloud-location"},
			want: "google-cloud-location",
		},
		{
			name: "path trailing slash",
			dep:  SwiftDependency{Path: "packages/auth/"},
			want: "auth",
		},
		{
			name: "url without git",
			dep:  SwiftDependency{URL: "https://github.com/apple/swift-protobuf"},
			want: "swift-protobuf",
		},
		{
			name: "url with git",
			dep:  SwiftDependency{URL: "https://github.com/apple/swift-protobuf.git"},
			want: "swift-protobuf",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.dep.LocalName()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
