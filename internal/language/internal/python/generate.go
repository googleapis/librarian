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

package python

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

const (
	googleapisRepo = "github.com/googleapis/googleapis"
)

// Generate generates a Python client library.
// Files and directories specified in library.Keep will be preserved during regeneration.
// If library.Keep is not specified, a default list of paths is used.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library) error {
	googleapisDir, err := sourceDir(cfg.Sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}

	if err := prepareLibrary(cfg, library, googleapisDir); err != nil {
		return err
	}

	// Convert to absolute path since protoc runs from a different directory
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output directory: %w", err)
	}

	// Clean output directory before generation
	if err := cleanOutputDirectory(outdir, library.Keep); err != nil {
		return fmt.Errorf("failed to clean output directory: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get API paths to generate (used later, I believe...)
	apiPaths := library.Channels
	if library.Channel != "" {
		apiPaths = []string{library.Channel}
	}
	if len(apiPaths) == 0 {
		return fmt.Errorf("no APIs specified for library %s", library.Name)
	}

	// Get transport from library or defaults
	transport := library.Transport
	defaults := cfg.Default
	if transport == "" && defaults != nil && defaults.Generate != nil {
		transport = defaults.Generate.Transport
	}

	// Get rest_numeric_enums from library or defaults
	restNumericEnums := defaults != nil && defaults.Generate != nil && defaults.Generate.RestNumericEnums
	if library.RestNumericEnums != nil {
		restNumericEnums = *library.RestNumericEnums
	}

	// Generate each API with its own service config
	for apiPath, apiServiceConfig := range library.APIServiceConfigs {
		if err := generateAPI(ctx, apiPath, library, googleapisDir, apiServiceConfig, outdir, transport, restNumericEnums); err != nil {
			return fmt.Errorf("failed to generate API %s: %w", apiPath, err)
		}
	}

	// Copy files needed for post processing (e.g., .repo-metadata.json, scripts)
	repoRoot := filepath.Dir(filepath.Dir(outdir)) // Go up two levels from packages/libname to repo root
	if err := copyFilesNeededForPostProcessing(outdir, library, repoRoot); err != nil {
		return fmt.Errorf("failed to copy files for post processing: %w", err)
	}

	// Generate .repo-metadata.json from service config
	if err := generateRepoMetadataFile(outdir, cfg, library, apiPaths); err != nil {
		return fmt.Errorf("failed to generate .repo-metadata.json: %w", err)
	}

	// Run post processor (synthtool/owlbot)
	// The post processor needs to run from the repository root, not the package directory
	if err := runPostProcessor(repoRoot, library.Name); err != nil {
		return fmt.Errorf("failed to run post processor: %w", err)
	}

	// Copy README.rst to docs/README.rst
	if err := copyReadmeToDocsDir(outdir, library.Name); err != nil {
		return fmt.Errorf("failed to copy README to docs: %w", err)
	}

	// Clean up files that shouldn't be in the final output
	if err := cleanUpFilesAfterPostProcessing(outdir, library.Name); err != nil {
		return fmt.Errorf("failed to cleanup after post processing: %w", err)
	}

	return nil
}

// generateAPI generates code for a single API.
func generateAPI(ctx context.Context, apiPath string, library *config.Library, googleapisDir, serviceConfigPath, outdir, transport string, restNumericEnums bool) error {
	// The Python Librarian container generates to a temporary directory,
	// then the results into owl-bot-staging. We generate straight into
	// owl-bot-staging instead. The post-processor then moves the files into
	// the correct final position in the repository.

	// Check if this is a proto-only library
	isProtoOnly := library.Python != nil && library.Python.IsProtoOnly

	stagingChildDirectory := getStagingChildDirectory(apiPath, isProtoOnly)
	repoRoot := filepath.Dir(filepath.Dir(outdir))
	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name, stagingChildDirectory)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return err
	}

	protoPattern := filepath.Join(apiPath, "*.proto")
	var args []string
	var cmdStr string

	if isProtoOnly {
		// Proto-only library: generate Python proto files only
		args = []string{
			protoPattern,
			fmt.Sprintf("--python_out=%s", stagingDir),
			fmt.Sprintf("--pyi_out=%s", stagingDir),
		}
		cmdStr = "protoc " + strings.Join(args, " ")
	} else {
		// GAPIC library: generate full client library
		var opts []string

		// Add transport option
		if transport != "" {
			opts = append(opts, fmt.Sprintf("transport=%s", transport))
		}

		// Add rest_numeric_enums option
		if restNumericEnums {
			opts = append(opts, "rest-numeric-enums")
		}

		// Add Python-specific options
		// First common options that apply to all channels
		if library.Python != nil && len(library.Python.OptArgs) > 0 {
			opts = append(opts, library.Python.OptArgs...)
		}
		// Then options that apply to this specific channel
		if library.Python != nil && len(library.Python.OptArgsByChannel) > 0 {
			apiOptArgs, ok := library.Python.OptArgsByChannel[apiPath]
			if ok {
				opts = append(opts, apiOptArgs...)
			}
		}

		// Add gapic-version from library version
		if library.Version != "" {
			opts = append(opts, fmt.Sprintf("gapic-version=%s", library.Version))
		}

		// Add gRPC service config (retry/timeout settings)
		// Try library config first, then auto-discover
		grpcConfigPath := ""
		if library.GRPCServiceConfig != "" {
			// GRPCServiceConfig is relative to the API directory
			grpcConfigPath = filepath.Join(googleapisDir, apiPath, library.GRPCServiceConfig)
		} else {
			// Auto-discover: look for *_grpc_service_config.json in the API directory
			apiDir := filepath.Join(googleapisDir, apiPath)
			matches, err := filepath.Glob(filepath.Join(apiDir, "*_grpc_service_config.json"))
			if err == nil && len(matches) > 0 {
				grpcConfigPath = matches[0]
			}
		}
		if grpcConfigPath != "" {
			opts = append(opts, fmt.Sprintf("retry-config=%s", grpcConfigPath))
		}

		// Add service YAML (API metadata) if provided
		if serviceConfigPath != "" {
			opts = append(opts, fmt.Sprintf("service-yaml=%s", serviceConfigPath))
		}

		args = []string{
			protoPattern,
			fmt.Sprintf("--python_gapic_out=%s", stagingDir),
		}

		// Add options if any
		if len(opts) > 0 {
			optString := "metadata," + strings.Join(opts, ",")
			args = append(args, fmt.Sprintf("--python_gapic_opt=%s", optString))
		}

		cmdStr = "protoc " + strings.Join(args, " ")
	}

	// Debug: print the protoc command
	fmt.Fprintf(os.Stderr, "\nRunning: %s\n", cmdStr)

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = googleapisDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("protoc command failed: %w", err)
	}

	// Copy the proto files as well as the generated code.
	if isProtoOnly {
		apiDir := filepath.Join(googleapisDir, apiPath)
		matches, err := filepath.Glob(filepath.Join(apiDir, "*.proto"))
		if err != nil {
			return fmt.Errorf("copying proto file failed: %w", err)
		}
		protoStagingDir := filepath.Join(stagingDir, apiPath)
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				return fmt.Errorf("reading proto file failed: %w", err)
			}
			if err := os.WriteFile(filepath.Join(protoStagingDir, filepath.Base(match)), content, 0644); err != nil {
				return fmt.Errorf("writing proto file failed: %w", err)
			}
		}
	}

	return nil
}

