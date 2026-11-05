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
	"log/slog"
	"maps"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/tool/protoc"
)

const (
	defaultRetryBackoff = 10 * time.Second
	envPath             = "PATH"
	maxRetryAttempts    = 3
)

// runProtoc runs the protoc command with the given environment variable and arguments.
func runProtoc(ctx context.Context, pc *config.Protoc, args ...string) error {
	// Ensure that the toolchain environment variables are set before calling protoc.
	env, err := toolsEnv()
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
	additionalEnv, err := toolsEnv()
	if err != nil {
		return err
	}
	mergedEnv := make(map[string]string)
	maps.Copy(mergedEnv, additionalEnv)
	maps.Copy(mergedEnv, env)
	return command.RunInDirWithEnv(ctx, dir, mergedEnv, cmd, args...)
}

// runInDirWithEnvRetry runs a command in the given directory with the given
// environment, retrying up to maxRetryAttempts times with exponential backoff
// on failure.
func runInDirWithEnvRetry(ctx context.Context, dir string, env map[string]string, cmd string, args ...string) error {
	return runInDirWithEnvAndBackoff(ctx, dir, env, defaultRetryBackoff, cmd, args...)
}

// runInDirWithEnvAndBackoff runs a command in the given directory with the
// given environment, retrying up to maxRetryAttempts times starting with
// initialBackoff on failure.
func runInDirWithEnvAndBackoff(ctx context.Context, dir string, env map[string]string, initialBackoff time.Duration, cmd string, args ...string) error {
	var err error
	backoff := initialBackoff
	for attempt := range maxRetryAttempts {
		if attempt > 0 {
			slog.Warn("retrying command after failure", "cmd", cmd, "args", args, "attempt", attempt+1, "max_attempts", maxRetryAttempts, "backoff", backoff, "error", err)
			if waitErr := backoffWait(ctx, backoff); waitErr != nil {
				return waitErr
			}
			backoff *= 2
		}
		err = runInDirWithEnv(ctx, dir, env, cmd, args...)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return fmt.Errorf("%s failed after %d attempts: %w", cmd, maxRetryAttempts, err)
}

// backoffWait waits for d or until ctx is canceled.
func backoffWait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// toolsEnv returns the environment variables required to run the Go tools.
func toolsEnv() (map[string]string, error) {
	toolsBinDir, err := InstallDir()
	if err != nil {
		return nil, err
	}
	return map[string]string{envPath: toolsBinDir}, nil
}
