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
	if len(library.Channels) == 0 {
		return fmt.Errorf("no channels specified for library %s", library.Name)
	}

	// Convert library.Output to absolute path since protoc runs from a
	// different directory.
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output directory: %w", err)
	}

	// Create output directory in case it's a new library
	// (or cleaning has removed everything).
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Some aspects of generation currently require the repo root. Compute it once here
	// and pass it down.
	repoRoot := filepath.Dir(filepath.Dir(outdir))

	// Generate each channel separately.
	for _, channel := range library.Channels {
		if err := generateChannel(ctx, channel, library, googleapisDir, repoRoot); err != nil {
			return fmt.Errorf("failed to generate channel %s: %w", channel.Path, err)
		}
	}

	// TODO(https://github.com/googleapis/librarian/issues/3157):
	// Copy files from .librarian/generator-input/client-post-processing
	// for post processing, or reimplement.

	// TODO(https://github.com/googleapis/librarian/issues/3146):
	// Remove the default version fudget here, as GenerateRepoMetadata should
	// compute it. For now, use the last component of the first channel path as
	// the default version.
	defaultVersion := filepath.Base(library.Channels[0].Path)
	source := map[string]string{
		"googleapisDir": googleapisDir,
	}
	// Generate .repo-metadata.json from the service config in the first
	// channel.
	// TODO(https://github.com/googleapis/librarian/issues/3159): stop
	// hardcoding the language and repo name, instead getting it passed in.
	channel, err := serviceconfig.Find(source, library.Channels[0].Path)
	if err != nil {
		return fmt.Errorf("failed to lookup service config: %w", err)
	}
	absoluteServiceConfig := filepath.Join(googleapisDir, channel.ServiceConfig)
	if err := repometadata.GenerateRepoMetadata(library, "python", "googleapis/google-cloud-python", absoluteServiceConfig, defaultVersion, outdir); err != nil {
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

// generateChannel generates part of a library for a single channel.
func generateChannel(ctx context.Context, channel *config.Channel, library *config.Library, googleapisDir, repoRoot string) error {
	// Note: the Python Librarian container generates to a temporary directory,
	// then the results into owl-bot-staging. We generate straight into
	// owl-bot-staging instead. The post-processor then moves the files into
	// the correct final position in the repository.
	// TODO(https://github.com/googleapis/librarian/issues/3210): generate
	// directly in place.

	stagingChildDirectory := getStagingChildDirectory(channel.Path)
	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name, stagingChildDirectory)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return err
	}
	protocOptions, err := createProtocOptions(channel, library, googleapisDir, stagingDir)
	if err != nil {
		return err
	}

	apiDir := filepath.Join(googleapisDir, channel.Path)
	protos, err := filepath.Glob(apiDir + "/*.proto")
	if err != nil {
		return fmt.Errorf("globbing for protos failed: %w", err)
	}
	if len(protos) == 0 {
		return fmt.Errorf("channel has no protos: %s", channel.Path)
	}

	// We want the proto filenames to be relative to googleapisDir
	for index, protoFile := range protos {
		rel, err := filepath.Rel(googleapisDir, protoFile)
		if err != nil {
			return fmt.Errorf("can't find relative path to proto")
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

func createProtocOptions(ch *config.Channel, library *config.Library, googleapisDir, stagingDir string) ([]string, error) {
	// GAPIC library: generate full client library
	var opts []string

	// Add transport option
	if library.Transport != "" {
		opts = append(opts, fmt.Sprintf("transport=%s", library.Transport))
	}

	// TODO(https://github.com/googleapis/librarian/issues/3161):
	// Make these conditional on configuration.
	opts = append(opts, "rest-numeric-enums")
	opts = append(opts, "metadata")

	// Add Python-specific options
	// First common options that apply to all channels
	if library.Python != nil && len(library.Python.OptArgs) > 0 {
		opts = append(opts, library.Python.OptArgs...)
	}
	// Then options that apply to this specific channel
	if library.Python != nil && len(library.Python.OptArgsByChannel) > 0 {
		apiOptArgs, ok := library.Python.OptArgsByChannel[ch.Path]
		if ok {
			opts = append(opts, apiOptArgs...)
		}
	}

	// Add gapic-version from library version
	if library.Version != "" {
		opts = append(opts, fmt.Sprintf("gapic-version=%s", library.Version))
	}

	// Add gRPC service config (retry/timeout settings)
	// Auto-discover: look for *_grpc_service_config.json in the API directory
	apiDir := filepath.Join(googleapisDir, ch.Path)
	grpcConfigPath := ""
	matches, err := filepath.Glob(filepath.Join(apiDir, "*_grpc_service_config.json"))
	if err == nil && len(matches) > 0 {
		if len(matches) > 1 {
			return nil, fmt.Errorf("multiple _grpc_service_config.json files found in %s", apiDir)
		}
		rel, err := filepath.Rel(googleapisDir, matches[0])
		if err != nil {
			return nil, fmt.Errorf("unable to make path relative: %s", matches[0])
		}
		grpcConfigPath = rel
	}
	if grpcConfigPath != "" {
		opts = append(opts, fmt.Sprintf("retry-config=%s", grpcConfigPath))
	}
	source := map[string]string{
		"googleapisDir": googleapisDir,
	}
	channel, err := serviceconfig.Find(source, ch.Path)
	if err != nil {
		return nil, err
	}
	if channel != nil && channel.ServiceConfig != "" {
		opts = append(opts, fmt.Sprintf("service-yaml=%s", channel.ServiceConfig))
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
