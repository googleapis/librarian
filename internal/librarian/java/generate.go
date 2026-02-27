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
	"regexp"
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
	// Output directories for Java
	gapicDir := filepath.Join(outdir, version, "gapic")
	grpcDir := filepath.Join(outdir, version, "grpc")
	protoDir := filepath.Join(outdir, version, "proto")
	for _, dir := range []string{gapicDir, grpcDir, protoDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	protocOptions, err := createProtocOptions(api, library, googleapisDir, protoDir, grpcDir, gapicDir)
	if err != nil {
		return fmt.Errorf("failed to create protoc options: %w", err)
	}
	args, protos, err := constructProtocCommandArgs(api, googleapisDir, protocOptions)
	if err != nil {
		return fmt.Errorf("failed to construct protoc command args: %w", err)
	}
	if err := command.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("failed to run protoc: %w", err)
	}
	if err := postProcess(ctx, outdir, library.Name, version, googleapisDir, gapicDir, grpcDir, protoDir, protos); err != nil {
		return fmt.Errorf("failed to post process: %w", err)
	}
	return nil
}

func constructProtocCommandArgs(api *config.API, googleapisDir string, protocOptions []string) ([]string, []string, error) {
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
	// hardcoded default to start, should get additionals from proto_library_with_info in BUILD.bazel
	protos = append(protos, filepath.Join(googleapisDir, filepath.FromSlash(commonProtos)))
	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
	}
	args = append(args, protos...)
	args = append(args, protocOptions...)
	return args, protos, nil
}

func postProcess(ctx context.Context, outdir, libraryName, version, googleapisDir, gapicDir, grpcDir, protoDir string, protos []string) error {
	// Unzip the temp-codegen.srcjar into temporary version/ directory.
	srcjarPath := filepath.Join(gapicDir, "temp-codegen.srcjar")
	if _, err := os.Stat(srcjarPath); err == nil {
		if err := filesystem.Unzip(ctx, srcjarPath, gapicDir); err != nil {
			return fmt.Errorf("failed to unzip %s: %w", srcjarPath, err)
		}
	}
	for _, dir := range []string{gapicDir, grpcDir, protoDir} {
		if err := fixHeaders(dir); err != nil {
			return fmt.Errorf("failed to fix headers in %s: %w", dir, err)
		}
	}
	if err := restructureOutput(outdir, libraryName, version, googleapisDir, protos); err != nil {
		return fmt.Errorf("failed to restructure output: %w", err)
	}
	// Cleanup intermediate protoc output directory after restructuring
	if err := os.RemoveAll(filepath.Join(outdir, version)); err != nil {
		return fmt.Errorf("failed to cleanup intermediate files: %w", err)
	}
	return nil
}

var headerRegex = regexp.MustCompile(`\* Copyright \d{4} Google LLC`)

// fixHeaders prepends the license header to all Java files in the given directory
// if they don't already have one.
func fixHeaders(dir string) error {
	year := time.Now().Year()
	lines := license.Header(strconv.Itoa(year))
	var b strings.Builder
	b.WriteString("/*\n")
	for _, line := range lines {
		b.WriteString(" *")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(" */\n")
	licenseText := b.String()

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".java" {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if headerRegex.Match(content) {
			return nil
		}
		return os.WriteFile(path, append([]byte(licenseText), content...), 0644)
	})
}

func createProtocOptions(api *config.API, library *config.Library, googleapisDir, protoDir, grpcDir, gapicDir string) ([]string, error) {
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

	apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, serviceconfig.LangJava)
	if err != nil {
		return nil, fmt.Errorf("failed to find api config: %w", err)
	}
	if apiCfg != nil && apiCfg.ServiceConfig != "" {
		// api-service-config specifies the service YAML (e.g., logging_v2.yaml) which
		// contains documentation, HTTP rules, and other API-level configuration.
		gapicOpts = append(gapicOpts, gapicOpt("api-service-config", filepath.Join(googleapisDir, apiCfg.ServiceConfig)))
	}

	gc, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find grpc service config: %w", err)
	}
	if gc != "" {
		// grpc-service-config specifies the retry and timeout settings for the gRPC client.
		gapicOpts = append(gapicOpts, gapicOpt("grpc-service-config", filepath.Join(googleapisDir, gc)))
	}
	// transport specifies whether to generate gRPC, REST, or both types of clients.
	gapicOpts = append(gapicOpts, gapicOpt("transport", transport))
	// rest-numeric-enums ensures that enums in REST requests are encoded as numbers
	// rather than strings.
	// TODO(https://github.com/googleapis/librarian/issues/4130):
	// assign this according to config
	gapicOpts = append(gapicOpts, "rest-numeric-enums")

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
func restructureOutput(outputDir, libraryID, version, googleapisDir string, protos []string) error {
	modules := deriveModuleNames(libraryID, version)
	// Temporary source directories (from protoc/generator output)
	tempGapicSrcDir := filepath.Join(outputDir, version, "gapic", "src", "main")
	tempGapicTestDir := filepath.Join(outputDir, version, "gapic", "src", "test")
	tempProtoSrcDir := filepath.Join(outputDir, version, "proto")
	tempGrpcSrcDir := filepath.Join(outputDir, version, "grpc")
	tempResourceNameSrcDir := filepath.Join(outputDir, version, "gapic", "proto", "src", "main", "java")
	tempSamplesDir := filepath.Join(outputDir, version, "gapic", "samples", "snippets", "generated", "src", "main", "java")
	// Final destination directories
	gapicDestDir := filepath.Join(outputDir, modules.gapic, "src", "main")
	gapicTestDestDir := filepath.Join(outputDir, modules.gapic, "src", "test")
	protoDestDir := filepath.Join(outputDir, modules.proto, "src", "main", "java")
	grpcDestDir := filepath.Join(outputDir, modules.grpc, "src", "main", "java")
	samplesDestDir := filepath.Join(outputDir, "samples", "snippets", "generated")

	// Ensure destination directories exist
	destDirs := []string{gapicDestDir, gapicTestDestDir, protoDestDir, grpcDestDir, samplesDestDir}
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
		{src: tempSamplesDir, dest: samplesDestDir, description: "samples"},
		{src: tempResourceNameSrcDir, dest: protoDestDir, description: "resource name source"},
	}
	for _, action := range actions {
		if _, err := os.Stat(action.src); err == nil {
			if err := filesystem.MoveAndMerge(action.src, action.dest); err != nil {
				return fmt.Errorf("failed to move %s: %w", action.description, err)
			}
		}
	}
	// Copy proto files to proto-*/src/main/proto
	protoFilesDestDir := filepath.Join(outputDir, modules.proto, "src", "main", "proto")
	if err := copyProtos(googleapisDir, protos, protoFilesDestDir); err != nil {
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
