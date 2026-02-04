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
	"github.com/googleapis/librarian/internal/librarian/source"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
	errEmptySources            = errors.New("sources required in librarian.yaml")
	errSkipGenerate            = errors.New("library has skip_generate set")
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
			cfg, err := loadConfig(ctx)
			if err != nil {
				return err
			}
			return runGenerate(ctx, cfg, all, libraryName)
		},
	}
}

func runGenerate(ctx context.Context, cfg *config.Config, all bool, libraryName string) error {
	if cfg.Sources == nil {
		return errEmptySources
	}
	return generateLibraries(ctx, all, cfg, libraryName)
}

func generateLibraries(ctx context.Context, all bool, cfg *config.Config, libraryName string) error {
	// Fetch sources.
	sources, err := source.FetchSources(ctx, cfg.Sources)
	if err != nil {
		return err
	}

	// Prepare and clean libraries sequentially.
	// This avoids race conditions when output directories are nested.
	var libraries []*config.Library
	for _, lib := range cfg.Libraries {
		if !shouldGenerate(lib, all, libraryName) {
			continue
		}
		prepared, err := prepareLibrary(cfg.Language, lib, cfg.Default)
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

	// Generate all libraries in parallel.
	g, gctx := errgroup.WithContext(ctx)
	for _, lib := range libraries {
		lib := lib
		g.Go(func() error {
			return generate(gctx, cfg.Language, lib, sources.Googleapis, sources)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	// Format all libraries sequentially.
	for _, lib := range libraries {
		if err := formatLibrary(ctx, cfg.Language, lib); err != nil {
			return err
		}
	}
	return postGenerate(ctx, cfg.Language)
}

// postGenerate performs repository-level actions after all individual
// libraries have been generated.
func postGenerate(ctx context.Context, language string) error {
	switch language {
	case languageRust:
		return rust.UpdateWorkspace(ctx)
	case languageFake:
		return fakePostGenerate()
	default:
		return nil
	}
}

func defaultOutput(language, name, api, defaultOut string) string {
	switch language {
	case languageRust:
		return rust.DefaultOutput(api, defaultOut)
	case languagePython:
		return python.DefaultOutputByName(name, defaultOut)
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

func shouldGenerate(lib *config.Library, all bool, libraryName string) bool {
	if lib.SkipGenerate {
		return false
	}
	return all || lib.Name == libraryName
}

// prepareLibrary applies defaults and cleans the output directory.
func prepareLibrary(language string, lib *config.Library, defaults *config.Default) (*config.Library, error) {
	library, err := applyDefaults(language, lib, defaults)
	if err != nil {
		return nil, err
	}
	switch language {
	case languageFake:
		// No cleaning needed.
	case languageDart, languagePython:
		if err := cleanOutput(library.Output, library.Keep); err != nil {
			return nil, err
		}
	case languageRust:
		keep, err := rust.Keep(library)
		if err != nil {
			return nil, fmt.Errorf("library %q: %w", library.Name, err)
		}
		if err := cleanOutput(library.Output, keep); err != nil {
			return nil, err
		}
	}
	return library, nil
}

func generate(ctx context.Context, language string, library *config.Library, googleapisDir string, sources *source.Sources) error {
	switch language {
	case languageFake:
		if err := fakeGenerate(library); err != nil {
			return err
		}
	case languageDart:
		if err := dart.Generate(ctx, library, sources); err != nil {
			return err
		}
	case languagePython:
		if err := python.Generate(ctx, library, googleapisDir); err != nil {
			return err
		}
	case languageRust:
		if err := rust.Generate(ctx, library, sources); err != nil {
			return err
		}
	default:
		return fmt.Errorf("language %q does not support generation", language)
	}
	return nil
}

func formatLibrary(ctx context.Context, language string, library *config.Library) error {
	switch language {
	case languageFake:
		return fakeFormat(library)
	case languageDart:
		return dart.Format(ctx, library)
	case languageRust:
		return rust.Format(ctx, library)
	case languagePython:
		// Python formatting is currently performed in the generate phase.
		// TODO(https://github.com/googleapis/librarian/issues/3730): separate
		// generation and formatting for Python.
		return nil
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