// getStagingChildDirectory determines where within owl-bot-staging/{library-name} the
// generated code the given API path should be staged. This is not quite equivalent
// to _get_staging_child_directory in the Python container, as for proto-only directories
// we don't want the apiPath suffix.
func getStagingChildDirectory(apiPath string, isProtoOnly bool) string {
	versionCandidate := filepath.Base(apiPath)
	if strings.HasPrefix(versionCandidate, "v") && !isProtoOnly {
		return versionCandidate
	} else {
		return versionCandidate + "-py"
	}
}

// copyFilesNeededForPostProcessing copies files needed during post processing.
// This includes .repo-metadata.json and client-post-processing scripts from the input directory.
func copyFilesNeededForPostProcessing(outdir string, library *config.Library, repoRoot string) error {
	if repoRoot == "" {
		return nil
	}

	inputDir := filepath.Join(repoRoot, ".librarian", "generator-input")
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		// No input directory, nothing to copy
		return nil
	}

	pathToLibrary := filepath.Join("packages", library.Name)
	sourceDir := filepath.Join(inputDir, pathToLibrary)

	// Copy files from input/packages/{library_name} to output, excluding client-post-processing
	if _, err := os.Stat(sourceDir); err == nil {
		if err := copyDirExcluding(sourceDir, outdir, "client-post-processing"); err != nil {
			return fmt.Errorf("failed to copy input files: %w", err)
		}
	}

	// Create scripts/client-post-processing directory
	scriptsDir := filepath.Join(outdir, pathToLibrary, "scripts", "client-post-processing")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Copy relevant client-post-processing YAML files
	// Try both locations: .librarian/generator-input/client-post-processing and synthtool-input
	postProcessingDirs := []string{
		filepath.Join(inputDir, "client-post-processing"),
		filepath.Join(repoRoot, "synthtool-input"),
	}

	for _, postProcessingDir := range postProcessingDirs {
		yamlFiles, err := filepath.Glob(filepath.Join(postProcessingDir, "*.yaml"))
		if err != nil {
			continue // If glob fails, try next location
		}

		for _, yamlFile := range yamlFiles {
			// Read the file to check if it applies to this library
			content, err := os.ReadFile(yamlFile)
			if err != nil {
				continue
			}

			// Check if the file references this library's path
			if strings.Contains(string(content), pathToLibrary+"/") {
				destPath := filepath.Join(scriptsDir, filepath.Base(yamlFile))
				if err := copyFile(yamlFile, destPath); err != nil {
					return fmt.Errorf("failed to copy post-processing file %s: %w", yamlFile, err)
				}
			}
		}
	}

	return nil
}

