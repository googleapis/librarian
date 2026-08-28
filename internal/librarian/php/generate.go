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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/proto"
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
	errNoAPIs                      = errors.New("no APIs configured")
)

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

// Generate generates a PHP client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) (err error) {
	if len(library.APIs) == 0 {
		return fmt.Errorf("%w: %q", errNoAPIs, library.Name)
	}
	if cfg.Tools == nil || cfg.Tools.Protoc == nil {
		if _, err := exec.LookPath("protoc"); err != nil {
			return fmt.Errorf("failed to find protoc: %w", err)
		}
	}

	bin, err := binDir()
	if err != nil {
		return err
	}
	wrapperPath := filepath.Join(bin, "gapic-generator-php")
	if _, err := os.Stat(wrapperPath); err != nil {
		return fmt.Errorf("PHP generator wrapper not found (did you run 'librarian install'?): %w", err)
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

	for _, api := range library.APIs {
		if api.PHP == nil || api.PHP.StagingSubdir == "" {
			return fmt.Errorf("API %q: %w", api.Path, errMissingStagingSubdir)
		}
	}
	srcCfg := sources.NewSourceConfig(src, library.Roots)
	googleapisDir := srcCfg.Root("googleapis")
	componentName, err := initComponentIfMissing(ctx, library, googleapisDir)
	if err != nil {
		return err
	}
	stagingDir := filepath.Join(owlBotStagingDir, componentName)
	if err := os.RemoveAll(stagingDir); err != nil {
		return err
	}
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return err
	}
	for _, api := range library.APIs {
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
	if err := postProcessLibrary(ctx, library, componentName); err != nil {
		return fmt.Errorf("failed to postprocess: %w", err)
	}
	return nil
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
	var pc *config.Protoc
	if params.cfg.Tools != nil {
		pc = params.cfg.Tools.Protoc
	}
	// Run 1: GAPIC Client Generation
	if err := generateGAPIC(ctx, params, pc, googleapisDir, gapicZipPath); err != nil {
		return err
	}
	// Run 2: Proto Message Generation
	return generateProto(ctx, params, pc, googleapisDir, protoZipPath)
}

// generateGAPIC generates the GAPIC client surface.
func generateGAPIC(ctx context.Context, params *generateAPIParams, pc *config.Protoc, googleapisDir, gapicZipPath string) error {
	if !shouldGenerateGAPIC(params.api) {
		return nil
	}
	apiMetadata, err := serviceconfig.Find(googleapisDir, params.api.Path, config.LanguagePhp)
	if err != nil {
		return err
	}
	grpcSrvConfigAbsPath, err := grpcServiceConfigPath(params.api, googleapisDir)
	if err != nil {
		return err
	}
	serviceConfigPath := ""
	if apiMetadata != nil {
		serviceConfigPath = apiMetadata.ServiceConfig
	}
	serviceYamlAbsPath, err := absConfigPath(googleapisDir, serviceConfigPath)
	if err != nil {
		return err
	}
	gapicYamlAbsPath, err := absConfigPath(googleapisDir, params.api.PHP.GapicYAML)
	if err != nil {
		return err
	}
	opts := gapicOpts(params.api, apiMetadata, grpcSrvConfigAbsPath, serviceYamlAbsPath, gapicYamlAbsPath)
	additionalProtos := params.api.PHP.AdditionalProtos
	includeCommonResources := *params.api.PHP.CommonResources
	gapicProtos, err := gatherGAPICProtos(googleapisDir, params.api.Path, additionalProtos, params.api.PHP.ExcludedProtos, includeCommonResources)
	if err != nil {
		return err
	}
	gapicArgs := buildGapicProtocArgs(params, gapicZipPath, opts, gapicProtos)
	if err := protoc.RunOrSystem(ctx, map[string]string{"GOOGLEAPIS_DIR": googleapisDir}, pc, gapicArgs...); err != nil {
		return fmt.Errorf("failed to generate PHP GAPIC API %s: %w", params.api.Path, err)
	}
	return extractOutput(ctx, gapicZipPath, params.gapicDestDir)
}

