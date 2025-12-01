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

// Package release contains the implementation of the release-stage command.
package release

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacycontainer/java/languagecontainer/release"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacycontainer/java/message"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacycontainer/java/pom"
)

// Stage executes the release stage command.
func Stage(ctx context.Context, cfg *release.Config) (*message.ReleaseStageResponse, error) {
	slog.Info("release-stage: invoked", "config", cfg)
	response := &message.ReleaseStageResponse{}
	for _, lib := range cfg.Request.Libraries {
		for _, path := range lib.SourcePaths {
			slog.Info("release-stage: processing library", "libraryID", lib.ID, "version", lib.Version, "sourcePath", path)
			if err := pom.UpdateVersions(
				cfg.Context.RepoDir,
				filepath.Join(cfg.Context.RepoDir, path),
				cfg.Context.OutputDir, lib.ID, lib.Version); err != nil {
				response.Error = err.Error()
				return response, err
			}
		}
	}
	return response, nil
}
