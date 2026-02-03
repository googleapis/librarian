// Copyright 2025 Google LLC
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

// Package python provides Python specific functionality for librarian.
package python

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// Generate generates a Python client library.
func Generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	if len(library.APIs) == 0 {
		return fmt.Errorf("no apis configured for library %q", library.Name)
	}

	// Convert library.Output to absolute path since protoc runs from a
	// different directory.
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}

	// Create output directory in case it's a new library
	// (or cleaning has removed everything).
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Some aspects of generation currently require the repo root. Compute it once here
	// and pass it down.
	repoRoot := filepath.Dir(filepath.Dir(outdir))
	for _, api := range library.APIs {
		if err := generateAPI(ctx, api, library, googleapisDir, repoRoot); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}

	// TODO(https://github.com/googleapis/librarian/issues/3157):
	// Copy files from .librarian/generator-input/client-post-processing
	// for post processing, or reimplement.

	// TODO(https://github.com/googleapis/librarian/issues/3146):
	// Remove the default version fudget here, as Generate should
	// compute it. For now, use the last component of the first api path as
	// the default version.
	defaultVersion := filepath.Base(library.APIs[0].Path)

	// Generate .repo-metadata.json from the service config in the first
	// api.
	// TODO(https://github.com/googleapis/librarian/issues/3159): stop
	// hardcoding the language and repo name, instead getting it passed in.
	api, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path)
	if err != nil {
		return fmt.Errorf("failed to find service config: %w", err)
	}
	absoluteServiceConfig := filepath.Join(googleapisDir, api.ServiceConfig)
	if err := repometadata.Generate(library, "python", "googleapis/google-cloud-python", absoluteServiceConfig, defaultVersion, outdir); err != nil {
		return fmt.Errorf("failed to generate .repo-metadata.json: %w", err)
	}

	// Run post processor (synthtool)
	// The post processor needs to run from the repository root, not the package directory.
	if err := runPostProcessor(ctx, repoRoot, outdir); err != nil {
		return fmt.Errorf("failed to run post processor: %w", err)
	}

	// Copy README.rst to docs/README.rst
	if err := copyReadmeToDocsDir(outdir); err != nil {
		return fmt.Errorf("failed to copy README to docs: %w", err)
	}

	// Clean up files that shouldn't be in the final output.
	if err := cleanUpFilesAfterPostProcessing(repoRoot); err != nil {
		return fmt.Errorf("failed to cleanup after post processing: %w", err)
	}

	return nil
}

// generateAPI generates part of a library for a single api.
func generateAPI(ctx context.Context, api *config.API, library *config.Library, googleapisDir, repoRoot string) error {
	// Note: the Python Librarian container generates to a temporary directory,
	// then the results into owl-bot-staging. We generate straight into
	// owl-bot-staging instead. The post-processor then moves the files into
	// the correct final position in the repository.
	// TODO(https://github.com/googleapis/librarian/issues/3210): generate
	// directly in place.

	stagingChildDirectory := getStagingChildDirectory(api.Path)
	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name, stagingChildDirectory)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return err
	}
	protocOptions, err := createProtocOptions(api, library, googleapisDir, stagingDir)
	if err != nil {
		return err
	}

	apiDir := filepath.Join(googleapisDir, api.Path)
	protos, err := filepath.Glob(apiDir + "/*.proto")
	if err != nil {
		return fmt.Errorf("failed to find protos: %w", err)
	}
	if len(protos) == 0 {
		return fmt.Errorf("no protos found in api %q", api.Path)
	}

	// We want the proto filenames to be relative to googleapisDir
	for index, protoFile := range protos {
		rel, err := filepath.Rel(googleapisDir, protoFile)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %q: %w", protoFile, err)
		}
		protos[index] = rel
	}

	cmdArgs := []string{"protoc"}
	cmdArgs = append(cmdArgs, protos...)
	cmdArgs = append(cmdArgs, protocOptions...)

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = googleapisDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", cmd.String(), err)
	}

	return nil
}

