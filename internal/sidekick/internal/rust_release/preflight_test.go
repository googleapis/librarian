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

package rustrelease

import (
	"os/exec"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/internal/config"
)

func TestPreflightSuccess(t *testing.T) {
	const echo = "/bin/echo"
	requireCommand(t, echo)
	release := config.Release{
		Preinstalled: map[string]string{
			"git":   echo,
			"cargo": echo,
		},
	}
	if err := PreFlight(&release); err != nil {
		t.Fatal(err)
	}
}

func TestPreflightMissingGit(t *testing.T) {
	release := config.Release{
		Preinstalled: map[string]string{
			"git": "git-is-not-installed",
		},
	}
	if err := PreFlight(&release); err == nil {
		t.Fatal(err)
	}
}

func TestPreflightMissingCargo(t *testing.T) {
	requireCommand(t, "git")
	release := config.Release{
		Preinstalled: map[string]string{
			"cargo": "cargo-is-not-installed",
		},
	}
	if err := PreFlight(&release); err == nil {
		t.Fatal(err)
	}
}

func TestGitExe(t *testing.T) {
	release := config.Release{}
	if got := gitExe(&release); got != "git" {
		t.Errorf("mismatch in gitExe(), want=git, got=%s", got)
	}
	release = config.Release{
		Preinstalled: map[string]string{
			"git": "alternative",
		},
	}
	if got := gitExe(&release); got != "alternative" {
		t.Errorf("mismatch in gitExe(), want=alternative, got=%s", got)
	}
}

func TestCargoExe(t *testing.T) {
	release := config.Release{}
	if got := cargoExe(&release); got != "cargo" {
		t.Errorf("mismatch in cargoExe(), want=cargo, got=%s", got)
	}
	release = config.Release{
		Preinstalled: map[string]string{
			"cargo": "alternative",
		},
	}
	if got := cargoExe(&release); got != "alternative" {
		t.Errorf("mismatch in cargoExe(), want=alternative, got=%s", got)
	}
}

func requireCommand(t *testing.T, command string) {
	t.Helper()
	if _, err := exec.LookPath(command); err != nil {
		t.Skipf("skipping test because %s is not installed", command)
	}
}
