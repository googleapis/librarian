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

package rust

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmdtest "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerate(t *testing.T) {
	cmdtest.RequireCommand(t, "protoc")
	testdataDir, err := filepath.Abs("../../../sidekick/testdata")
	if err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	googleapisDir := filepath.Join(testdataDir, "googleapis")
	library := &config.Library{
		Name:          "secretmanager",
		Output:        outDir,
		ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		Channel:       "google/cloud/secretmanager/v1",
		Version:       "0.1.0",
		ReleaseLevel:  "preview",
		CopyrightYear: "2025",
	}
	if err := Generate(t.Context(), library, googleapisDir); err != nil {
		t.Fatal(err)
	}

	// Verify generated files and their content.
	filesToValidate := []struct {
		path    string
		content string
	}{
		{filepath.Join(outDir, "Cargo.toml"), "name"},
		{filepath.Join(outDir, "Cargo.toml"), "secretmanager"},
		{filepath.Join(outDir, "README.md"), "# Google Cloud Client Libraries for Rust - Secret Manager API"},
		{filepath.Join(outDir, "src", "lib.rs"), "pub mod model;"},
		{filepath.Join(outDir, "src", "lib.rs"), "pub mod client;"},
	}

	for _, tt := range filesToValidate {
		info, err := os.Stat(tt.path)
		if err != nil {
			t.Fatalf("Failed to stat %q: %v", tt.path, err)
		}
		if info.IsDir() {
			t.Fatalf("%q is a directory, expected a file", tt.path)
		}

		got, err := os.ReadFile(tt.path)
		if err != nil {
			t.Fatalf("Failed to read %q: %v", tt.path, err)
		}
		if !strings.Contains(string(got), tt.content) {
			t.Errorf("File %q content missing expected string. Got: %q, Want substring: %q", tt.path, string(got), tt.content)
		}
	}
}
