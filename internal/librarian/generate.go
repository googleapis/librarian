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
	"github.com/googleapis/librarian/internal/librarian/swift"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
	errSkipGenerate            = errors.New("library has skip_generate set")
	errNoPreviewVariant        = errors.New("library does not have a preview variant")
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
			cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
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

	isPreview := isPreviewName(libraryName)
	baseName := trimPreviewName(libraryName)

	// Prepare the libraries to generate by skipping as specified and applying
	// defaults.
	var libraries []*config.Library
	for _, lib := range cfg.Libraries {
		if !all && isPreview && lib.Name == baseName && lib.Preview == nil {
			return fmt.Errorf("%w: %q", errNoPreviewVariant, baseName)
		}
		if !shouldGenerate(lib, all, libraryName) {
			continue
		}
		prepared, err := applyDefaults(cfg.Language, lib, cfg.Default)
		if err != nil {
			return err
		}
		if !all && isPreview {
			prepared = ResolvePreview(prepared, cfg.Language)
		} else if all && lib.Preview != nil {
			// Generate both stable and preview libraries by first appending the
			// resolved library config for the preview variant.
			libraries = append(libraries, ResolvePreview(prepared, cfg.Language))
		}
		libraries = append(libraries, prepared)
	}
	if len(libraries) == 0 {
		if all {
			return errors.New("no libraries to generate: all libraries have skip_generate set")
		}
		for _, lib := range cfg.Libraries {
			if lib.Name == baseName {
				return fmt.Errorf("%w: %q", errSkipGenerate, libraryName)
			}
		}
		return fmt.Errorf("%w: %q", ErrLibraryNotFound, libraryName)
	}

	if err := cleanLibraries(cfg.Language, libraries); err != nil {
		return err
	}
	return generateLibraries(ctx, cfg, libraries, sources)
}

// cleanLibraries iterates over all the given libraries sequentially,
// delegating to language-specific code to clean each library.
func cleanLibraries(language string, libraries []*config.Library) error {
	var err error
	for _, library := range libraries {
		switch language {
		case config.LanguageDart:
			err = checkAndClean(library.Output, library.Keep)
		case config.LanguageFake:
			err = fakeClean(library)
		case config.LanguageGo:
			err = golang.Clean(library)
		case config.LanguageJava:
			err = java.Clean(library)
		case config.LanguageNodejs:
			err = checkAndClean(library.Output, library.Keep)
		case config.LanguagePython:
			err = python.Clean(library)
		case config.LanguageRust:
			keep, keepErr := rust.Keep(library)
			if keepErr != nil {
				return fmt.Errorf("generating keep list: %w", keepErr)
			}
			err = checkAndClean(library.Output, keep)
		case config.LanguageSwift:
			err = checkAndClean(library.Output, library.Keep)
		default:
			err = fmt.Errorf("language %q does not support cleaning", language)
		}
		if err != nil {
			return fmt.Errorf("clean library %q (%s): %w", library.Name, language, err)
		}
	}
	return nil
}

// generateLibraries generates and formats all the given libraries,
// delegating to language-specific code. Each language chooses its own
// concurrency strategy for these two steps.
func generateLibraries(ctx context.Context, cfg *config.Config, libraries []*config.Library, src *sources.Sources) error {
	switch cfg.Language {
	case config.LanguageDart:
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error {
				if err := dart.Generate(gctx, library, src); err != nil {
					return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
				}
				if err := dart.Format(gctx, library); err != nil {
					return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
				}
				return nil
			})
		}
		return g.Wait()
	case config.LanguageFake:
		for _, library := range libraries {
			if err := fakeGenerate(library); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			if err := fakeFormat(library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
		}
		return fakePostGenerate()
	case config.LanguageGo:
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error {
				if err := golang.Generate(gctx, library, src); err != nil {
					return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
		g, gctx = errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error {
				if err := golang.Format(gctx, library, cfg.Tools); err != nil {
					return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
				}
				return nil
			})
		}
		return g.Wait()
	case config.LanguageJava:
		for _, library := range libraries {
			if err := java.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			if err := java.Format(ctx, library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
		}
		return java.PostGenerate(ctx, ".", cfg)
	case config.LanguageNodejs:
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error {
				if err := nodejs.Generate(gctx, cfg, library, src); err != nil {
					return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
				}
				return nil
			})
		}
		return g.Wait()
	case config.LanguagePython:
		for _, library := range libraries {
			// TODO(https://github.com/googleapis/librarian/issues/3730): separate
			// generation and formatting for Python.
			if err := python.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
		}
	case config.LanguageRust:
		// Generation can be parallelized but formatting cannot because
		// cargo fmt shares the Cargo.toml workspace file across libraries.
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error {
				if err := rust.Generate(gctx, cfg, library, src); err != nil {
					return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
		for _, library := range libraries {
			if err := rust.Format(ctx, library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
		}
		return rust.UpdateWorkspace(ctx)
	case config.LanguageSwift:
		g, gctx := errgroup.WithContext(ctx)
		for _, library := range libraries {
			g.Go(func() error {
				if err := swift.Generate(gctx, cfg, library, src); err != nil {
					return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
				}
				if err := swift.Format(gctx, library); err != nil {
					return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
				}
				return nil
			})
		}
		return g.Wait()
	default:
		return fmt.Errorf("language %q does not support generation", cfg.Language)
	}
	return nil
}

func defaultOutput(language string, name, api, defaultOut string) string {
	switch language {
	case config.LanguageDart:
		return dart.DefaultOutput(name, defaultOut)
	case config.LanguageGo:
		return golang.DefaultOutput(name, defaultOut)
	case config.LanguageNodejs:
		return nodejs.DefaultOutput(name, defaultOut)
	case config.LanguagePython:
		return python.DefaultOutput(name, defaultOut)
	case config.LanguageRust:
		return rust.DefaultOutput(api, defaultOut)
	case config.LanguageSwift:
		return swift.DefaultOutput(api, defaultOut)
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
	isPreview := isPreviewName(libraryName)
	if lib.SkipGenerate && !isPreview {
		return false
	}
	if isPreview && lib.Preview != nil && lib.Preview.SkipGenerate {
		return false
	}
	return all || lib.Name == libraryName || (isPreview && lib.Name == trimPreviewName(libraryName))
}

func isPreviewName(libraryName string) bool {
	return strings.HasSuffix(libraryName, "-preview")
}

func trimPreviewName(libraryName string) string {
	return strings.TrimSuffix(libraryName, "-preview")
}
