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

package rustrelease

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPublishedCrateSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	addCrate(t, tmpDir, "google-cloud-storage")
	manifest := path.Join(tmpDir, "Cargo.toml")
	got, err := publishedCrate(manifest)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"google-cloud-storage"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPublishedCrateReadError(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	if got, err := publishedCrate(manifest); err == nil {
		t.Errorf("expected error on missing manifest, got=%v", got)
	}
}

func TestPublishedCrateUnmarshalError(t *testing.T) {
	tmpDir := t.TempDir()
	addCrate(t, tmpDir, "google-cloud-storage")
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := os.WriteFile(manifest, []byte("invalid-toml={\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got, err := publishedCrate(manifest); err == nil {
		t.Errorf("expected error on unmarshaling error, got=%v", got)
	}
}

func TestPublishedCrateNotForPublication(t *testing.T) {
	tmpDir := t.TempDir()
	addCrate(t, tmpDir, "google-cloud-storage")
	manifest := path.Join(tmpDir, "Cargo.toml")
	contents := fmt.Sprintf(initialCargoContents, "google-cloud-storage")
	contents = contents + "\npublish = false\n"
	if err := os.WriteFile(manifest, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := publishedCrate(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
