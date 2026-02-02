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

package dart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/common"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "dart")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	library := &config.Library{
		Name:          "google-cloud-secretmanager-v1",
		Version:       "0.1.0",
		Output:        outDir,
		CopyrightYear: "2025",
		APIs: []*config.API{
			{
				Path: "google/cloud/secretmanager/v1",
			},
		},
		Dart: &config.DartPackage{
			APIKeysEnvironmentVariables: "GOOGLE_API_KEY",
			IssueTrackerURL:             "https://github.com/googleapis/google-cloud-dart/issues",
			Packages: map[string]string{
				"package:googleapis_auth":           "^2.0.0",
				"package:http":                      "^1.3.0",
				"package:google_cloud_api":          "^0.4.0",
				"package:google_cloud_iam_v1":       "^0.4.0",
				"package:google_cloud_protobuf":     "^0.4.0",
				"package:google_cloud_location":     "^0.4.0",
				"package:google_cloud_longrunning":  "^0.4.0",
				"package:google_cloud_logging_type": "^0.4.0",
				"package:google_cloud_rpc":          "^0.4.0",
				"package:google_cloud_type":         "^0.4.0",
			},
		},
	}
	sources := &common.Sources{
		Googleapis: googleapisDir,
	}
	if err := Generate(t.Context(), library, sources); err != nil {
		t.Fatal(err)
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path string
		want string
	}{
		{filepath.Join(outDir, "pubspec.yaml"), "name:"},
		{filepath.Join(outDir, "pubspec.yaml"), "google_cloud_secretmanager_v1"},
		{filepath.Join(outDir, "README.md"), "Secret Manager"},
		{filepath.Join(outDir, "lib", "secretmanager.dart"), "library"},
	} {
		t.Run(test.path, func(t *testing.T) {
			if _, err := os.Stat(test.path); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), test.want) {
				t.Errorf("%q missing expected string: %q", test.path, test.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "dart")
	outDir := t.TempDir()
	dartFile := filepath.Join(outDir, "test.dart")
	if err := os.WriteFile(dartFile, []byte("void main() { print('hello'); }"), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Output: outDir,
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}
}
