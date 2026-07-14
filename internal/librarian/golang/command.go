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

package golang

import (
	"context"
	"maps"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/tool/protoc"
)

const envPath = "PATH"

// runProtoc runs the protoc command with the given environment variable and arguments.
func runProtoc(ctx context.Context, pc *config.Protoc, args ...string) error {
	// Ensure that the toolchain environment variables are set before calling protoc.
	env, err := installEnv()
	if err != nil {
		return err
	}
	return protoc.RunOrSystem(ctx, env, pc, args...)
}

// runWithEnv runs a command with the given environment.
func runWithEnv(ctx context.Context, env map[string]string, cmd string, args ...string) error {
	return runInDirWithEnv(ctx, "", env, cmd, args...)
}

// runInDirWithEnv runs a command in the given directory with the given environment.
func runInDirWithEnv(ctx context.Context, dir string, env map[string]string, cmd string, args ...string) error {
	additionalEnv, err := installEnv()
	if err != nil {
		return err
	}
	maps.Copy(env, additionalEnv)
	return command.RunInDirWithEnv(ctx, dir, env, cmd, args...)
}

// installEnv returns the environment variables required to run the Go tools.
func installEnv() (map[string]string, error) {
	toolsBinDir, err := InstallDir()
	if err != nil {
		return nil, err
	}
	return map[string]string{envPath: toolsBinDir}, nil
}
