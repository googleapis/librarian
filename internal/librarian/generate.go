// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
)

func generateSingleLibrary(ctx context.Context, containerClient ContainerClient, state *config.LibrarianState, libraryState *config.LibraryState, repo gitrepo.Repository, sourceRepo gitrepo.Repository, outputDir string) error {
	// For each library, create a separate output directory. This avoids
	// libraries interfering with each other, and makes it easier to see what
	// was generated for each library when debugging.
	safeLibraryDirectory := getSafeDirectoryName(libraryState.ID)
	libraryOutputDir := filepath.Join(outputDir, safeLibraryDirectory)
	if err := os.MkdirAll(libraryOutputDir, 0755); err != nil {
		return fmt.Errorf("error making output directory %w", err)
	}

	apiRoot, err := filepath.Abs(sourceRepo.GetDir())
	if err != nil {
		return err
	}

	generateRequest := &docker.GenerateRequest{
		ApiRoot:   apiRoot,
		LibraryID: libraryState.ID,
		Output:    libraryOutputDir,
		RepoDir:   repo.GetDir(),
		State:     state,
	}
	slog.Info("Performing generation for library", "id", libraryState.ID, "outputDir", libraryOutputDir)
	if err := containerClient.Generate(ctx, generateRequest); err != nil {
		return err
	}

	// Read the library state from the response.
	if _, err := readLibraryState(
		filepath.Join(generateRequest.RepoDir, config.LibrarianDir, config.GenerateResponse)); err != nil {
		return err
	}

	if err := cleanAndCopyLibrary(state, repo.GetDir(), libraryState.ID, libraryOutputDir); err != nil {
		return err
	}

	slog.Info("Generation succeeds", "id", libraryState.ID)
	return nil
}

func restoreLibrary(libraryState *config.LibraryState, repo gitrepo.Repository) error {
	if err := repo.Restore(libraryState.SourceRoots); err != nil {
		return err
	}
	return repo.CleanUntracked(libraryState.SourceRoots)
}

// getSafeDirectoryName returns a directory name which doesn't contain slashes
// based on a library ID. This avoids cases where a library ID contains
// slashes, but we want generateSingleLibrary to create a directory which
// is not a subdirectory of some other directory. For example, if there
// are library IDs of "pubsub" and "pubsub/v2" we don't want to create
// "output/pubsub/v2" and then "output/pubsub" later. This function does
// not protect against malicious library IDs, e.g. ".", ".." or deliberate
// collisions (e.g. "pubsub/v2" and "pubsub-slash-v2").
//
// The exact implementation may change over time - nothing should rely on this.
// The current implementation simply replaces any slashes with "-slash-".
func getSafeDirectoryName(libraryID string) string {
	return strings.ReplaceAll(libraryID, "/", "-slash-")
}
