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
	"github.com/googleapis/librarian/internal/config"
)

func TestLocalName(t *testing.T) {
	for _, test := range []struct {
		name string
		dep  Dependency
		want string
	}{
		{
			name: "path simple",
			dep:  Dependency{SwiftDependency: config.SwiftDependency{Path: "packages/auth"}},
			want: "auth",
		},
		{
			name: "path nested",
			dep:  Dependency{SwiftDependency: config.SwiftDependency{Path: "generated/google-cloud-location"}},
			want: "google-cloud-location",
		},
		{
			name: "path trailing slash",
			dep:  Dependency{SwiftDependency: config.SwiftDependency{Path: "packages/auth/"}},
			want: "auth",
		},
		{
			name: "url without git",
			dep:  Dependency{SwiftDependency: config.SwiftDependency{URL: "https://github.com/apple/swift-protobuf"}},
			want: "swift-protobuf",
		},
		{
			name: "url with git",
			dep:  Dependency{SwiftDependency: config.SwiftDependency{URL: "https://github.com/apple/swift-protobuf.git"}},
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

func TestDependency_Mode(t *testing.T) {
	for _, test := range []struct {
		name              string
		dep               Dependency
		wantLocalOrRemote bool
		wantLocalOnly     bool
		wantRemoteOnly    bool
	}{
		{
			name:              "local only",
			dep:               Dependency{SwiftDependency: config.SwiftDependency{Path: "pkgs/swift-google-auth"}},
			wantLocalOrRemote: false,
			wantLocalOnly:     true,
			wantRemoteOnly:    false,
		},
		{
			name:              "remote only",
			dep:               Dependency{SwiftDependency: config.SwiftDependency{URL: "https://github.com/apple/swift-log", Version: "1.12.0"}},
			wantLocalOrRemote: false,
			wantLocalOnly:     false,
			wantRemoteOnly:    true,
		},
		{
			name:              "local or remote",
			dep:               Dependency{SwiftDependency: config.SwiftDependency{URL: "https://github.com/googleapis/swift-google-auth", Path: "pkgs/swift-google-auth", Version: "0.2.0"}},
			wantLocalOrRemote: true,
			wantLocalOnly:     false,
			wantRemoteOnly:    false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.dep.IsLocalOrRemote(); got != test.wantLocalOrRemote {
				t.Errorf("IsLocalOrRemote() = %v, want %v", got, test.wantLocalOrRemote)
			}
			if got := test.dep.IsLocalOnly(); got != test.wantLocalOnly {
				t.Errorf("IsLocalOnly() = %v, want %v", got, test.wantLocalOnly)
			}
			if got := test.dep.IsRemoteOnly(); got != test.wantRemoteOnly {
				t.Errorf("IsRemoteOnly() = %v, want %v", got, test.wantRemoteOnly)
			}
		})
	}
}
