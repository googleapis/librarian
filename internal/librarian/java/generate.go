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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	cloudPrefix  = "google-cloud-"
	grpcPrefix   = "grpc-"
	protoPrefix  = "proto-"
	commonProtos = "google/cloud/common_resources.proto"
)

// Generate generates a Java client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, googleapisDir string) error {
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}
	// Ensure googleapisDir is absolute to avoid issues with relative paths in protoc.
	googleapisDir, err = filepath.Abs(googleapisDir)
	if err != nil {
		return fmt.Errorf("failed to resolve googleapis directory path: %w", err)
	}
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	for _, api := range library.APIs {
		if err := generateAPI(ctx, api, library, googleapisDir, outdir); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}
	return postProcessLibrary(cfg, library, outdir, googleapisDir)
}

func generateAPI(ctx context.Context, api *config.API, library *config.Library, googleapisDir, outdir string) error {
	version := serviceconfig.ExtractVersion(api.Path)
	if version == "" {
		return fmt.Errorf("failed to generate api: failed to extract version from api path %q", api.Path)
	}
	javaAPI := resolveJavaAPI(library, api)
	p := postProcessParams{
		outDir:         outdir,
		libraryName:    library.Name,
		version:        version,
		googleapisDir:  googleapisDir,
		includeSamples: !javaAPI.NoSamples,
		gapicDir:       filepath.Join(outdir, version, "gapic"),
		grpcDir:        filepath.Join(outdir, version, "grpc"),
		protoDir:       filepath.Join(outdir, version, "proto"),
	}
	for _, dir := range []string{p.gapicDir, p.grpcDir, p.protoDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	apiDir := filepath.Join(googleapisDir, api.Path)
	// TODO(https://github.com/googleapis/librarian/issues/4198):
	// Consider recursive gathering and explicit sorting
	// of proto files to match the behavior of the hermetic build, ensuring
	// a deterministic order in the generated gapic_metadata.json.
	apiProtos, err := filepath.Glob(apiDir + "/*.proto")
	if err != nil {
		return fmt.Errorf("failed to find protos: %w", err)
	}
	if len(apiProtos) == 0 {
		return fmt.Errorf("failed to generate api: no protos found in api %q", api.Path)
	}
	p.apiProtos = apiProtos

	// 1. Generate standard Protocol Buffer Java classes.
	if err := runProtoc(ctx, protoProtocArgs(apiProtos, googleapisDir, p.protoDir)); err != nil {
		return fmt.Errorf("failed to generate proto: %w", err)
	}
	// 2. Generate gRPC service stubs (skipped if transport is rest).
	apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageJava)
	if err != nil {
		return fmt.Errorf("failed to find api config: %w", err)
	}
	transport := serviceconfig.GRPCRest
	if apiCfg != nil {
		transport = apiCfg.Transport(config.LanguageJava)
	}
	if transport != "rest" {
		if err := runProtoc(ctx, grpcProtocArgs(apiProtos, googleapisDir, p.grpcDir)); err != nil {
			return fmt.Errorf("failed to generate grpc: %w", err)
		}
	}
	// 3. Generate GAPIC library.
	gapicOpts, err := resolveGAPICOptions(api, javaAPI, googleapisDir, apiCfg)
	if err != nil {
		return fmt.Errorf("failed to resolve gapic options: %w", err)
	}
	var additionalProtos []string
	for _, p := range javaAPI.AdditionalProtos {
		additionalProtos = append(additionalProtos, filepath.Join(googleapisDir, filepath.FromSlash(p)))
	}
	if err := runProtoc(ctx, gapicProtocArgs(apiProtos, additionalProtos, googleapisDir, p.gapicDir, gapicOpts)); err != nil {
		return fmt.Errorf("failed to generate gapic: %w", err)
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

func grpcProtocArgs(apiProtos []string, googleapisDir, grpcDir string) []string {
	args := baseProtocArgs(googleapisDir)
	args = append(args, fmt.Sprintf("--java_grpc_out=%s", grpcDir))
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

func resolveGAPICOptions(api *config.API, javaAPI *config.JavaAPI, googleapisDir string, apiCfgs *serviceconfig.API) ([]string, error) {
	// gapicOpts are passed to the GAPIC generator via --java_gapic_opt.
	// "metadata" enables the generation of gapic_metadata.json and GraalVM reflect-config.json.
	gapicOpts := []string{"metadata"}
	if apiCfgs != nil && apiCfgs.ServiceConfig != "" {
		// api-service-config specifies the service YAML (e.g., logging_v2.yaml) which
		// contains documentation, HTTP rules, and other API-level configuration.
		gapicOpts = append(gapicOpts, gapicOpt("api-service-config", filepath.Join(googleapisDir, apiCfgs.ServiceConfig)))
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

	grpcServiceConfig, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find grpc service config: %w", err)
	}
	if grpcServiceConfig != "" {
		// grpc-service-config specifies the retry and timeout settings for the gRPC client.
		gapicOpts = append(gapicOpts, gapicOpt("grpc-service-config", filepath.Join(googleapisDir, grpcServiceConfig)))
	}

	// transport specifies whether to generate gRPC, REST, or both types of clients.
	transport := serviceconfig.GRPCRest
	if apiCfgs != nil {
		transport = apiCfgs.Transport(config.LanguageJava)
	}
	gapicOpts = append(gapicOpts, gapicOpt("transport", string(transport)))

	// rest-numeric-enums ensures that enums in REST requests are encoded as numbers
	// rather than strings.
	if !javaAPI.NoRestNumericEnums {
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
		return fmt.Errorf("formatting failed: %w", err)
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

// resolveJavaAPI returns the Java-specific configuration for the given API,
// applying default values if no explicit configuration is found in the library.
func resolveJavaAPI(library *config.Library, api *config.API) *config.JavaAPI {
	res := &config.JavaAPI{
		Path:             api.Path,
		AdditionalProtos: []string{commonProtos},
	}
	if library.Java == nil {
		return res
	}
	for _, javaAPI := range library.Java.JavaAPIs {
		if javaAPI.Path != api.Path {
			continue
		}
		*res = *javaAPI
		if len(res.AdditionalProtos) == 0 {
			res.AdditionalProtos = []string{commonProtos}
		}
		return res
	}
	return res
}
