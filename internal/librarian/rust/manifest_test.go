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
	"testing"

	"github.com/googleapis/librarian/internal/testhelper"
)

func TestUpdateCargoVersionSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	testhelper.AddCrate(t, tmpDir, "google-cloud-storage")
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := UpdateCargoVersion(manifest, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if idx := bytes.Index(contents, []byte(`version                = "1.2.3"`)); idx == -1 {
		t.Errorf("version 1.2.3 not found in %s", contents)
	}
}

func TestUpdateCargoVersionMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := UpdateCargoVersion(manifest, "1.2.3"); err == nil {
		t.Error("expected an error, got none")
	}
}

func TestUpdateCargoVersionMissingVersion(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := os.WriteFile(manifest, []byte("[package]\nname=\"foo\""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateCargoVersion(manifest, "1.2.3"); err == nil {
		t.Error("expected an error, got none")
	}
}