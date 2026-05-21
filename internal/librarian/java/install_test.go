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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInstall(t *testing.T) {
	tmpDir := t.TempDir()
	stubLogPath := filepath.Join(tmpDir, "pip_invocations.log")
	stubContent := fmt.Sprintf(`#!/bin/bash
echo "pip $@" >> %q
`, stubLogPath)
	stubDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}
	stubPath := filepath.Join(stubDir, "pip")
	if err := os.WriteFile(stubPath, []byte(stubContent), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)
	err := Install(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(stubLogPath)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	want := "pip install PyYAML==6.0.2 jinja2==3.1.6"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
