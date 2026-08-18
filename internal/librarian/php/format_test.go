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
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFormat(t *testing.T) {
	requirePrettier(t)
	tmpDir := t.TempDir()
	library := &config.Library{
		Name:   "test-library",
		Output: tmpDir,
	}
	// Write an unformatted PHP file.
	unformatted := `<?php
class Foo {
    public function bar( $a, $b ) {
        return "hello" ;
    }
}
`
	// Prettier PHP formatting should standardize spaces and convert double quotes to single quotes.
	want := `<?php
class Foo
{
    public function bar($a, $b)
    {
        return 'hello';
    }
}
`
	targetFile := filepath.Join(tmpDir, "Client", "Foo.php")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetFile, []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}
	gotBytes, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormat_NoMatchingFiles(t *testing.T) {
	requirePrettier(t)
	tmpDir := t.TempDir()
	library := &config.Library{
		Name:   "test-library",
		Output: tmpDir,
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}
}

func TestFormat_ErrorPrettierMissing(t *testing.T) {
	t.Setenv("LIBRARIAN_BIN", t.TempDir())
	library := &config.Library{
		Name:   "test-library",
		Output: t.TempDir(),
	}
	err := Format(t.Context(), library)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Format() error = %v, want wrap of %v", err, fs.ErrNotExist)
	}
}
func TestPrettierEnv(t *testing.T) {
	got := prettierEnv("/path/to/bin/prettier")
	want := map[string]string{
		"PATH": "/path/to/bin",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// phpSupported checks if prettier supports PHP by attempting to format a dummy snippet.
// We must test actual formatting with the --plugin flag because:
// 1. pnpm's symlinked node_modules structure prevents automatic plugin discovery.
// 2. Prettier 2's --support-info command ignores the --plugin flag, so we cannot query it directly.
func phpSupported(ctx context.Context, prettierPath, pluginPath string) bool {
	cmd := exec.CommandContext(ctx, prettierPath, "--plugin="+pluginPath, "--parser=php")
	cmd.Stdin = strings.NewReader("<?php class Foo {}")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

// requirePrettier skips the test if prettier or the PHP plugin is not available.
func requirePrettier(t *testing.T) {
	t.Helper()
	// TODO(https://github.com/googleapis/librarian/issues/7118):
	// Use testhelper.RequireCommand once it supports cached tools.
	prettierPath, pluginPath, err := prettierToolPaths()
	if err != nil {
		t.Skipf("prettier tools not available: %v", err)
	}
	if !phpSupported(t.Context(), prettierPath, pluginPath) {
		t.Skip("prettier does not support PHP (missing plugin?)")
	}
}
