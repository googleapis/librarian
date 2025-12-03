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

// Command migrate-librarian is a tool for migrating .librarian/state.yaml and .librarian/config.yaml to librarian
// configuration.
package main

import (
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	librarianDir        = ".librarian"
	librarianStateFile  = "state.yaml"
	librarianConfigFile = "config.yaml"
)

var (
	errRepoNotFound = errors.New("-repo flag is required")
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("migrate-librarian failed", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flagSet := flag.NewFlagSet("migrate-librarian", flag.ContinueOnError)
	repoPath := flagSet.String("repo", "", "Path to the google-cloud-rust repository (required)")
	outputPath := flagSet.String("output", "./.librarian.yaml", "Output file path (default: ./.librarian.yaml)")
	if err := flagSet.Parse(args[1:]); err != nil {
		return err
	}
	if *repoPath == "" {
		return errRepoNotFound
	}

	librarianState, err := readState(*repoPath)
	if err != nil {
		return err
	}

	librarianConfig, err := readConfig(*repoPath)
	if err != nil {
		return err
	}

	cfg := buildConfig(librarianState, librarianConfig)

	return yaml.Write(*outputPath, cfg)
}

func buildConfig(state *legacyconfig.LibrarianState, config *legacyconfig.LibrarianConfig) *config.Config {

}

func readState(path string) (*legacyconfig.LibrarianState, error) {
	stateFile := filepath.Join(path, librarianDir, librarianStateFile)
	return yaml.Read[legacyconfig.LibrarianState](stateFile)
}

func readConfig(path string) (*legacyconfig.LibrarianConfig, error) {
	configFile := filepath.Join(path, librarianDir, librarianConfigFile)
	return yaml.Read[legacyconfig.LibrarianConfig](configFile)
}
