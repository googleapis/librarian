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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
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

func TestRunInDir(t *testing.T) {
	dir := t.TempDir()
	if err := RunInDir(t.Context(), dir, "go", "mod", "init", "example.com/foo"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Errorf("go.mod was not created in the specified directory: %v", err)
	}
}

func TestRunWithEnv_SetsAndVerifiesVariable(t *testing.T) {
	ctx := t.Context()
	const (
		name  = "LIBRARIAN_TEST_VAR"
		value = "value"
	)
	err := RunWithEnv(ctx, map[string]string{name: value},
		"sh", "-c", fmt.Sprintf("test \"$%s\" = \"%s\"", name, value))
	if err != nil {
		t.Fatalf("RunWithEnv() = %v, want %v", err, nil)
	}
}

func TestRunWithEnv_VariableNotSetFailsValidation(t *testing.T) {
	ctx := t.Context()
	const (
		name  = "LIBRARIAN_TEST_VAR"
		value = "value"
	)
	err := RunWithEnv(ctx, map[string]string{}, "sh", "-c", fmt.Sprintf("test \"$%s\" = \"%s\"", name, value))
	if err == nil {
		t.Fatalf("RunWithEnv() = %v, want non-nil", err)
	}
}

func TestGetExecutablePath(t *testing.T) {
	tests := []struct {
		name           string
		releaseConfig  *config.Release
		executableName string
		want           string
	}{
		{
			name: "Preinstalled tool found",
			releaseConfig: &config.Release{
				Preinstalled: map[string]string{
					"cargo": "/usr/bin/cargo",
					"git":   "/usr/bin/git",
				},
			},
			executableName: "cargo",
			want:           "/usr/bin/cargo",
		},
		{
			name: "Preinstalled tool not found",
			releaseConfig: &config.Release{
				Preinstalled: map[string]string{
					"git": "/usr/bin/git",
				},
			},
			executableName: "cargo",
			want:           "cargo",
		},
		{
			name:           "No preinstalled section",
			releaseConfig:  &config.Release{},
			executableName: "cargo",
			want:           "cargo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := GetExecutablePath(test.releaseConfig.Preinstalled, test.executableName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestVerbose(t *testing.T) {
	t.Cleanup(func() {
		Verbose = false
	})

	for _, test := range []struct {
		name    string
		verbose bool
	}{
		{"verbose enabled", true},
		{"verbose disabled", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			Verbose = test.verbose

			got := captureStdout(t, func() {
				if err := Run(t.Context(), "go", "version"); err != nil {
					t.Fatal(err)
				}
			})

			if test.verbose {
				if !strings.Contains(got, "go version") {
					t.Errorf("expected stdout to contain command, got: %q", got)
				}
			} else {
				if got != "" {
					t.Errorf("expected empty stdout, got: %q", got)
				}
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = stdout
	})

	fn()
	w.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestSetExecCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     []string
		wantCommand []string
	}{
		{
			name:        "command with args",
			command:     []string{"my-command", "arg1", "arg2"},
			wantCommand: []string{"my-command", "arg1", "arg2"},
		},
		{
			name:        "command without args",
			command:     []string{"my-other-command"},
			wantCommand: []string{"my-other-command"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var executedCommand []string
			SetExecCommand(func(ctx context.Context, name string, arg ...string) *exec.Cmd {
				executedCommand = append([]string{name}, arg...)
				return exec.CommandContext(ctx, "true")
			})
			t.Cleanup(ResetExecCommand)

			if err := Run(t.Context(), test.command[0], test.command[1:]...); err != nil {
				t.Fatalf("Run() with mock command failed: %v", err)
			}

			if diff := cmp.Diff(test.wantCommand, executedCommand); diff != "" {
				t.Errorf("mock command executed with wrong arguments, diff (-want +got):\n%s", diff)
			}
		})
	}
}
