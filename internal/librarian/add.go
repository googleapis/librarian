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
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/librarian/java"
	"github.com/googleapis/librarian/internal/librarian/nodejs"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/librarian/swift"
	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errAPIAlreadyExists       = errors.New("api already exists in library")
	errLibraryAlreadyExists   = errors.New("library already exists in config")
	errPreviewAlreadyExists   = errors.New("preview library config already exists")
	errPreviewRequiresLibrary = errors.New("only APIs with an existing Library can have a Preview")
	errWrongAPICount          = errors.New("must provide exactly one API path")
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "add a new client library",
		UsageText: "librarian add <api>",
		Description: `add registers a single API in librarian.yaml.

The <api> is a path within the configured googleapis source, such as
"google/cloud/secretmanager/v1". The library name and other defaults are
derived from the first API path using language-specific rules.

If the API path should naturally be included in an existing library, and if the
language supports doing so, that library is modified. Otherwise, a new library
is created.

While release-please is responsible for library releases, the relevant
release-please configuration will be updated as necessary to onboard any new
library.

To add a preview client of an existing library, prefix the API path with
"preview/".

Examples:

	librarian add google/cloud/secretmanager/v1
	librarian add preview/google/cloud/secretmanager/v1beta

A typical librarian workflow for adding a new client library is:

	librarian add <api>            # onboard a new API into librarian.yaml
	librarian generate <library>   # generate the client library`,
		Action: func(ctx context.Context, c *cli.Command) error {
			apis := c.Args().Slice()
			if len(apis) != 1 {
				return errWrongAPICount
			}
			cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				return err
			}
			return runAdd(ctx, cfg, apis[0])
		},
	}
}

func runAdd(ctx context.Context, cfg *config.Config, api string) error {
	name, cfg, err := addLibrary(cfg, api)
	if err != nil {
		return err
	}
	cfg, err = resolveDependencies(ctx, cfg, name)
	if err != nil {
		return err
	}
	if cfg.Language == config.LanguageGo || cfg.Language == config.LanguagePython || cfg.Language == config.LanguageNodejs {
		if hasBulkReleasePleaseConfigs(".", cfg) {
			if err := syncToReleasePlease(".", cfg, name); err != nil {
				return err
			}
		}
	}
	return RunTidyOnConfig(ctx, ".", cfg)
}

func resolveDependencies(ctx context.Context, cfg *config.Config, name string) (*config.Config, error) {
	switch cfg.Language {
	case config.LanguageJava:
		lib, sources, err := setupResolve(ctx, cfg, name)
		if err != nil {
			return nil, err
		}
		return java.ResolveMixinDependencies(cfg, lib, sources)
	case config.LanguageRust:
		lib, sources, err := setupResolve(ctx, cfg, name)
		if err != nil {
			return nil, err
		}
		return rust.ResolveDependencies(ctx, cfg, lib, sources)
	default:
		return cfg, nil
	}
}

