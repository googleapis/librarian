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
)

func TestGetPackageName(t *testing.T) {
	expectedPackageName := "new-lib-format"
	got, err := getPackageName("testdata/new-lib-format")
	if err != nil {
		t.Fatalf("error getting package name %v", err)
	}
	if got != expectedPackageName {
		t.Errorf("want packageName %s, got %s", expectedPackageName, got)
	}
}

func TestPrepareCargoWorkspace(t *testing.T) {
	cmdtest.RequireCommand(t, "cargo")
	cmdtest.RequireCommand(t, "taplo")
	libName := "new-lib"
	testdataDir, err := filepath.Abs("./testdata")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(testdataDir)
	outputDir := path.Join(testdataDir, libName)
	if err := PrepareCargoWorkspace(t.Context(), outputDir); err != nil {
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
	expectedCargoContent := "name = \"new-lib\""
	if !strings.Contains(string(got), expectedCargoContent) {
		t.Errorf("%q missing expected string: %q", got, expectedCargoContent)
	}
	os.RemoveAll(outputDir)
	cmdtest.Run(t.Context(), "git", "restore", "--source=HEAD", "--staged", "--worktree", ".")
}

func TestFormatAndValidateCreatedLibrary(t *testing.T) {
	cmdtest.RequireCommand(t, "cargo")
	cmdtest.RequireCommand(t, "env")
	cmdtest.RequireCommand(t, "git")
	testdataDir, err := filepath.Abs("./testdata")
	libName := "new-lib-format"
	t.Chdir(testdataDir)
	fileToFormat := path.Join(testdataDir, libName, "src", "main.rs")
	if err := FormatAndValidateLibrary(t.Context(), path.Join(testdataDir, libName)); err != nil {
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
