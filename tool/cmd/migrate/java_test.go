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

package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/license"
)

func licenseHeader() string {
	var b strings.Builder
	year := time.Now().Year()
	for _, line := range license.Header(strconv.Itoa(year)) {
		b.WriteString("#")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func TestRunJavaMigration(t *testing.T) {
	for _, test := range []struct {
		name                string
		genConfigContent    string
		wantLibrarianYAML   string
		wantErr             bool
		expectLibrarianFile bool
	}{
		{
			name: "prioritize library_name over api_shortname",
			genConfigContent: `libraries:
  - api_shortname: language
    library_name: language-v1
    GAPICs:
      - proto_path: google/cloud/language/v1
`,
			wantLibrarianYAML: licenseHeader() + `language: java
sources:
  googleapis:
    commit: ""
    dir: ../../googleapis
default: {}
libraries:
  - name: language-v1
    apis:
      - path: google/cloud/language/v1
    output: java-language-v1
`,
			expectLibrarianFile: true,
		},
		{
			name: "fallback to api_shortname",
			genConfigContent: `libraries:
  - api_shortname: language
    GAPICs:
      - proto_path: google/cloud/language/v1
`,
			wantLibrarianYAML: licenseHeader() + `language: java
sources:
  googleapis:
    commit: ""
    dir: ../../googleapis
default: {}
libraries:
  - name: language
    apis:
      - path: google/cloud/language/v1
    output: java-language
`,
			expectLibrarianFile: true,
		},
		{
			name: "multiple libraries",
			genConfigContent: `libraries:
  - api_shortname: language
    GAPICs:
      - proto_path: google/cloud/language/v1
  - library_name: vision
    GAPICs:
      - proto_path: google/cloud/vision/v1
  - api_shortname: texttospeech
    library_name: tts
    GAPICs:
      - proto_path: google/cloud/texttospeech/v1
`,
			wantLibrarianYAML: licenseHeader() + `language: java
sources:
  googleapis:
    commit: ""
    dir: ../../googleapis
default: {}
libraries:
  - name: language
    apis:
      - path: google/cloud/language/v1
    output: java-language
  - name: tts
    apis:
      - path: google/cloud/texttospeech/v1
    output: java-tts
  - name: vision
    apis:
      - path: google/cloud/vision/v1
    output: java-vision
`,
			expectLibrarianFile: true,
		},
		{
			name:             "no generation config",
			genConfigContent: "",
			wantErr:          true,
		},
		{
			name: "empty libraries list",
			genConfigContent: `libraries: []
`,
		},
		{
			name:             "invalid yaml",
			genConfigContent: "{",
			wantErr:          true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.genConfigContent != "" {
				genConfigFile := filepath.Join(tmpDir, "generation_config.yaml")
				if err := os.WriteFile(genConfigFile, []byte(test.genConfigContent), 0644); err != nil {
					t.Fatalf("failed to write test generation_config.yaml: %v", err)
				}
			}

			// RunTidyOnConfig writes to librarian.yaml in the current working directory.
			t.Chdir(tmpDir)

			err := runJavaMigration(context.Background(), ".")

			if (err != nil) != test.wantErr {
				t.Fatalf("runJavaMigration() error = %v, wantErr %v", err, test.wantErr)
			}

			librarianConfigFile := filepath.Join(tmpDir, "librarian.yaml")
			_, statErr := os.Stat(librarianConfigFile)

			if test.expectLibrarianFile {
				if os.IsNotExist(statErr) {
					t.Fatalf("expected librarian.yaml to be created, but it was not")
				}
				got, readErr := os.ReadFile(librarianConfigFile)
				if readErr != nil {
					t.Fatalf("failed to read librarian.yaml: %v", readErr)
				}
				// The marshaler adds a trailing newline.
				want := test.wantLibrarianYAML
				if len(want) > 0 && want[len(want)-1] != '\n' {
					want += "\n"
				}

				if diff := cmp.Diff(want, string(got)); diff != "" {
					t.Errorf(`mismatch (-want +got):
%s`, diff)
				}
			} else {
				if !os.IsNotExist(statErr) {
					t.Fatalf("expected librarian.yaml not to be created, but it was")
				}
			}
		})
	}
}
