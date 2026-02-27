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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateInternalVersionFile(t *testing.T) {
	for _, test := range []struct {
		name        string
		version     string
		wantVersion string
	}{
		{
			name:        "with version",
			version:     "1.2.3",
			wantVersion: `const Version = "1.2.3"`,
		},
		{
			name:        "empty version",
			version:     "",
			wantVersion: `const Version = "0.0.0"`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := generateInternalVersionFile(dir, test.version); err != nil {
				t.Fatal(err)
			}

			content, err := os.ReadFile(filepath.Join(dir, "internal", "version.go"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), test.wantVersion) {
				t.Errorf("want %q in output, got:\n%s", test.wantVersion, content)
			}
			if !strings.Contains(string(content), "package internal") {
				t.Errorf("want package internal in output, got:\n%s", content)
			}
		})
	}
}

func TestGenerateClientVersionFile(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		apiPath string
		wantDir string
	}{
		{
			name: "basic",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "", // set in test
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientDirectory: "secretmanager",
							ImportPath:      "secretmanager/apiv1",
							Path:            "google/cloud/secretmanager/v1",
						},
					},
				},
			},
			apiPath: "google/cloud/secretmanager/v1",
			wantDir: "secretmanager/apiv1",
		},
		{
			name: "custom client directory",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "", // set in test
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientDirectory: "secretmanager",
							ImportPath:      "secretmanager/customdir/apiv1",
							Path:            "google/cloud/secretmanager/v1",
						},
					},
				},
			},
			apiPath: "google/cloud/secretmanager/v1",
			wantDir: "secretmanager/customdir/apiv1",
		},
		{
			name: "custom client package",
			library: &config.Library{
				Name:   "storage",
				Output: "", // set in test
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							ClientPackageOverride: "storage",
							ClientDirectory:       "internal",
							ImportPath:            "storage/internal/apiv2",
							Path:                  "google/cloud/storage/v2",
						},
					},
				},
			},
			apiPath: "google/cloud/storage/v2",
			wantDir: "storage/internal/apiv2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.library.Output = dir

			if err := generateClientVersionFile(test.library, test.apiPath); err != nil {
				t.Fatal(err)
			}

			versionPath := filepath.Join(dir, test.wantDir, "version.go")
			content, err := os.ReadFile(versionPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), "versionClient = internal.Version") {
				t.Errorf("want versionClient assignment in output, got:\n%s", content)
			}
		})
	}
}

func TestGenerateClientVersionFile_Skipped(t *testing.T) {
	dir := t.TempDir()
	library := &config.Library{
		Name:   "alloydb",
		Output: dir,
		Go: &config.GoModule{
			GoAPIs: []*config.GoAPI{
				{
					ClientDirectory: "connectors",
					ImportPath:      "alloydb/connectors/apiv1",
					DisableGAPIC:    true,
					Path:            "google/cloud/alloydb/connectors/v1",
				},
			},
		},
	}
	apiPath := "google/cloud/alloydb/connectors/v1"

	if err := generateClientVersionFile(library, apiPath); err != nil {
		t.Fatal(err)
	}

	gotPath := filepath.Join(library.Output, "alloydb/connectors/apiv1")
	versionPath := filepath.Join(gotPath, "version.go")
	if _, err := os.Stat(versionPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("client version file should not exist")
	}
}
