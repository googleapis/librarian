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

package pom

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateVersion(t *testing.T) {
	tests := []struct {
		name        string
		initial     string
		libraryID   string
		version     string
		expected    string
		expectError bool
	}{
		{
			name: "happy path",
			initial: `<project>
  <version>1.0.0</version><!-- {x-version-update:google-cloud-java:current} -->
</project>`,
			libraryID: "google-cloud-java",
			version:   "2.0.0",
			expected: `<project>
  <version>2.0.0</version><!-- {x-version-update:google-cloud-java:current} -->
</project>`,
		},
		{
			name: "no match",
			initial: `<project>
  <version>1.0.0</version><!-- {x-version-update:google-cloud-java:current} -->
</project>`,
			libraryID: "wrong-library-id",
			version:   "2.0.0",
			expected: `<project>
  <version>1.0.0</version><!-- {x-version-update:google-cloud-java:current} -->
</project>`,
		},
		{
			name: "multiple versions",
			initial: `<project>
  <version>1.0.0</version><!-- {x-version-update:google-cloud-java:current} -->
  <dependency>
    <groupId>com.google.cloud</groupId>
    <artifactId>google-cloud-secretmanager</artifactId>
    <version>1.2.3</version><!-- {x-version-update:google-cloud-secretmanager:current} -->
  </dependency>
</project>`,
			libraryID: "google-cloud-secretmanager",
			version:   "2.0.0",
			expected: `<project>
  <version>1.0.0</version><!-- {x-version-update:google-cloud-java:current} -->
  <dependency>
    <groupId>com.google.cloud</groupId>
    <artifactId>google-cloud-secretmanager</artifactId>
    <version>2.0.0</version><!-- {x-version-update:google-cloud-secretmanager:current} -->
  </dependency>
</project>`,
		},
		{
			name: "no comment",
			initial: `<project>
  <version>1.0.0</version>
</project>`,
			libraryID: "google-cloud-java",
			version:   "2.0.0",
			expected: `<project>
  <version>1.0.0</version>
</project>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pomPath := filepath.Join(tmpDir, "pom.xml")
			outPath := filepath.Join(tmpDir, "out", "pom.xml")
			if err := os.WriteFile(pomPath, []byte(test.initial), 0644); err != nil {
				t.Fatalf("failed to write initial pom.xml: %v", err)
			}

			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				t.Fatalf("failed to create output directory: %v", err)
			}
			err := updateVersion(pomPath, outPath, test.libraryID, test.version)

			if test.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				content, readErr := os.ReadFile(outPath)
				if readErr != nil {
					t.Fatalf("failed to read pom.xml: %v", readErr)
				}
				if string(content) != test.expected {
					t.Errorf("expected:\n%s\ngot:\n%s", test.expected, string(content))
				}
			}
		})
	}
}
