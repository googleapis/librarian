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

package librarian

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"golang.org/x/sync/errgroup"
)

const (
	discoveryRepo = "github.com/googleapis/discovery-artifact-manager"
	protobufRepo  = "github.com/protocolbuffers/protobuf"
	showcaseRepo  = "github.com/googleapis/gapic-showcase"
)

// FetchRustDartSources fetches all source repositories needed for Rust and Dart generation in parallel.
// It returns a *sidekickconfig.Sources struct with all directories populated.
func FetchRustDartSources(ctx context.Context, cfgSources *config.Sources) (*sidekickconfig.Sources, error) {
	sources := &sidekickconfig.Sources{}
	// fetchSource fetches a repository source.
	fetchSource := func(ctx context.Context, source *config.Source, repo string) (string, error) {
		if source == nil {
			return "", nil
		}
		if source.Dir != "" {
			return source.Dir, nil
		}

		dir, err := fetch.RepoDir(ctx, repo, source.Commit, source.SHA256)
		if err != nil {
			return "", fmt.Errorf("failed to fetch %s: %w", repo, err)
		}
		return dir, nil
	}
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		dir, err := fetchSource(ctx, cfgSources.Conformance, protobufRepo)
		if err != nil {
			return err
		}
		sources.Conformance = dir
		return nil
	})
	g.Go(func() error {
		dir, err := fetchSource(ctx, cfgSources.Discovery, discoveryRepo)
		if err != nil {
			return err
		}
		sources.Discovery = dir
		return nil
	})
	g.Go(func() error {
		dir, err := fetchSource(ctx, cfgSources.Showcase, showcaseRepo)
		if err != nil {
			return err
		}
		sources.Showcase = dir
		return nil
	})
	if cfgSources.ProtobufSrc != nil {
		g.Go(func() error {
			dir, err := fetchSource(ctx, cfgSources.ProtobufSrc, protobufRepo)
			if err != nil {
				return err
			}
			sources.ProtobufSrc = filepath.Join(dir, cfgSources.ProtobufSrc.Subpath)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return sources, nil
}
