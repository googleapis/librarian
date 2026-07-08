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

// Package php provides PHP specific functionality for librarian.
package php

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

const (
	generatorVersion = "v1.21.2"
	generatorSHA256  = "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518"
)

// Generate generates a PHP client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) (err error) {
	if len(library.APIs) == 0 {
		return fmt.Errorf("no apis configured for library %q", library.Name)
	}
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		return fmt.Errorf("failed to find protoc: %w", err)
	}

	// Fetch PHP generator and install dependencies
	// Temp step, remove once install is ready
	generatorDir, err := installGenerator(ctx)
	if err != nil {
		return err
	}

	// Setup sandbox staging dir
	tempDir, err := os.MkdirTemp("", "librarian-php-")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
	}()

	wrapperPath := filepath.Join(generatorDir, "wrapper.sh")
	outputZipPath := filepath.Join(tempDir, "output.zip")
	srcCfg := sources.NewSourceConfig(src, library.Roots)
	for _, api := range library.APIs {
		params := &generateAPIParams{
			api:           api,
			library:       library,
			srcCfg:        srcCfg,
			wrapperPath:   wrapperPath,
			outputZipPath: outputZipPath,
			protocPath:    protocPath,
		}
		if err := generateAPI(ctx, params); err != nil {
			return err
		}
		// Cleanup output zip for subsequent APIs in the same library package
		_ = os.Remove(outputZipPath)
	}

	return nil
}

type generateAPIParams struct {
	api           *config.API
	library       *config.Library
	srcCfg        *sources.SourceConfig
	wrapperPath   string
	outputZipPath string
	protocPath    string
}

// generateAPI generates a single target API by resolving its service config, gathering
// all target proto files, and executing the PHP generator plugin via protoc.
// It extracts the resulting ZIP archive directly to the library output directory.
func generateAPI(ctx context.Context, params *generateAPIParams) error {
	googleapisDir := params.srcCfg.Root("googleapis")
	// Resolve service config files
	grpcConfigPath, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, params.api.Path)
	if err != nil {
		return err
	}
	apiMetadata, err := serviceconfig.Find(googleapisDir, params.api.Path, config.LanguagePhp)
	if err != nil {
		return err
	}
	opts := gapicOpts(params.api, apiMetadata, grpcConfigPath)

	// Gather target protos
	var targetProtos []string
	apiDir := filepath.Join(googleapisDir, params.api.Path)
	protos, err := gatherProtos(apiDir)
	if err != nil {
		return err
	}
	targetProtos = append(targetProtos, protos...)
	// Always include common resources if present
	commonResources := filepath.Join(googleapisDir, "google/cloud/common_resources.proto")
	if _, err := os.Stat(commonResources); err == nil {
		targetProtos = append(targetProtos, commonResources)
	}
	if len(targetProtos) == 0 {
		return fmt.Errorf("no target protos found for API %s", params.api.Path)
	}
	// Build protoc command arguments
	gapicOutArg := fmt.Sprintf("--gapic_out=%s:%s", strings.Join(opts, ","), params.outputZipPath)
	protocArgs := []string{
		"--experimental_allow_proto3_optional",
		"--plugin=protoc-gen-gapic=" + params.wrapperPath,
		gapicOutArg,
	}

	// Append active root directories as include paths (-I) to resolve proto imports.
	for _, root := range params.srcCfg.ActiveRoots {
		if r := params.srcCfg.Root(root); r != "" {
			protocArgs = append(protocArgs, "-I", r)
		}
	}
	protocArgs = append(protocArgs, targetProtos...)
	// Run compilation
	if err := command.RunWithEnv(ctx, map[string]string{"GOOGLEAPIS_DIR": googleapisDir}, params.protocPath, protocArgs...); err != nil {
		return fmt.Errorf("failed to generate PHP API %s: %w", params.api.Path, err)
	}

	// Extract output
	outDir := params.library.Output
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}
	if err := filesystem.Unzip(ctx, params.outputZipPath, outDir); err != nil {
		return fmt.Errorf("failed to extract generated output to %s: %w", outDir, err)
	}

	return nil
}

// gatherProtos walks the directory tree recursively from root and returns
// a sorted list of absolute paths for all found proto files.
func gatherProtos(root string) ([]string, error) {
	var protos []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type().IsRegular() && filepath.Ext(path) == ".proto" {
			protos = append(protos, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(protos)
	return protos, nil
}

func gapicOpts(api *config.API, apiMetadata *serviceconfig.API, grpcConfigPath string) []string {
	transport := serviceconfig.GRPCRest
	if apiMetadata != nil {
		transport = apiMetadata.Transport(config.LanguagePhp)
	}
	opts := []string{"metadata", "transport=" + string(transport)}
	migrationMode := "NEW_SURFACE_ONLY"
	if api.PHP != nil && api.PHP.MigrationMode != "" {
		migrationMode = api.PHP.MigrationMode
	}
	opts = append(opts, "migration-mode="+migrationMode)
	if apiMetadata != nil && apiMetadata.HasRESTNumericEnums(config.LanguagePhp) {
		opts = append(opts, "rest-numeric-enums")
	}
	opts = append(opts, "generate-snippets")

	if grpcConfigPath != "" {
		opts = append(opts, "grpc_service_config="+grpcConfigPath)
	}
	if apiMetadata != nil && apiMetadata.ServiceConfig != "" {
		opts = append(opts, "service_yaml="+apiMetadata.ServiceConfig)
	}

	return opts
}

// installGenerator fetches the PHP generator and installs its Composer dependencies.
// TODO(https://github.com/googleapis/librarian/issues/6630): remove after install command is ready.
func installGenerator(ctx context.Context) (string, error) {
	phpPath, err := exec.LookPath("php")
	if err != nil {
		return "", fmt.Errorf("php is required on host to run PHP generator: %w", err)
	}

	repo := "github.com/googleapis/gapic-generator-php"
	generatorDir, err := fetch.Repo(ctx, repo, generatorVersion, generatorSHA256)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PHP generator: %w", err)
	}
	// Ensure Composer dependencies are installed
	vendorDir := filepath.Join(generatorDir, "vendor")
	if _, err := os.Stat(vendorDir); os.IsNotExist(err) {
		composerPhar := filepath.Join(generatorDir, "rules_php_gapic", "resources", "composer.phar")
		if _, err := os.Stat(composerPhar); err == nil {
			if err := command.RunInDir(ctx, generatorDir, phpPath, composerPhar, "install"); err != nil {
				return "", fmt.Errorf("failed to run composer.phar install: %w", err)
			}
		} else {
			if _, err := exec.LookPath("composer"); err != nil {
				return "", fmt.Errorf("neither composer.phar nor system composer was found: %w", err)
			}
			if err := command.RunInDir(ctx, generatorDir, "composer", "install"); err != nil {
				return "", fmt.Errorf("failed to run composer install: %w", err)
			}
		}
	}

	// Write wrapper script
	wrapperPath := filepath.Join(generatorDir, "wrapper.sh")
	wrapperContent := fmt.Sprintf(`#!/bin/bash
exec %q -d display_errors=stderr -d memory_limit=1024M %q --side_loaded_root_dir "$GOOGLEAPIS_DIR" "$@"
`, phpPath, filepath.Join(generatorDir, "src/Main.php"))

	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755); err != nil {
		return "", fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return generatorDir, nil
}
