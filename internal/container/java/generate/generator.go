// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generate

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/container/java/bazel"
	"github.com/googleapis/librarian/internal/container/java/execv"
	"github.com/googleapis/librarian/internal/container/java/languagecontainer/generate"
	"github.com/googleapis/librarian/internal/container/java/message"
	"github.com/googleapis/librarian/internal/container/java/pom"
	"github.com/googleapis/librarian/internal/container/java/protoc"
)

// Test substitution vars.
var (
	bazelParse  = bazel.Parse
	execvRun    = execv.Run
	protocBuild = protoc.Build
)

// Generate is the main entrypoint for the `generate` command. It orchestrates
// the entire generation process.
func Generate(ctx context.Context, cfg *generate.Config) error {
	slog.Debug("librariangen: generate command started")
	libraryID := cfg.Request.ID
	for _, api := range cfg.Request.APIs {
		if err := processAPI(ctx, cfg, libraryID, api); err != nil {
			return err
		}		
	}

	// Generate pom.xml files
	if err := pom.Generate(cfg.Context.OutputDir, libraryID); err != nil {
		return fmt.Errorf("librariangen: failed to generate poms for API %s: %w", libraryID, err)
	}

	slog.Debug("librariangen: generate command finished")
	return nil
}

func processAPI(ctx context.Context, cfg *generate.Config, libraryID string, api message.API) error {
	version := extractVersion(api.Path)
	if version == "" {
		slog.Warn("skipping api with no version", "api", api.Path)
		return nil
	}
	slog.Info("processing api", "path", api.Path, "version", version)
	outputConfig := &protoc.OutputConfig{
		GAPICDir: filepath.Join(cfg.Context.OutputDir, version, "gapic"),
		GRPCDir:  filepath.Join(cfg.Context.OutputDir, version, "grpc"),
		ProtoDir: filepath.Join(cfg.Context.OutputDir, version, "proto"),
	}
	defer func() {
		if err := cleanupIntermediateFiles(outputConfig); err != nil {
			slog.Error("failed to cleanup", "err", err)
		}
	}()

	if err := invokeProtoc(ctx, cfg.Context, &api, outputConfig); err != nil {
		return fmt.Errorf("librariangen: gapic generation failed: %w", err)
	}
	// Unzip the temp-codegen.srcjar.
	srcjarPath := filepath.Join(outputConfig.GAPICDir, "temp-codegen.srcjar")
	srcjarDest := outputConfig.GAPICDir
	if err := unzip(srcjarPath, srcjarDest); err != nil {
		return fmt.Errorf("librariangen: failed to unzip %s: %w", srcjarPath, err)
	}

	if err := restructureOutput(cfg.Context.OutputDir, libraryID, version); err != nil {
		return fmt.Errorf("librariangen: failed to restructure output: %w", err)
	}

	return nil
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

// invokeProtoc handles the protoc GAPIC generation logic for the 'generate' CLI command.
// It reads a request file, and for each API specified, it invokes protoc
// to generate the client library. It returns the module path and the path to the service YAML.
func invokeProtoc(ctx context.Context, genCtx *generate.Context, api *message.API, outputConfig *protoc.OutputConfig) error {
	apiServiceDir := filepath.Join(genCtx.SourceDir, api.Path)
	slog.Info("processing api", "service_dir", apiServiceDir)
	bazelConfig, err := bazelParse(apiServiceDir)
	if err != nil {
		return fmt.Errorf("librariangen: failed to parse BUILD.bazel for %s: %w", apiServiceDir, err)
	}
	args, err := protocBuild(apiServiceDir, bazelConfig, genCtx.SourceDir, outputConfig)
	if err != nil {
		return fmt.Errorf("librariangen: failed to build protoc command for api %q: %w", api.Path, err)
	}

	// Create protoc output directories.
	for _, dir := range []string{outputConfig.ProtoDir, outputConfig.GRPCDir, outputConfig.GAPICDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	if err := execvRun(ctx, args, genCtx.OutputDir); err != nil {
		return fmt.Errorf("librariangen: protoc failed for api %q: %w, execvRun error: %v", api.Path, err, err)
	}
	return nil
}

// moveFiles moves all files (and directories) from sourceDir to targetDir.
func moveFiles(sourceDir, targetDir string) error {
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("librariangen: failed to read dir %s: %w", sourceDir, err)
	}
	for _, f := range files {
		oldPath := filepath.Join(sourceDir, f.Name())
		newPath := filepath.Join(targetDir, f.Name())
		slog.Debug("librariangen: moving file", "from", oldPath, "to", newPath)
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("librariangen: failed to move %s to %s: %w, os.Rename error: %v", oldPath, newPath, err, err)
		}
	}
	return nil
}

