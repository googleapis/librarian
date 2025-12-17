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

package command

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	if err := Run(t.Context(), "go", "version"); err != nil {
		t.Fatal(err)
	}
}

func TestRunError(t *testing.T) {
	err := Run(t.Context(), "go", "invalid-subcommand-bad-bad-bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid-subcommand-bad-bad-bad") {
		t.Errorf("error should mention the invalid subcommand, got: %v", err)
	}
}

func TestRunWithEnvSuccess(t *testing.T) {
	ctx := t.Context()
	const (
		name  = "LIBRARIAN_TEST_VAR"
		value = "value"
	)
	err := RunWithEnv(ctx, []string{name + "=" + value}, "sh", "-c", `test "$`+name+`" = "`+value+`"`)
	if err != nil {
		t.Fatalf("RunWithEnv() = %v, want %v", err, nil)
	}
}

func TestRunWithEnvFailure(t *testing.T) {
	ctx := t.Context()
	const (
		name  = "LIBRARIAN_TEST_VAR"
		value = "value"
	)
	err := RunWithEnv(ctx, []string{}, "sh", "-c", `test "$`+name+`" = "`+value+`"`)
	if err == nil {
		t.Fatalf("RunWithEnv() = %v, want non-nil", err)
	}
}
