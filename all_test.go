// Copyright 2024 Google LLC
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
	"os/exec"
	"strings"
	"testing"
)

func TestGolangCILint(t *testing.T) {
	rungo(t, "tool", "golangci-lint", "run")
}

func TestGoImports(t *testing.T) {
	// goimports -d exits 0 even when it finds unformatted files, so check
	// stdout instead. Only stdout is checked because stderr may contain
	// download progress messages. See
	// https://github.com/googleapis/librarian/pull/2915.
	if out := rungo(t, "tool", "goimports", "-d", "."); out != "" {
		t.Fatalf("goimports -d .:\n%s", out)
	}
}

func TestGoModTidy(t *testing.T) {
	rungo(t, "mod", "tidy", "-diff")
}

func TestYAMLFormat(t *testing.T) {
	rungo(t, "tool", "yamlfmt", "-lint", ".")
}

func rungo(t *testing.T, args ...string) string {
	t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go %s: %v\nStdout:\n%s\nStderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}
