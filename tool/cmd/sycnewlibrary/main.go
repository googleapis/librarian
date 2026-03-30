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

// The syncnewlibrary command creates a new library configuration to legacylibrarian state.yaml
// based on the library name in librarian.yaml.
package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	errRepoNotFound = errors.New("repo argument is required")
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	flagSet := flag.NewFlagSet("syncnewlibrary", flag.ContinueOnError)
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
	cfg, err := yaml.Read[config.Config](filepath.Join(abs, "librarian.yaml"))
	if err != nil {
		return err
	}
	state = syncNewLibrary(state, cfg)
	return yaml.Write(stateFile, state)
}

func syncNewLibrary(state *legacyconfig.LibrarianState, cfg *config.Config) *legacyconfig.LibrarianState {
	for _, lib := range cfg.Libraries {
		legacyLib := state.LibraryByID(lib.Name)
		if legacyLib != nil {
			continue
		}
		state.Libraries = append(state.Libraries, createLegacyGoLibrary(lib.Name, lib.Version))
	}
	sort.Slice(state.Libraries, func(i, j int) bool {
		return state.Libraries[i].ID < state.Libraries[j].ID
	})
	return state
}

func createLegacyGoLibrary(id, version string) *legacyconfig.LibraryState {
	return &legacyconfig.LibraryState{
		ID:          id,
		Version:     version,
		SourceRoots: []string{id},
		TagFormat:   "{id}/v{version}",
	}
}
