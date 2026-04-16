// Copyright 2025 Google LLC
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

package yaml

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGet(t *testing.T) {
	m := map[string]any{
		"sources": map[string]any{
			"googleapis": map[string]any{
				"commit": "abcd123",
			},
		},
		"version": "v1.0.0",
	}

	for _, tt := range []struct {
		path    string
		want    any
		wantErr bool
	}{
		{
			path:    "version",
			want:    "v1.0.0",
			wantErr: false,
		},
		{
			path:    "sources.googleapis.commit",
			want:    "abcd123",
			wantErr: false,
		},
		{
			path: "sources.googleapis",
			want: map[string]any{
				"commit": "abcd123",
			},
			wantErr: false,
		},
		{
			path:    "sources.googleapis.sha256",
			want:    nil,
			wantErr: true,
		},
		{
			path:    "nonexistent",
			want:    nil,
			wantErr: true,
		},
	} {
		t.Run(tt.path, func(t *testing.T) {
			got, err := Get(m, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestSet(t *testing.T) {
	m := map[string]any{
		"version": "v1.0.0",
	}

	for _, tt := range []struct {
		path  string
		value any
		want  map[string]any
	}{
		{
			path:  "version",
			value: "v1.0.1",
			want: map[string]any{
				"version": "v1.0.1",
			},
		},
		{
			path:  "sources.googleapis.commit",
			value: "abcd123",
			want: map[string]any{
				"version": "v1.0.1",
				"sources": map[string]any{
					"googleapis": map[string]any{
						"commit": "abcd123",
					},
				},
			},
		},
	} {
		t.Run(tt.path, func(t *testing.T) {
			updated, err := Set(m, tt.path, tt.value)
			if err != nil {
				t.Fatalf("Set(%q, %v) error = %v", tt.path, tt.value, err)
			}
			key := strings.Split(tt.path, ".")[0]
			if diff := cmp.Diff(tt.want[key], updated[key]); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
