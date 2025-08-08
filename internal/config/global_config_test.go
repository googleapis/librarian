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
		config     *GlobalConfig
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid config",
			config: &GlobalConfig{
				GlobalFilesAllowlist: []*GlobalFile{
					{
						Path:        "a/path",
						Permissions: ReadOnly,
					},
					{
						Path:        "another/path",
						Permissions: WriteOnly,
					},
					{
						Path:        "other/paths",
						Permissions: ReadWrite,
					},
				},
			},
		},
		{
			name: "invalid path in config",
			config: &GlobalConfig{
				GlobalFilesAllowlist: []*GlobalFile{
					{
						Path:        "a/path",
						Permissions: ReadOnly,
					},
					{
						Path:        "/another/absolute/path",
						Permissions: WriteOnly,
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "invalid global file path",
		},
		{
			name: "invalid permission in config",
			config: &GlobalConfig{
				GlobalFilesAllowlist: []*GlobalFile{
					{
						Path:        "a/path",
						Permissions: ReadOnly,
					},
					{
						Path:        "another/path",
						Permissions: Unknown,
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "invalid global file permission",
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
