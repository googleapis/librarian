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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

var (
	errOwlBotNotFound = errors.New("owlbot.py not found")
)

func postProcessLibrary(ctx context.Context, library *config.Library, componentName string) (err error) {
	stagingDir := filepath.Join(owlBotStagingDir, componentName)
	defer func() {
		if cleanupErr := os.RemoveAll(stagingDir); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
	}()

	// TODO(https://github.com/googleapis/librarian/issues/7153): We need to use component name as library output to maintain backward compatibility. Change this to library.Output when ready.
	owlbotPy := filepath.Join(componentName, "owlbot.py")
	if _, err := os.Stat(owlbotPy); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("library %q: %w", library.Name, errOwlBotNotFound)
		}
		return err
	}

	if err := command.RunInDir(ctx, componentName, "python3", "owlbot.py"); err != nil {
		return fmt.Errorf("failed to run owlbot.py: %w", err)
	}
	bin, err := binDir()
	if err != nil {
		return fmt.Errorf("failed to get bin dir: %w", err)
	}
	postProcessor := filepath.Join(bin, "php-post-processor")
	if err := command.RunInDir(ctx, componentName, postProcessor, "--input", "."); err != nil {
		return fmt.Errorf("failed to run php-post-processor: %w", err)
	}
	return nil
}
