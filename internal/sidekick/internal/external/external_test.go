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

package external

import (
	"os/exec"
	"testing"
)

func TestSuccess(t *testing.T) {
	// "go" must be installed, otherwise: how are you running the unit tests?
	cmd := exec.Command("go", "help")
	if err := Exec(cmd); err != nil {
		t.Fatal(err)
	}
}

func TestError(t *testing.T) {
	// Seems unlikely that `go` will gain this subcommand. I will buy you a cold
	// beverage if I am wrong.
	cmd := exec.Command("go", "invalid-subcommand-bad-bad-bad")
	if err := Exec(cmd); err == nil {
		t.Errorf("expected an error using go invalid-subcommand-bad-bad-bad")
	}
}

func TestRun(t *testing.T) {
	if err := Run("go", "help"); err != nil {
		t.Fatal(err)
	}
}
