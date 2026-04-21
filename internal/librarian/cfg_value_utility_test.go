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

package librarian

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGetConfigValue(t *testing.T) {
	currentConfig := &config.Config{
		Version: "v1.0.0",
	}

	for _, test := range []struct {
		path string
		want string
	}{
		{
			path: "version",
			want: "v1.0.0",
		},
	} {
		t.Run(test.path, func(t *testing.T) {
			got, err := getConfigValue(currentConfig, test.path)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetConfigValue_Error(t *testing.T) {
	currentConfig := &config.Config{
		Version: "v1.0.0",
	}
	for _, test := range []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "invalid path",
			path:    "invalid.path",
			wantErr: errUnsupportedPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := getConfigValue(currentConfig, test.path)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("getConfigValue(%q) error = %v, wantErr %v", test.path, err, test.wantErr)
			}
		})
	}
}

func TestSetConfigValue(t *testing.T) {
	for _, test := range []struct {
		path  string
		value string
		want  *config.Config
	}{
		{
			path:  "version",
			value: "v1.0.1",
			want: &config.Config{
				Version: "v1.0.1",
			},
		},
	} {
		t.Run(test.path, func(t *testing.T) {
			cfg := &config.Config{
				Version: "v1.0.0",
			}
			got, err := setConfigValue(cfg, test.path, test.value)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetConfigValue_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "unsupported path",
			path:    "unknown.field",
			wantErr: errUnsupportedPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Version: "v1.0.0",
			}
			_, err := setConfigValue(cfg, test.path, "some-value")
			if !errors.Is(err, test.wantErr) {
				t.Errorf("setConfigValue(%q) error = %v, wantErr %v", test.path, err, test.wantErr)
			}
		})
	}
}
