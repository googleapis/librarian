// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language"
)

func TestReleaseCommand(t *testing.T) {
	const testlib = "test-lib"

	for _, test := range []struct {
		name         string
		args         []string
		wantErr      error
		wantVersions map[string]string
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "release"},
			wantErr: errMissingLibraryOrAllFlag,
		},
		{
			name:    "library name and all flag",
			args:    []string{"librarian", "release", testlib, "--all"},
			wantErr: errBothLibraryAndAllFlag,
		},
		{
			name:    "execute flag",
			args:    []string{"librarian", "release", "--all", "--execute"},
			wantErr: errExecuteNotImplemented,
		},
		{
			name: "library name",
			args: []string{"librarian", "release", testlib},
			wantVersions: map[string]string{
				testlib: language.TestReleaseVersion,
			},
		},
		{
			name: "all flag",
			args: []string{"librarian", "release", "--all"},
			wantVersions: map[string]string{
				testlib: language.TestReleaseVersion,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			configPath := filepath.Join(tempDir, librarianConfigPath)
			configContent := fmt.Sprintf(`language: testhelper
versions:
  %s: 0.1.0
`, testlib)
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatal(err)
			}
			if err := Run(t.Context(), test.args...); err != nil {
				if errors.Is(err, test.wantErr) {
					return
				}
				t.Fatal(err)
			}
			if test.wantVersions != nil {
				cfg, err := config.Read(configPath)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(test.wantVersions, cfg.Versions); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