func setupResolve(ctx context.Context, cfg *config.Config, name string) (*config.Library, *sources.Sources, error) {
	lib, err := FindLibrary(cfg, name)
	if err != nil {
		return nil, nil, err
	}
	sources, err := LoadSources(ctx, cfg.Sources)
	if err != nil {
		return nil, nil, err
	}
	return lib, sources, nil
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
	case config.LanguageJava:
		return java.DefaultLibraryName(api)
	case config.LanguageNodejs:
		return nodejs.DefaultLibraryName(api)
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

// addLibrary adds a new library to the config based on the provided API.
// It returns the name of the new or updated library, the updated config, and an
// error if the API cannot be added (e.g. because it already exists, or the new
// API is a preview and there is no corresponding stable library).
func addLibrary(cfg *config.Config, apiPath string) (string, *config.Config, error) {
	stablePath, isPreview := strings.CutPrefix(apiPath, "preview/")
	api := &config.API{Path: stablePath}
	existingLib := findExistingLibraryForNewAPI(cfg, stablePath)
	if isPreview {
		if existingLib == nil {
			return "", nil, fmt.Errorf("%w: API path %s", errPreviewRequiresLibrary, apiPath)
		}
		return addPreviewLibrary(cfg, existingLib, api)
	}
	if existingLib != nil {
		return updateExistingLibrary(cfg, existingLib, api)
	}
	return addNewLibrary(cfg, api)
}

// findExistingLibraryForNewAPI determines if an existing library in cfg is
// the natural library to contain apiPath, and returns it if so. If no existing
// library is found, nil is returned. In most languages this check is performed
// by deriving the library name from the API path and seeing if that library
// already exists. In Python the mapping from API path to library name isn't
// always as simple for historical reasons.
func findExistingLibraryForNewAPI(cfg *config.Config, apiPath string) *config.Library {
	switch cfg.Language {
	case config.LanguageNodejs:
		return nodejs.FindExistingLibraryForNewAPI(cfg.Libraries, apiPath)
	case config.LanguagePython:
		return python.FindExistingLibraryForNewAPI(cfg.Libraries, apiPath)
	default:
		name := deriveLibraryName(cfg.Language, apiPath)
		// Not using FindLibrary as the error handling becomes awkward.
		for _, library := range cfg.Libraries {
			if library.Name == name {
				return library
			}
		}
		return nil
	}
}

// addPreviewLibrary adds a new preview library to the config.
func addPreviewLibrary(cfg *config.Config, lib *config.Library, api *config.API) (string, *config.Config, error) {
	if lib.Preview != nil {
		return "", nil, fmt.Errorf("%w: %s", errPreviewAlreadyExists, lib.Name)
	}
	// Derive an initial version for the preview client, starting from the
	// containing stable client's version as if it were a preview, then
	// determining the actual preview version relative from the current stable
	// version. For example, if the stable was 1.0.0, the initial preview would
	// be 1.1.0-preview.1.
	v, err := semver.DeriveNextPreview(lib.Version+"-preview.1", lib.Version, languageVersioningOptions[cfg.Language])
	if err != nil {
		return "", nil, err
	}
	lib.Preview = &config.Library{
		Version: v,
		APIs:    []*config.API{api},
	}
	return lib.Name, cfg, nil
}

// addNewLibrary adds a new library to the config.
func addNewLibrary(cfg *config.Config, api *config.API) (string, *config.Config, error) {
	name := deriveLibraryName(cfg.Language, api.Path)
	lib := &config.Library{
		Name:          name,
		CopyrightYear: strconv.Itoa(time.Now().Year()),
		APIs:          []*config.API{api},
	}
	switch cfg.Language {
	case config.LanguageGo:
		lib = golang.Add(lib)
	case config.LanguageJava:
		lib = java.Add(lib)
	case config.LanguagePython:
		var err error
		lib, err = python.Add(cfg, lib)
		if err != nil {
			return "", nil, err
		}
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

func updateExistingLibrary(cfg *config.Config, existingLib *config.Library, api *config.API) (string, *config.Config, error) {
	if slices.ContainsFunc(existingLib.APIs, func(a *config.API) bool { return api.Path == a.Path }) {
		return "", nil, fmt.Errorf("%w: %s in library %s", errAPIAlreadyExists, api.Path, existingLib.Name)
	}
	switch cfg.Language {
	case config.LanguagePython:
		if err := python.ValidateNewAPIs(existingLib); err != nil {
			return "", nil, err
		}
		existingLib.APIs = append(existingLib.APIs, api)
	case config.LanguageGo:
		existingLib.APIs = append(existingLib.APIs, api)
		existingLib = golang.Add(existingLib)
	case config.LanguageJava, config.LanguageNodejs:
		existingLib.APIs = append(existingLib.APIs, api)
	default:
		return "", nil, fmt.Errorf("%w: %s", errLibraryAlreadyExists, existingLib.Name)
	}
	return existingLib.Name, cfg, nil
}
