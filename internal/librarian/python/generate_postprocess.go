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

package python

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	changelog         = "CHANGELOG.md"
	changelogTemplate = `# Changelog

[PyPI History][1]

[1]: https://pypi.org/project/%s/#history
`
)

// runPostProcessor runs the synthtool post processor on the output directory.
func runPostProcessor(ctx context.Context, repoRoot, outDir, generationRoot string) error {
	// The post-processor expects the string replacement scripts to be in the
	// output directory, so we need to copy them there.
	// TODO(https://github.com/googleapis/librarian/issues/3008): reimplement
	// the string replacements in Go, and at that point stop copying the files.
	scriptsOutput := filepath.Join(outDir, "scripts", "client-post-processing")
	scriptsInput := filepath.Join(repoRoot, ".librarian", "generator-input", "client-post-processing")
	if err := os.CopyFS(scriptsOutput, os.DirFS(scriptsInput)); err != nil {
		return err
	}

	pythonCode := fmt.Sprintf(`
from synthtool.languages import python_mono_repo
python_mono_repo.owlbot_main(%q)
`, outDir)
	templateDir, err := templateDirectory()
	if err != nil {
		return err
	}
	env := map[string]string{"SYNTHTOOL_TEMPLATES": templateDir}
	if err := command.RunInDirWithEnv(ctx, generationRoot, env, "python3", "-c", pythonCode); err != nil {
		return fmt.Errorf("failed to run post-processor: %w", err)
	}

	// synthtool runs formatting, then applies string replacements. This leaves
	// some files unformatted. We format again just to get everything straight.
	// (Changing synthtool's ordering would require changes in the replacements
	// as well... we can do all of that after migration, when we remove
	// synthtool entirely - see
	// https://github.com/googleapis/librarian/issues/3008)
	if err := command.RunInDir(ctx, outDir, "nox", "-s", "format", "--no-venv", "--no-install"); err != nil {
		return fmt.Errorf("failed to format code after post-processing: %w", err)
	}
	return nil
}

// copyReadmeToDocsDir copies README.rst to docs/README.rst.
// This handles symlinks properly by reading content and writing a real file.
// This is a no-op if either the source doesn't exist, or the library is
// handwritten and the target doesn't already exist.
func copyReadmeToDocsDir(lib *config.Library, outdir string) error {
	sourcePath := filepath.Join(outdir, "README.rst")
	docsPath := filepath.Join(outdir, "docs")
	destPath := filepath.Join(docsPath, "README.rst")

	// If source doesn't exist, nothing to copy
	if _, err := os.Lstat(sourcePath); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	// If the library is handwritten and the target doesn't already exist, skip
	// copying.
	if len(lib.APIs) == 0 {
		if _, err := os.Lstat(destPath); errors.Is(err, fs.ErrNotExist) {
			return nil
		}
	}
	// Read content from source (follows symlinks)
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Create docs directory if it doesn't exist
	if err := os.MkdirAll(docsPath, 0o755); err != nil {
		return err
	}

	// Remove any existing symlink at destination
	if info, err := os.Lstat(destPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(destPath); err != nil {
				return err
			}
		}
	}

	// Write content to destination as a real file
	return os.WriteFile(destPath, content, 0o644)
}

// cleanUpFilesAfterPostProcessing cleans up files after post processing.
// TODO(https://github.com/googleapis/librarian/issues/3210): generate
// directly in place and remove the owl-bot-staging directory entirely.
// TODO(https://github.com/googleapis/librarian/issues/3008): perform string
// replacements in Go code, so we don't need to copy files.
func cleanUpFilesAfterPostProcessing(generationRoot, outdir string) error {
	// Remove the temporary generation directory. RemoveAll will remove the
	// packages/xyz symlink rather than following it and deleting the whole
	// package.
	if err := os.RemoveAll(generationRoot); err != nil {
		return err
	}
	// Remove the post-processing scripts. This will leave the "scripts"
	// directory, but that's okay if it's empty - git ignores empty directories.
	// If it's *not* empty, then there must have been files there before, which
	// we'd want to keep anyway.
	if err := os.RemoveAll(filepath.Join(outdir, "scripts", "client-post-processing")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove client-post-processing directory: %w", err)
	}
	return nil
}

// DefaultOutput derives an output path from a library name and a default
// output directory. Currently, this just assumes each library is a directory
// directly underneath the default output directory.
func DefaultOutput(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}

// DefaultLibraryName derives a library name from an API path by stripping
// the version suffix and replacing "/" with "-".
// For example: "google/cloud/secretmanager/v1" ->
// "google-cloud-secretmanager".
func DefaultLibraryName(api string) string {
	path := api
	if serviceconfig.ExtractVersion(api) != "" {
		// Strip version suffix (v1, v1beta2, v2alpha, etc.).
		path = filepath.Dir(api)
	}
	return strings.ReplaceAll(path, "/", "-")
}

// isPreview determines if the given output directory contains the canonical
// preview subdirectory segments as a means of identifying the library as a
// preview library.
func isPreview(output string) bool {
	return strings.Contains(output, "preview-packages")
}

// createChangelog creates a regular changelog file for the library with the
// specified name in the given output directory, if it doesn't already exist.
// It also creates a symlink to the new file from a docs subdirectory. If the
// changelog file already exists in the output directory, this function returns
// immediately with no error.
func createChangelog(libName, output string) error {
	rootChangelog := filepath.Join(output, changelog)
	_, statErr := os.Stat(rootChangelog)
	// If the file exists, we're done.
	if statErr == nil {
		return nil
	}
	if !errors.Is(statErr, fs.ErrNotExist) {
		return statErr
	}
	docs := filepath.Join(output, "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(changelogTemplate, libName)
	if err := os.WriteFile(rootChangelog, []byte(content), 0o644); err != nil {
		return err
	}
	// Create a relative symlink in docs: CHANGELOG.md => ../CHANGELOG.md
	// The target is created directly rather than using filepath.Join to make
	// sure it always uses a forward-slash, even on Windows.
	if err := os.Symlink("../"+changelog, filepath.Join(docs, changelog)); err != nil {
		return err
	}
	return nil
}
