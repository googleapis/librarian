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

// Package java provides Java specific functionality for librarian.
package java

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

const (
	commonResourcesProto = "google/cloud/common_resources.proto"
	commonProtosLibrary  = "common-protos"
)

// nonRecursivePaths is a set of paths where proto gathering should not be recursive.
var nonRecursivePaths = map[string]bool{
	"google/api":   true,
	"google/cloud": true,
	"google/rpc":   true,
}

var (
	errNoProtos          = errors.New("no protos found")
	errMonorepoVersion   = fmt.Errorf("failed to find monorepo version for %q in config", rootLibrary)
	errBOMVersionMissing = errors.New("libraries bom version not found in config")
)

// Generate generates a Java client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, srcs *sources.Sources) error {
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %q: %w", outdir, err)
	}
	srcCfg := sources.NewSourceConfig(srcs, library.Roots)
	primaryDir := srcCfg.Root(srcCfg.ActiveRoots[0])

	// generate repo metadata prior to client because info is needed for
	// owlbot.py to generate README.md
	metadata, err := generateRepoMetadata(cfg, library, outdir, primaryDir)
	if err != nil {
		return fmt.Errorf("failed to generate .repo-metadata.json: %w", err)
	}

	transports := make(map[string]serviceconfig.Transport)
	for _, api := range library.APIs {
		apiCfg, err := serviceconfig.Find(primaryDir, api.Path, config.LanguageJava)
		if err != nil {
			return fmt.Errorf("failed to find api config for %s: %w", api.Path, err)
		}
		transports[api.Path] = apiCfg.Transport(config.LanguageJava)
		// metadata is needed for pom.xml generation in post process
		if err := generateAPI(ctx, generateAPIParams{
			cfg:      cfg,
			api:      api,
			library:  library,
			srcCfg:   srcCfg,
			outdir:   outdir,
			metadata: metadata,
			apiCfg:   apiCfg,
		}); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}

	if err := postProcessLibrary(ctx, libraryPostProcessParams{
		cfg:        cfg,
		library:    library,
		outDir:     outdir,
		metadata:   metadata,
		transports: transports,
	}); err != nil {
		return err
	}
	return nil
}

func deriveAPIBase(library *config.Library, apiPath string) string {
	// TODO(https://github.com/googleapis/librarian/issues/5728):
	// remove this after updated owlbot.py
	if library.Name == commonProtosLibrary {
		return "v1"
	}
	return path.Base(apiPath)
}

type generateAPIParams struct {
	cfg      *config.Config
	api      *config.API
	library  *config.Library
	srcCfg   *sources.SourceConfig
	outdir   string
	metadata *repoMetadata
	apiCfg   *serviceconfig.API
}