// shouldGenerateGAPIC checks if GAPIC client generation should proceed.
func shouldGenerateGAPIC(api *config.API) bool {
	if api.PHP.GenerateGAPIC != nil {
		return *api.PHP.GenerateGAPIC
	}
	return true
}

func grpcServiceConfigPath(api *config.API, googleapisDir string) (string, error) {
	if api.PHP.SkipGRPCServiceConfig {
		return "", nil
	}
	grpcServiceConfigPath, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return "", err
	}
	grpcServiceConfigAbsPath, err := absConfigPath(googleapisDir, grpcServiceConfigPath)
	if err != nil {
		return "", err
	}
	return grpcServiceConfigAbsPath, nil
}

func generateProto(ctx context.Context, params *generateAPIParams, pc *config.Protoc, googleapisDir, protoZipPath string) error {
	mainProtos, err := gatherMainProtos(googleapisDir, params.api.Path, params.api.PHP.ExcludedProtos)
	if err != nil {
		return err
	}
	protoArgs := buildProtoProtocArgs(params, protoZipPath, mainProtos)
	if err := protoc.RunOrSystem(ctx, map[string]string{"GOOGLEAPIS_DIR": googleapisDir}, pc, protoArgs...); err != nil {
		return fmt.Errorf("failed to generate PHP Proto API %s: %w", params.api.Path, err)
	}
	return extractOutput(ctx, protoZipPath, params.protoDestDir)
}

// gatherGAPICProtos collects all proto files inside the target API directory,
// appends common resources, and appends any configured additional protos.
func gatherGAPICProtos(googleapisDir, apiPath string, additionalProtos, excludeProtos []string, includeCommonResources bool) ([]string, error) {
	targetProtos, err := gatherMainProtos(googleapisDir, apiPath, excludeProtos)
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

func gatherMainProtos(googleapisDir, apiPath string, excludeProtos []string) ([]string, error) {
	apiDir := filepath.Join(googleapisDir, filepath.FromSlash(apiPath))
	protos, err := proto.Gather(apiDir, apiPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w for API %s: %w", errNoProtos, apiPath, err)
		}
		return nil, err
	}
	protos = filterProtos(googleapisDir, protos, excludeProtos)
	if len(protos) == 0 {
		return nil, fmt.Errorf("%w for API %s", errNoProtos, apiPath)
	}
	return protos, nil
}

func filterProtos(googleapisDir string, protos, excludeProtos []string) []string {
	for _, exclude := range excludeProtos {
		fullPath := filepath.Join(googleapisDir, filepath.FromSlash(exclude))
		protos = slices.DeleteFunc(protos, func(p string) bool {
			return p == fullPath
		})
	}
	return protos
}

func absConfigPath(baseDir, configPath string) (string, error) {
	if configPath == "" {
		return "", nil
	}
	return filepath.Abs(filepath.Join(baseDir, configPath))
}

func gapicOpts(api *config.API, apiMetadata *serviceconfig.API, grpcSrvConfigAbsPath, serviceYamlAbsPath, gapicYamlAbsPath string) []string {
	transport := serviceconfig.GRPCRest
	if apiMetadata != nil {
		transport = apiMetadata.Transport(config.LanguagePhp)
	}
	opts := []string{"metadata", "transport=" + string(transport)}
	if apiMetadata != nil && apiMetadata.HasRESTNumericEnums(config.LanguagePhp) {
		opts = append(opts, "rest-numeric-enums")
	}
	if shouldGenerateSamples(api) {
		opts = append(opts, "generate-snippets")
	}

	if grpcSrvConfigAbsPath != "" {
		opts = append(opts, "grpc_service_config="+grpcSrvConfigAbsPath)
	}
	if serviceYamlAbsPath != "" {
		opts = append(opts, "service_yaml="+serviceYamlAbsPath)
	}
	if gapicYamlAbsPath != "" {
		opts = append(opts, "gapic_yaml="+gapicYamlAbsPath)
	}
	return opts
}

func shouldGenerateSamples(api *config.API) bool {
	if api.PHP.Samples != nil {
		return *api.PHP.Samples
	}
	return true
}
