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

package librarian

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunEnv(t *testing.T) {
	cacheDir := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cacheDir)
	t.Setenv("LIBRARIAN_BIN", binDir)
	var buf bytes.Buffer
	if err := runEnv(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	wants := []string{
		fmt.Sprintf("LIBRARIAN_CACHE=%s", cacheDir),
		fmt.Sprintf("LIBRARIAN_BIN=%s", binDir),
		fmt.Sprintf("golang: %s", filepath.Join(binDir, "go_tools")),
		fmt.Sprintf("java: %s", filepath.Join(binDir, "java_tools")),
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("runEnv() output missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestRunEnv_Error(t *testing.T) {
	// Unset environment variables to force path resolution errors.
	t.Setenv("LIBRARIAN_CACHE", "")
	t.Setenv("LIBRARIAN_BIN", "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	var buf bytes.Buffer
	if err := runEnv(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	wants := []string{
		"LIBRARIAN_CACHE=<error:",
		"LIBRARIAN_BIN=<error:",
		"golang: <error:",
		"java: <error:",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("runEnv() output missing %q\ngot:\n%s", want, got)
		}
	}
}
