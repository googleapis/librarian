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

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

const (
	generatorVersion = "v1.21.2"
	generatorSHA256  = "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518"
)

// Install installs the PHP generator tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools != nil {
		return nil
	}
	_, err := installGenerator(ctx)
	return err
}

// installGenerator is a temp function for testing purposes.
// TODO(https://github.com/googleapis/librarian/issues/6630): remove after install command is ready.
func installGenerator(ctx context.Context) (string, error) {
	phpPath, err := exec.LookPath("php")
	if err != nil {
		return "", fmt.Errorf("failed to find php: %w", err)
	}

	repo := "github.com/googleapis/gapic-generator-php"
	generatorDir, err := fetch.Repo(ctx, repo, generatorVersion, generatorSHA256)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PHP generator: %w", err)
	}
	// Ensure Composer dependencies are installed
	vendorDir := filepath.Join(generatorDir, "vendor")
	if _, err := os.Stat(vendorDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if _, err := exec.LookPath("composer"); err == nil {
			if err := command.RunInDir(ctx, generatorDir, "composer", "install"); err != nil {
				return "", fmt.Errorf("failed to run composer install: %w", err)
			}
		} else {
			composerPhar := filepath.Join(generatorDir, "rules_php_gapic", "resources", "composer.phar")
			if _, err := os.Stat(composerPhar); err == nil {
				if err := command.RunInDir(ctx, generatorDir, phpPath, composerPhar, "install"); err != nil {
					return "", fmt.Errorf("failed to run composer.phar install: %w", err)
				}
			} else {
				return "", fmt.Errorf("neither system composer nor composer.phar was found")
			}
		}
	}
	return generatorDir, nil
}