func generateAPI(ctx context.Context, params generateAPIParams) error {
	javaAPI := ResolveJavaAPI(params.library, params.api)
	primaryDir := params.srcCfg.Root(params.srcCfg.ActiveRoots[0])
	googleapisDir := params.srcCfg.Root("googleapis")
	additionalProtos := deriveAdditionalProtoPaths(javaAPI, googleapisDir)
	additionalProtosToGenerateAndCopyAbs := deriveAbsoluteProtoPaths(javaAPI.AdditionalProtosToGenerateAndCopy, googleapisDir)

	postParams := postProcessParams{
		cfg:            params.cfg,
		library:        params.library,
		javaAPI:        javaAPI,
		metadata:       params.metadata,
		outDir:         params.outdir,
		apiBase:        deriveAPIBase(params.library, params.api.Path),
		includeSamples: *javaAPI.Samples,
	}
	gapicDir := postParams.gapicDir()
	gRPCDir := postParams.gRPCDir()
	protoDir := postParams.protoDir()
	for _, dir := range []string{gapicDir, gRPCDir, protoDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
	}

	apiDir := filepath.Join(primaryDir, params.api.Path)
	apiProtos, err := gatherProtos(apiDir, params.api.Path)
	if err != nil {
		return fmt.Errorf("failed to find protos: %w", err)
	}
	apiProtos = filterProtos(apiProtos, javaAPI.ExcludedProtos, primaryDir)
	if len(apiProtos) == 0 {
		return fmt.Errorf("%s: %w", params.api.Path, errNoProtos)
	}
	postParams.protosToCopy, err = deriveProtosToCopy(apiProtos, primaryDir, javaAPI.AdditionalProtosToGenerateAndCopy, googleapisDir)
	if err != nil {
		return err
	}

	// 1. Generate standard Protocol Buffer Java classes.
	if shouldGenerateProtoGRPC(javaAPI) {
		protoProtos := filterProtos(apiProtos, javaAPI.SkipProtoClassGeneration, primaryDir)
		protoProtos = append(protoProtos, additionalProtosToGenerateAndCopyAbs...)
		args := protoProtocArgs(protoProtos, params.srcCfg, protoDir)
		if err := runProtoc(ctx, args); err != nil {
			return fmt.Errorf("failed to generate proto: %w", err)
		}
	}
	// 2. Generate gRPC service stubs (skipped if transport is rest).
	transport := params.apiCfg.Transport(config.LanguageJava)
	if shouldGenerateProtoGRPC(javaAPI) && transport != "rest" {
		if err := runProtoc(ctx, gRPCProtocArgs(apiProtos, params.srcCfg, gRPCDir)); err != nil {
			return fmt.Errorf("failed to generate gRPC module: %w", err)
		}
	}
	// 3. Generate GAPIC library.
	if shouldGenerateGAPIC(javaAPI) || shouldGenerateResourceNames(javaAPI) {
		gapicOpts, err := resolveGAPICOptions(params.cfg, params.library, params.api, primaryDir, params.apiCfg)
		if err != nil {
			return fmt.Errorf("failed to resolve gapic options: %w", err)
		}
		args := gapicProtocArgs(apiProtos, append(additionalProtos, additionalProtosToGenerateAndCopyAbs...), params.srcCfg, gapicDir, gapicOpts)
		if err := runProtoc(ctx, args); err != nil {
			return fmt.Errorf("failed to generate gapic: %w", err)
		}
	}

	if err := postProcessAPI(ctx, postParams); err != nil {
		return fmt.Errorf("failed to post process: %w", err)
	}
	return nil
}

// deriveAdditionalProtoPaths returns the absolute paths to additional proto files
// configured via AdditionalProtos to include in GAPIC generation. It includes
// google/cloud/common_resources.proto unless opted out via OmitCommonResources.
func deriveAdditionalProtoPaths(javaAPI *config.JavaAPI, googleapisDir string) []string {
	var additionalProtos []string
	if !javaAPI.OmitCommonResources {
		additionalProtos = append(additionalProtos, filepath.Join(googleapisDir, filepath.FromSlash(commonResourcesProto)))
	}
	for _, p := range javaAPI.AdditionalProtos {
		if p == commonResourcesProto {
			continue
		}
		additionalProtos = append(additionalProtos, filepath.Join(googleapisDir, filepath.FromSlash(p)))
	}
	return additionalProtos
}

func deriveAbsoluteProtoPaths(paths []string, baseDir string) []string {
	var absPaths []string
	for _, p := range paths {
		absPaths = append(absPaths, filepath.Join(baseDir, filepath.FromSlash(p)))
	}
	return absPaths
}

// deriveProtosToCopy resolves absolute and relative paths for API and additional protos.
// The returned structs are eventually used by copyProtos to copy these files into the generated proto module.
func deriveProtosToCopy(apiProtos []string, primaryDir string, additionalRel []string, googleapisDir string) ([]protoFileToCopy, error) {
	var res []protoFileToCopy
	for _, apiProto := range apiProtos {
		rel, err := filepath.Rel(primaryDir, apiProto)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate relative path for %s: %w", apiProto, err)
		}
		res = append(res, protoFileToCopy{
			absolutePath: apiProto,
			relativePath: rel,
		})
	}
	for _, relPath := range additionalRel {
		res = append(res, protoFileToCopy{
			absolutePath: filepath.Join(googleapisDir, filepath.FromSlash(relPath)),
			relativePath: relPath,
		})
	}
	return res, nil
}

