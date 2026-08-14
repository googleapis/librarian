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
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/librarian/java"
	"github.com/googleapis/librarian/internal/librarian/nodejs"
	"github.com/googleapis/librarian/internal/librarian/php"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/ruby"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/librarian/swift"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/timing"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
	errSkipGenerate            = errors.New("library has skip_generate set")
	errNoPreviewVariant        = errors.New("library does not have a preview variant")
	errUnsupportedLanguage     = errors.New("language does not support generation")
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate <library>",
		Description: `generate produces client library code from the APIs configured in
librarian.yaml.

The library name argument selects a single library to regenerate. Use the
--all flag to regenerate every library in the workspace instead. Exactly
one of <library> or --all must be provided.

Generation is delegated to the language-specific tooling configured in
librarian.yaml. Libraries marked with skip_generate are skipped.

Examples:

	librarian generate <library>   # regenerate one library
	librarian generate --all       # regenerate every library

[after-flags]
A typical librarian workflow for regenerating every library against the
latest API definitions is:

	librarian update googleapis
	librarian generate --all`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "generate all libraries",
			},
			&cli.BoolFlag{
				Name:  "timing",
				Usage: "print a per-phase timing summary to stderr after generation",
			},
			&cli.IntFlag{
				Name:    "jobs",
				Aliases: []string{"j"},
				Usage:   "maximum number of libraries to generate concurrently (default: number of CPUs)",
			},
			&cli.BoolFlag{
				Name:  "changed-only",
				Usage: "skip regenerating libraries whose inputs are unchanged (Java only; opt-in)",
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
			if cmd.Bool("timing") {
				tc := timing.New(cfg.Language)
				ctx = timing.WithCollector(ctx, tc)
				defer func() { fmt.Fprint(os.Stderr, tc.Summary()) }()
			}
			return runGenerate(ctx, cfg, all, libraryName, cmd.Int("jobs"), cmd.Bool("changed-only"))
		},
	}
}

func runGenerate(ctx context.Context, cfg *config.Config, all bool, libraryName string, jobs int, changedOnly bool) error {
	tc := timing.FromContext(ctx)
	defer tc.Span("generate.total")()

	srcStop := tc.Span("sources.load")
	sources, err := LoadSources(ctx, cfg.Sources)
	srcStop()
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

	// --changed-only (Java) skips libraries whose inputs match their recorded
	// manifest. This must happen before cleaning so skipped libraries keep
	// their output (and manifest) untouched.
	var fingerprints map[string]string
	if changedOnly && cfg.Language == config.LanguageJava {
		filterStop := tc.Span("changed_only.filter")
		libraries, fingerprints = filterChangedLibraries(libraries, sources, configExtra(cfg))
		filterStop()
		if len(libraries) == 0 {
			slog.Info("changed-only: no libraries changed, nothing to generate")
			return nil
		}
	}

	cleanStop := tc.Span("clean.total")
	err = cleanLibraries(cfg.Language, libraries)
	cleanStop()
	if err != nil {
		return err
	}

	defer tc.Span("generate_libraries.total")()
	return generateLibraries(ctx, cfg, libraries, sources, jobs, fingerprints)
}

