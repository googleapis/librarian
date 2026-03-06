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

package golang

import (
	"context"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// Format formats a generated Go library.
func Format(ctx context.Context, library *config.Library) error {
	outDir, err := filepath.Abs(library.Output)
	if err != nil {
		return err
	}
	args := []string{"-w"}
	libraryDir := filepath.Join(outDir, library.Name)
	if _, err := os.Stat(libraryDir); err == nil {
		args = append(args, libraryDir)
	}
	snippetDir := snippetDirectory(outDir, library.Name)
	if _, err := os.Stat(snippetDir); err == nil {
		args = append(args, snippetDir)
	}
	if len(args) == 1 {
		// No need to format the library if library directory doesn't exist,
		// e.g., root-module.
		return nil
	}
	return command.Run(ctx, "goimports", args...)
}
