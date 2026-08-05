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

package java

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

const maxFilesPerFormatBatch = 2000

// Format formats Java client libraries using google-java-format in batches via argument files.
func Format(ctx context.Context, libraries ...*config.Library) error {
	var allFiles []string
	for _, lib := range libraries {
		files, err := collectJavaFiles(lib.Output)
		if err != nil {
			return fmt.Errorf("failed to find java files for formatting in %q: %w", lib.Name, err)
		}
		allFiles = append(allFiles, files...)
	}
	if len(allFiles) == 0 {
		return nil
	}
	// Format files in chunks of maxFilesPerFormatBatch (2,000 files).
	// Batching prevents JVM heap exhaustion (GC thrashing) on RAM-constrained CI runners
	// while reducing 250+ JVM invocations down to ~10.
	for i := 0; i < len(allFiles); i += maxFilesPerFormatBatch {
		end := i + maxFilesPerFormatBatch
		if end > len(allFiles) {
			end = len(allFiles)
		}
		chunk := allFiles[i:end]
		if err := formatBatch(ctx, chunk); err != nil {
			return fmt.Errorf("failed to format batch [%d:%d]: %w", i, end, err)
		}
	}
	return nil
}

// formatBatch formats a single chunk of Java files using an argument file.
func formatBatch(ctx context.Context, files []string) error {
	// We write file paths into a temporary argument file (@filename) rather than
	// passing them as discrete CLI arguments to avoid exceeding the OS command-line
	// length limit (ARG_MAX) when formatting thousands of files.
	argFile, err := createArgFile(files)
	if err != nil {
		return fmt.Errorf("failed to create format argument file: %w", err)
	}
	defer os.Remove(argFile)
	env, err := getToolsEnv()
	if err != nil {
		return err
	}
	if err := command.RunWithEnv(ctx, env, "google-java-format", "@"+argFile); err != nil {
		return fmt.Errorf("failed to format files: %w", err)
	}
	return nil
}

// createArgFile creates a temporary file containing --replace and all Java file paths,
// flushes and closes the write handle, and returns the absolute file path.
func createArgFile(files []string) (string, error) {
	tmpFile, err := os.CreateTemp("", "gjf-args-*.txt")
	if err != nil {
		return "", err
	}

	content := "--replace\n" + strings.Join(files, "\n")
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}
	return tmpFile.Name(), nil
}

func collectJavaFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".java" {
			return nil
		}
		// Exclude generated samples and Spanner-specific sample source directory.
		// Spanner stores its samples in a different location than other libraries.
		// TODO(https://github.com/googleapis/librarian/issues/6095): Remove spanner
		// samples exclusion once we got confirm from the spanner team.
		if strings.Contains(path, filepath.Join("samples", "snippets", "generated")) ||
			strings.Contains(path, filepath.Join("samples", "snippets", "src")) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}
