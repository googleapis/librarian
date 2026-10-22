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

package ruby

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	installDir := filepath.Join(binDir, "ruby_tools")
	configPath := filepath.Join(installDir, "gems", "gapic-generator-cloud-0.52.0", "cloud-rubocop.yml")
	for _, dir := range []string{filepath.Dir(configPath), filepath.Join(installDir, "bin")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(configPath, []byte("AllCops:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.WriteExecutable(t, filepath.Join(installDir, "bin", "rubocop"), `#!/bin/sh
while [ $# -gt 0 ]; do
  case "$1" in
    --cache|--fail-level|-c) shift 2 ;;
    -*) shift ;;
    *) echo "# formatted" >> "$1"; shift ;;
  esac
done
`)
	outDir := t.TempDir()
	want := map[string]string{
		"lib/foo.rb":       "x = 1\n# formatted\n",
		"lib/foo_pb.rb":    "x = 1\n",
		"lib/kept.rb":      "x = 1\n",
		"test/foo_test.rb": "x = 1\n",
	}
	for relPath := range want {
		full := filepath.Join(outDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Format(t.Context(), &config.Library{Name: "foo", Output: outDir, Keep: []string{"lib/kept.rb"}}); err != nil {
		t.Fatal(err)
	}
	got := make(map[string]string)
	for relPath := range want {
		content, err := os.ReadFile(filepath.Join(outDir, relPath))
		if err != nil {
			t.Fatal(err)
		}
		got[relPath] = string(content)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormat_Error(t *testing.T) {
	t.Setenv("LIBRARIAN_BIN", t.TempDir())
	outDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outDir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "lib", "foo.rb"), []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gotErr := Format(t.Context(), &config.Library{Name: "foo", Output: outDir})
	if !errors.Is(gotErr, errRubocopConfigNotFound) {
		t.Errorf("Format() error = %v, wantErr %v", gotErr, errRubocopConfigNotFound)
	}
}
