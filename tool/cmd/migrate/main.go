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
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	sidekickFile             = ".sidekick.toml"
	googleapisArchivePrefix  = "https://github.com/googleapis/googleapis/archive/"
	showcaseArchivePrefix    = "https://github.com/googleapis/gapic-showcase/archive/"
	protobufArchivePrefix    = "https://github.com/protocolbuffers/protobuf/archive/"
	conformanceArchivePrefix = "https://github.com/protocolbuffers/protobuf/archive/"
	tarballSuffix            = ".tar.gz"
	librarianDir             = ".librarian"
	librarianStateFile       = "state.yaml"
	librarianConfigFile      = "config.yaml"
	defaultTagFormat         = "{name}/v{version}"
	googleapisRepo           = "github.com/googleapis/googleapis"
)

var (
	errRepoNotFound                = errors.New("-repo flag is required")
	errSidekickNotFound            = errors.New(".sidekick.toml not found")
	errTidyFailed                  = errors.New("librarian tidy failed")
	errUnableToCalculateOutputPath = errors.New("unable to calculate output path")
	errFetchSource                 = errors.New("cannot fetch source")

	fetchSource = fetchGoogleapis

	pathToName = map[string]string{
		"google_cloud_protojson_conformance": "google_cloud_protobuf_test_messages_proto3",
		"google_cloud_showcase_v1beta1":      "google_cloud_showcase_v1beta1",
	}

	libraryToKeep = map[string][]string{
		"google_cloud_protobuf": {"lib/src/encoding.dart"},
		"google_cloud_rpc": {
			"lib/src/exceptions.dart",
			"lib/src/versions.dart",
			"lib/src/vm.dart",
			"lib/src/web.dart",
			"lib/exceptions.dart",
			"lib/service_client.dart",
		},
		"google_cloud_showcase_v1beta1": {"dart_test.yaml"},
	}
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("migrate", flag.ContinueOnError)
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
	case "google-cloud-python", "google-cloud-go":
		parts := strings.SplitN(base, "-", 3)
		return runLibrarianMigration(ctx, parts[2], abs)
	default:
		return fmt.Errorf("invalid path: %q", repoPath)
	}
}
