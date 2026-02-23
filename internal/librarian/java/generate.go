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
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
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
		return fmt.Errorf("no apis configured for library %q", library.Name)
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
	version := extractVersion(api.Path)
	if version == "" {
		return fmt.Errorf("failed to extract version from api path %q", api.Path)
	}
	// Output directories for Java as seen in v0.7.0
	gapicDir := filepath.Join(outdir, version, "gapic")
	grpcDir := filepath.Join(outdir, version, "grpc")
	protoDir := filepath.Join(outdir, version, "proto")
	for _, dir := range []string{gapicDir, grpcDir, protoDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	protocOptions, err := createProtocOptions(api, library, googleapisDir, protoDir, grpcDir, gapicDir)
	if err != nil {
		return err
	}

	cmd, protos, err := constructProtocCommand(ctx, api, googleapisDir, protocOptions)
	if err != nil {
		return err
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", cmd.String(), err)
	}

	return postProcess(outdir, library.Name, version, googleapisDir, gapicDir, grpcDir, protoDir, protos)
}

func constructProtocCommand(ctx context.Context, api *config.API, googleapisDir string, protocOptions []string) (*exec.Cmd, []string, error) {
	apiDir := filepath.Join(googleapisDir, api.Path)
	protos, err := filepath.Glob(apiDir + "/*.proto")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find protos: %w", err)
	}
	if len(protos) == 0 {
		return nil, nil, fmt.Errorf("no protos found in api %q", api.Path)
	}
	// hardcoded default to start, should get additionals from proto_library_with_info in BUILD.bazel
	protos = append(protos, filepath.Join(googleapisDir, "google", "cloud", "common_resources.proto"))
	cmdArgs := []string{"protoc", "--experimental_allow_proto3_optional", "-I=" + googleapisDir}
	cmdArgs = append(cmdArgs, protos...)
	cmdArgs = append(cmdArgs, protocOptions...)
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	return cmd, protos, nil
}

func postProcess(outdir, libraryName, version, googleapisDir, gapicDir, grpcDir, protoDir string, protos []string) error {
	// Unzip the temp-codegen.srcjar.
	srcjarPath := filepath.Join(gapicDir, "temp-codegen.srcjar")
	if _, err := os.Stat(srcjarPath); err == nil {
		if err := unzip(srcjarPath, gapicDir); err != nil {
			return fmt.Errorf("failed to unzip %s: %w", srcjarPath, err)
		}
	}
	if err := restructureOutput(outdir, libraryName, version, googleapisDir, protos); err != nil {
		return fmt.Errorf("failed to restructure output: %w", err)
	}
	// Cleanup intermediate protoc output directory before restructuring
	if err := os.RemoveAll(filepath.Join(outdir, version)); err != nil {
		return fmt.Errorf("failed to cleanup intermediate files: %w", err)
	}
	return nil
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

	sc, err := serviceconfig.Find(googleapisDir, api.Path, serviceconfig.LangJava)
	if err != nil {
		return nil, err
	}
	if sc != nil && sc.ServiceConfig != "" {
		// api-service-config specifies the service YAML (e.g., logging_v2.yaml) which
		// contains documentation, HTTP rules, and other API-level configuration.
		gapicOpts = append(gapicOpts, fmt.Sprintf("api-service-config=%s", filepath.Join(googleapisDir, sc.ServiceConfig)))
	}

	gc, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, err
	}
	if gc != "" {
		// grpc-service-config specifies the retry and timeout settings for the gRPC client.
		gapicOpts = append(gapicOpts, fmt.Sprintf("grpc-service-config=%s", filepath.Join(googleapisDir, gc)))
	}

	// transport specifies whether to generate gRPC, REST, or both types of clients.
	gapicOpts = append(gapicOpts, fmt.Sprintf("transport=%s", transport))

	// rest-numeric-enums ensures that enums in REST requests are encoded as numbers
	// rather than strings, which is the standard for Google Cloud APIs.
	gapicOpts = append(gapicOpts, "rest-numeric-enums")

	// --java_gapic_out invokes the GAPIC generator.
	// The "metadata:" prefix is a parameter that tells the generator to include
	// the metadata files mentioned above in the output srcjar/zip for GraalVM support.
	args = append(args, fmt.Sprintf("--java_gapic_out=metadata:%s", gapicDir))
	args = append(args, "--java_gapic_opt="+strings.Join(gapicOpts, ","))

	return args, nil
}

