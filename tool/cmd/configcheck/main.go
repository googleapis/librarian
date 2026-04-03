// Copyright 2026 Google LLC
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

// Command configcheck is a tool to verify the consistency of library versions
// between librarian.yaml and .librarian/state.yaml.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	errRepoNotFound               = errors.New("repo argument is required")
	errLibNotFoundInLibrarianYAML = errors.New("library not found in librarian.yaml")
	errLibNotFoundInStateYAML     = errors.New("library not found in state.yaml")
	errLibraryVersionNotSame      = errors.New("library version not same")
	errLibraryAPINotSame          = errors.New("library API not same")
)

type library struct {
	version string
	apis    []string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	flagSet := flag.NewFlagSet("configcheck", flag.ContinueOnError)
	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if flagSet.NArg() < 1 {
		return errRepoNotFound
	}

	repoPath := flagSet.Arg(0)
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	stateFile := filepath.Join(abs, legacyconfig.LibrarianDir, legacyconfig.LibrarianStateFile)
	state, err := yaml.Read[legacyconfig.LibrarianState](stateFile)
	if err != nil {
		return err
	}
	cfg, err := yaml.Read[config.Config](filepath.Join(abs, config.LibrarianYAML))
	if err != nil {
		return err
	}
	return configCheck(state, cfg)
}

// configCheck verifies that the libraries and their versions defined in the
// state.yaml match those in librarian.yaml.
func configCheck(state *legacyconfig.LibrarianState, cfg *config.Config) error {
	legacyLibs := convertLegacyLibs(state.Libraries)
	libs := convertLibs(cfg.Libraries)
	for id, legacyLib := range legacyLibs {
		lib, ok := libs[id]
		if !ok {
			return fmt.Errorf("library %s: %w", id, errLibNotFoundInLibrarianYAML)
		}
		if lib.version != legacyLib.version {
			return fmt.Errorf("library %s: %w", id, errLibraryVersionNotSame)
		}
		if !slices.Equal(lib.apis, legacyLib.apis) {
			return fmt.Errorf("library %s: %w", id, errLibraryAPINotSame)
		}
	}
	for name := range libs {
		if _, ok := legacyLibs[name]; !ok {
			return fmt.Errorf("library %s: %w", name, errLibNotFoundInStateYAML)
		}
	}
	return nil
}

func convertLegacyLibs(libs []*legacyconfig.LibraryState) map[string]*library {
	res := make(map[string]*library)
	for _, lib := range libs {
		apis := make([]string, 0, len(lib.APIs))
		for _, api := range lib.APIs {
			apis = append(apis, api.Path)
		}
		slices.Sort(apis)
		res[lib.ID] = &library{
			version: lib.Version,
			apis:    apis,
		}
	}
	return res
}

func convertLibs(libs []*config.Library) map[string]*library {
	res := make(map[string]*library)
	for _, lib := range libs {
		apis := make([]string, 0, len(lib.APIs))
		for _, api := range lib.APIs {
			apis = append(apis, api.Path)
		}
		slices.Sort(apis)
		res[lib.Name] = &library{
			version: lib.Version,
			apis:    apis,
		}
	}
	return res
}
