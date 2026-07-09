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

package php

import (
	"path/filepath"
	"testing"
)

func TestInstallDir(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	got, err := InstallDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(binDir, "php_tools")
	if got != want {
		t.Errorf("InstallDir() = %q, want %q", got, want)
	}
}

func TestBinDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", dir)
	got, err := binDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "php_tools", "bin")
	if got != want {
		t.Errorf("binDir() = %q, want %q", got, want)
	}
}
