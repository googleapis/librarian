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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/license"
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

	bin, err := binDir()
	if err != nil {
		return fmt.Errorf("failed to get bin dir: %w", err)
	}
	postProcessor := filepath.Join(bin, "php-post-processor")
	if err := runPostProcessors(ctx, library, stagingDir, postProcessor); err != nil {
		return err
	}
	if err := restoreCopyrightYear(stagingDir, componentName, library.CopyrightYear); err != nil {
		return err
	}
	if err := command.RunInDir(ctx, componentName, "python3", "owlbot.py"); err != nil {
		return fmt.Errorf("failed to run owlbot.py: %w", err)
	}

	return nil
}

func runPostProcessors(ctx context.Context, library *config.Library, stagingDir, postProcessor string) error {
	for _, api := range library.APIs {
		apiStagingDir := filepath.Join(stagingDir, api.PHP.StagingSubdir)
		if err := command.RunInDir(ctx, apiStagingDir, postProcessor, "--input", "."); err != nil {
			return fmt.Errorf("failed to run php-post-processor on %s: %w", api.Path, err)
		}
	}
	return nil
}

// restoreCopyrightYear replaces the copyright year in generated source files.
func restoreCopyrightYear(outDir, originalDir, fallbackYear string) error {
	re := regexp.MustCompile(`Copyright \d{4} Google`)
	err := filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".php") {
			return nil
		}

		relPath, err := filepath.Rel(outDir, path)
		if err != nil {
			return err
		}

		mappedRelPath := relPath
		for _, prefix := range []string{"src/", "tests/", "samples/", "metadata/"} {
			if idx := strings.Index(filepath.ToSlash(relPath), "/"+prefix); idx != -1 {
				mappedRelPath = relPath[idx+1:]
				break
			} else if strings.HasPrefix(filepath.ToSlash(relPath), prefix) {
				mappedRelPath = relPath
				break
			}
		}

		yearToUse := fallbackYear
		origPath := filepath.Join(originalDir, mappedRelPath)
		// Assuming the current working directory is the repository root (which is true for `librarian generate`).
		if origContent, err := exec.Command("git", "show", "HEAD:"+filepath.ToSlash(origPath)).CombinedOutput(); err == nil {
			if year := license.Year(origContent); year != "" {
				yearToUse = year
			}
		}

		if yearToUse == "" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		replacement := []byte(fmt.Sprintf("Copyright %s Google", yearToUse))
		updated := re.ReplaceAll(content, replacement)
		if bytes.Equal(content, updated) {
			return nil
		}
		return os.WriteFile(path, updated, 0644)
	})
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
