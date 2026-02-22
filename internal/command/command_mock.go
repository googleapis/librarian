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
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
)

// MockResult defines the exact output and exit status of a mocked command.
type MockResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error // Convenience: If set, overrides Stderr and sets ExitCode to 1
}

// MockCommander tracks executed commands and provisions simulated outputs.
// It is fully safe for parallel test execution and cross-platform runs.
type MockCommander struct {
	mu          sync.Mutex
	GotCommands [][]string
	MockResults map[string]MockResult // Replaces MockErrors to allow stdout/stderr mocking
	Default     MockResult            // Fallback if a command isn't explicitly mocked
}

type contextKey struct{}

var (
	installOnce sync.Once
	realExec    = exec.CommandContext // Keep a reference to the real execution function
)

// FormatCmd generates an unambiguous string representation of a command.
func FormatCmd(name string, arg ...string) string {
	return exec.Command(name, arg...).String()
}

// InjectContext attaches the MockCommander to the context.
// It ensures a global context-aware router is installed exactly once.
func (m *MockCommander) InjectContext(ctx context.Context) context.Context {
	installOnce.Do(func() {
		execCommand = func(execCtx context.Context, name string, arg ...string) *exec.Cmd {
			if mocker, ok := execCtx.Value(contextKey{}).(*MockCommander); ok {
				return mocker.executeMock(execCtx, name, arg...)
			}
			return realExec(execCtx, name, arg...)
		}
	})

	return context.WithValue(ctx, contextKey{}, m)
}

func (m *MockCommander) executeMock(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := append([]string{name}, arg...)
	key := FormatCmd(name, arg...)

	m.mu.Lock()
	m.GotCommands = append(m.GotCommands, cmd)

	result, ok := m.MockResults[key]
	if !ok {
		result = m.Default
	}
	m.mu.Unlock()

	// Apply the convenience Error field if it was provided
	if result.Error != nil {
		if result.Stderr == "" {
			result.Stderr = result.Error.Error() + "\n"
		}
		if result.ExitCode == 0 {
			result.ExitCode = 1
		}
	}

	exitCodeStr := strconv.Itoa(result.ExitCode)
	var mockCmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Safe PowerShell execution using environment variables
		mockCmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
			"[Console]::Out.Write($env:MOCK_STDOUT); [Console]::Error.Write($env:MOCK_STDERR); exit $env:MOCK_EXIT_CODE")
	} else {
		// Safe Unix execution using environment variables (printf '%s' prevents backslash interpretation)
		mockCmd = exec.CommandContext(ctx, "sh", "-c",
			`printf '%s' "$MOCK_STDOUT"; printf '%s' "$MOCK_STDERR" >&2; exit "$MOCK_EXIT_CODE"`)
	}

	// Attach the mocked outputs as environment variables to completely prevent shell injection
	mockCmd.Env = append(os.Environ(),
		"MOCK_STDOUT="+result.Stdout,
		"MOCK_STDERR="+result.Stderr,
		"MOCK_EXIT_CODE="+exitCodeStr,
	)

	return mockCmd
}
