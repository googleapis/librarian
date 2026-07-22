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

package java

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	if err := os.WriteFile(pomPath, []byte(mockPOM), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := parsePOM(pomPath)
	if err != nil {
		t.Fatal(err)
	}
	want := &pomProject{
		ArtifactID: "gapic-generator-java",
		Version:    "2.28.0-SNAPSHOT",
	}
	if diff := cmp.Diff(want.ArtifactID, got.ArtifactID); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.Version, got.Version); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParsePOM_Error(t *testing.T) {
	tmpDir := t.TempDir()
	for _, test := range []struct {
		name    string
		pomPath string
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name:    "missing file error",
			pomPath: filepath.Join(tmpDir, "nonexistent.xml"),
			wantErr: errReadPOM,
		},
		{
			name:    "invalid XML syntax",
			pomPath: filepath.Join(tmpDir, "invalid.xml"),
			setup: func(t *testing.T) {
				if err := os.WriteFile(filepath.Join(tmpDir, "invalid.xml"), []byte("<project><invalid"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errParsePOM,
		},
		{
			name:    "missing artifactId or version",
			pomPath: filepath.Join(tmpDir, "empty.xml"),
			setup: func(t *testing.T) {
				if err := os.WriteFile(filepath.Join(tmpDir, "empty.xml"), []byte("<project></project>"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errInvalidPOM,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			_, err := parsePOM(test.pomPath)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("parsePOM() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}
