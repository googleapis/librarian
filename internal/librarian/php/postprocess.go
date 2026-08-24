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
	"path/filepath"
	"regexp"
	"slices"
	"strings"

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

	bin, err := binDir()
	if err != nil {
		return fmt.Errorf("failed to get bin dir: %w", err)
	}
	postProcessor := filepath.Join(bin, "php-post-processor")
	if err := runPostProcessors(ctx, library, stagingDir, postProcessor); err != nil {
		return err
	}
	if err := restoreCopyrightYear(stagingDir, library.CopyrightYear); err != nil {
		return err
	}
	if err := command.RunInDir(ctx, componentName, "python3", "owlbot.py"); err != nil {
		return fmt.Errorf("failed to run owlbot.py: %w", err)
	}

	return nil
}

func runPostProcessors(ctx context.Context, library *config.Library, stagingDirBase, postProcessor string) error {
	stagingDirs := rootStagingDirs(library)
	for _, stagingDir := range stagingDirs {
		apiStagingDir := filepath.Join(stagingDirBase, stagingDir)
		if err := command.RunInDir(ctx, apiStagingDir, postProcessor, "--input", "."); err != nil {
			return fmt.Errorf("failed to run php-post-processor on %s: %w", apiStagingDir, err)
		}
	}
	return nil
}

// rootStagingDirs returns a list of unique root directories from the library's staging directories.
func rootStagingDirs(library *config.Library) []string {
	stagingDirs := make([]string, 0, len(library.APIs))
	for _, api := range library.APIs {
		stagingDirs = append(stagingDirs, filepath.Clean(filepath.FromSlash(api.PHP.StagingSubdir)))
	}
	var res []string
	prefixes := make(map[string]bool)
	slices.Sort(stagingDirs)
	for _, dir := range stagingDirs {
		originalDir := dir
		found := false
		for dir != "." && dir != "/" {
			if prefixes[dir] {
				found = true
				break
			}
			dir = filepath.Dir(dir)
		}
		if !found {
			res = append(res, originalDir)
			prefixes[originalDir] = true
		}
	}
	return res
}

// restoreCopyrightYear replaces the copyright year in generated source files.
func restoreCopyrightYear(outDir, year string) error {
	if year == "" {
		return nil
	}
	re := regexp.MustCompile(`Copyright \d{4} Google`)
	err := filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".php") {
			return nil
		}
		return updateCopyrightYearInFile(path, year, re)
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func updateCopyrightYearInFile(path, year string, re *regexp.Regexp) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	replacement := fmt.Appendf(nil, "Copyright %s Google", year)
	updated := re.ReplaceAll(content, replacement)
	if bytes.Equal(content, updated) {
		return nil
	}
	return os.WriteFile(path, updated, 0o644)
}