// copyDirExcluding copies a directory tree, excluding files/dirs matching the exclude pattern.
func copyDirExcluding(src, dst, exclude string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories
		if info.IsDir() && info.Name() == exclude {
			return filepath.SkipDir
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := destFile.ReadFrom(sourceFile); err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

// generateRepoMetadataFile generates the .repo-metadata.json
func generateRepoMetadataFile(outdir string, cfg *config.Config, library *config.Library, apiPaths []string) error {
	// Use the first service config path (all point to the same service YAML)
	var primaryServiceConfigPath string
	for _, serviceConfigPath := range library.APIServiceConfigs {
		primaryServiceConfigPath = serviceConfigPath
		break
	}

	return config.GenerateRepoMetadata(library, cfg.Language, cfg.Repo, primaryServiceConfigPath, outdir, apiPaths)
}

// runPostProcessor runs the synthtool post processor on the output directory.
func runPostProcessor(outdir, libraryName string) error {
	pathToLibrary := filepath.Join("packages", libraryName)

	fmt.Fprintf(os.Stderr, "\nRunning Python post-processor...\n")

	// Run python_mono_repo.owlbot_main
	pythonCode := fmt.Sprintf(`
from synthtool.languages import python_mono_repo
python_mono_repo.owlbot_main(%q)
`, pathToLibrary)
	cmd := exec.Command("python3", "-c", pythonCode)
	cmd.Dir = outdir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	fmt.Fprintf(os.Stderr, "Running: python3 -c %q\n", pythonCode)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("post processor failed: %w", err)
	}

	// If there is no noxfile, run isort and black
	// This is required for proto-only libraries which are not GAPIC
	libraryDir := filepath.Join(outdir, pathToLibrary)
	noxfilePath := filepath.Join(libraryDir, "noxfile.py")
	if _, err := os.Stat(noxfilePath); os.IsNotExist(err) {
		if err := runIsort(libraryDir); err != nil {
			return err
		}
		if err := runBlackFormatter(libraryDir); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "Python post-processor ran successfully.\n")
	return nil
}

// runIsort runs the isort import sorter on Python files in the library's directory.
// The --fss flag forces strict alphabetical sorting within sections.
func runIsort(libraryDir string) error {
	fmt.Fprintf(os.Stderr, "\nRunning: isort --fss %s\n", libraryDir)
	cmd := exec.Command("isort", "--fss", libraryDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("isort failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// runBlackFormatter runs the black code formatter on Python files in the output directory.
// Black enforces double quotes and consistent Python formatting.
func runBlackFormatter(libraryDir string) error {
	fmt.Fprintf(os.Stderr, "\nRunning: black %s\n", libraryDir)
	cmd := exec.Command("black", libraryDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("black formatter failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// copyReadmeToDocsDir copies README.rst to docs/README.rst.
// This handles symlinks properly by reading content and writing a real file.
func copyReadmeToDocsDir(outdir, libraryName string) error {
	pathToLibrary := filepath.Join(outdir, "packages", libraryName)
	sourcePath := filepath.Join(pathToLibrary, "README.rst")
	docsPath := filepath.Join(pathToLibrary, "docs")
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
func cleanUpFilesAfterPostProcessing(outdir, libraryName string) error {
	pathToLibrary := filepath.Join(outdir, "packages", libraryName)

	// Remove .nox directory
	if err := os.RemoveAll(filepath.Join(pathToLibrary, ".nox")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove .nox: %w", err)
	}

	// Remove owl-bot-staging
	repoRoot := filepath.Dir(filepath.Dir(outdir))
	if err := os.RemoveAll(filepath.Join(repoRoot, "owl-bot-staging")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove owl-bot-staging: %w", err)
	}

	// Remove CHANGELOG.md files
	os.Remove(filepath.Join(pathToLibrary, "CHANGELOG.md"))
	os.Remove(filepath.Join(pathToLibrary, "docs", "CHANGELOG.md"))

	// Remove client-post-processing YAML files
	scriptsPath := filepath.Join(pathToLibrary, "scripts", "client-post-processing")
	if yamlFiles, err := filepath.Glob(filepath.Join(scriptsPath, "*.yaml")); err == nil {
		for _, f := range yamlFiles {
			os.Remove(f)
		}
	}

	return nil
}
