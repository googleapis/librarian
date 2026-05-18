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

package testhelper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTempDir(t *testing.T) {
	dir := tempDir(t)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("tempDir was not created: %v", err)
	}
	tempBase := os.TempDir()
	rel, err := filepath.Rel(tempBase, dir)
	if err != nil || filepath.IsAbs(rel) {
		t.Fatalf("tempDir %s was not created inside TempDir %s", dir, tempBase)
	}
}

func TestRequireCommand(t *testing.T) {
	RequireCommand(t, "git")
}
