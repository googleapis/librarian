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

package maven

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePOM(t *testing.T) {
	tmpDir := t.TempDir()
	mockPOM := `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <parent>
    <groupId>com.google.api.generator</groupId>
    <version>2.28.0-SNAPSHOT</version>
  </parent>
  <artifactId>gapic-generator-java</artifactId>
</project>`
	pomPath := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(mockPOM), 0644); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name          string
		pomPath       string
		wantArtifact  string
		wantVersion   string
		wantErrSubstr string
	}{
		{
			name:         "parse POM successfully",
			pomPath:      pomPath,
			wantArtifact: "gapic-generator-java",
			wantVersion:  "2.28.0-SNAPSHOT",
		},
		{
			name:          "missing file error",
			pomPath:       filepath.Join(tmpDir, "nonexistent.xml"),
			wantErrSubstr: "failed to read pom.xml",
		},
		{
			name:          "invalid XML syntax",
			pomPath:       filepath.Join(tmpDir, "invalid.xml"),
			wantErrSubstr: "failed to parse pom.xml",
		},
		{
			name:          "missing artifactId or version",
			pomPath:       filepath.Join(tmpDir, "empty.xml"),
			wantErrSubstr: "missing artifactId or version",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "invalid XML syntax" {
				if err := os.WriteFile(test.pomPath, []byte("<project><invalid"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if test.name == "missing artifactId or version" {
				if err := os.WriteFile(test.pomPath, []byte("<project></project>"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			gotArtifact, gotVersion, err := ParsePOM(test.pomPath)
			if test.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("ParsePOM() error = %v, wantErrSubstr = %q", err, test.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotArtifact != test.wantArtifact {
				t.Errorf("ParsePOM() gotArtifact = %q, wantArtifact = %q", gotArtifact, test.wantArtifact)
			}
			if gotVersion != test.wantVersion {
				t.Errorf("ParsePOM() gotVersion = %q, wantVersion = %q", gotVersion, test.wantVersion)
			}
		})
	}
}
