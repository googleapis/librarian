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
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// Format formats a generated Go library.
func Format(ctx context.Context, library *config.Library) error {
	// No need to format the root module because it does not
	// have a dedicated output directory.
	if library.Name == rootModule {
		return nil
	}
	args, err := processArgs(library)
	if err != nil {
		return err
	}
	return command.Run(ctx, "goimports", args...)
}

func processArgs(library *config.Library) ([]string, error) {
	args := []string{"-w"}
	args = append(args, library.Output)
	for _, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if goAPI == nil {
			return nil, errGoAPINotFound
		}
		if goAPI.ProtoOnly {
			continue
		}
		snippetDir := snippetDirectory(repoRootPath(library), clientPathFromRepoRoot(library, goAPI))
		skip := false
		// No need to format the snippet directory if the directory is within one of
		// paths to delete after generation. The snippet directory does not exist.
		for _, path := range library.Go.DeleteGenerationOutputPaths {
			pathToDelete := filepath.Join(library.Output, path)
			if strings.HasPrefix(snippetDir, pathToDelete) {
				skip = true
			}
		}
		if skip {
			continue
		}
		args = append(args, snippetDir)
	}
	return args, nil
}
