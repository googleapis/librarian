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

// Package source provides common functionalities for generating and releasing clients in all languages.
package source

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"golang.org/x/sync/errgroup"
)

const (
	discoveryRepo  = "github.com/googleapis/discovery-artifact-manager"
	googleapisRepo = "github.com/googleapis/googleapis"
	protobufRepo   = "github.com/protocolbuffers/protobuf"
	showcaseRepo   = "github.com/googleapis/gapic-showcase"
)

// Sources contains the directory paths for source repositories used by
// sidekick.
type Sources struct {
	Conformance string
	Discovery   string
	Googleapis  string
	ProtobufSrc string
	Showcase    string
}

// FetchSources fetches all source repositories needed for library generation in parallel.
// It returns a Sources struct with all directories populated.
func FetchSources(ctx context.Context, cfgSources *config.Sources) (*Sources, error) {
	sources := &Sources{}

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
		dir, err := fetchSource(ctx, cfgSources.Googleapis, googleapisRepo)
		if err != nil {
			return err
		}
		sources.Googleapis = dir
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
	g.Go(func() error {
		dir, err := fetchSource(ctx, cfgSources.Showcase, showcaseRepo)
		if err != nil {
			return err
		}
		sources.Showcase = dir
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return sources, nil
}

// fetchSource fetches a repository source.
func fetchSource(ctx context.Context, source *config.Source, repo string) (string, error) {
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
