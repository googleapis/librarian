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
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestUpdateSidekickConfigSuccess(t *testing.T) {
	const contentsA = `# With version line
[codec]
a = 123
version = "1.2.3"
copyright-year = '2038'
`
	const contentsB = `# With version line
[codec]
a = 123
version        = "1.2.3"
copyright-year = '2038'
`

	for _, contents := range []string{contentsA, contentsB} {
		tmpDir := t.TempDir()
		manifest := path.Join(tmpDir, "Cargo.toml")
		sidekick := path.Join(tmpDir, ".sidekick.toml")
		const want = `# With version line
[codec]
a = 123
version        = '2.3.4'
copyright-year = '2038'
`
		if err := os.WriteFile(sidekick, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
		if err := updateSidekickConfig(manifest, "2.3.4"); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(sidekick)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(want, string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestUpdateSidekickConfigSuccessNoVersion(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	sidekick := path.Join(tmpDir, ".sidekick.toml")
	const contents = `# With version line
[codec]
copyright-year = '2038'
c = 345
`
	const want = `# With version line
[codec]
version        = '2.3.4'
copyright-year = '2038'
c = 345
`
	if err := os.WriteFile(sidekick, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := updateSidekickConfig(manifest, "2.3.4"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(sidekick)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateSidekickConfigSuccessEmptyCodec(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	sidekick := path.Join(tmpDir, ".sidekick.toml")
	const contents = `# With version line

[codec]
`
	const want = `# With version line

[codec]
version        = '2.3.4'
`
	if err := os.WriteFile(sidekick, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := updateSidekickConfig(manifest, "2.3.4"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(sidekick)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateSidekickConfigSuccessOnlyVersion(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	sidekick := path.Join(tmpDir, ".sidekick.toml")
	const contents = `# With version line

[codec]
version        = '1.2.3'
`
	const want = `# With version line

[codec]
version        = '2.3.4'
`
	if err := os.WriteFile(sidekick, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := updateSidekickConfig(manifest, "2.3.4"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(sidekick)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateSidekickConfigNoErrorOnMissingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := updateSidekickConfig(manifest, "2.3.4"); err != nil {
		t.Errorf("no errors expected when sidekick config does not exist, got=%v", err)
	}
}

func TestUpdateSidekickConfigStatError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows, file permissions are not the same as on Unix")
	}
	tmpDir := path.Join(t.TempDir(), "not-readable")
	_ = os.MkdirAll(tmpDir, 0000)
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := updateSidekickConfig(manifest, "2.3.4"); err == nil {
		t.Errorf("expected an error with non-readable file")
	}
}

func TestUpdateSidekickConfigReadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	sidekick := path.Join(tmpDir, ".sidekick.toml")
	if err := os.WriteFile(sidekick, []byte{}, 0000); err != nil {
		t.Fatal(err)
	}
	if err := updateSidekickConfig(manifest, "2.3.4"); err == nil {
		t.Errorf("expected an error with non-readable file")
	}
}

func TestUpdateSidekickConfigNoCodec(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	sidekick := path.Join(tmpDir, ".sidekick.toml")
	if err := os.WriteFile(sidekick, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := updateSidekickConfig(manifest, "2.3.4"); err == nil {
		t.Errorf("expected an error with missing [codec] line")
	}
}