func restructureOutput(outputDir, libraryID, version string) error {
	slog.Debug("librariangen: restructuring output directory", "dir", outputDir)

	// Define source and destination directories.
	gapicSrcDir := filepath.Join(outputDir, version, "gapic", "src", "main", "java")
	gapicTestDir := filepath.Join(outputDir, version, "gapic", "src", "test", "java")
	protoSrcDir := filepath.Join(outputDir, version, "proto")
	resourceNameSrcDir := filepath.Join(outputDir, version, "gapic", "proto", "src", "main", "java")
	samplesDir := filepath.Join(outputDir, version, "gapic", "samples", "snippets")

	gapicDestDir := filepath.Join(outputDir, fmt.Sprintf("google-cloud-%s", libraryID), "src", "main", "java")
	gapicTestDestDir := filepath.Join(outputDir, fmt.Sprintf("google-cloud-%s", libraryID), "src", "test", "java")
	protoDestDir := filepath.Join(outputDir, fmt.Sprintf("proto-google-cloud-%s-%s", libraryID, version), "src", "main", "java")
	resourceNameDestDir := filepath.Join(outputDir, fmt.Sprintf("proto-google-cloud-%s-%s", libraryID, version), "src", "main", "java")
	grpcDestDir := filepath.Join(outputDir, fmt.Sprintf("grpc-google-cloud-%s-%s", libraryID, version), "src", "main", "java")
	samplesDestDir := filepath.Join(outputDir, "samples", "snippets")

	// Create destination directories.
	destDirs := []string{gapicDestDir, gapicTestDestDir, protoDestDir, samplesDestDir, grpcDestDir}
	for _, dir := range destDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// The resource name directory is not created if there are no resource names
	// to generate. We create it here to avoid errors later.
	if _, err := os.Stat(resourceNameSrcDir); os.IsNotExist(err) {
		if err := os.MkdirAll(resourceNameSrcDir, 0755); err != nil {
			return err
		}
	}

	// Remove the location classes from the proto output to avoid conflicts with
	// proto-google-common-protos.
	if err := os.RemoveAll(filepath.Join(protoSrcDir, "com", "google", "cloud", "location")); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(protoSrcDir, "google", "cloud", "CommonResources.java")); err != nil {
		return err
	}

	// Move files that won't have conflicts.
	moves := map[string]string{
		filepath.Join(outputDir, version, "proto"): protoDestDir,
		filepath.Join(outputDir, version, "grpc"):  grpcDestDir,
	}
	for src, dest := range moves {
		if err := moveFiles(src, dest); err != nil {
			return err
		}
	}

	// Merge the gapic source and test files.
	if err := copyAndMerge(gapicSrcDir, gapicDestDir); err != nil {
		return err
	}
	if err := copyAndMerge(gapicTestDir, gapicTestDestDir); err != nil {
		return err
	}
	if err := copyAndMerge(samplesDir, samplesDestDir); err != nil {
		return err
	}

	// Merge the resource name files into the proto destination.
	if err := copyAndMerge(resourceNameSrcDir, resourceNameDestDir); err != nil {
		return err
	}

	return nil
}

// copyAndMerge recursively copies the contents of src to dest, merging directories.
func copyAndMerge(src, dest string) error {
	entries, err := os.ReadDir(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			if err := copyAndMerge(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if err := os.Rename(srcPath, destPath); err != nil {
				return fmt.Errorf("librariangen: failed to move %s to %s: %w, os.Rename error: %v", srcPath, destPath, err, err)
			}
		}
	}
	return nil
}

func cleanupIntermediateFiles(outputConfig *protoc.OutputConfig) error {
	slog.Debug("librariangen: cleaning up intermediate files")
	return os.RemoveAll(filepath.Dir(outputConfig.GAPICDir))
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
			return fmt.Errorf("librariangen: illegal file path: %s", fpath)
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
		rc.Close() // Error on read-only file close is less critical
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