func createProtocOptions(ch *config.API, library *config.Library, googleapisDir, stagingDir string) ([]string, error) {
	// GAPIC library: generate full client library
	opts := []string{"metadata"}

	// Add Python-specific options
	// First common options that apply to all apis
	if library.Python != nil && len(library.Python.OptArgs) > 0 {
		opts = append(opts, library.Python.OptArgs...)
	}
	// Then options that apply to this specific api
	if library.Python != nil && len(library.Python.OptArgsByAPI) > 0 {
		apiOptArgs, ok := library.Python.OptArgsByAPI[ch.Path]
		if ok {
			opts = append(opts, apiOptArgs...)
		}
	}
	restNumericEnums := true
	addTransport := library.Transport != ""
	for _, opt := range opts {
		if strings.HasPrefix(opt, "rest-numeric-enums") {
			restNumericEnums = false
		}
		if strings.HasPrefix(opt, "transport=") {
			addTransport = false
		}
	}

	// Add rest-numeric-enums, if we haven't already got it.
	if restNumericEnums {
		opts = append(opts, "rest-numeric-enums")
	}

	// Add transport option, if we haven't already got it.
	if addTransport {
		opts = append(opts, fmt.Sprintf("transport=%s", library.Transport))
	}

	// Add gapic-version from library version
	if library.Version != "" {
		opts = append(opts, fmt.Sprintf("gapic-version=%s", library.Version))
	}

	// Add gRPC service config (retry/timeout settings)
	grpcConfigPath, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, ch.Path)
	if err != nil {
		return nil, err
	}
	// TODO(https://github.com/googleapis/librarian/issues/3827): remove this
	// hardcoding once we can use the gRPC service config for Compute.
	if strings.HasPrefix(library.Name, "google-cloud-compute") {
		grpcConfigPath = ""
	}
	if grpcConfigPath != "" {
		opts = append(opts, fmt.Sprintf("retry-config=%s", grpcConfigPath))
	}

	api, err := serviceconfig.Find(googleapisDir, ch.Path)
	if err != nil {
		return nil, err
	}
	if api != nil && api.ServiceConfig != "" {
		opts = append(opts, fmt.Sprintf("service-yaml=%s", api.ServiceConfig))
	}

	return []string{
		fmt.Sprintf("--python_gapic_out=%s", stagingDir),
		fmt.Sprintf("--python_gapic_opt=%s", strings.Join(opts, ",")),
	}, nil
}

// getStagingChildDirectory determines where within owl-bot-staging/{library-name} the
// generated code the given API path should be staged. This is not quite equivalent
// to _get_staging_child_directory in the Python container, as for proto-only directories
// we don't want the apiPath suffix.
func getStagingChildDirectory(apiPath string) string {
	versionCandidate := filepath.Base(apiPath)
	if strings.HasPrefix(versionCandidate, "v") {
		return versionCandidate
	} else {
		return versionCandidate + "-py"
	}
}

// runPostProcessor runs the synthtool post processor on the output directory.
func runPostProcessor(ctx context.Context, repoRoot, outDir string) error {
	pythonCode := fmt.Sprintf(`
from synthtool.languages import python_mono_repo
python_mono_repo.owlbot_main(%q)
`, outDir)
	cmd := exec.CommandContext(ctx, "python3", "-c", pythonCode)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", cmd.String(), err)
	}
	return nil
}

// copyReadmeToDocsDir copies README.rst to docs/README.rst.
// This handles symlinks properly by reading content and writing a real file.
func copyReadmeToDocsDir(outdir string) error {
	sourcePath := filepath.Join(outdir, "README.rst")
	docsPath := filepath.Join(outdir, "docs")
	destPath := filepath.Join(docsPath, "README.rst")

	// If source doesn't exist, nothing to copy
	if _, err := os.Lstat(sourcePath); os.IsNotExist(err) {
		return nil
	}

	// Read content from source (follows symlinks)
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Create docs directory if it doesn't exist
	if err := os.MkdirAll(docsPath, 0755); err != nil {
		return err
	}

	// Remove any existing symlink at destination
	if info, err := os.Lstat(destPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(destPath); err != nil {
				return err
			}
		}
	}

	// Write content to destination as a real file
	return os.WriteFile(destPath, content, 0644)
}

// cleanUpFilesAfterPostProcessing cleans up files after post processing.
// TODO(https://github.com/googleapis/librarian/issues/3210): generate
// directly in place and remove this code entirely.
func cleanUpFilesAfterPostProcessing(repoRoot string) error {
	// Remove owl-bot-staging
	if err := os.RemoveAll(filepath.Join(repoRoot, "owl-bot-staging")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove owl-bot-staging: %w", err)
	}

	return nil
}

// DefaultOutputByName derives an output path from a library name and a default
// output directory. Currently this just assumes each library is a directory
// directly underneath the default output directory.
func DefaultOutputByName(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}

// DefaultLibraryName derives a library name from an API path by stripping
// the version suffix and replacing "/" with "-".
// For example: "google/cloud/secretmanager/v1" ->
// "google-cloud-secretmanager".
func DefaultLibraryName(api string) string {
	path := api
	if v := filepath.Base(api); len(v) > 1 && v[0] == 'v' && v[1] >= '0' && v[1] <= '9' {
		// Strip version suffix (v1, v1beta2, v2alpha, etc.).
		path = filepath.Dir(api)
	}
	return strings.ReplaceAll(path, "/", "-")
}
