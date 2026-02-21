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
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// MockCommander tracks executed commands and provisions simulated errors.
type MockCommander struct {
	GotCommands [][]string
	MockErrors  map[string]error // Use for specific command errors
	DefaultErr  error            // Use if you want all commands to fail with this error
}

var mu sync.Mutex

// SetupMock injects the MockCommander into the package's execution path.
// It returns a restore function designed to be passed directly to t.Cleanup().
func (m *MockCommander) SetupMock() func() {
	mu.Lock()
	defer mu.Unlock()

	// Save the original execution function
	oldExec := execCommand

	// Inject the mock closure directly
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := append([]string{name}, arg...)
		m.GotCommands = append(m.GotCommands, cmd)

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
			return exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("echo %q >&2; exit 1", err.Error()))
		}
		return exec.CommandContext(ctx, "true")
	}

	// Return the cleanup closure
	return func() {
		mu.Lock()
		defer mu.Unlock()
		execCommand = oldExec
	}
}
