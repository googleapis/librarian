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

package swift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateVersion(t *testing.T) {
	const expectedFile = "Package+Version.swift"
	library := &config.Library{
		CopyrightYear: "2038",
		Version:       "1.2.3-test",
	}
	outDir := t.TempDir()
	if err := GenerateVersion(t.Context(), outDir, library); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(outDir, expectedFile)
	contents, err := os.ReadFile(filepath.Join(outDir, expectedFile))
	if err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(filename)
	if stat.Mode().Perm()|0666 != 0666 {
		t.Errorf("generated files should just be read-write %s: %o", filename, stat.Mode())
	}

	contentStr := string(contents)
	for _, test := range []struct {
		start string
		want  string
	}{
		{"// Copyright ", "// Copyright 2038 Google LLC\n"},
		{"static let version:", "static let version: String = \"1.2.3-test\"\n"},
	} {
		t.Run(test.start, func(t *testing.T) {
			got := extractBlock(t, contentStr, test.start, "\n")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
