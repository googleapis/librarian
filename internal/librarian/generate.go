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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/librarian/java"
	"github.com/googleapis/librarian/internal/librarian/nodejs"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
	errSkipGenerate            = errors.New("library has skip_generate set")
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate <library>",
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
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			return runGenerate(ctx, cfg, all, libraryName)
		},
	}
}

func runGenerate(ctx context.Context, cfg *config.Config, all bool, libraryName string) error {
	sources, err := LoadSources(ctx, cfg.Sources)
	if err != nil {
		return err
	}

	// Prepare the libraries to generate by skipping as specified and applying
	// defaults.
	var libraries []*config.Library
	for _, lib := range cfg.Libraries {
		if !shouldGenerate(lib, all, libraryName) {
			continue
		}
		prepared, err := applyDefaults(cfg.Language, lib, cfg.Default)
		if err != nil {
			return err
		}
		libraries = append(libraries, prepared)
	}
	if len(libraries) == 0 {
		if all {
			return errors.New("no libraries to generate: all libraries have skip_generate set")
		}
		for _, lib := range cfg.Libraries {
			if lib.Name == libraryName {
				return fmt.Errorf("%w: %q", errSkipGenerate, libraryName)
			}
		}
		return fmt.Errorf("%w: %q", ErrLibraryNotFound, libraryName)
	}

	if err := cleanLibraries(cfg.Language, libraries); err != nil {
		return err
	}
	if err := generateLibraries(ctx, cfg, libraries, sources); err != nil {
		return err
	}
	return postGenerate(ctx, cfg.Language, cfg)
}

// cleanLibraries iterates over all the given libraries sequentially,
// delegating to language-specific code to clean each library.
func cleanLibraries(language string, libraries []*config.Library) error {
	for _, library := range libraries {
		switch language {
		case config.LanguageDart:
			if err := checkAndClean(library.Output, library.Keep); err != nil {
				return err
			}
		case config.LanguageFake:
			if err := fakeClean(library); err != nil {
				return err
			}
		case config.LanguageGo:
			if err := golang.Clean(library); err != nil {
				return err
			}
		case config.LanguageJava:
			if err := java.Clean(library); err != nil {
				return err
			}
		case config.LanguageNodejs:
			if err := checkAndClean(library.Output, library.Keep); err != nil {
				return err
			}
		case config.LanguagePython:
			if err := python.Clean(library); err != nil {
				return err
			}
		case config.LanguageRust:
			keep, err := rust.Keep(library)
			if err != nil {
				return fmt.Errorf("library %q: %w", library.Name, err)
			}
			if err := checkAndClean(library.Output, keep); err != nil {
				return err
			}
		}
	}
	return nil
}

// generateLibraries generates and formats all the given libraries,
// delegating to language-specific code. Each language chooses its own
// concurrency strategy for these two steps.
func generateLibraries(ctx context.Context, cfg *config.Config, libraries []*config.Library, src *sidekickconfig.Sources) error {
	switch cfg.Language {
	case config.LanguageDart:
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error {
				if err := dart.Generate(gctx, library, src); err != nil {
					return err
				}
				return dart.Format(gctx, library)
			})
		}
		return g.Wait()
	case config.LanguageFake:
		for _, library := range libraries {
			if err := fakeGenerate(library); err != nil {
				return err
			}
			if err := fakeFormat(library); err != nil {
				return err
			}
		}
	case config.LanguageGo:
		for _, library := range libraries {
			// Generation cannot be parallelized because protoc writes to a
			// shared cloud.google.com/go directory tree under each library's
			// output, and concurrent MoveAndMerge calls would race.
			if err := golang.Generate(ctx, library, src.Googleapis); err != nil {
				return err
			}
		}
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error { return golang.Format(gctx, library) })
		}
		return g.Wait()
	case config.LanguageJava:
		for _, library := range libraries {
			if err := java.Generate(ctx, cfg, library, src.Googleapis); err != nil {
				return err
			}
			if err := java.Format(ctx, library); err != nil {
				return err
			}
		}
	case config.LanguageNodejs:
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error {
				if err := nodejs.Generate(gctx, library, src.Googleapis); err != nil {
					return err
				}
				return nodejs.Format(gctx, library)
			})
		}
		return g.Wait()
	case config.LanguagePython:
		for _, library := range libraries {
			// TODO(https://github.com/googleapis/librarian/issues/3730): separate
			// generation and formatting for Python.
			if err := python.Generate(ctx, cfg, library, src.Googleapis); err != nil {
				return err
			}
		}
	case config.LanguageRust:
		// Generation can be parallelized but formatting cannot because
		// cargo fmt shares the Cargo.toml workspace file across libraries.
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error { return rust.Generate(gctx, cfg, library, src) })
		}
		if err := g.Wait(); err != nil {
			return err
		}
		for _, library := range libraries {
			if err := rust.Format(ctx, library); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("language %q does not support generation", cfg.Language)
	}
	return nil
}

// postGenerate performs repository-level actions after all individual
// libraries have been generated.
func postGenerate(ctx context.Context, language string, cfg *config.Config) error {
	switch language {
	case config.LanguageFake:
		return fakePostGenerate()
	case config.LanguageJava:
		return java.PostGenerate(ctx, cfg)
	case config.LanguageRust:
		return rust.UpdateWorkspace(ctx)
	default:
		return nil
	}
}

func defaultOutput(language string, name, api, defaultOut string) string {
	switch language {
	case config.LanguageDart:
		return dart.DefaultOutput(name, defaultOut)
	case config.LanguageNodejs:
		return nodejs.DefaultOutput(name, defaultOut)
	case config.LanguagePython:
		return python.DefaultOutput(name, defaultOut)
	case config.LanguageRust:
		return rust.DefaultOutput(api, defaultOut)
	default:
		return defaultOut
	}
}

func deriveAPIPath(language string, name string) string {
	switch language {
	case config.LanguageDart:
		return dart.DeriveAPIPath(name)
	case config.LanguageRust:
		return rust.DeriveAPIPath(name)
	default:
		return strings.ReplaceAll(name, "-", "/")
	}
}

func shouldGenerate(lib *config.Library, all bool, libraryName string) bool {
	if lib.SkipGenerate {
		return false
	}
	return all || lib.Name == libraryName
}
