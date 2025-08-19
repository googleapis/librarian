// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"strings"
	"testing"
)

func TestGlobalConfig_Validate(t *testing.T) {
	for _, test := range []struct {
		name       string
		config     *LibrarianConfig
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid config",
			config: &LibrarianConfig{
				GlobalFilesAllowlist: []*GlobalFile{
					{
						Path:        "a/path",
						Permissions: "read-only",
					},
					{
						Path:        "another/path",
						Permissions: "write-only",
					},
					{
						Path:        "other/paths",
						Permissions: "read-write",
					},
				},
			},
		},
		{
			name: "invalid path in config",
			config: &LibrarianConfig{
				GlobalFilesAllowlist: []*GlobalFile{
					{
						Path:        "a/path",
						Permissions: "read-only",
					},
					{
						Path:        "/another/absolute/path",
						Permissions: "write-only",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "invalid global file path",
		},
		{
			name: "invalid permission in config",
			config: &LibrarianConfig{
				GlobalFilesAllowlist: []*GlobalFile{
					{
						Path:        "a/path",
						Permissions: "write-only",
					},
					{
						Path:        "another/path",
						Permissions: "unknown",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "invalid global file permissions",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			if test.wantErr {
				if err == nil {
					t.Errorf("GlobalConfig.Validate() should return error")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("GlobalConfig.Validate() err = %v, want error containing %q", err, test.wantErrMsg)
				}

				return
			}

			if err != nil {
				t.Errorf("GlobalConfig.Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
