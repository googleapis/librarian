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

// import-metadata is a tool to import metadata from existing
// .repo-metadata.json files
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

func main() {
	ctx := context.Background()
	cmd := &cli.Command{
		Name:  "import-metadata",
		Usage: "commands for import configs",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "python-repo",
				Usage:    "path to the python repository",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "librarian-repo",
				Usage:    "path to the librarian repository",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			pythonRepoDir := cmd.String("python-repo")
			librarianRepoDir := cmd.String("librarian-repo")
			return importMetadata(ctx, pythonRepoDir, librarianRepoDir)
		},
	}
	if err := cmd.Run(ctx, os.Args); err != nil {
		log.Fatalf("import-metadata: %v", err)
	}
}

// importMetadata examines every API in serviceconfig.APIs. If the API is
// configured as part of any library within the Python librarian config, and
// if resolving the API (i.e. loading its service config) doesn't provide either
// NewIssueURI or DocumentationURI (indicating that the service config doesn't)
// have a publishing section) then the repo metadata from the Python library is
// used, and overrides for those fields are added to the API. The file is then
// saved back to internal/serviceconfig/sdk.yaml.
func importMetadata(ctx context.Context, pythonRepoDir, librarianRepoDir string) error {
	pythonCfg, err := yaml.Read[config.Config](filepath.Join(pythonRepoDir, "librarian.yaml"))
	if err != nil {
		return fmt.Errorf("error loading Python librarian configuration: %w", err)
	}
	defaultOutputDir := filepath.Join(pythonRepoDir, pythonCfg.Default.Output)

	googleapisDir, _, err := librarian.LoadSources(ctx, pythonCfg)
	if err != nil {
		return fmt.Errorf("error loading sources: %w", err)
	}

	apiPathToLibrary := make(map[string]*config.Library)
	for _, library := range pythonCfg.Libraries {
		for _, libraryAPI := range library.APIs {
			apiPathToLibrary[libraryAPI.Path] = library
		}
	}

	apis := serviceconfig.APIs
	for index, api := range apis {
		library, found := apiPathToLibrary[api.Path]
		if !found {
			continue
		}
		resolvedAPI, err := serviceconfig.Find(googleapisDir, api.Path, pythonCfg.Language)
		if err != nil {
			return fmt.Errorf("error finding service for path %s: %w", api.Path, err)
		}
		if resolvedAPI.NewIssueURI != "" || resolvedAPI.DocumentationURI != "" {
			continue
		}
		libraryOutputDir := python.DefaultOutputByName(library.Name, defaultOutputDir)
		repoMetadata, err := repometadata.Read(libraryOutputDir)
		if err != nil {
			return fmt.Errorf("error finding service for path %s: %w", api.Path, err)
		}
		api.DocumentationURI = repoMetadata.ProductDocumentation
		api.NewIssueURI = repoMetadata.IssueTracker
		apis[index] = api
	}
	yamlFile := filepath.Join(librarianRepoDir, "internal", "serviceconfig", "sdk.yaml")
	if err := yaml.Write(yamlFile, serviceconfig.APIs); err != nil {
		return fmt.Errorf("error writing YAML file: %w", err)
	}

	return nil
}
