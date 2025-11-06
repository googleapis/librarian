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

	"github.com/google/go-cmp/cmp"
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
					t.Fatal("GlobalConfig.Validate() should return error")
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

func TestLibraryConfigFor(t *testing.T) {
	cases := []struct {
		name          string
		config        *LibrarianConfig
		LibraryID     string
		wantLibrary   *LibraryConfig
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "library found",
			config: &LibrarianConfig{
				Libraries: []*LibraryConfig{
					{LibraryID: "lib1", NextVersion: "1.0.0"},
					{LibraryID: "lib2", NextVersion: "2.0.0"},
				},
			},
			LibraryID:   "lib1",
			wantLibrary: &LibraryConfig{LibraryID: "lib1", NextVersion: "1.0.0"},
		},
		{
			name: "library not found",
			config: &LibrarianConfig{
				Libraries: []*LibraryConfig{
					{LibraryID: "lib1", NextVersion: "1.0.0"},
					{LibraryID: "lib2", NextVersion: "2.0.0"},
				},
			},
			LibraryID:   "lib3",
			wantLibrary: nil,
		},
		{
			name: "empty libraries",
			config: &LibrarianConfig{
				Libraries: []*LibraryConfig{},
			},
			LibraryID:   "lib1",
			wantLibrary: nil,
		},
		{
			name: "multiple libraries with target in middle",
			config: &LibrarianConfig{
				Libraries: []*LibraryConfig{
					{LibraryID: "lib1", NextVersion: "1.0.0"},
					{LibraryID: "lib2", NextVersion: "2.0.0"},
					{LibraryID: "lib3", NextVersion: "3.0.0"},
				},
			},
			LibraryID:   "lib2",
			wantLibrary: &LibraryConfig{LibraryID: "lib2", NextVersion: "2.0.0"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotLibrary := tc.config.LibraryConfigFor(tc.LibraryID)

			if diff := cmp.Diff(tc.wantLibrary, gotLibrary); diff != "" {
				t.Errorf("LibraryConfigFor() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetGlobalFiles(t *testing.T) {
	for _, test := range []struct {
		name   string
		config *LibrarianConfig
		want   []string
	}{
		{
			name: "get_global_files",
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
			want: []string{
				"a/path",
				"another/path",
				"other/paths",
			},
		},
		{
			name: "empty_global_files",
			config: &LibrarianConfig{
				GlobalFilesAllowlist: []*GlobalFile{},
			},
			want: nil,
		},
		{
			name:   "nil_global_files",
			config: &LibrarianConfig{},
			want:   nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.config.GetGlobalFiles()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetGlobalFiles() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsGenerationBlocked(t *testing.T) {
	for _, test := range []struct {
		name      string
		config    *LibrarianConfig
		libraryID string
		want      bool
	}{
		{
			name:      "nil config",
			config:    nil,
			libraryID: "lib1",
			want:      false,
		},
		{
			name: "library not in config",
			config: &LibrarianConfig{
				Libraries: []*LibraryConfig{
					{LibraryID: "lib2", GenerateBlocked: true},
				},
			},
			libraryID: "lib1",
			want:      false,
		},
		{
			name: "library in config, generate_blocked is false",
			config: &LibrarianConfig{
				Libraries: []*LibraryConfig{
					{LibraryID: "lib1", GenerateBlocked: false},
				},
			},
			libraryID: "lib1",
			want:      false,
		},
		{
			name: "library in config, generate_blocked is true",
			config: &LibrarianConfig{
				Libraries: []*LibraryConfig{
					{LibraryID: "lib1", GenerateBlocked: true},
				},
			},
			libraryID: "lib1",
			want:      true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.config.IsGenerationBlocked(test.libraryID)
			if got != test.want {
				t.Errorf("IsGenerationBlocked() = %v, want %v", got, test.want)
			}
		})
	}
}
