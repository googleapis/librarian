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
	"errors"
	"os/exec"
	"testing"
)

func TestGolangCILint(t *testing.T) {
	rungo(t, "tool", "golangci-lint", "run")
}

func TestGoImports(t *testing.T) {
	cmd := exec.Command("go", "tool", "goimports", "-d", ".")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("goimports failed to run: %v\nStdout:\n%s\nStderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stdout.Len() > 0 {
		t.Errorf("goimports found unformatted files:\n%s", stdout.String())
	}
}

func TestGoModTidy(t *testing.T) {
	rungo(t, "mod", "tidy", "-diff")
}

func TestYAMLFormat(t *testing.T) {
	cmd := exec.Command("go", "tool", "yamlfmt", "-lint", ".")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("yamlfmt failed to run: %v\nStdout:\n%s\nStderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stdout.Len() > 0 {
		t.Errorf("yamlfmt found unformatted files:\n%s", stdout.String())
	}
}

func rungo(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("go", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		if ee := (*exec.ExitError)(nil); errors.As(err, &ee) && len(ee.Stderr) > 0 {
			t.Fatalf("%v: %v\n%s", cmd, err, ee.Stderr)
		}
		t.Fatalf("%v: %v\n%s", cmd, err, output)
	}
}
