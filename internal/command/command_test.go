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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

const invalidSubcommand = "invalid-subcommand"

func TestRun(t *testing.T) {
	if err := Run(t.Context(), "go", "version"); err != nil {
		t.Fatal(err)
	}
}

func TestRunError(t *testing.T) {
	err := Run(t.Context(), "go", invalidSubcommand)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), invalidSubcommand) {
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

func TestOutput(t *testing.T) {
	got, err := Output(t.Context(), "go", "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "go version") {
		t.Errorf("expected output to contain %q, got: %q", "go version", got)
	}
}

func TestOutput_Error(t *testing.T) {
	_, err := Output(t.Context(), "go", invalidSubcommand)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Output() error = %v, want type *exec.ExitError", err)
	}
	if !strings.Contains(string(exitErr.Stderr), invalidSubcommand) {
		t.Errorf("stderr should mention the invalid subcommand; got %q", string(exitErr.Stderr))
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
			t.Cleanup(func() {
				Verbose = false
				stdout = os.Stdout
			})
			Verbose = test.verbose
			var outBuf bytes.Buffer
			stdout = &outBuf
			if err := Run(t.Context(), "go", "version"); err != nil {
				t.Fatal(err)
			}
			got := outBuf.String()

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

func TestRunStreaming(t *testing.T) {
	for _, test := range []struct {
		name    string
		command string
		args    []string
		verbose bool
		wantOut string
		wantErr string
	}{
		{
			name:    "simple output and err",
			command: "/bin/sh",
			args:    []string{"-c", "echo test-output && echo >&2 test-error"},
			wantOut: "test-output\n",
			wantErr: "test-error\n",
		},
		{
			name:    "verbose output",
			command: "/bin/sh",
			args:    []string{"-c", "echo test-output"},
			verbose: true,
			wantOut: "/bin/sh -c echo test-output\ntest-output\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(func() {
				Verbose = false
				stdout = os.Stdout
				stderr = os.Stderr
			})
			Verbose = test.verbose
			var outBuf, errBuf bytes.Buffer
			stdout = &outBuf
			stderr = &errBuf
			err := RunStreaming(t.Context(), test.command, test.args...)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantOut, outBuf.String()); diff != "" {
				t.Errorf("mismatch of stdout (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantErr, errBuf.String()); diff != "" {
				t.Errorf("mismatch of stderr (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunStreaming_Error(t *testing.T) {
	err := RunStreaming(t.Context(), "go", invalidSubcommand)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("RunWithStreamingOutput() error = %v, want type *exec.ExitError", err)
	}
	if !strings.Contains(string(err.Error()), invalidSubcommand) {
		t.Errorf("err.Error() should mention the invalid subcommand; got %q", err.Error())
	}
}