var runProtoc = func(ctx context.Context, args []string) error {
	return command.Run(ctx, "protoc", args...)
}

func baseProtocArgs(srcCfg *sources.SourceConfig) []string {
	args := []string{
		"--experimental_allow_proto3_optional",
	}
	for _, root := range srcCfg.ActiveRoots {
		args = append(args, "-I="+srcCfg.Root(root))
	}
	return args
}

func protoProtocArgs(apiProtos []string, srcCfg *sources.SourceConfig, protoDir string) []string {
	args := baseProtocArgs(srcCfg)
	args = append(args, fmt.Sprintf("--java_out=%s", protoDir))
	args = append(args, apiProtos...)
	return args
}

func gRPCProtocArgs(apiProtos []string, srcCfg *sources.SourceConfig, gRPCDir string) []string {
	args := baseProtocArgs(srcCfg)
	args = append(args, fmt.Sprintf("--java_grpc_out=%s", gRPCDir))
	args = append(args, apiProtos...)
	return args
}

func gapicProtocArgs(apiProtos, additionalProtos []string, srcCfg *sources.SourceConfig, gapicDir string, gapicOpts []string) []string {
	args := baseProtocArgs(srcCfg)
	args = append(args, fmt.Sprintf("--java_gapic_out=metadata:%s", gapicDir))
	args = append(args, "--java_gapic_opt="+strings.Join(gapicOpts, ","))
	args = append(args, apiProtos...)
	args = append(args, additionalProtos...)
	return args
}

func resolveGAPICOptions(cfg *config.Config, library *config.Library, api *config.API, sourceDir string, apiCfg *serviceconfig.API) ([]string, error) {
	// gapicOpts are passed to the GAPIC generator via --java_gapic_opt.
	// "metadata" enables the generation of gapic_metadata.json and GraalVM reflect-config.json.
	gapicOpts := []string{"metadata"}

	gapicOpts = append(gapicOpts, gapicOpt("repo", cfg.Repo))
	gapicOpts = append(gapicOpts, gapicOpt("artifact", DeriveDistributionName(library)))

	if apiCfg.ServiceConfig != "" {
		// api-service-config specifies the service YAML (e.g., logging_v2.yaml) which
		// contains documentation, HTTP rules, and other API-level configuration.
		gapicOpts = append(gapicOpts, gapicOpt("api-service-config", filepath.Join(sourceDir, apiCfg.ServiceConfig)))
	}

	gapicConfig, err := serviceconfig.FindGAPICConfig(sourceDir, api.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find gapic config: %w", err)
	}
	if gapicConfig != "" {
		// gapic-config specifies the GAPIC configuration (e.g., logging_gapic.yaml) which
		// contains batching, LRO retries, and language settings.
		gapicOpts = append(gapicOpts, gapicOpt("gapic-config", filepath.Join(sourceDir, gapicConfig)))
	}

	gRPCServiceConfig, err := serviceconfig.FindGRPCServiceConfig(sourceDir, api.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find gRPC service config: %w", err)
	}
	if gRPCServiceConfig != "" {
		// grpc-service-config specifies the retry and timeout settings for the gRPC client.
		gapicOpts = append(gapicOpts, gapicOpt("grpc-service-config", filepath.Join(sourceDir, gRPCServiceConfig)))
	}

	// transport specifies whether to generate gRPC, REST, or both types of clients.
	transport := apiCfg.Transport(config.LanguageJava)
	gapicOpts = append(gapicOpts, gapicOpt("transport", string(transport)))
	// rest-numeric-enums ensures that enums in REST requests are encoded as numbers
	// rather than strings.
	if apiCfg.HasRESTNumericEnums(config.LanguageJava) {
		gapicOpts = append(gapicOpts, "rest-numeric-enums")
	}
	// generate-version-java ensures that the Version.java file is generated.
	gapicOpts = append(gapicOpts, "generate-version-java")
	return gapicOpts, nil
}

