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

// Package source provides functionalities to fetch and process source for generating and releasing clients in all
// languages.
package source

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"golang.org/x/sync/errgroup"
)

const (
	discoveryRepo = "github.com/googleapis/discovery-artifact-manager"
	protobufRepo  = "github.com/protocolbuffers/protobuf"
	showcaseRepo  = "github.com/googleapis/gapic-showcase"
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

// FetchRustSources fetches all source repositories needed for Rust generation
// in parallel. It returns a rust.Sources struct with all directories populated.
func FetchRustSources(ctx context.Context, cfgSources *config.Sources) (*Sources, error) {
	sources := &Sources{}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if cfgSources.Discovery.Dir != "" {
			sources.Discovery = cfgSources.Discovery.Dir
			return nil
		}
		dir, err := fetch.RepoDir(ctx, discoveryRepo, cfgSources.Discovery.Commit, cfgSources.Discovery.SHA256)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", discoveryRepo, err)
		}
		sources.Discovery = dir
		return nil
	})
	g.Go(func() error {
		if cfgSources.Conformance.Dir != "" {
			sources.Conformance = cfgSources.Conformance.Dir
			return nil
		}
		dir, err := fetch.RepoDir(ctx, protobufRepo, cfgSources.Conformance.Commit, cfgSources.Conformance.SHA256)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", protobufRepo, err)
		}
		sources.Conformance = dir
		return nil
	})
	g.Go(func() error {
		if cfgSources.Showcase.Dir != "" {
			sources.Showcase = cfgSources.Showcase.Dir
			return nil
		}
		dir, err := fetch.RepoDir(ctx, showcaseRepo, cfgSources.Showcase.Commit, cfgSources.Showcase.SHA256)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", showcaseRepo, err)
		}
		sources.Showcase = dir
		return nil
	})

	if cfgSources.ProtobufSrc != nil {
		g.Go(func() error {
			if cfgSources.ProtobufSrc.Dir != "" {
				sources.ProtobufSrc = cfgSources.ProtobufSrc.Dir
				return nil
			}
			dir, err := fetch.RepoDir(ctx, protobufRepo, cfgSources.ProtobufSrc.Commit, cfgSources.ProtobufSrc.SHA256)
			if err != nil {
				return fmt.Errorf("failed to fetch %s: %w", protobufRepo, err)
			}
			sources.ProtobufSrc = dir
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return sources, nil
}
