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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/nodejs"
	"github.com/googleapis/librarian/internal/tool/composer"
	"github.com/googleapis/librarian/internal/tool/pip"
)

const (
	generatorVersion = "v1.21.2"
	generatorSHA256  = "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518"
	toolsDir         = "php_tools"
)

var (
	errMissingTools    = errors.New("tools configuration is missing")
	errMissingComposer = errors.New("tools.composer configuration is missing")
	errMissingPip      = errors.New("tools.pip configuration is missing")
	errMissingPNPM     = errors.New("tools.pnpm configuration is missing")
)

// Install installs the PHP generator tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil {
		return errMissingTools
	}
	if len(tools.Composer) == 0 {
		return errMissingComposer
	}
	if len(tools.Pip) == 0 {
		return errMissingPip
	}
	if len(tools.PNPM) == 0 {
		return errMissingPNPM
	}

	phpPath, err := checkRequiredCommands()
	if err != nil {
		return err
	}

	bin, err := binDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	if err := composer.Install(ctx, tools.Composer, phpPath, bin); err != nil {
		return err
	}
	// The PHP client library generation process relies on Python-based
	// tools (such as synthtool or owlbot) for post-processing and generation.
	if err := pip.Install(ctx, tools.Pip); err != nil {
		return err
	}
	// Install PNPM tools.
	// We wrap the error here because InstallPNPM does not wrap its exec errors with context,
	if err := nodejs.InstallPNPM(ctx, tools.PNPM); err != nil {
		return fmt.Errorf("failed to install pnpm tools: %w", err)
	}
	return nil
}

func checkRequiredCommands() (string, error) {
	if _, err := exec.LookPath("composer"); err != nil {
		return "", fmt.Errorf("failed to find composer: %w", err)
	}
	phpPath, err := exec.LookPath("php")
	if err != nil {
		return "", fmt.Errorf("failed to find php: %w", err)
	}
	return phpPath, nil
}

// InstallDir gets the directory where tools should be installed.
func InstallDir() (string, error) {
	dir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	absDir, err := filepath.Abs(filepath.Join(dir, toolsDir))
	if err != nil {
		return "", fmt.Errorf("failed to get install directory: %w", err)
	}
	return absDir, nil
}

// binDir gets the directory where PHP tool executables are stored.
func binDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, "bin"), nil
}