func gapicOpt(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

// Format formats a Java client library using google-java-format.
func Format(ctx context.Context, library *config.Library) error {
	files, err := collectJavaFiles(library.Output)
	if err != nil {
		return fmt.Errorf("failed to find java files for formatting: %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	if _, err := exec.LookPath("google-java-format"); err != nil {
		return fmt.Errorf("google-java-format not found in PATH: %w", err)
	}

	args := append([]string{"--replace"}, files...)
	if err := command.Run(ctx, "google-java-format", args...); err != nil {
		return fmt.Errorf("failed to format files: %w", err)
	}
	return nil
}

func collectJavaFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".java" {
			return nil
		}
		// exclude samples/snippets/generated
		if strings.Contains(path, filepath.Join("samples", "snippets", "generated")) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// ResolveJavaAPI returns the Java-specific configuration for the given API.
// TODO(https://github.com/googleapis/librarian/issues/5050):
// Exported to use in migrate tool, unexport after migrate is done.
func ResolveJavaAPI(library *config.Library, api *config.API) *config.JavaAPI {
	res := &config.JavaAPI{
		Path: api.Path,
	}
	if library.Java == nil {
		return res
	}
	for _, javaAPI := range library.Java.JavaAPIs {
		if javaAPI.Path == api.Path {
			return javaAPI
		}
	}
	return res
}

// TODO(https://github.com/googleapis/librarian/issues/5152):
// BOM version should be required and pre-validated, remove this and inline when done.
func findBOMVersion(cfg *config.Config) (string, error) {
	if cfg.Default != nil && cfg.Default.Java != nil && cfg.Default.Java.LibrariesBOMVersion != "" {
		return cfg.Default.Java.LibrariesBOMVersion, nil
	}
	return "", errBOMVersionMissing
}

// gatherProtos returns a sorted list of proto files in the given root directory,
// ensuring that subpackage protos (e.g., in a "schema" directory) are included
// in the generation.
//
// recursion is disabled for certain base paths in nonRecursivePaths.
func gatherProtos(root, relPath string) ([]string, error) {
	var protos []string
	recursive := !nonRecursivePaths[filepath.ToSlash(relPath)]

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && filepath.Ext(path) == ".proto" {
			protos = append(protos, path)
		}
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, errNoProtos
	}
	if err != nil {
		return nil, err
	}
	sort.Strings(protos)
	return protos, nil
}

// filterProtos returns entries from fullPaths that excludes root + relPath in relExcludes.
func filterProtos(fullPaths []string, relExcludes []string, root string) []string {
	if len(relExcludes) == 0 {
		return fullPaths
	}
	excludedSet := make(map[string]bool, len(relExcludes))
	for _, e := range relExcludes {
		fullPath := filepath.ToSlash(filepath.Join(root, filepath.FromSlash(e)))
		excludedSet[fullPath] = true
	}
	filtered := make([]string, 0, len(fullPaths))
	for _, p := range fullPaths {
		if excludedSet[filepath.ToSlash(p)] {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

func shouldGenerateGAPIC(javaAPI *config.JavaAPI) bool {
	if javaAPI.GenerateGAPIC != nil {
		return *javaAPI.GenerateGAPIC
	}
	return true
}

func shouldGenerateProtoGRPC(javaAPI *config.JavaAPI) bool {
	if javaAPI.GenerateProtoGRPC != nil {
		return *javaAPI.GenerateProtoGRPC
	}
	return true
}

func shouldGenerateResourceNames(javaAPI *config.JavaAPI) bool {
	if javaAPI.GenerateResourceNames != nil {
		return *javaAPI.GenerateResourceNames
	}
	return shouldGenerateGAPIC(javaAPI)
}
