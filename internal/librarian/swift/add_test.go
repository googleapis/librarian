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

func TestAdd(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{
			name: "no config",
			cfg:  &config.Config{},
			want: defaultVersion},
		{
			name: "no swift default",
			cfg:  &config.Config{Default: &config.Default{}},
			want: defaultVersion,
		},
		{
			name: "no swift default version",
			cfg:  &config.Config{Default: &config.Default{Swift: &config.SwiftDefault{}}},
			want: defaultVersion,
		},
		{
			name: "with version",
			cfg:  &config.Config{Default: &config.Default{Swift: &config.SwiftDefault{DefaultVersion: "0.1.0-preview"}}},
			want: "0.1.0-preview",
		},
		{
			name: "with stable version",
			cfg:  &config.Config{Default: &config.Default{Swift: &config.SwiftDefault{DefaultVersion: "1.0.0"}}},
			want: "1.0.0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			lib := &config.Library{}
			got := Add(lib, test.cfg)
			if diff := cmp.Diff(test.want, got.Version); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
