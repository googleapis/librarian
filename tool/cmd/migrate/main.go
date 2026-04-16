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

// Command migrate is a tool for migrating .sidekick.toml or .librarian configuration to librarian.yaml.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/bazelbuild/buildtools/build"
)

const (
	librarianDir        = ".librarian"
	librarianStateFile  = "state.yaml"
	librarianConfigFile = "config.yaml"
	defaultTagFormat    = "{name}/v{version}"
	googleapisRepo      = "github.com/googleapis/googleapis"
)

var (
	errRepoNotFound = errors.New("repo argument is required")
	errTidyFailed   = errors.New("librarian tidy failed")
	errFetchSource  = errors.New("cannot fetch source")
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string) error {
	// TODO(https://github.com/googleapis/librarian/issues/4567): change this
	// to use github.com/urfave/cli/v3 consistently with other tooling.
	flagSet := flag.NewFlagSet("migrate", flag.ContinueOnError)
	insertMarkersFlag := flagSet.Bool("insert-markers", false, "whether to insert markers in Java pom.xml files")
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
	base := filepath.Base(abs)
	switch base {
	case "google-cloud-java":
		return runJavaMigration(ctx, abs, *insertMarkersFlag)
	case "google-cloud-node":
		return runNodejsMigration(ctx, abs)
	case "google-cloud-dotnet":
		return runDotnetMigration(ctx, abs)
	default:
		return fmt.Errorf("invalid path: %q", repoPath)
	}
}

func parseBazel(googleapisDir, dir string) (*build.File, error) {
	path := filepath.Join(googleapisDir, dir, "BUILD.bazel")
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file, err := build.ParseBuild(path, data)
	if err != nil {
		return nil, err
	}
	return file, nil
}
