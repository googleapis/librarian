// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rust

import (
	"context"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
)

const (
	googleapisRepo = "github.com/googleapis/googleapis"
	discoveryRepo  = "github.com/googleapis/discovery-artifact-manager"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, sources *config.Sources) error {
	googleapisDir, err := sourceDir(sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	discoveryDir, err := sourceDir(sources.Discovery, discoveryRepo)
	if err != nil {
		return err
	}
	sidekickConfig := toSidekickConfig(library, library.ServiceConfig, googleapisDir, discoveryDir)
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	return sidekickrust.Generate(model, library.Output, sidekickConfig)
}

func sourceDir(source *config.Source, repo string) (string, error) {
	if source == nil {
		return "", nil
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir(repo, source.Commit, source.SHA256)
}
