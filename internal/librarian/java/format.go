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
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

const maxFilesPerFormatBatch = 4000

// Format formats Java client libraries using google-java-format in batches.
func Format(ctx context.Context, libraries ...*config.Library) error {
	var allFiles []string
	for _, lib := range libraries {
		files, err := collectJavaFiles(lib.Output)
		if err != nil {
			return fmt.Errorf("failed to find java files for formatting in %q: %w", lib.Name, err)
		}
		allFiles = append(allFiles, files...)
	}
	env, err := getToolsEnv()
	if err != nil {
		return err
	}
	totalBatches := (len(allFiles) + maxFilesPerFormatBatch - 1) / maxFilesPerFormatBatch
	slog.Info("starting java format step", "total_files", len(allFiles), "total_batches", totalBatches)

	// Batch file paths in chunks of maxFilesPerFormatBatch (4,000 files).
	// Passing 4,000 files per CLI invocation avoids exceeding OS command-line length limits (ARG_MAX)
	// while keeping JVM heap memory safe on RAM-constrained CI runners (~700MB Heap).
	for i := 0; i < len(allFiles); i += maxFilesPerFormatBatch {
		end := min(i+maxFilesPerFormatBatch, len(allFiles))
		chunk := allFiles[i:end]
		batchIdx := i/maxFilesPerFormatBatch + 1

		batchStart := time.Now()
		args := append([]string{"--replace"}, chunk...)
		if err := command.RunWithEnv(ctx, env, "google-java-format", args...); err != nil {
			return fmt.Errorf("failed to format batch %d/%d [%d:%d]: %w", batchIdx, totalBatches, i, end, err)
		}
		slog.Info("formatted java batch", "batch", batchIdx, "total", totalBatches, "files", len(chunk), "duration", time.Since(batchStart))
	}
	return nil
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
