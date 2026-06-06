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

package golang

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
)

func TestMergeEnv(t *testing.T) {
	for _, test := range []struct {
		name string
		env  map[string]string
		path string
		want func(base string) map[string]string
	}{
		{
			name: "nil env",
			env:  nil,
			path: "/original/path",
			want: func(base string) map[string]string {
				return map[string]string{
					envPath: base + ":/original/path",
				}
			},
		},
		{
			name: "custom env keys merged",
			env: map[string]string{
				"FOO": "bar",
			},
			path: "/original/path",
			want: func(base string) map[string]string {
				return map[string]string{
					envPath: base + ":/original/path",
					"FOO":   "bar",
				}
			},
		},
		{
			name: "env overrides PATH",
			env: map[string]string{
				envPath: "/env/custom/path",
			},
			path: "/original/path",
			want: func(base string) map[string]string {
				return map[string]string{
					envPath: "/env/custom/path",
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			baseDir := t.TempDir()
			t.Setenv(cache.EnvLibrarianBin, baseDir)
			t.Setenv(envPath, test.path)
			got, err := mergeEnv(test.env)
			if err != nil {
				t.Fatal(err)
			}
			wantAbsBase, err := filepath.Abs(baseDir)
			if err != nil {
				t.Fatal(err)
			}
			wantMap := test.want(wantAbsBase)
			if diff := cmp.Diff(wantMap, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