// concurrencyLimit resolves the effective worker count from the --jobs flag: a
// positive value is used as-is, otherwise it falls back to the number of CPUs.
func concurrencyLimit(jobs int) int {
	if jobs > 0 {
		return jobs
	}
	return runtime.NumCPU()
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
			err = nodejs.Clean(library)
		case config.LanguagePhp:
			err = php.Clean(library)
		case config.LanguagePython:
			err = python.Clean(library)
		case config.LanguageRuby:
			err = ruby.Clean(library)
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

// forEachLibrary runs fn for every library concurrently, bounded by limit, and
// records a per-library timing span named phase. It is the single place that
// owns the generate/format concurrency, so every language gets consistent
// parallelism and per-library benchmarking for free — instrumenting a new
// language is just routing its work through here.
func forEachLibrary(ctx context.Context, libraries []*config.Library, limit int, phase string, fn func(context.Context, *config.Library) error) error {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	tc := timing.FromContext(ctx)
	for _, library := range libraries {
		g.Go(func() error {
			defer tc.Span(phase)()
			return fn(gctx, library)
		})
	}
	return g.Wait()
}

// generateLibraries generates and formats all the given libraries, delegating
// to language-specific code via forEachLibrary, which provides the shared
// concurrency and per-library timing.
func generateLibraries(ctx context.Context, cfg *config.Config, libraries []*config.Library, src *sources.Sources, jobs int, fingerprints map[string]string) error {
	limit := concurrencyLimit(jobs)
	switch cfg.Language {
	case config.LanguageDart:
		return forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := dart.Generate(ctx, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			if err := dart.Format(ctx, library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		})
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
		if err := forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := golang.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		}); err != nil {
			return err
		}
		return forEachLibrary(ctx, libraries, limit, timing.PhaseFormatLibrary, func(ctx context.Context, library *config.Library) error {
			if err := golang.Format(ctx, library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		})
	case config.LanguageJava:
		// Each library writes only within its own output directory (syncPOMs is
		// scoped to library.Output); the repo-shared root/BOM POMs are written
		// by the sequential PostGenerate below, after all libraries complete.
		if err := forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := java.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			// Record the input fingerprint so a later --changed-only run can
			// skip this library while its inputs are unchanged.
			if fp := fingerprints[library.Name]; fp != "" {
				if werr := writeGeneratedManifest(library.Output, fp); werr != nil {
					slog.Warn("changed-only: failed to write manifest", "library", library.Name, "err", werr)
				}
			}
			return nil
		}); err != nil {
			return err
		}
		tc := timing.FromContext(ctx)
		fmtStop := tc.Span("format.all")
		err := java.Format(ctx, libraries...)
		fmtStop()
		if err != nil {
			return fmt.Errorf("format java libraries (%s): %w", cfg.Language, err)
		}
		defer tc.Span("postgenerate")()
		return java.PostGenerate(ctx, ".", cfg)
	case config.LanguageNodejs:
		return forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := nodejs.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		})
	case config.LanguagePhp:
		return forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := php.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			if err := php.Format(ctx, library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		})
	case config.LanguagePython:
		// TODO(https://github.com/googleapis/librarian/issues/3730):
		// separate generation and formatting for Python.
		return forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := python.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		})
	case config.LanguageRuby:
		return forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := ruby.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			if err := ruby.Format(ctx, library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		})
	case config.LanguageRust:
		if err := forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := rust.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		}); err != nil {
			return err
		}
		// Formatting must run after generation: files are removed during
		// generation, and formatting reads the Cargo.toml of dependencies.
		if err := forEachLibrary(ctx, libraries, limit, timing.PhaseFormatLibrary, func(ctx context.Context, library *config.Library) error {
			if err := rust.Format(ctx, library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		}); err != nil {
			return err
		}
		return rust.UpdateWorkspace(ctx)
	case config.LanguageSwift:
		return forEachLibrary(ctx, libraries, limit, timing.PhaseGenerateLibrary, func(ctx context.Context, library *config.Library) error {
			if err := swift.Generate(ctx, cfg, library, src); err != nil {
				return fmt.Errorf("generate library %q (%s): %w", library.Name, cfg.Language, err)
			}
			if err := swift.Format(ctx, library); err != nil {
				return fmt.Errorf("format library %q (%s): %w", library.Name, cfg.Language, err)
			}
			return nil
		})
	default:
		return fmt.Errorf("%w: %q", errUnsupportedLanguage, cfg.Language)
	}
}

func defaultOutput(language string, name, api, defaultOut string) string {
	switch language {
	case config.LanguageDart:
		return dart.DefaultOutput(name, defaultOut)
	case config.LanguageGo:
		return golang.DefaultOutput(name, defaultOut)
	case config.LanguageJava:
		return java.DefaultOutput(name, defaultOut)
	case config.LanguageNodejs:
		return nodejs.DefaultOutput(name, defaultOut)
	case config.LanguagePhp:
		return php.DefaultOutput(name, defaultOut)
	case config.LanguagePython:
		return python.DefaultOutput(name, defaultOut)
	case config.LanguageRuby:
		return ruby.DefaultOutput(name, defaultOut)
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
