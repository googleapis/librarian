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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errDuplicateLibraryName  = errors.New("duplicate library name")
	errDuplicateAPIPath      = errors.New("duplicate api path")
	errNoGoogleapiSourceInfo = errors.New("googleapis source not configured in librarian.yaml")
)

func tidyCommand() *cli.Command {
	return &cli.Command{
		Name:      "tidy",
		Usage:     "format and validate librarian.yaml",
		UsageText: "librarian tidy",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			return RunTidyOnConfig(ctx, ".", cfg)
		},
	}
}

// RunTidyOnConfig formats and validates the provided librarian configuration
// and writes it to disk, relative to the specified repository root directory.
func RunTidyOnConfig(ctx context.Context, repoDir string, cfg *config.Config) error {
	if err := validateLibraries(cfg); err != nil {
		return err
	}

	if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
		return errNoGoogleapiSourceInfo
	}
	cfg.Libraries = tidyLibraries(cfg)
	return yaml.Write(filepath.Join(repoDir, librarianConfigPath), formatConfig(cfg))
}

func tidyLibraries(cfg *config.Config) []*config.Library {
	for i, lib := range cfg.Libraries {
		cfg.Libraries[i] = tidyLibrary(cfg, lib)
	}
	return cfg.Libraries
}

func tidyLibrary(cfg *config.Config, lib *config.Library) *config.Library {
	if lib.Output != "" && len(lib.APIs) == 1 && isDerivableOutput(cfg, lib) {
		lib.Output = ""
	}
	if isVeneer(cfg.Language, lib) {
		// Veneers are never generated, so ensure skip_generate is false.
		lib.SkipGenerate = false
	}
	if len(lib.Roots) == 1 && lib.Roots[0] == "googleapis" {
		lib.Roots = nil
	}
	if lib.SpecificationFormat == config.SpecProtobuf {
		lib.SpecificationFormat = ""
	}
	// Only remove derivable API paths when there's exactly one API.
	// When there are multiple APIs, preserve all of them.
	if len(lib.APIs) == 1 && canDeriveAPIPath(cfg.Language) {
		if lib.APIs[0].Path == deriveAPIPath(cfg.Language, lib.Name) {
			lib.APIs[0].Path = ""
		}
	}
	lib.APIs = slices.DeleteFunc(lib.APIs, func(ch *config.API) bool {
		return ch.Path == ""
	})
	if cfg.Default != nil && lib.ReleaseLevel != "" && lib.ReleaseLevel == cfg.Default.ReleaseLevel {
		lib.ReleaseLevel = ""
	}
	return tidyLanguageConfig(lib, cfg.Language)
}

func isDerivableOutput(cfg *config.Config, lib *config.Library) bool {
	derivedOutput := defaultOutput(cfg.Language, lib.Name, lib.APIs[0].Path, cfg.Default.Output)
	return lib.Output == derivedOutput
}

func validateLibraries(cfg *config.Config) error {
	var (
		errs      []error
		nameCount = make(map[string]int)
		pathCount = make(map[string]int)
	)
	for _, lib := range cfg.Libraries {
		if lib.Name != "" {
			nameCount[lib.Name]++
		}
		for _, ch := range lib.APIs {
			if ch.Path != "" {
				pathCount[ch.Path]++
			}
		}
	}
	for name, count := range nameCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("%w: %s (appears %d times)", errDuplicateLibraryName, name, count))
		}
	}
	for path, count := range pathCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("%w: %s (appears %d times)", errDuplicateAPIPath, path, count))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// languageTidiers maps a language to a function that tidies the language-specific
// configuration.
var languageTidiers = map[string]func(*config.Library) *config.Library{
	config.LanguageGo:   golang.Tidy,
	config.LanguageRust: tidyRustConfig,
}

// tidyLanguageConfig finds and executes the language-specific tidier for a library.
func tidyLanguageConfig(lib *config.Library, language string) *config.Library {
	if tidier, ok := languageTidiers[language]; ok {
		return tidier(lib)
	}
	return lib
}

// isEmptyRustModule returns true if the module is a placeholder that can be removed.
func isEmptyRustModule(module *config.RustModule) bool {
	if module.Language == config.LanguageRustStorage {
		// The Rust storage module has hardcoded API paths and templates, so it is never empty.
		return false
	}
	return module.APIPath == "" && module.Template == ""
}

// deleteEmptyRustModules returns a new slice of modules with empty modules removed.
func deleteEmptyRustModules(modules []*config.RustModule) []*config.RustModule {
	return slices.DeleteFunc(modules, isEmptyRustModule)
}

func tidyRustConfig(lib *config.Library) *config.Library {
	if lib.Rust != nil && lib.Rust.Modules != nil {
		lib.Rust.Modules = deleteEmptyRustModules(lib.Rust.Modules)
	}

	// TODO(https://github.com/googleapis/librarian/issues/4276): Remove veneer field
	lib.Veneer = false
	return lib
}

func formatConfig(cfg *config.Config) *config.Config {
	if cfg.Default != nil && cfg.Default.Rust != nil {
		slices.SortFunc(cfg.Default.Rust.PackageDependencies, func(a, b *config.RustPackageDependency) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	slices.SortFunc(cfg.Libraries, func(a, b *config.Library) int {
		return strings.Compare(a.Name, b.Name)
	})
	for _, lib := range cfg.Libraries {
		serviceconfig.SortAPIs(lib.APIs)
		if lib.Rust != nil {
			slices.SortFunc(lib.Rust.PackageDependencies, func(a, b *config.RustPackageDependency) int {
				return strings.Compare(a.Name, b.Name)
			})
		}
	}
	return cfg
}
