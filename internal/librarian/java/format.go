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
	"runtime"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"golang.org/x/sync/errgroup"
)

const maxFilesPerFormatBatch = 2000

// Format formats Java client libraries using google-java-format in batches.
func Format(ctx context.Context, libraries ...*config.Library) error {
	var allFiles []string
	for _, library := range libraries {
		files, err := collectJavaFiles(library.Output)
		if err != nil {
			return fmt.Errorf("failed to find java files for formatting in %q: %w", library.Name, err)
		}
		allFiles = append(allFiles, files...)
	}
	if len(allFiles) == 0 {
		return nil
	}
	env, err := getToolsEnv()
	if err != nil {
		return err
	}
	// Chunk file paths to avoid exceeding OS command-line length limits (ARG_MAX) and to bound per-batch JVM heap usage.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.GOMAXPROCS(0))
	for i := 0; i < len(allFiles); i += maxFilesPerFormatBatch {
		end := min(i+maxFilesPerFormatBatch, len(allFiles))
		g.Go(func() error {
			if err := command.RunWithEnv(gctx, env, "google-java-format", append([]string{"--replace"}, allFiles[i:end]...)...); err != nil {
				return fmt.Errorf("failed to format batch [%d:%d]: %w", i, end, err)
			}
			return nil
		})
	}
	return g.Wait()
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
