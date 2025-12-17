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

// Package command provides helpers to execute external commands with logging.
package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Run executes a program (with arguments) and captures any error output. It is a
// convenience wrapper around RunWithEnv.
func Run(ctx context.Context, command string, arg ...string) error {
	return RunWithEnv(ctx, nil, command, arg...)
}

// RunWithEnv executes a program (with arguments) and optional environment
// variables and captures any error output. If env is nil or empty, the command
// inherits the environment of the calling process.
func RunWithEnv(ctx context.Context, env []string, command string, arg ...string) error {
	cmd := exec.CommandContext(ctx, command, arg...)
	if len(env) > 0 {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, env...)
	}
	fmt.Fprintf(os.Stderr, "Running: %s\n", cmd.String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %v\n%s", cmd, err, output)
	}
	return nil
}
