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
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	cloudPrefix  = "google-cloud-"
	grpcPrefix   = "grpc-"
	protoPrefix  = "proto-"
	commonProtos = "google/cloud/common_resources.proto"
)

// GenerateLibraries generates all the given libraries in sequence.
func GenerateLibraries(ctx context.Context, libraries []*config.Library, googleapisDir string) error {
	for _, library := range libraries {
		if err := generate(ctx, library, googleapisDir); err != nil {
			return err
		}
	}
	return nil
}

// generate generates a Java client library.
func generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	if len(library.APIs) == 0 {
		return fmt.Errorf("failed to generate library: no apis configured for library %q", library.Name)
	}
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
	return nil
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
	protocOptions, err := createProtocOptions(api, javaAPI, library, googleapisDir, p.protoDir, p.grpcDir, p.gapicDir)
	if err != nil {
		return fmt.Errorf("failed to create protoc options: %w", err)
	}
	args, protos, err := constructProtocCommandArgs(api, javaAPI, googleapisDir, protocOptions)
	if err != nil {
		return fmt.Errorf("failed to construct protoc command args: %w", err)
	}
	p.protos = protos
	if err := command.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("failed to run protoc: %w", err)
	}
	if err := postProcessAPI(ctx, p); err != nil {
		return fmt.Errorf("failed to post process: %w", err)
	}
	return nil
}

type postProcessParams struct {
	outDir         string
	libraryName    string
	version        string
	googleapisDir  string
	protos         []string
	includeSamples bool
	gapicDir       string
	grpcDir        string
	protoDir       string
}

func constructProtocCommandArgs(api *config.API, javaAPI *config.JavaAPI, googleapisDir string, protocOptions []string) ([]string, []string, error) {
	apiDir := filepath.Join(googleapisDir, api.Path)
	// TODO(https://github.com/googleapis/librarian/issues/4198):
	// Consider recursive gathering and explicit sorting
	// of proto files to match the behavior of the hermetic build, ensuring
	// a deterministic order in the generated gapic_metadata.json.
	protos, err := filepath.Glob(apiDir + "/*.proto")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find protos: %w", err)
	}
	if len(protos) == 0 {
		return nil, nil, fmt.Errorf("failed to construct protoc command args: no protos found in api %q", api.Path)
	}

	for _, p := range javaAPI.AdditionalProtos {
		protos = append(protos, filepath.Join(googleapisDir, filepath.FromSlash(p)))
	}

	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
	}
	args = append(args, protos...)
	args = append(args, protocOptions...)
	return args, protos, nil
}

func postProcessAPI(ctx context.Context, p postProcessParams) error {
	// Unzip the temp-codegen.srcjar into temporary version/ directory.
	srcjarPath := filepath.Join(p.gapicDir, "temp-codegen.srcjar")
	if _, err := os.Stat(srcjarPath); err == nil {
		if err := filesystem.Unzip(ctx, srcjarPath, p.gapicDir); err != nil {
			return fmt.Errorf("failed to unzip %s: %w", srcjarPath, err)
		}
	}
	for _, dir := range []string{p.grpcDir, p.protoDir} {
		if err := addMissingHeaders(dir); err != nil {
			return fmt.Errorf("failed to fix headers in %s: %w", dir, err)
		}
	}
	if err := restructureOutput(p); err != nil {
		return fmt.Errorf("failed to restructure output: %w", err)
	}

	// Generate clirr-ignored-differences.xml for the proto module.
	modules := deriveModuleNames(p.libraryName, p.version)
	protoModuleRoot := filepath.Join(p.outDir, modules.proto)
	if err := GenerateClirr(protoModuleRoot); err != nil {
		return fmt.Errorf("failed to generate clirr ignore file: %w", err)
	}

	// Cleanup intermediate protoc output directory after restructuring
	if err := os.RemoveAll(filepath.Join(p.outDir, p.version)); err != nil {
		return fmt.Errorf("failed to cleanup intermediate files: %w", err)
	}
	return nil
}

