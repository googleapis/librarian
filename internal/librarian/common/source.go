package common

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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

func AddLibraryRoots(library *config.Library, sources *Sources) map[string]string {
	source := make(map[string]string)
	if len(library.Roots) == 0 && sources.Googleapis != "" {
		// Default to googleapis if no roots are specified.
		source["googleapis-root"] = sources.Googleapis
		source["roots"] = "googleapis"
	} else {
		source["roots"] = strings.Join(library.Roots, ",")
		rootMap := map[string]struct {
			path string
			key  string
		}{
			"googleapis":   {path: sources.Googleapis, key: "googleapis-root"},
			"discovery":    {path: sources.Discovery, key: "discovery-root"},
			"showcase":     {path: sources.Showcase, key: "showcase-root"},
			"protobuf-src": {path: sources.ProtobufSrc, key: "protobuf-src-root"},
			"conformance":  {path: sources.Conformance, key: "conformance-root"},
		}
		for _, root := range library.Roots {
			if r, ok := rootMap[root]; ok && r.path != "" {
				source[r.key] = r.path
			}
		}
	}

	return source
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
