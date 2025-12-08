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

package python

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name     string
		args     []string
		wantErr  bool
		wantExit int
	}{
		{
			name:    "valid command",
			args:    []string{"echo", "hello"},
			wantErr: false,
		},
		{
			name:    "invalid command",
			args:    []string{"command-that-does-not-exist"},
			wantErr: true,
		},
		{
			name:     "command with non-zero exit",
			args:     []string{"sh", "-c", "exit 1"},
			wantErr:  true,
			wantExit: 1,
		},
		{
			name:     "command with stderr output",
			args:     []string{"sh", "-c", "echo 'test error' >&2; exit 1"},
			wantErr:  true,
			wantExit: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := run(context.Background(), test.args, ".")
			if (err != nil) != test.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, test.wantErr)
			}

			if !test.wantErr {
				return
			}

			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				if test.wantExit != 0 && exitErr.ExitCode() != test.wantExit {
					t.Errorf("Run() exit code = %d, want %d", exitErr.ExitCode(), test.wantExit)
				}
			}
		})
	}
}