func extractVersion(path string) string {
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasPrefix(parts[i], "v") {
			return parts[i]
		}
	}
	return ""
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, copyErr := io.Copy(outFile, rc)
		rc.Close()
		closeErr := outFile.Close()

		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func restructureOutput(outputDir, libraryID, version, googleapisDir string, protos []string) error {
	gapicSrcDir := filepath.Join(outputDir, version, "gapic", "src", "main")
	gapicTestDir := filepath.Join(outputDir, version, "gapic", "src", "test")
	protoSrcDir := filepath.Join(outputDir, version, "proto")
	resourceNameSrcDir := filepath.Join(outputDir, version, "gapic", "proto", "src", "main", "java")
	samplesDir := filepath.Join(outputDir, version, "gapic", "samples", "snippets", "generated", "src", "main", "java")

	// Adjusting libraryID for Java naming convention as seen in v0.7.0.
	// This logic derives destination directory names (e.g., google-cloud-secretmanager,
	// proto-google-cloud-secretmanager-v1) from the 'name' field in librarian.yaml.
	// This currently handles cases where the API path (e.g., google/cloud/secrets)
	// differs from the desired library name (e.g., secretmanager).
	// TODO: Consider making sub-module naming patterns customizable in librarian.yaml.
	libraryName := libraryID
	if !strings.HasPrefix(libraryName, "google-cloud-") {
		libraryName = "google-cloud-" + libraryID
	}

	gapicDestDir := filepath.Join(outputDir, libraryName, "src", "main")
	gapicTestDestDir := filepath.Join(outputDir, libraryName, "src", "test")
	protoModuleName := fmt.Sprintf("proto-%s-%s", libraryName, version)
	protoDestDir := filepath.Join(outputDir, protoModuleName, "src", "main", "java")
	grpcDestDir := filepath.Join(outputDir, fmt.Sprintf("grpc-%s-%s", libraryName, version), "src", "main", "java")
	samplesDestDir := filepath.Join(outputDir, "samples", "snippets", "generated")
	destDirs := []string{gapicDestDir, gapicTestDestDir, protoDestDir, samplesDestDir, grpcDestDir}
	for _, dir := range destDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	// Remove location classes and CommonResources to avoid conflicts.
	os.RemoveAll(filepath.Join(protoSrcDir, "com", "google", "cloud", "location"))
	os.Remove(filepath.Join(protoSrcDir, "google", "cloud", "CommonResources.java"))
	moves := map[string]string{
		protoSrcDir: protoDestDir,
		filepath.Join(outputDir, version, "grpc"): grpcDestDir,
		gapicSrcDir:        gapicDestDir,
		gapicTestDir:       gapicTestDestDir,
		samplesDir:         samplesDestDir,
		resourceNameSrcDir: protoDestDir,
	}
	for src, dest := range moves {
		if _, err := os.Stat(src); err == nil {
			if err := moveAndMerge(src, dest); err != nil {
				return err
			}
		}
	}
	// Copy proto files to proto-*/src/main/proto
	protoFilesDestDir := filepath.Join(outputDir, protoModuleName, "src", "main", "proto")
	if err := copyProtos(googleapisDir, protos, protoFilesDestDir); err != nil {
		return fmt.Errorf("failed to copy proto files: %w", err)
	}
	return nil
}

func copyProtos(googleapisDir string, protos []string, destDir string) error {
	for _, proto := range protos {
		if strings.HasSuffix(proto, "google/cloud/common_resources.proto") {
			continue
		}
		// Calculate relative path from googleapisDir to preserve directory structure
		rel, err := filepath.Rel(googleapisDir, proto)
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		if err := copyFile(proto, target); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func moveAndMerge(sourceDir, targetDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		oldPath := filepath.Join(sourceDir, entry.Name())
		newPath := filepath.Join(targetDir, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(newPath, 0755); err != nil {
				return err
			}
			if err := moveAndMerge(oldPath, newPath); err != nil {
				return err
			}
		} else {
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// Format formats a Java client library using google-java-format.
func Format(ctx context.Context, library *config.Library) error {
	return nil
}

// Clean removes files in the library's output directory that are not in the keep list.
// It targets patterns like proto-*, grpc-*, and the main GAPIC module.
func Clean(library *config.Library) error {
	return nil
}
