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

package php

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPostProcess_MissingOwlBot(t *testing.T) {
	ctx := t.Context()
	destDir := t.TempDir()
	lib := &config.Library{
		Name:   "SecretManager",
		Output: destDir,
	}
	err := postProcessLibrary(ctx, lib)
	if !errors.Is(err, errOwlBotNotFound) {
		t.Errorf("postProcessLibrary() error = %v, want = %v", err, errOwlBotNotFound)
	}
}

func TestPostProcess_OwlBot(t *testing.T) {
	testhelper.RequireCommand(t, "python3")
	ctx := t.Context()
	absOwlbotRan, err := filepath.Abs(filepath.Join("testdata", "owlbot_ran.py"))
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	destDir := filepath.Join(repoRoot, "SecretManager")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Symlink mock owlbot.py from testdata that writes "owlbot_ran.txt" when executed.
	if err := os.Symlink(absOwlbotRan, filepath.Join(destDir, "owlbot.py")); err != nil {
		t.Fatal(err)
	}
	lib := &config.Library{
		Name:   "SecretManager",
		Output: destDir,
	}
	if err := postProcessLibrary(ctx, lib); err != nil {
		t.Fatal(err)
	}
	// Verify owlbot.py ran
	expectedFile := filepath.Join(destDir, "owlbot_ran.txt")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("expected file %s to exist (indicating owlbot.py ran)", expectedFile)
	}
}

func TestPostProcess_OwlBotError(t *testing.T) {
	testhelper.RequireCommand(t, "python3")
	ctx := t.Context()
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	destDir := filepath.Join(repoRoot, "SecretManager")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	owlbotPy := filepath.Join(destDir, "owlbot.py")
	if err := os.WriteFile(owlbotPy, []byte("import sys; sys.exit(1)"), 0755); err != nil {
		t.Fatal(err)
	}
	lib := &config.Library{
		Name:   "SecretManager",
		Output: destDir,
	}
	err := postProcessLibrary(ctx, lib)
	if err == nil {
		t.Fatal("postProcessLibrary() expected error, got nil")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("expected exit error, got: %v", err)
	}
}

func TestPostProcess_StatError(t *testing.T) {
	ctx := t.Context()
	repoRoot := t.TempDir()
	destDir := filepath.Join(repoRoot, "SecretManager")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	inaccessibleDir := filepath.Join(destDir, "inaccessible")
	if err := os.MkdirAll(inaccessibleDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(inaccessibleDir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(inaccessibleDir, 0755)
	})

	lib := &config.Library{
		Name:   "SecretManager",
		Output: inaccessibleDir,
	}
	err := postProcessLibrary(ctx, lib)
	if err == nil {
		t.Fatal("postProcessLibrary() expected error, got nil")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("expected permission error, got: %v", err)
	}
}

func TestPostProcess_CleanupError(t *testing.T) {
	ctx := t.Context()
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	destDir := filepath.Join(repoRoot, "SecretManager")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	owlbotPy := filepath.Join(destDir, "owlbot.py")
	if err := os.WriteFile(owlbotPy, []byte("import sys; sys.exit(0)"), 0755); err != nil {
		t.Fatal(err)
	}

	stagingDir := filepath.Join(repoRoot, owlBotStagingDir, "SecretManager")
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		t.Fatal(err)
	}
	inaccessibleSubdir := filepath.Join(stagingDir, "inaccessible")
	if err := os.MkdirAll(inaccessibleSubdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stagingDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(stagingDir, 0755)
	})

	lib := &config.Library{
		Name:   "SecretManager",
		Output: destDir,
	}
	err := postProcessLibrary(ctx, lib)
	if err == nil {
		t.Fatal("postProcessLibrary() expected error, got nil")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("expected permission error, got: %v", err)
	}
}
