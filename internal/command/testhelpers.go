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

package command

import (
	"os/exec"
	"testing"
)

// RequireCommand skips the test if the specified command is not found in PATH.
// Use this to skip tests that depend on external tools like protoc, cargo, or taplo,
// so that `go test ./...` will always pass on a fresh clone of the repo.
func RequireCommand(t *testing.T, command string) {
	t.Helper()
	if _, err := exec.LookPath(command); err != nil {
		t.Skipf("skipping test because %s is not installed", command)
	}
}
