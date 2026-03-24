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
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// Verbose controls whether commands are printed to stderr before execution.
//
// TODO(https://github.com/googleapis/librarian/issues/3687): pass in as
// config.
var Verbose bool

// Run executes a program (with arguments). On error, stderr is included in the
// error message. It is a convenience wrapper around RunWithEnv.
func Run(ctx context.Context, command string, arg ...string) error {
	return RunWithEnv(ctx, nil, command, arg...)
}

// RunInDir executes a program in a specific directory.
func RunInDir(ctx context.Context, dir, command string, arg ...string) error {
	_, err := runCmd(ctx, dir, nil, command, arg...)
	return err
}

// RunWithEnv executes a program (with arguments) and optional environment
// variables and captures any error output. If env is nil or empty, the command
// inherits the environment of the calling process.
func RunWithEnv(ctx context.Context, env map[string]string, command string, arg ...string) error {
	_, err := runCmd(ctx, "", env, command, arg...)
	return err
}

// RunStreaming runs the given binary with the specified args,
// setting its output and errors streams to those of the current process. The
// output is not otherwise captured. This is primarily for use in tools which
// call potentially long-running commands. It should not be used within
// librarian itself, where the Run and Output functions are generally preferred,
// as those observe the Verbose flag for output. The Verbose flag only affects
// the behavior of this function in terms of whether the command being executed
// is first written to stdout.
func RunStreaming(ctx context.Context, command string, arg ...string) error {
	cmd := buildCmd(ctx, "", nil, command, arg...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	return nil
}

// Output executes a program (with arguments) and returns stdout. It is a
// convenience wrapper around OutputWithEnv.
func Output(ctx context.Context, command string, arg ...string) (string, error) {
	return OutputWithEnv(ctx, nil, command, arg...)
}

// OutputWithEnv executes a program (with arguments) and optional environment
// variables and returns stdout. If env is nil or empty, the command inherits
// the environment of the calling process. On error, stderr is included in the
// error message.
func OutputWithEnv(ctx context.Context, env map[string]string, command string, arg ...string) (string, error) {
	return runCmd(ctx, "", env, command, arg...)
}

func buildCmd(ctx context.Context, dir string, env map[string]string, command string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, command, arg...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	if Verbose {
		fmt.Fprintf(os.Stdout, "%s\n", cmd.String())
	}
	return cmd
}

func runCmd(ctx context.Context, dir string, env map[string]string, command string, arg ...string) (string, error) {
	cmd := buildCmd(ctx, dir, env, command, arg...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("%s: %s: %w", cmd, exitErr.Stderr, err)
		}
		return "", fmt.Errorf("%s: %w", cmd, err)
	}
	return string(output), nil
}

// GetExecutablePath finds the path for a given command, checking for an
// override in the provided commandOverrides map first.
func GetExecutablePath(commandOverrides map[string]string, commandName string) string {
	if exe, ok := commandOverrides[commandName]; ok {
		return exe
	}
	return commandName
}
