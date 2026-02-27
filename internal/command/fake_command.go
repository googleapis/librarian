// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
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

// FakeResult defines the exact output and exit status of a faked command.
type FakeResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error // Convenience: If set, overrides Stderr and sets ExitCode to 1
}

// FakeCommander tracks executed commands and provisions simulated outputs.
// It is fully safe for parallel test execution and cross-platform runs.
type FakeCommander struct {
	mu          sync.Mutex
	GotCommands [][]string
	FakeResults map[string]FakeResult // Stores results mapped by command string
	Default     FakeResult            // Fallback if a command isn't explicitly faked
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

// InjectContext attaches the FakeCommander to the context.
// It ensures a global context-aware router is installed exactly once.
func (f *FakeCommander) InjectContext(ctx context.Context) context.Context {
	installOnce.Do(func() {
		execCommand = func(execCtx context.Context, name string, arg ...string) *exec.Cmd {
			if faker, ok := execCtx.Value(contextKey{}).(*FakeCommander); ok {
				return faker.executeFake(execCtx, name, arg...)
			}
			return realExec(execCtx, name, arg...)
		}
	})

	return context.WithValue(ctx, contextKey{}, f)
}

func (f *FakeCommander) executeFake(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := append([]string{name}, arg...)
	key := FormatCmd(name, arg...)

	f.mu.Lock()
	f.GotCommands = append(f.GotCommands, cmd)

	result, ok := f.FakeResults[key]
	if !ok {
		result = f.Default
	}
	f.mu.Unlock()

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
	var fakeCmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Safe PowerShell execution using environment variables
		fakeCmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
			"[Console]::Out.Write($env:FAKE_STDOUT); [Console]::Error.Write($env:FAKE_STDERR); exit $env:FAKE_EXIT_CODE")
	} else {
		// Safe Unix execution using environment variables (printf '%s' prevents backslash interpretation)
		fakeCmd = exec.CommandContext(ctx, "sh", "-c",
			`printf '%s' "$FAKE_STDOUT"; printf '%s' "$FAKE_STDERR" >&2; exit "$FAKE_EXIT_CODE"`)
	}

	// Attach the faked outputs as environment variables to completely prevent shell injection
	fakeCmd.Env = append(os.Environ(),
		"FAKE_STDOUT="+result.Stdout,
		"FAKE_STDERR="+result.Stderr,
		"FAKE_EXIT_CODE="+exitCodeStr,
	)

	return fakeCmd
}
