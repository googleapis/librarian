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
	"io/fs"
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
	errNoProtos                    = errors.New("no target protos found")
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
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return err
	}
	srcCfg := sources.NewSourceConfig(src, library.Roots)
	for _, api := range library.APIs {
		if api.PHP == nil || api.PHP.StagingSubdir == "" {
			return fmt.Errorf("API %q: %w", api.Path, errMissingStagingSubdir)
		}
		gapicDestDir := filepath.Join(stagingDir, api.PHP.StagingSubdir)
		protoDestDir := filepath.Join(gapicDestDir, "proto/src")

		params := &generateAPIParams{
			cfg:          cfg,
			api:          api,
			library:      library,
			srcCfg:       srcCfg,
			wrapperPath:  wrapperPath,
			tempDir:      tempDir,
			gapicDestDir: gapicDestDir,
			protoDestDir: protoDestDir,
		}
		if err := generateAPI(ctx, params); err != nil {
			return err
		}
	}
	if err := postProcessLibrary(ctx, library); err != nil {
		return fmt.Errorf("failed to postprocess: %w", err)
	}
	return nil
}

type generateAPIParams struct {
	cfg          *config.Config
	api          *config.API
	library      *config.Library
	srcCfg       *sources.SourceConfig
	wrapperPath  string
	tempDir      string
	gapicDestDir string
	protoDestDir string
}

// generateAPI generates a single target API by resolving its service config, gathering
// all target proto files, and executing the PHP generator plugin via protoc.
// It extracts the resulting ZIP archive directly to the library output directory.
func generateAPI(ctx context.Context, params *generateAPIParams) (retErr error) {
	if params.api.PHP == nil || params.api.PHP.CommonResources == nil {
		return errCommonResourcesUnconfigured
	}
	sanitizedPath := strings.ReplaceAll(params.api.Path, "/", "_")
	gapicZipPath := filepath.Join(params.tempDir, sanitizedPath+"-gapic.zip")
	protoZipPath := filepath.Join(params.tempDir, sanitizedPath+"-proto.zip")

	defer func() {
		if cleanupErr := os.Remove(gapicZipPath); cleanupErr != nil && !errors.Is(cleanupErr, fs.ErrNotExist) {
			retErr = errors.Join(retErr, fmt.Errorf("failed to remove gapic zip: %w", cleanupErr))
		}
		if cleanupErr := os.Remove(protoZipPath); cleanupErr != nil && !errors.Is(cleanupErr, fs.ErrNotExist) {
			retErr = errors.Join(retErr, fmt.Errorf("failed to remove proto zip: %w", cleanupErr))
		}
	}()
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
	opts := gapicOpts(apiMetadata, grpcConfigPath)
	additionalProtos := params.api.PHP.AdditionalProtos
	includeCommonResources := *params.api.PHP.CommonResources
	gapicProtos, err := gatherGAPICProtos(googleapisDir, params.api.Path, additionalProtos, includeCommonResources)
	if err != nil {
		return err
	}
	mainProtos, err := gatherMainProtos(googleapisDir, params.api.Path)
	if err != nil {
		return err
	}

	var pc *config.Protoc
	if params.cfg.Tools != nil && params.cfg.Tools.Protoc != nil {
		pc = params.cfg.Tools.Protoc
	}
	// Run 1: GAPIC Client Generation
	gapicArgs := buildGapicProtocArgs(params, gapicZipPath, opts, gapicProtos)
	if err := protoc.RunOrSystem(ctx, map[string]string{"GOOGLEAPIS_DIR": googleapisDir}, pc, gapicArgs...); err != nil {
		return fmt.Errorf("failed to generate PHP GAPIC API %s: %w", params.api.Path, err)
	}
	if err := extractOutput(ctx, gapicZipPath, params.gapicDestDir); err != nil {
		return err
	}
	// Run 2: Proto Message Generation
	protoArgs := buildProtoProtocArgs(params, protoZipPath, mainProtos)
	if err := protoc.RunOrSystem(ctx, map[string]string{"GOOGLEAPIS_DIR": googleapisDir}, pc, protoArgs...); err != nil {
		return fmt.Errorf("failed to generate PHP Proto API %s: %w", params.api.Path, err)
	}
	return extractOutput(ctx, protoZipPath, params.protoDestDir)
}

// gatherGAPICProtos collects all proto files inside the target API directory,
// appends common resources, and appends any configured additional protos.
func gatherGAPICProtos(googleapisDir, apiPath string, additionalProtos []string, includeCommonResources bool) ([]string, error) {
	targetProtos, err := gatherMainProtos(googleapisDir, apiPath)
	if err != nil {
		return nil, err
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

func buildGapicProtocArgs(params *generateAPIParams, gapicZipPath string, opts []string, targetProtos []string) []string {
	gapicOutArg := fmt.Sprintf("--gapic_out=%s:%s", strings.Join(opts, ","), gapicZipPath)
	outputArgs := []string{
		"--plugin=protoc-gen-gapic=" + params.wrapperPath,
		gapicOutArg,
	}
	return buildBaseProtocArgs(params.srcCfg, outputArgs, targetProtos)
}

func buildProtoProtocArgs(params *generateAPIParams, protoZipPath string, targetProtos []string) []string {
	phpOutArg := fmt.Sprintf("--php_out=%s", protoZipPath)
	return buildBaseProtocArgs(params.srcCfg, []string{phpOutArg}, targetProtos)
}

func buildBaseProtocArgs(srcCfg *sources.SourceConfig, outputArgs []string, targetProtos []string) []string {
	args := []string{
		"--experimental_allow_proto3_optional",
	}
	args = append(args, outputArgs...)
	// Append active root directories as include paths (-I) to resolve proto imports.
	for _, root := range srcCfg.ActiveRoots {
		if r := srcCfg.Root(root); r != "" {
			args = append(args, "-I", r)
		}
	}
	return append(args, targetProtos...)
}

func extractOutput(ctx context.Context, zipPath, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}
	if err := filesystem.Unzip(ctx, zipPath, outDir); err != nil {
		return fmt.Errorf("failed to extract generated output to %s: %w", outDir, err)
	}
	return nil
}

func gatherMainProtos(googleapisDir, apiPath string) ([]string, error) {
	apiDir := filepath.Join(googleapisDir, filepath.FromSlash(apiPath))
	protos, err := gatherProtos(apiDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w for API %s: %w", errNoProtos, apiPath, err)
		}
		return nil, err
	}
	if len(protos) == 0 {
		return nil, fmt.Errorf("%w for API %s", errNoProtos, apiPath)
	}
	return protos, nil
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

func gapicOpts(apiMetadata *serviceconfig.API, grpcConfigPath string) []string {
	transport := serviceconfig.GRPCRest
	if apiMetadata != nil {
		transport = apiMetadata.Transport(config.LanguagePhp)
	}
	opts := []string{"metadata", "transport=" + string(transport)}
	opts = append(opts, "migration-mode=NEW_SURFACE_ONLY")
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