// addMissingHeaders prepends the license header to all Java files in the given directory
// if they don't already have one.
func addMissingHeaders(dir string) error {
	year := time.Now().Year()
	licenseText := buildLicenseText(year)
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() || filepath.Ext(path) != ".java" {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if license.HasHeader(content) {
			return nil
		}
		return os.WriteFile(path, append([]byte(licenseText), content...), 0644)
	})
}

// buildLicenseText constructs the complete license header text for the given year.
func buildLicenseText(year int) string {
	lines := license.Header(strconv.Itoa(year))
	var b strings.Builder
	b.WriteString("/*\n")
	for _, line := range lines {
		b.WriteString(" *")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(" */\n")
	return b.String()
}

func createProtocOptions(api *config.API, javaAPI *config.JavaAPI, library *config.Library, googleapisDir, protoDir, grpcDir, gapicDir string) ([]string, error) {
	args := []string{
		// --java_out generates standard Protocol Buffer Java classes.
		fmt.Sprintf("--java_out=%s", protoDir),
	}
	transport := library.Transport
	if transport == "" {
		transport = "grpc+rest" // Default to grpc+rest
	}
	// --java_grpc_out generates the gRPC service stubs.
	// This is omitted if the transport is purely REST-based.
	if transport != "rest" {
		args = append(args, fmt.Sprintf("--java_grpc_out=%s", grpcDir))
	}
	// gapicOpts are passed to the GAPIC generator via --java_gapic_opt.
	// "metadata" enables the generation of gapic_metadata.json and GraalVM reflect-config.json.
	gapicOpts := []string{"metadata"}

	apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageJava)
	if err != nil {
		return nil, fmt.Errorf("failed to find api config: %w", err)
	}
	if apiCfg != nil && apiCfg.ServiceConfig != "" {
		// api-service-config specifies the service YAML (e.g., logging_v2.yaml) which
		// contains documentation, HTTP rules, and other API-level configuration.
		gapicOpts = append(gapicOpts, gapicOpt("api-service-config", filepath.Join(googleapisDir, apiCfg.ServiceConfig)))
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
	gapicOpts = append(gapicOpts, gapicOpt("transport", transport))

	// rest-numeric-enums ensures that enums in REST requests are encoded as numbers
	// rather than strings.
	if !javaAPI.NoRestNumericEnums {
		gapicOpts = append(gapicOpts, "rest-numeric-enums")
	}

	// --java_gapic_out invokes the GAPIC generator.
	// The "metadata:" prefix is a parameter that tells the generator to include
	// the metadata files mentioned above in the output srcjar/zip for GraalVM support.
	args = append(args, fmt.Sprintf("--java_gapic_out=metadata:%s", gapicDir))
	args = append(args, "--java_gapic_opt="+strings.Join(gapicOpts, ","))
	return args, nil
}

func gapicOpt(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

type javaModules struct {
	gapic string // e.g., google-cloud-secretmanager
	proto string // e.g., proto-google-cloud-secretmanager-v1
	grpc  string // e.g., grpc-google-cloud-secretmanager-v1
}

func deriveModuleNames(libraryID, version string) javaModules {
	name := libraryID
	if !strings.HasPrefix(name, cloudPrefix) {
		name = cloudPrefix + libraryID
	}
	return javaModules{
		gapic: name,
		proto: fmt.Sprintf("%s%s-%s", protoPrefix, name, version),
		grpc:  fmt.Sprintf("%s%s-%s", grpcPrefix, name, version),
	}
}

func removeConflictingFiles(protoSrcDir string) error {
	// These files are removed because they are often duplicated across
	// multiple artifacts in the Google Cloud Java ecosystem, leading
	// to classpath conflicts.
	if err := os.RemoveAll(filepath.Join(protoSrcDir, "com", "google", "cloud", "location")); err != nil {
		return fmt.Errorf("failed to remove location classes: %w", err)
	}
	if err := os.Remove(filepath.Join(protoSrcDir, "google", "cloud", "CommonResources.java")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove CommonResources.java: %w", err)
	}
	return nil
}

// restructureOutput moves the generated code from the temporary versioned directory
// tree into the final directory structure for GAPIC, Proto, gRPC, and samples.
func restructureOutput(p postProcessParams) error {
	modules := deriveModuleNames(p.libraryName, p.version)
	// Temporary source directories (from protoc/generator output)
	tempGapicSrcDir := filepath.Join(p.outDir, p.version, "gapic", "src", "main")
	tempGapicTestDir := filepath.Join(p.outDir, p.version, "gapic", "src", "test")
	tempProtoSrcDir := filepath.Join(p.outDir, p.version, "proto")
	tempGrpcSrcDir := filepath.Join(p.outDir, p.version, "grpc")
	tempResourceNameSrcDir := filepath.Join(p.outDir, p.version, "gapic", "proto", "src", "main", "java")
	tempSamplesDir := filepath.Join(p.outDir, p.version, "gapic", "samples", "snippets", "generated", "src", "main", "java")
	// Final destination directories
	gapicDestDir := filepath.Join(p.outDir, modules.gapic, "src", "main")
	gapicTestDestDir := filepath.Join(p.outDir, modules.gapic, "src", "test")
	protoDestDir := filepath.Join(p.outDir, modules.proto, "src", "main", "java")
	grpcDestDir := filepath.Join(p.outDir, modules.grpc, "src", "main", "java")
	samplesDestDir := filepath.Join(p.outDir, "samples", "snippets", "generated")

	// Ensure destination directories exist
	destDirs := []string{gapicDestDir, gapicTestDestDir, protoDestDir, grpcDestDir}
	if p.includeSamples {
		destDirs = append(destDirs, samplesDestDir)
	}
	for _, dir := range destDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := removeConflictingFiles(tempProtoSrcDir); err != nil {
		return err
	}

	type moveAction struct {
		src, dest   string
		description string
	}
	actions := []moveAction{
		{src: tempProtoSrcDir, dest: protoDestDir, description: "proto source"},
		{src: tempGrpcSrcDir, dest: grpcDestDir, description: "grpc source"},
		{src: tempGapicSrcDir, dest: gapicDestDir, description: "gapic source"},
		{src: tempGapicTestDir, dest: gapicTestDestDir, description: "gapic test"},
		{src: tempResourceNameSrcDir, dest: protoDestDir, description: "resource name source"},
	}
	if p.includeSamples {
		actions = append(actions, moveAction{src: tempSamplesDir, dest: samplesDestDir, description: "samples"})
	}
	for _, action := range actions {
		if _, err := os.Stat(action.src); err == nil {
			if err := filesystem.MoveAndMerge(action.src, action.dest); err != nil {
				return fmt.Errorf("failed to move %s: %w", action.description, err)
			}
		}
	}
	// Copy proto files to proto-*/src/main/proto
	protoFilesDestDir := filepath.Join(p.outDir, modules.proto, "src", "main", "proto")
	if err := copyProtos(p.googleapisDir, p.protos, protoFilesDestDir); err != nil {
		return fmt.Errorf("failed to copy proto files: %w", err)
	}
	return nil
}

func copyProtos(googleapisDir string, protos []string, destDir string) error {
	for _, proto := range protos {
		if strings.HasSuffix(proto, commonProtos) {
			continue
		}
		// Calculate relative path from googleapisDir to preserve directory structure
		rel, err := filepath.Rel(googleapisDir, proto)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path for %s: %w", proto, err)
		}
		target := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(target), err)
		}
		if err := filesystem.CopyFile(proto, target); err != nil {
			return fmt.Errorf("failed to copy file %s to %s: %w", proto, target, err)
		}
	}
	return nil
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
