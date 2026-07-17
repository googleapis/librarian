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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/tool/protoc"
)

const (
	commonResourcesProto = "google/cloud/common_resources.proto"
	owlBotStagingDir     = "owl-bot-staging"
)

var (
	errCommonResourcesUnconfigured = errors.New("common_resources must be set (either per-API or globally under default.php)")
	errMissingStagingSubdir        = errors.New("staging_subdir is required for PHP configurations")
)

// Generate generates a PHP client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) (err error) {
	if len(library.APIs) == 0 {
		return fmt.Errorf("no apis configured for library %q", library.Name)
	}
	if cfg.Tools == nil || cfg.Tools.Protoc == nil {
		if _, err := exec.LookPath("protoc"); err != nil {
			return fmt.Errorf("failed to find protoc: %w", err)
		}
	}

	// Locate PHP generator
	// TODO(https://github.com/googleapis/librarian/issues/6629 & 6630): remove this wrapper path once `generate` is done
	// and we're ready to migrate onto `install`
	//
	var wrapperPath string
	bin, err := binDir()
	if err == nil {
		dynamicWrapper := filepath.Join(bin, "gapic-generator-php")
		if _, err := os.Stat(dynamicWrapper); err == nil {
			wrapperPath = dynamicWrapper
		}
	}

	if wrapperPath == "" {
		generatorDir, err := generatorDir(ctx)
		if err != nil {
			return fmt.Errorf("failed to locate PHP generator: %w", err)
		}
		wrapperPath = filepath.Join(generatorDir, "wrapper.sh")
		if _, err := os.Stat(wrapperPath); err != nil {
			return fmt.Errorf("PHP generator wrapper not found (did you run 'librarian install'?): %w", err)
		}
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

	stagingDir := filepath.Join(owlBotStagingDir, library.Name)
	if err := os.RemoveAll(stagingDir); err != nil {
		return err
	}
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return err
	}
	outputZipPath := filepath.Join(tempDir, "output.zip")
	srcCfg := sources.NewSourceConfig(src, library.Roots)
	for _, api := range library.APIs {
		if api.PHP == nil || api.PHP.StagingSubdir == "" {
			return fmt.Errorf("API %q: %w", api.Path, errMissingStagingSubdir)
		}
		destDir := filepath.Join(stagingDir, api.PHP.StagingSubdir)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		params := &generateAPIParams{
			cfg:           cfg,
			api:           api,
			library:       library,
			srcCfg:        srcCfg,
			wrapperPath:   wrapperPath,
			outputZipPath: outputZipPath,
			destDir:       destDir,
		}
		if err := generateAPI(ctx, params); err != nil {
			return err
		}
		// Cleanup output zip for subsequent APIs in the same library package
		_ = os.Remove(outputZipPath)
	}
	if err := postProcessLibrary(ctx, library); err != nil {
		return fmt.Errorf("failed to postprocess: %w", err)
	}
	return nil
}

type generateAPIParams struct {
	cfg           *config.Config
	api           *config.API
	library       *config.Library
	srcCfg        *sources.SourceConfig
	wrapperPath   string
	outputZipPath string
	destDir       string
}

// generateAPI generates a single target API by resolving its service config, gathering
// all target proto files, and executing the PHP generator plugin via protoc.
// It extracts the resulting ZIP archive directly to the library output directory.
func generateAPI(ctx context.Context, params *generateAPIParams) error {
	if params.api.PHP == nil || params.api.PHP.CommonResources == nil {
		return errCommonResourcesUnconfigured
	}
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
	var additionalProtos []string
	if params.api.PHP != nil {
		additionalProtos = params.api.PHP.AdditionalProtos
	}
	includeCommonResources := *params.api.PHP.CommonResources
	targetProtos, err := gatherTargetProtos(googleapisDir, params.api.Path, additionalProtos, includeCommonResources)
	if err != nil {
		return err
	}
	protocArgs := buildProtocArgs(params, opts, targetProtos)
	// Run compilation
	var pc *config.Protoc
	if params.cfg.Tools != nil && params.cfg.Tools.Protoc != nil {
		pc = params.cfg.Tools.Protoc
	}
	if err := protoc.RunOrSystem(ctx, map[string]string{"GOOGLEAPIS_DIR": googleapisDir}, pc, protocArgs...); err != nil {
		return fmt.Errorf("failed to generate PHP API %s: %w", params.api.Path, err)
	}
	return extractOutput(ctx, params.outputZipPath, params.destDir)
}

// gatherTargetProtos collects all proto files inside the target API directory,
// appends common resources, and appends any configured additional protos.
func gatherTargetProtos(googleapisDir, apiPath string, additionalProtos []string, includeCommonResources bool) ([]string, error) {
	apiDir := filepath.Join(googleapisDir, filepath.FromSlash(apiPath))
	targetProtos, err := gatherProtos(apiDir)
	if err != nil {
		return nil, err
	}
	if len(targetProtos) == 0 {
		return nil, fmt.Errorf("no target protos found for API %s", apiPath)
	}

	if includeCommonResources {
		commonResources := filepath.Join(googleapisDir, commonResourcesProto)
		targetProtos = append(targetProtos, commonResources)
	}
	for _, p := range additionalProtos {
		targetProtos = append(targetProtos, filepath.Join(googleapisDir, filepath.FromSlash(p)))
	}
	return targetProtos, nil
}

func buildProtocArgs(params *generateAPIParams, opts []string, targetProtos []string) []string {
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
	return append(protocArgs, targetProtos...)
}

func extractOutput(ctx context.Context, zipPath, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}
	if err := filesystem.Unzip(ctx, zipPath, outDir); err != nil {
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

// DefaultOutput derives an output path from a library name and a default
// output directory.
func DefaultOutput(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}
