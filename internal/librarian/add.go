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
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/librarian/swift"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errLibraryAlreadyExists      = errors.New("library already exists in config")
	errMissingAPI                = errors.New("must provide at least one API")
	errMixedPreviewAndNonPreview = errors.New("cannot mix preview and non-preview APIs")
	errPreviewRequiresLibrary    = errors.New("only APIs with an existing Library can have a Preview")
	errPreviewAlreadyExists      = errors.New("preview library config already exists")
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "add a new client library to librarian.yaml",
		UsageText: "librarian add <apis...>",
		Action: func(ctx context.Context, c *cli.Command) error {
			apis := c.Args().Slice()
			if len(apis) == 0 {
				return errMissingAPI
			}
			cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				return err
			}
			return runAdd(ctx, cfg, apis...)
		},
	}
}

func runAdd(ctx context.Context, cfg *config.Config, apis ...string) error {
	name, cfg, err := addLibrary(cfg, apis...)
	if err != nil {
		return err
	}
	cfg, err = resolveDependencies(ctx, cfg, name)
	if err != nil {
		return err
	}
	switch cfg.Language {
	case config.LanguageGo:
		// TODO(https://github.com/googleapis/librarian/issues/5029): Remove this function after
		// fully migrating off legacylibrarian.
		if err := syncToStateYAML(".", cfg); err != nil {
			return err
		}
	}
	return RunTidyOnConfig(ctx, ".", cfg)
}

func resolveDependencies(ctx context.Context, cfg *config.Config, name string) (*config.Config, error) {
	switch cfg.Language {
	case config.LanguageRust:
		lib, err := FindLibrary(cfg, name)
		if err != nil {
			return nil, err
		}
		sources, err := LoadSources(ctx, cfg.Sources)
		if err != nil {
			return nil, err
		}
		return rust.ResolveDependencies(ctx, cfg, lib, sources)
	default:
		return cfg, nil
	}
}

// deriveLibraryName derives a library name from an API path.
// The derivation is language-specific.
func deriveLibraryName(language string, api string) string {
	switch language {
	case config.LanguageDart:
		return dart.DefaultLibraryName(api)
	case config.LanguageFake:
		return fakeDefaultLibraryName(api)
	case config.LanguageGo:
		return golang.DefaultLibraryName(api)
	case config.LanguagePython:
		return python.DefaultLibraryName(api)
	case config.LanguageRust:
		return rust.DefaultLibraryName(api)
	case config.LanguageSwift:
		return swift.DefaultLibraryName(api)
	default:
		return strings.ReplaceAll(api, "/", "-")
	}
}

// addLibrary adds a new library to the config based on the provided APIs.
// It returns the name of the new library, the updated config, and an error
// if the library already exists.
func addLibrary(cfg *config.Config, apis ...string) (string, *config.Config, error) {
	isPreview := slices.ContainsFunc(apis, func(a string) bool {
		return strings.HasPrefix(a, "preview/")
	})
	mixed := slices.ContainsFunc(apis, func(a string) bool {
		return isPreview && !strings.HasPrefix(a, "preview/")
	})
	if mixed {
		return "", nil, errMixedPreviewAndNonPreview
	}

	paths := make([]*config.API, 0, len(apis))
	for _, a := range apis {
		if isPreview {
			a = strings.TrimPrefix(a, "preview/")
		}
		paths = append(paths, &config.API{Path: a})
	}

	name := deriveLibraryName(cfg.Language, paths[0].Path)

	if isPreview {
		lib, err := FindLibrary(cfg, name)
		if err != nil {
			return "", nil, fmt.Errorf("%s: %w", name, errPreviewRequiresLibrary)
		}
		if lib.Preview != nil {
			return "", nil, fmt.Errorf("%s: %w", name, errPreviewAlreadyExists)
		}
		lib.Preview = &config.Library{
			APIs: paths,
		}
		return name, cfg, nil
	}

	exists := slices.ContainsFunc(cfg.Libraries, func(lib *config.Library) bool {
		return lib.Name == name
	})
	if exists {
		return "", nil, fmt.Errorf("%w: %s", errLibraryAlreadyExists, name)
	}

	lib := &config.Library{
		Name:          name,
		CopyrightYear: strconv.Itoa(time.Now().Year()),
		APIs:          paths,
	}

	switch cfg.Language {
	case config.LanguageGo:
		lib = golang.Add(lib)
	case config.LanguageRust:
		lib = rust.Add(lib)
	case config.LanguageFake:
		lib = fakeAdd(lib, defaultVersion)
	}

	cfg.Libraries = append(cfg.Libraries, lib)
	sort.Slice(cfg.Libraries, func(i, j int) bool {
		return cfg.Libraries[i].Name < cfg.Libraries[j].Name
	})
	return name, cfg, nil
}

// syncToStateYAML updates the .librarian/state.yaml with any new libraries.
func syncToStateYAML(repoDir string, cfg *config.Config) error {
	stateFile := filepath.Join(repoDir, legacyconfig.LibrarianDir, legacyconfig.LibrarianStateFile)
	state, err := yaml.Read[legacyconfig.LibrarianState](stateFile)
	if err != nil {
		return err
	}
	for _, lib := range cfg.Libraries {
		if state.LibraryByID(lib.Name) != nil {
			continue
		}
		state.Libraries = append(state.Libraries, createLegacyLibrary(lib))
	}
	sort.Slice(state.Libraries, func(i, j int) bool {
		return state.Libraries[i].ID < state.Libraries[j].ID
	})
	return yaml.Write(stateFile, state)
}

func createLegacyLibrary(lib *config.Library) *legacyconfig.LibraryState {
	return &legacyconfig.LibraryState{
		ID:          lib.Name,
		Version:     lib.Version,
		SourceRoots: []string{lib.Name},
		TagFormat:   "{id}/v{version}",
	}
}
