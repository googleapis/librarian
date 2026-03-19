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

// The syncversion command synchronizes library versions from the legacy
// librarian state file into the new librarian.yaml configuration file.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	errRepoNotFound      = errors.New("repo argument is required")
	errVersionRegression = errors.New("version is regression")
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("syncversion", flag.ContinueOnError)
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

	state, err := yaml.Read[legacyconfig.LibrarianState](filepath.Join(abs, legacyconfig.LibrarianDir, legacyconfig.LibrarianStateFile))
	if err != nil {
		return err
	}
	cfg, err := yaml.Read[config.Config](filepath.Join(abs, "librarian.yaml"))
	if err != nil {
		return err
	}
	cfg, err = syncVersion(state, cfg)
	if err != nil {
		return err
	}
	return librarian.RunTidyOnConfig(ctx, repoPath, cfg)
}

func syncVersion(state *legacyconfig.LibrarianState, cfg *config.Config) (*config.Config, error) {
	legacyLibraries := make(map[string]string)
	for _, lib := range state.Libraries {
		legacyLibraries[lib.ID] = lib.Version
	}
	for _, lib := range cfg.Libraries {
		newVersion, ok := legacyLibraries[lib.Name]
		if !ok || newVersion == "" {
			continue
		}
		maxVersion := semver.MaxVersion(lib.Version, newVersion)
		if maxVersion == lib.Version && newVersion != lib.Version {
			// lib.Version is greater than newVersion, something is
			// wrong, fail in this case.
			return nil, errVersionRegression
		}
		lib.Version = newVersion
	}
	return cfg, nil
}
