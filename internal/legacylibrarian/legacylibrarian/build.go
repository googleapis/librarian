// Copyright 2025 Google LLC
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

package legacylibrarian

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"cloud.google.com/go/librarian/internal/legacylibrarian/legacyconfig"
	"cloud.google.com/go/librarian/internal/legacylibrarian/legacydocker"
	"cloud.google.com/go/librarian/internal/legacylibrarian/legacygitrepo"
)

func buildSingleLibrary(ctx context.Context, containerClient ContainerClient, state *legacyconfig.LibrarianState, libraryState *legacyconfig.LibraryState, repo legacygitrepo.Repository) error {
	if libraryState == nil {
		return fmt.Errorf("no libraryState provided")
	}
	buildRequest := &legacydocker.BuildRequest{
		LibraryID: libraryState.ID,
		RepoDir:   repo.GetDir(),
		State:     state,
	}
	slog.Info("performing build for library", "id", libraryState.ID)
	if containerErr := containerClient.Build(ctx, buildRequest); containerErr != nil {
		if restoreErr := restoreLibrary(libraryState, repo); restoreErr != nil {
			return errors.Join(containerErr, restoreErr)
		}

		return containerErr
	}

	// Read the library state from the response.
	if _, responseErr := readLibraryState(
		filepath.Join(buildRequest.RepoDir, legacyconfig.LibrarianDir, legacyconfig.BuildResponse)); responseErr != nil {
		if restoreErr := restoreLibrary(libraryState, repo); restoreErr != nil {
			return errors.Join(responseErr, restoreErr)
		}

		return responseErr
	}

	slog.Info("build succeeds", "id", libraryState.ID)
	return nil
}
