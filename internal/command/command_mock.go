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

package command

import (
	"context"
	"os/exec"
	"strings"
	"sync"
)

// MockCommander tracks executed commands and provisions simulated errors.
// It is fully safe for parallel test execution.
type MockCommander struct {
	mu          sync.Mutex // Protects GotCommands during concurrent test steps
	GotCommands [][]string
	MockErrors  map[string]error // Use for specific command errors
	DefaultErr  error            // Use if you want all commands to fail with this error
}

type contextKey struct{}

var (
	installOnce sync.Once
	realExec    = exec.CommandContext // Keep a reference to the real execution function
)

// InjectContext attaches the MockCommander to the context.
// It ensures a global context-aware router is installed exactly once,
// safely allowing t.Parallel() tests to run simultaneously without cross-talk.
func (m *MockCommander) InjectContext(ctx context.Context) context.Context {
	installOnce.Do(func() {
		// Replace the package-level execCommand with a router.
		// sync.Once makes this completely thread-safe, no mutex needed!
		execCommand = func(execCtx context.Context, name string, arg ...string) *exec.Cmd {
			// If the context contains a mock instance, route to it.
			if mocker, ok := execCtx.Value(contextKey{}).(*MockCommander); ok {
				return mocker.executeMock(execCtx, name, arg...)
			}
			// Otherwise, fall back to real execution.
			return realExec(execCtx, name, arg...)
		}
	})

	return context.WithValue(ctx, contextKey{}, m)
}

// executeMock contains the isolated logic for a specific test's MockCommander instance.
func (m *MockCommander) executeMock(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := append([]string{name}, arg...)

	m.mu.Lock()
	m.GotCommands = append(m.GotCommands, cmd)
	m.mu.Unlock()

	key := strings.Join(cmd, " ")

	// Check for a specific error in the map first, fallback to DefaultErr
	var err error
	if specificErr, ok := m.MockErrors[key]; ok {
		err = specificErr
	} else if m.DefaultErr != nil {
		err = m.DefaultErr
	}

	// Return the dummy command
	if err != nil {
		return exec.CommandContext(ctx, "sh", "-c", "printf '%s\n' \"$1\" >&2; exit 1", "sh", err.Error())
	}
	return exec.CommandContext(ctx, "true")
}
