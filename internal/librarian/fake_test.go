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

package librarian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerate(t *testing.T) {
	const (
		libraryName = "test-library"
		outputDir   = "output"
	)
	library := &config.Library{
		Name:   libraryName,
		Output: outputDir,
	}

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	if err := generate(t.Context(), "fake", library, "", nil); err != nil {
		t.Fatal(err)
	}

	readmePath := filepath.Join(outputDir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	want := "# test-library\n\nGenerated library\n"
	if diff := cmp.Diff(want, string(content)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
