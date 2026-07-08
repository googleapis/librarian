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

package rust

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestCargoPreFlightSuccess(t *testing.T) {
	setupFakeCargoScript(t, "#!/bin/bash\nexit 0")

	tools := []*config.CargoTool{
		{Name: "cargo-semver-checks"},
	}
	if err := cargoPreFlight(t.Context(), tools); err != nil {
		t.Fatal(err)
	}
}

func TestCargoPreFlightBadCargo(t *testing.T) {
	setupFakeCargoScript(t, "#!/bin/bash\nexit 1")

	tools := []*config.CargoTool{
		{Name: "cargo-semver-checks"},
	}
	if err := cargoPreFlight(t.Context(), tools); err == nil {
		t.Error("expected an error, got none")
	}
}

func TestCargoPreFlightBadTool(t *testing.T) {
	script := `#!/bin/bash
if [ "$1" = "install" ]; then exit 1; fi
exit 0
`
	setupFakeCargoScript(t, script)

	tools := []*config.CargoTool{
		{Name: "not-a-valid-tool", Version: "0.0.1"},
	}
	if err := cargoPreFlight(t.Context(), tools); err == nil {
		t.Error("expected an error, got none")
	}
}

func TestPreFlightMissingGit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	if err := preFlight(t.Context(), nil); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestPreFlightMissingCargo(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	tmpDir := t.TempDir()
	gitScript := filepath.Join(tmpDir, "git")
	os.WriteFile(gitScript, []byte("#!/bin/sh\nexit 0"), 0755)
	t.Setenv("PATH", tmpDir)
	if err := preFlight(t.Context(), nil); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestPreFlightMissingUpstream(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.ContinueInNewGitRepository(t, t.TempDir())
	if err := preFlight(t.Context(), nil); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestPreFlightWithTools(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	setupFakeCargoScript(t, "#!/bin/bash\nexit 0")

	tools := []*config.CargoTool{
		{
			Name:    "cargo-semver-checks",
			Version: "0.42.0",
		},
	}
	testhelper.SetupForVersionBump(t, "test-preflight-with-tools")
	if err := preFlight(t.Context(), tools); err != nil {
		t.Errorf("expected a successful run, got=%v", err)
	}
}

func TestPreFlightToolFailure(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	script := `#!/bin/bash
if [ "$1" = "install" ]; then exit 1; fi
exit 0
`
	setupFakeCargoScript(t, script)

	tools := []*config.CargoTool{
		{
			Name:    "invalid-tool-name---",
			Version: "a.b.c",
		},
	}
	testhelper.SetupForVersionBump(t, "test-preflight-with-tools")
	if err := preFlight(t.Context(), tools); err == nil {
		t.Errorf("expected an error installing cargo-semver-checks")
	}
}
