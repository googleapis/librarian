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
	"strings"

	"cloud.google.com/go/librarian/internal/config"
	"cloud.google.com/go/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errDuplicateLibraryName = errors.New("duplicate library name")
	errDuplicateChannelPath = errors.New("duplicate channel path")
)

func tidyCommand() *cli.Command {
	return &cli.Command{
		Name:      "tidy",
		Usage:     "format and validate librarian.yaml",
		UsageText: "librarian tidy [path]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return RunTidy()
		},
	}
}

// RunTidy formats and validates the librarian configuration file.
func RunTidy() error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	if err := validateLibraries(cfg); err != nil {
		return err
	}
	for _, lib := range cfg.Libraries {
		if lib.Output != "" && len(lib.Channels) == 1 && isDerivableOutput(cfg, lib) {
			lib.Output = ""
		}
		for _, ch := range lib.Channels {
			if isDerivableChannelPath(cfg.Language, lib, ch) {
				ch.Path = ""
			}
			if isDerivableServiceConfig(cfg.Language, lib, ch) {
				ch.ServiceConfig = ""
			}
		}
		lib.Channels = slices.DeleteFunc(lib.Channels, func(ch *config.Channel) bool {
			return ch.Path == "" && ch.ServiceConfig == "" && !ch.ServiceConfigDoesNotExist
		})

		tidyLanguageConfig(lib, cfg.Language)
	}
	return yaml.Write(librarianConfigPath, formatConfig(cfg))
}

func isDerivableOutput(cfg *config.Config, lib *config.Library) bool {
	derivedOutput := defaultOutput(cfg.Language, lib.Channels[0].Path, cfg.Default.Output)
	return lib.Output == derivedOutput
}

func isDerivableChannelPath(language string, lib *config.Library, ch *config.Channel) bool {
	return ch.Path == deriveChannelPath(language, lib)
}

func isDerivableServiceConfig(language string, lib *config.Library, ch *config.Channel) bool {
	path := ch.Path
	if path == "" {
		path = deriveChannelPath(language, lib)
	}
	return ch.ServiceConfig != "" && ch.ServiceConfig == deriveServiceConfig(path)
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
		for _, ch := range lib.Channels {
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
			errs = append(errs, fmt.Errorf("%w: %s (appears %d times)", errDuplicateChannelPath, path, count))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func tidyLanguageConfig(lib *config.Library, language string) {
	switch language {
	case "rust":
		tidyRustConfig(lib)
	}
}

func tidyRustConfig(lib *config.Library) {
	if lib.Rust != nil && lib.Rust.Modules != nil {
		lib.Rust.Modules = slices.DeleteFunc(lib.Rust.Modules, func(module *config.RustModule) bool {
			return module.Source == "none" && module.Template == ""
		})
	}
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
		slices.SortFunc(lib.Channels, func(a, b *config.Channel) int {
			return strings.Compare(a.Path, b.Path)
		})
		if lib.Rust != nil {
			slices.SortFunc(lib.Rust.PackageDependencies, func(a, b *config.RustPackageDependency) int {
				return strings.Compare(a.Name, b.Name)
			})
		}
	}
	return cfg
}
