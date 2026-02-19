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
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/sidekick/source"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

const (
	googleapisRepo = "github.com/googleapis/googleapis"
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
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
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

	googleapisDir, rustDartSources, err := LoadSources(ctx, cfg)
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

	// Clean, generate and format libraries. Each of these steps is completed
	// before the next one starts, but each language can choose whether to
	// implement the step in parallel across all libraries or in sequence.
	if err := cleanLibraries(cfg.Language, libraries); err != nil {
		return err
	}
	if err := generateLibraries(ctx, cfg.Language, libraries, googleapisDir, rustDartSources); err != nil {
		return err
	}
	if err := formatLibraries(ctx, cfg.Language, libraries); err != nil {
		return err
	}
	return postGenerate(ctx, cfg.Language)
}

// LoadSources fetches and loads the sources required for generation.
func LoadSources(ctx context.Context, cfg *config.Config) (string, *source.Sources, error) {
	var googleapisDir string
	if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
		return "", nil, errors.New("must specify --googleapis flag")
	}
	if cfg.Sources.Googleapis.Dir != "" {
		googleapisDir = cfg.Sources.Googleapis.Dir
	} else {
		dir, err := fetch.RepoDir(ctx, googleapisRepo, cfg.Sources.Googleapis.Commit, cfg.Sources.Googleapis.SHA256)
		if err != nil {
			return "", nil, fmt.Errorf("failed to fetch %s: %w", googleapisRepo, err)
		}
		googleapisDir = dir
	}

	var rustDartSources *source.Sources
	if cfg.Language == languageRust || cfg.Language == languageDart {
		sources, err := source.FetchRustDartSources(ctx, cfg.Sources)
		if err != nil {
			return "", nil, err
		}
		rustDartSources = sources
		rustDartSources.Googleapis = googleapisDir
	}
	return googleapisDir, rustDartSources, nil
}

// cleanLibraries iterates over all the given libraries sequentially,
// delegating to language-specific code to clean each library.
func cleanLibraries(language string, libraries []*config.Library) error {
	for _, library := range libraries {
		switch language {
		case languageFake:
			// No cleaning needed.
		case languageDart:
			if err := checkAndClean(library.Output, library.Keep); err != nil {
				return err
			}
		case languagePython:
			if err := python.CleanLibrary(library); err != nil {
				return err
			}
		case languageGo:
			if err := golang.Clean(library); err != nil {
				return err
			}
		case languageRust:
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

// generateLibraries delegates to language-specific code to generate all the
// given libraries.
func generateLibraries(ctx context.Context, language string, libraries []*config.Library, googleapisDir string, src *source.Sources) error {
	switch language {
	case languageFake:
		return fakeGenerateLibraries(libraries)
	case languageDart:
		return dart.GenerateLibraries(ctx, libraries, src)
	case languagePython:
		return python.GenerateLibraries(ctx, libraries, googleapisDir)
	case languageGo:
		return golang.GenerateLibraries(ctx, libraries, googleapisDir)
	case languageRust:
		return rust.GenerateLibraries(ctx, libraries, src)
	default:
		return fmt.Errorf("language %q does not support generation", language)
	}
}

// formatLibraries iterates over all the given libraries sequentially,
// delegating to language-specific code to format each library.
func formatLibraries(ctx context.Context, language string, libraries []*config.Library) error {
	for _, library := range libraries {
		switch language {
		case languageFake:
			if err := fakeFormat(library); err != nil {
				return err
			}
		case languageDart:
			if err := dart.Format(ctx, library); err != nil {
				return err
			}
		case languageGo:
			if err := golang.Format(ctx, library); err != nil {
				return err
			}
		case languageRust:
			if err := rust.Format(ctx, library); err != nil {
				return err
			}
		case languagePython:
			// TODO(https://github.com/googleapis/librarian/issues/3730): separate
			// generation and formatting for Python.
			return nil
		default:
			return fmt.Errorf("language %q does not support formatting", language)
		}
	}
	return nil
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
	case languageDart:
		return dart.DefaultOutput(name, defaultOut)
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
	case languageDart:
		return dart.DeriveAPIPath(name)
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
