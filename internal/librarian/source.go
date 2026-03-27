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
	"errors"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/sources"
	"golang.org/x/sync/errgroup"
)

const (
	googleapisRepo = "github.com/googleapis/googleapis"
	discoveryRepo  = "github.com/googleapis/discovery-artifact-manager"
	protobufRepo   = "github.com/protocolbuffers/protobuf"
	showcaseRepo   = "github.com/googleapis/gapic-showcase"
)

// ErrMissingGoogleapisSource is returned when the googleapis source is missing.
var ErrMissingGoogleapisSource = errors.New("must specify googleapis source")

// LoadSources fetches all source repositories needed for generation in parallel.
// It returns a *sources.Sources struct with all directories populated.
func LoadSources(ctx context.Context, src *config.Sources) (*sources.Sources, error) {
	if src == nil || src.Googleapis == nil {
		return nil, ErrMissingGoogleapisSource
	}
	srcs := &sources.Sources{}
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		dir, err := fetchSource(ctx, src.Googleapis, googleapisRepo)
		if err != nil {
			return err
		}
		if dir == "" {
			return ErrMissingGoogleapisSource
		}
		srcs.Googleapis = dir
		return nil
	})
	g.Go(func() error {
		dir, err := fetchSource(ctx, src.Conformance, protobufRepo)
		if err != nil {
			return err
		}
		srcs.Conformance = dir
		return nil
	})
	g.Go(func() error {
		dir, err := fetchSource(ctx, src.Discovery, discoveryRepo)
		if err != nil {
			return err
		}
		srcs.Discovery = dir
		return nil
	})
	g.Go(func() error {
		dir, err := fetchSource(ctx, src.Showcase, showcaseRepo)
		if err != nil {
			return err
		}
		srcs.Showcase = dir
		return nil
	})
	if src.ProtobufSrc != nil {
		g.Go(func() error {
			dir, err := fetchSource(ctx, src.ProtobufSrc, protobufRepo)
			if err != nil {
				return err
			}
			srcs.ProtobufSrc = filepath.Join(dir, src.ProtobufSrc.Subpath)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return srcs, nil
}

func fetchSource(ctx context.Context, source *config.Source, repo string) (string, error) {
	if source == nil {
		return "", nil
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	dir, err := fetch.Repo(ctx, repo, source.Commit, source.SHA256)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", repo, err)
	}
	return dir, nil
}
