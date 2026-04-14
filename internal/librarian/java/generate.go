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
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

var nonRecursivePaths = map[string]bool{
	"google/api":   true,
	"google/cloud": true,
	"google/rpc":   true,
}

var (
	errExtractVersion    = errors.New("failed to extract version")
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
	// Ensure googleapisDir is absolute to avoid issues with relative paths in protoc.
	var googleapisDir string
	googleapisDir, err = filepath.Abs(srcs.Googleapis)
	if err != nil {
		return fmt.Errorf("failed to resolve googleapis directory path: %w", err)
	}
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %q: %w", outdir, err)
	}
	// generate repo metadata prior to client because info is needed for
	// owlbot.py to generate README.md
	metadata, err := generateRepoMetadata(cfg, library, outdir, googleapisDir)
	if err != nil {
		return fmt.Errorf("failed to generate .repo-metadata.json: %w", err)
	}

	transports := make(map[string]serviceconfig.Transport)
	for _, api := range library.APIs {
		apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageJava)
		if err != nil {
			return fmt.Errorf("failed to find api config for %s: %w", api.Path, err)
		}
		transports[api.Path] = apiCfg.Transport(config.LanguageJava)
		// metadata is needed for pom.xml generation in post process
		if err := generateAPI(ctx, cfg, api, library, googleapisDir, outdir, metadata, apiCfg); err != nil {
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

func generateAPI(ctx context.Context, cfg *config.Config, api *config.API, library *config.Library, googleapisDir, outdir string, metadata *repoMetadata, apiCfg *serviceconfig.API) error {
	version := serviceconfig.ExtractVersion(api.Path)
	if version == "" {
		return fmt.Errorf("%s: %w", api.Path, errExtractVersion)
	}
	javaAPI := ResolveJavaAPI(library, api)
	p := postProcessParams{
		cfg:            cfg,
		library:        library,
		javaAPI:        javaAPI,
		metadata:       metadata,
		outDir:         outdir,
		version:        version,
		googleapisDir:  googleapisDir,
		includeSamples: *javaAPI.Samples,
	}
	gapicDir := p.gapicDir()
	gRPCDir := p.gRPCDir()
	protoDir := p.protoDir()
	for _, dir := range []string{gapicDir, gRPCDir, protoDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
	}

	apiDir := filepath.Join(googleapisDir, api.Path)
	apiProtos, err := gatherProtos(apiDir, api.Path)
	if err != nil {
		return fmt.Errorf("failed to find protos: %w", err)
	}
	if len(apiProtos) == 0 {
		return fmt.Errorf("%s: %w", api.Path, errNoProtos)
	}
	p.apiProtos = apiProtos

	// 1. Generate standard Protocol Buffer Java classes.
	if err := runProtoc(ctx, protoProtocArgs(apiProtos, googleapisDir, protoDir)); err != nil {
		return fmt.Errorf("failed to generate proto: %w", err)
	}
	// 2. Generate gRPC service stubs (skipped if transport is rest).
	transport := apiCfg.Transport(config.LanguageJava)
	if transport != "rest" {
		if err := runProtoc(ctx, gRPCProtocArgs(apiProtos, googleapisDir, gRPCDir)); err != nil {
			return fmt.Errorf("failed to generate gRPC module: %w", err)
		}
	}
	// 3. Generate GAPIC library.
	if !javaAPI.ProtoOnly {
		gapicOpts, err := resolveGAPICOptions(cfg, library, api, googleapisDir, apiCfg)
		if err != nil {
			return fmt.Errorf("failed to resolve gapic options: %w", err)
		}
		var additionalProtos []string
		for _, p := range javaAPI.AdditionalProtos {
			additionalProtos = append(additionalProtos, filepath.Join(googleapisDir, filepath.FromSlash(p)))
		}
		if err := runProtoc(ctx, gapicProtocArgs(apiProtos, additionalProtos, googleapisDir, gapicDir, gapicOpts)); err != nil {
			return fmt.Errorf("failed to generate gapic: %w", err)
		}
	}

	if err := postProcessAPI(ctx, p); err != nil {
		return fmt.Errorf("failed to post process: %w", err)
	}
	return nil
}

var runProtoc = func(ctx context.Context, args []string) error {
	return command.Run(ctx, "protoc", args...)
}

func baseProtocArgs(googleapisDir string) []string {
	return []string{
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
	}
}

func protoProtocArgs(apiProtos []string, googleapisDir, protoDir string) []string {
	args := baseProtocArgs(googleapisDir)
	args = append(args, fmt.Sprintf("--java_out=%s", protoDir))
	args = append(args, apiProtos...)
	return args
}

func gRPCProtocArgs(apiProtos []string, googleapisDir, gRPCDir string) []string {
	args := baseProtocArgs(googleapisDir)
	args = append(args, fmt.Sprintf("--java_grpc_out=%s", gRPCDir))
	args = append(args, apiProtos...)
	return args
}

func gapicProtocArgs(apiProtos, additionalProtos []string, googleapisDir, gapicDir string, gapicOpts []string) []string {
	args := baseProtocArgs(googleapisDir)
	args = append(args, fmt.Sprintf("--java_gapic_out=metadata:%s", gapicDir))
	args = append(args, "--java_gapic_opt="+strings.Join(gapicOpts, ","))
	args = append(args, apiProtos...)
	args = append(args, additionalProtos...)
	return args
}

func resolveGAPICOptions(cfg *config.Config, library *config.Library, api *config.API, googleapisDir string, apiCfg *serviceconfig.API) ([]string, error) {
	// gapicOpts are passed to the GAPIC generator via --java_gapic_opt.
	// "metadata" enables the generation of gapic_metadata.json and GraalVM reflect-config.json.
	gapicOpts := []string{"metadata"}

	gapicOpts = append(gapicOpts, gapicOpt("repo", cfg.Repo))
	gapicOpts = append(gapicOpts, gapicOpt("artifact", DeriveDistributionName(library)))

	if apiCfg.ServiceConfig != "" {
		// api-service-config specifies the service YAML (e.g., logging_v2.yaml) which
		// contains documentation, HTTP rules, and other API-level configuration.
		gapicOpts = append(gapicOpts, gapicOpt("api-service-config", filepath.Join(googleapisDir, apiCfg.ServiceConfig)))
	}

	gapicConfig, err := serviceconfig.FindGAPICConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find gapic config: %w", err)
	}
	if gapicConfig != "" {
		// gapic-config specifies the GAPIC configuration (e.g., logging_gapic.yaml) which
		// contains batching, LRO retries, and language settings.
		gapicOpts = append(gapicOpts, gapicOpt("gapic-config", filepath.Join(googleapisDir, gapicConfig)))
	}

	gRPCServiceConfig, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find gRPC service config: %w", err)
	}
	if gRPCServiceConfig != "" {
		// grpc-service-config specifies the retry and timeout settings for the gRPC client.
		gapicOpts = append(gapicOpts, gapicOpt("grpc-service-config", filepath.Join(googleapisDir, gRPCServiceConfig)))
	}

	// transport specifies whether to generate gRPC, REST, or both types of clients.
	transport := apiCfg.Transport(config.LanguageJava)
	gapicOpts = append(gapicOpts, gapicOpt("transport", string(transport)))

	// rest-numeric-enums ensures that enums in REST requests are encoded as numbers
	// rather than strings.
	if apiCfg.HasRESTNumericEnums(config.LanguageJava) {
		gapicOpts = append(gapicOpts, "rest-numeric-enums")
	}
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
// recursion is disabled for certain base paths (google/api, google/cloud, google/rpc).
func gatherProtos(root, relPath string) ([]string, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, errNoProtos
	}
	var protos []string
	recursive := !nonRecursivePaths[relPath]

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
	if err != nil {
		return nil, err
	}
	sort.Strings(protos)
	return protos, nil
}
