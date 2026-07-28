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
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/nodejs"
)

func TestFormat(t *testing.T) {
	nodeInstallDir, err := nodejs.InstallDir()
	if err != nil {
		t.Skipf("nodejs InstallDir failed: %v", err)
	}
	prettierPath := filepath.Join(nodeInstallDir, "bin", "prettier")
	if _, err := os.Stat(prettierPath); err != nil {
		t.Skipf("prettier not found at %s: %v", prettierPath, err)
	}

	if !phpSupported(t.Context(), prettierPath) {
		t.Skip("prettier does not support PHP (missing plugin?)")
	}

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
		t.Fatalf("Format() failed: %v", err)
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

func phpSupported(ctx context.Context, prettierPath string) bool {
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, prettierPath, "--support-info")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.Contains(stdout.String(), `"PHP"`)
}
