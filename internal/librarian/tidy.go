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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
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
		removeDerivableChannelFields(lib)
	}
	return yaml.Write(librarianConfigPath, formatConfig(cfg))
}

// removeDerivableChannelFields removes 'path' and 'service_config' fields from a library's channels
// if they can be derived from the library's name.
//
// Derivation Rules:
//   - ch.Path: Derived by replacing all hyphens ('-') in the library's name with forward slashes ('/').
//     If an explicitly set path matches this derived value, it is removed.
//   - ch.ServiceConfig: Derived using the resolved path (either original or derived).
//     The pattern is '[resolved_path]/[service_name]_[version].yaml'.
//     If an explicitly set service_config matches this pattern, it is removed.
func removeDerivableChannelFields(lib *config.Library) {
	derivedPath := strings.ReplaceAll(lib.Name, "-", "/")
	for _, ch := range lib.Channels {
		resolvedPath := ch.Path
		if resolvedPath == "" {
			resolvedPath = derivedPath
		}

		if ch.ServiceConfig != "" {
			parts := strings.Split(resolvedPath, "/")
			if len(parts) >= 2 {
				version := parts[len(parts)-1]
				service := parts[len(parts)-2]
				if strings.HasPrefix(version, "v") {
					derivedSC := fmt.Sprintf("%s/%s_%s.yaml", resolvedPath, service, version)
					if ch.ServiceConfig == derivedSC {
						ch.ServiceConfig = ""
					}
				}
			}
		}

		if ch.Path == derivedPath {
			ch.Path = ""
		}
	}
	lib.Channels = slices.DeleteFunc(lib.Channels, func(ch *config.Channel) bool {
		return ch.Path == "" && ch.ServiceConfig == ""
	})
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
