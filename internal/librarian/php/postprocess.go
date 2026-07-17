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

func postProcessLibrary(ctx context.Context, library *config.Library) (err error) {
	stagingDir := filepath.Join(owlBotStagingDir, library.Name)
	defer func() {
		if cleanupErr := os.RemoveAll(stagingDir); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
	}()

	owlbotPy := filepath.Join(library.Output, "owlbot.py")
	if _, err := os.Stat(owlbotPy); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("library %q: %w", library.Name, errOwlBotNotFound)
		}
		return err
	}

	if err := command.RunInDir(ctx, library.Output, "python3", "owlbot.py"); err != nil {
		return fmt.Errorf("failed to run owlbot.py: %w", err)
	}
	return nil
}
