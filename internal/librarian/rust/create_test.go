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

package rust

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	cmdtest "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPrepareCargoWorkspace(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")
	testhelper.RequireCommand(t, "taplo")

	testdataDir, err := filepath.Abs("./testdata")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(testdataDir)
	outputDir := t.TempDir()

	if err := prepareCargoWorkspace(t.Context(), outputDir); err != nil {
		t.Fatal(err)
	}
	expectedFile := path.Join(outputDir, "Cargo.toml")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatal(err)
	}
	expectedCargoContent := "name = \"google-cloud-secretmanager-v1\""
	if !strings.Contains(string(got), expectedCargoContent) {
		t.Errorf("%q missing expected string: %q", got, expectedCargoContent)
	}
	os.RemoveAll(outputDir)
	cmdtest.Run(t.Context(), "git", "restore", "--source=HEAD", "--staged", "--worktree", ".")
}

func TestFormatAndValidateCreatedLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")
	testhelper.RequireCommand(t, "env")
	testhelper.RequireCommand(t, "git")
	testdataDir, err := filepath.Abs("./testdata")
	libName := "google-cloud-secretmanager-v1"
	t.Chdir(testdataDir)
	fileToFormat := path.Join(testdataDir, libName, "src", "main.rs")
	if err := formatAndValidateLibrary(t.Context(), path.Join(testdataDir, libName)); err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fileToFormat)
	if err != nil {
		t.Fatal(err)
	}
	lineCount := bytes.Count(data, []byte("\n"))
	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		lineCount++
	}
	if lineCount != 6 {
		t.Errorf("formatting should have given us 6 lines but got: %d", lineCount)
	}
	cmdtest.Run(t.Context(), "git", "restore", "--source=HEAD", "--staged", "--worktree", ".")
	cmdtest.Run(t.Context(), "git", "clean", "-fd", ".")
}
