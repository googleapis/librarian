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
	"fmt"
	"maps"
	"os"

	"github.com/googleapis/librarian/internal/command"
)

const envPath = "PATH"

// runWithEnv runs a command with the given environment.
func runWithEnv(ctx context.Context, env map[string]string, cmd string, args ...string) error {
	return runInDirWithEnv(ctx, "", env, cmd, args...)
}

// runInDirWithEnv runs a command in the given directory with the given environment.
func runInDirWithEnv(ctx context.Context, dir string, env map[string]string, cmd string, args ...string) error {
	env, err := mergeEnv(env)
	if err != nil {
		return err
	}
	return command.RunInDirWithEnv(ctx, dir, env, cmd, args...)
}

// mergeEnv merges the given environment with the installation directory.
func mergeEnv(env map[string]string) (map[string]string, error) {
	toolsBinDir, err := InstallDir()
	if err != nil {
		return nil, err
	}
	res := map[string]string{envPath: fmt.Sprintf("%s:%s", toolsBinDir, os.Getenv(envPath))}
	maps.Copy(res, env)
	return res, nil
}
