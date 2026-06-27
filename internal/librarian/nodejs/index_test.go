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

package nodejs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindVersion(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  []versionAndClient
	}{
		{
			name: "valid single version",
			setup: func(t *testing.T, dir string) {
				v1Dir := filepath.Join(dir, "src", "v1")
				if err := os.MkdirAll(v1Dir, 0755); err != nil {
					t.Fatal(err)
				}
				content := "export { AppConnectionsServiceClient } from './app_connections_service_client';\n"
				if err := os.WriteFile(filepath.Join(v1Dir, "index.ts"), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: []versionAndClient{
				{Version: "v1", Client: "AppConnectionsServiceClient"},
			},
		},
		{
			name: "valid multiple versions",
			setup: func(t *testing.T, dir string) {
				v1Dir := filepath.Join(dir, "src", "v1")
				v1beta1Dir := filepath.Join(dir, "src", "v1beta1")
				for _, d := range []string{v1Dir, v1beta1Dir} {
					if err := os.MkdirAll(d, 0755); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(v1Dir, "index.ts"), []byte("export { ClientV1Client } from './client';"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(v1beta1Dir, "index.ts"), []byte("export { ClientV1Beta1Client } from './client';"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: []versionAndClient{
				{Version: "v1", Client: "ClientV1Client"},
				{Version: "v1beta1", Client: "ClientV1Beta1Client"},
			},
		},
		{
			name: "ignores non-version dirs",
			setup: func(t *testing.T, dir string) {
				v1Dir := filepath.Join(dir, "src", "v1")
				helpersDir := filepath.Join(dir, "src", "helpers")
				if err := os.MkdirAll(v1Dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(helpersDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(v1Dir, "index.ts"), []byte("export { ClientV1Client } from './client';"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(helpersDir, "index.ts"), []byte("export { Helper } from './helper';"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: []versionAndClient{
				{Version: "v1", Client: "ClientV1Client"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			test.setup(t, tmpDir)
			got, err := findVersion(tmpDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(versionAndClient{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindVersion_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(t *testing.T, dir string)
		wantErr error
	}{
		{
			name: "missing src dir",
			setup: func(t *testing.T, dir string) {
				// Do nothing
			},
			wantErr: os.ErrNotExist,
		},
		{
			name: "missing client export in index.ts",
			setup: func(t *testing.T, dir string) {
				v1Dir := filepath.Join(dir, "src", "v1")
				if err := os.MkdirAll(v1Dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(v1Dir, "index.ts"), []byte("export const foo = 'bar';"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errNoClientFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			test.setup(t, tmpDir)
			_, err := findVersion(tmpDir)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("findVersion() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
