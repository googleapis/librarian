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

package librarian

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

const (
	discoveryRepo  = "github.com/googleapis/discovery-artifact-manager"
	googleapisRepo = "github.com/googleapis/googleapis"
	protobufRepo   = "github.com/protocolbuffers/protobuf"
	showcaseRepo   = "github.com/googleapis/gapic-showcase"
)

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
	errEmptySources            = errors.New("sources required in librarian.yaml")
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate [library] [--all]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "generate all libraries",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			all := cmd.Bool("all")
			libraryName := cmd.Args().First()
			if !all && libraryName == "" {
				return errMissingLibraryOrAllFlag
			}
			if all && libraryName != "" {
				return errBothLibraryAndAllFlag
			}
			return runGenerate(ctx, all, libraryName)
		},
	}
}

func runGenerate(ctx context.Context, all bool, libraryName string) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	if cfg.Sources == nil {
		return errEmptySources
	}
	return routeGenerate(ctx, all, cfg, libraryName)
}

func routeGenerate(ctx context.Context, all bool, cfg *config.Config, libraryName string) error {
	if all {
		return generateAll(ctx, cfg)
	}

	googleapisDir, err := fetchSource(ctx, cfg.Sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	var rustSources *rust.Sources
	if cfg.Language == languageRust {
		rustSources, err = fetchRustSources(ctx, cfg.Sources)
		if err != nil {
			return err
		}
		rustSources.Googleapis = googleapisDir
	}

	lib, err := generateLibrary(ctx, cfg, libraryName, googleapisDir, rustSources)
	if err != nil {
		return err
	}
	if lib == nil {
		// Skip formatting if generation skipped.
		return nil
	}
	return formatLibrary(ctx, cfg.Language, lib)
}

func generateAll(ctx context.Context, cfg *config.Config) error {
	googleapisDir, err := fetchSource(ctx, cfg.Sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	var rustSources *rust.Sources
	if cfg.Language == languageRust {
		rustSources, err = fetchRustSources(ctx, cfg.Sources)
		if err != nil {
			return err
		}
		rustSources.Googleapis = googleapisDir
	}

	g, gCtx := errgroup.WithContext(ctx)
	libraries := make([]*config.Library, len(cfg.Libraries))
	for i, lib := range cfg.Libraries {
		i := i
		name := lib.Name
		g.Go(func() error {
			lib, err := generateLibrary(gCtx, cfg, name, googleapisDir, rustSources)
			if err != nil {
				return err
			}
			libraries[i] = lib
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	for _, lib := range libraries {
		if lib == nil {
			// Skip formatting if generation skipped.
			continue
		}
		if err := formatLibrary(ctx, cfg.Language, lib); err != nil {
			return err
		}
	}
	return nil
}

func defaultOutput(language, channel, defaultOut string) string {
	switch language {
	case languageRust:
		return rust.DefaultOutput(channel, defaultOut)
	default:
		return defaultOut
	}
}

func deriveAPIPath(language, name string) string {
	switch language {
	case languageRust:
		return rust.DeriveAPIPath(name)
	default:
		return strings.ReplaceAll(name, "-", "/")
	}
}

func generateLibrary(ctx context.Context, cfg *config.Config, libraryName string, googleapisDir string, rustSources *rust.Sources) (*config.Library, error) {
	for _, lib := range cfg.Libraries {
		if lib.Name == libraryName {
			if lib.SkipGenerate {
				return nil, nil
			}
			lib, err := prepareLibrary(cfg.Language, lib, cfg.Default, true)
			if err != nil {
				return nil, err
			}
			return generate(ctx, cfg.Language, lib, googleapisDir, rustSources)
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrLibraryNotFound, libraryName)
}

func generate(ctx context.Context, language string, library *config.Library, googleapisDir string, rustSources *rust.Sources) (_ *config.Library, err error) {
	switch language {
	case languageFake:
		if err := fakeGenerate(library); err != nil {
			return nil, err
		}
	case languageDart:
		if err := cleanOutput(library.Output, library.Keep); err != nil {
			return nil, err
		}
		if err := dart.Generate(ctx, library, googleapisDir); err != nil {
			return nil, err
		}
	case languagePython:
		if err := cleanOutput(library.Output, library.Keep); err != nil {
			return nil, err
		}
		if err := python.Generate(ctx, library, googleapisDir); err != nil {
			return nil, err
		}
	case languageRust:
		keep, err := rust.Keep(library)
		if err != nil {
			return nil, fmt.Errorf("library %q: %w", library.Name, err)
		}
		if err := cleanOutput(library.Output, keep); err != nil {
			return nil, fmt.Errorf("library %q: %w", library.Name, err)
		}
		if err := rust.Generate(ctx, library, rustSources); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("language %q does not support generation", language)
	}
	return library, nil
}

// fetchRustSources fetches all source repositories needed for Rust generation
// in parallel. It returns a rust.Sources struct with all directories populated.
func fetchRustSources(ctx context.Context, cfgSources *config.Sources) (*rust.Sources, error) {
	sources := &rust.Sources{}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		dir, err := fetchSource(ctx, cfgSources.Discovery, discoveryRepo)
		if err != nil {
			return err
		}
		sources.Discovery = dir
		return nil
	})
	g.Go(func() error {
		dir, err := fetchSource(ctx, cfgSources.Conformance, protobufRepo)
		if err != nil {
			return err
		}
		sources.Conformance = dir
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

func formatLibrary(ctx context.Context, language string, library *config.Library) error {
	switch language {
	case languageFake:
		return fakeFormat(library)
	case languageDart:
		return dart.Format(ctx, library)
	case languageRust:
		return rust.Format(ctx, library)
	}
	return fmt.Errorf("language %q does not support formatting", language)
}

// cleanOutput removes all files in dir except those in keep. The keep list
// should contain paths relative to dir. It returns an error if any file
// in keep does not exist.
func cleanOutput(dir string, keep []string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cannot access output directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", dir)
	}

	keepSet := make(map[string]bool)
	for _, k := range keep {
		path := filepath.Join(dir, k)
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("keep file %q does not exist", k)
		}
		// Effectively get a canonical relative path. While in most cases
		// this will be equal to k, it might not be - in particular,
		// on Windows the directory separator in paths returned by Rel
		// will be a backslash.
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		keepSet[rel] = true
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if keepSet[rel] {
			return nil
		}
		return os.Remove(path)
	})
}
