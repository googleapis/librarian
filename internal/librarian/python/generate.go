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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

const (
	cloudGoogleComDocumentationTemplate = "https://cloud.google.com/python/docs/reference/%s/latest"
	googleapisDevDocumentationTemplate  = "https://googleapis.dev/python/%s/latest"
	transportOption                     = "transport"
	gapicNamespaceOption                = "python-gapic-namespace"
	gapicNameOption                     = "python-gapic-name"
	warehousePackageNameOption          = "warehouse-package-name"
)

var (
	errNoDefaultVersion        = errors.New("default version must be specified for every library with generated APIs")
	errExplicitTransportOption = errors.New("transport option is derived from sdk.yaml and must not be specified explicitly")
)

// Generate generates a Python client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, srcs *sources.Sources) error {
	googleapisDir := srcs.Googleapis
	// Convert library.Output to absolute path since protoc runs from a
	// different directory.
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}

	// For preview libraries, the API protos are rooted in the
	// googleapis/preview subdirectory, so change the googleapisDir to target
	// that root.
	if isPreview(outdir) {
		googleapisDir = filepath.Join(googleapisDir, "preview")
	}

	// Create output directory in case it's a new library
	// (or cleaning has removed everything).
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Some aspects of generation currently require the repo root. Compute it
	// once here and pass it down.
	repoRoot := filepath.Dir(filepath.Dir(outdir))

	// In order to make sure we generate google/cloud/firestore/v1 *after*
	// google/cloud/firestore/admin/v1 (etc), sort the APIs in descending path
	// length order before generation. This is pretty ghastly, but it works to
	// minimize the diff during generation. (And it's deterministic.)
	// TODO(https://github.com/googleapis/librarian/issues/4740): remove this
	// sorting and just use library.APIs.
	apisSortedByPathLength := slices.Clone(library.APIs)
	slices.SortFunc(apisSortedByPathLength, func(a, b *config.API) int {
		return len(b.Path) - len(a.Path)
	})
	for _, api := range apisSortedByPathLength {
		if err := generateAPI(ctx, api, library, googleapisDir, repoRoot); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}

	// Construct the repo metadata in memory, then write it to disk. This has
	// to be before post-processing, as the data in .repo-metadata.json is used
	// by the post-processor, primarily for documentation.
	repoMetadata, err := createRepoMetadata(cfg, library, googleapisDir)
	if err != nil {
		return err
	}
	if err := repoMetadata.Write(library.Output); err != nil {
		return err
	}

	// Run post processor (synthtool) and then clean up afterwards.
	// The post processor needs to run from the repository root, not the package
	// directory.
	if len(library.APIs) > 0 {
		if err := runPostProcessor(ctx, repoRoot, outdir); err != nil {
			return fmt.Errorf("failed to run post processor: %w", err)
		}
		if err := cleanUpFilesAfterPostProcessing(repoRoot, outdir); err != nil {
			return fmt.Errorf("failed to cleanup after post processing: %w", err)
		}
	}

	if library.Python == nil || !library.Python.SkipReadmeCopy {
		if err := copyReadmeToDocsDir(outdir); err != nil {
			return fmt.Errorf("failed to copy README to docs: %w", err)
		}
	}
	return nil
}

// createRepoMetadata creates (in memory, not on disk) a RepoMetadata suitable
// for the given library.
func createRepoMetadata(cfg *config.Config, library *config.Library, googleapisDir string) (*repometadata.RepoMetadata, error) {
	// Just to avoid lots of checks for library.Python being nil.
	packageOptions := library.Python
	if packageOptions == nil {
		packageOptions = &config.PythonPackage{}
	}
	var repoMetadata *repometadata.RepoMetadata
	if len(library.APIs) > 0 {
		var err error
		repoMetadata, err = repometadata.FromLibrary(cfg, library, googleapisDir)
		if err != nil {
			return nil, err
		}
		// Require the DefaultVersion field, even if we could have inferred
		// it. The default version affects the final code, and changes to it
		// should be explicit - if adding a new version of an API changes the
		// inferred default version, that would cause compatibility issues. This
		// in itself is far from ideal; keeping the default version is "safe"
		// but toilsome operationally.
		// TODO(https://github.com/googleapis/librarian/issues/4772): design away
		// from default versions.
		if packageOptions.DefaultVersion == "" {
			return nil, fmt.Errorf("error creating metadata for %s: %w", library.Name, errNoDefaultVersion)
		}
		repoMetadata.DefaultVersion = packageOptions.DefaultVersion
	} else {
		// Handwritten library: populate from scratch (and then apply overrides
		// as normal).
		releaseLevel := "stable"
		if library.Version == "" || strings.HasPrefix(library.Version, "0.") {
			releaseLevel = "preview"
		}
		repoMetadata = &repometadata.RepoMetadata{
			Name:             library.Name,
			DistributionName: library.Name,
			Language:         cfg.Language,
			ReleaseLevel:     releaseLevel,
			Repo:             cfg.Repo,
			// Allow even handwritten libraries to specify a default value in
			// the package options if they want to. This would be unusual, but
			// if it's specified, we should honor it.
			DefaultVersion: packageOptions.DefaultVersion,
		}
	}
	if packageOptions.MetadataNameOverride != "" {
		repoMetadata.Name = packageOptions.MetadataNameOverride
	} else {
		repoMetadata.Name = library.Name
	}
	repoMetadata.LibraryType = packageOptions.LibraryType
	repoMetadata.ClientDocumentation = BuildClientDocumentationURI(library.Name, repoMetadata.Name)
	// Even after migration oddities, just a few libraries don't fit into the
	// normal pattern for client documentation URI (e.g. the documentation is
	// in cloud.google.com when it would be expected to be in googleapis.dev).
	if packageOptions.ClientDocumentationOverride != "" {
		repoMetadata.ClientDocumentation = packageOptions.ClientDocumentationOverride
	}
	// TODO(https://github.com/googleapis/librarian/issues/4175): remove these.
	if packageOptions.NamePrettyOverride != "" {
		repoMetadata.NamePretty = packageOptions.NamePrettyOverride
	}
	if packageOptions.ProductDocumentationOverride != "" {
		repoMetadata.ProductDocumentation = packageOptions.ProductDocumentationOverride
	}
	if packageOptions.APIShortnameOverride != "" {
		repoMetadata.APIShortname = packageOptions.APIShortnameOverride
	}
	if packageOptions.APIIDOverride != "" {
		repoMetadata.APIID = packageOptions.APIIDOverride
	}
	if packageOptions.IssueTrackerOverride != "" {
		repoMetadata.IssueTracker = packageOptions.IssueTrackerOverride
	}
	return repoMetadata, nil
}

// BuildClientDocumentationURI builds the URI for the client documentation
// for the library.
// TODO(https://github.com/googleapis/librarian/issues/4175): make this function
// package-private (or inline it) after migration, when we won't need to
// determine whether or not to specify an override.
func BuildClientDocumentationURI(libraryName, repoMetadataName string) string {
	// Work out the right documentation URI based on whether this is a Cloud
	// or non-Cloud API.
	docTemplate := cloudGoogleComDocumentationTemplate
	if !strings.HasPrefix(libraryName, "google-cloud") {
		docTemplate = googleapisDevDocumentationTemplate
	}
	return fmt.Sprintf(docTemplate, repoMetadataName)
}

// generateAPI generates part of a library for a single api.
func generateAPI(ctx context.Context, api *config.API, library *config.Library, googleapisDir, repoRoot string) error {
	// Note: the Python Librarian container generates to a temporary directory,
	// then the results into owl-bot-staging. We generate straight into
	// owl-bot-staging instead. The post-processor then moves the files into
	// the correct final position in the repository.
	// TODO(https://github.com/googleapis/librarian/issues/3210): generate
	// directly in place.

	protoOnly := isProtoOnly(api, library)
	stagingChildDirectory := getStagingChildDirectory(api.Path, protoOnly)
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

	cmdArgs := append(protos, protocOptions...)
	if err := command.RunInDir(ctx, googleapisDir, "protoc", cmdArgs...); err != nil {
		return fmt.Errorf("failed to execute protoc: %w", err)
	}

	// Copy the proto files as well as the generated code for proto-only libraries.
	if protoOnly {
		if err := stageProtoFiles(googleapisDir, stagingDir, protos); err != nil {
			return err
		}
	}

	return nil
}

func stageProtoFiles(googleapisDir, targetDir string, relativeProtoPaths []string) error {
	for _, proto := range relativeProtoPaths {
		sourceProtoFile := filepath.Join(googleapisDir, proto)
		targetProtoFile := filepath.Join(targetDir, proto)
		dir := filepath.Dir(targetProtoFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s failed: %w", dir, err)
		}
		if err := filesystem.CopyFile(sourceProtoFile, targetProtoFile); err != nil {
			return fmt.Errorf("copying proto file %s failed: %w", sourceProtoFile, err)
		}
	}
	return nil
}

func createProtocOptions(api *config.API, library *config.Library, googleapisDir, stagingDir string) ([]string, error) {
	if isProtoOnly(api, library) {
		return []string{
			fmt.Sprintf("--python_out=%s", stagingDir),
			fmt.Sprintf("--pyi_out=%s", stagingDir),
		}, nil
	}
	// GAPIC library: generate full client library
	opts := []string{"metadata"}

	// Add Python-specific options that apply to this specific API.
	if library.Python != nil && len(library.Python.OptArgsByAPI) > 0 {
		apiOptArgs, ok := library.Python.OptArgsByAPI[api.Path]
		if ok {
			opts = append(opts, apiOptArgs...)
		}
	}
	apiMetadata, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguagePython)
	if err != nil {
		return nil, err
	}
	if apiMetadata.HasRESTNumericEnums(config.LanguagePython) {
		opts = append(opts, "rest-numeric-enums")
	}
	// The transport option should never be specified explicitly. Ensure it
	// hasn't been specified, and add the derived transport.
	if _, explicitTransport := findOption(opts, transportOption); explicitTransport {
		return nil, fmt.Errorf("error creating GAPIC options for %s: %w", api.Path, errExplicitTransportOption)
	}
	transport := serviceconfig.GRPCRest
	if apiMetadata != nil {
		transport = apiMetadata.Transport(config.LanguagePython)
	}
	opts = append(opts, fmt.Sprintf("%s=%s", transportOption, transport))

	// Add derived python-gapic-namespace option, if we haven't already got it.
	if _, ok := findOption(opts, gapicNamespaceOption); !ok {
		opts = append(opts, fmt.Sprintf("%s=%s", gapicNamespaceOption, deriveGAPICNamespace(api.Path)))
	}
	// Add derived python-gapic-name option, if we haven't already got it.
	if _, ok := findOption(opts, gapicNameOption); !ok {
		opts = append(opts, fmt.Sprintf("%s=%s", gapicNameOption, deriveGAPICName(api.Path)))
	}
	// Add the library name as warehouse-package-name option, if we haven't already got it.
	if _, ok := findOption(opts, warehousePackageNameOption); !ok {
		opts = append(opts, fmt.Sprintf("%s=%s", warehousePackageNameOption, library.Name))
	}

	// Add gapic-version from library version
	if library.Version != "" {
		opts = append(opts, fmt.Sprintf("gapic-version=%s", library.Version))
	}

	// Add gRPC service config (retry/timeout settings)
	grpcConfigPath, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, err
	}
	if grpcConfigPath != "" {
		opts = append(opts, fmt.Sprintf("retry-config=%s", grpcConfigPath))
	}

	if apiMetadata != nil && apiMetadata.ServiceConfig != "" {
		opts = append(opts, fmt.Sprintf("service-yaml=%s", apiMetadata.ServiceConfig))
	}

	return []string{
		fmt.Sprintf("--python_gapic_out=%s", stagingDir),
		fmt.Sprintf("--python_gapic_opt=%s", strings.Join(opts, ",")),
	}, nil
}

func isProtoOnly(api *config.API, library *config.Library) bool {
	return library.Python != nil && slices.Contains(library.Python.ProtoOnlyAPIs, api.Path)
}

// getStagingChildDirectory determines where within owl-bot-staging/{library-name} the
// generated code the given API path should be staged. This is not quite equivalent
// to _get_staging_child_directory in the Python container, as for proto-only directories
// we don't want the apiPath suffix.
func getStagingChildDirectory(apiPath string, isProtoOnly bool) string {
	versionCandidate := filepath.Base(apiPath)
	if strings.HasPrefix(versionCandidate, "v") || isProtoOnly {
		return versionCandidate
	} else {
		return versionCandidate + "-py"
	}
}

// runPostProcessor runs the synthtool post processor on the output directory.
func runPostProcessor(ctx context.Context, repoRoot, outDir string) error {
	// The post-processor expects the string replacement scripts to be in the
	// output directory, so we need to copy them there.
	// TODO(https://github.com/googleapis/librarian/issues/3008): reimplement
	// the string replacements in Go, and at that point stop copying the files.
	scriptsOutput := filepath.Join(outDir, "scripts", "client-post-processing")
	scriptsInput := filepath.Join(repoRoot, ".librarian", "generator-input", "client-post-processing")
	if err := os.CopyFS(scriptsOutput, os.DirFS(scriptsInput)); err != nil {
		return err
	}

	pythonCode := fmt.Sprintf(`
from synthtool.languages import python_mono_repo
python_mono_repo.owlbot_main(%q)
`, outDir)
	if err := command.RunInDir(ctx, repoRoot, "python3", "-c", pythonCode); err != nil {
		return fmt.Errorf("failed to run post-processor: %w", err)
	}

	// synthtool runs formatting, then applies string replacements. This leaves
	// some files unformatted. We format again just to get everything straight.
	// (Changing synthtool's ordering would require changes in the replacements
	// as well... we can do all of that after migration, when we remove
	// synthtool entirely - see
	// https://github.com/googleapis/librarian/issues/3008)
	if err := command.RunInDir(ctx, outDir, "nox", "-s", "format", "--no-venv", "--no-install"); err != nil {
		return fmt.Errorf("failed to format code after post-processing: %w", err)
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
	if _, err := os.Lstat(sourcePath); errors.Is(err, fs.ErrNotExist) {
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
// directly in place and remove the owl-bot-staging directory entirely.
// TODO(https://github.com/googleapis/librarian/issues/3008): perform string
// replacements in Go code, so we don't need to copy files.
func cleanUpFilesAfterPostProcessing(repoRoot, outdir string) error {
	// Remove owl-bot-staging from the repo root.
	if err := os.RemoveAll(filepath.Join(repoRoot, "owl-bot-staging")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove owl-bot-staging: %w", err)
	}
	// Remove the post-processing scripts. This will leave the "scripts"
	// directory, but that's okay if it's empty - git ignores empty directories.
	// If it's *not* empty, then there must have been files there before, which
	// we'd want to keep anyway.
	if err := os.RemoveAll(filepath.Join(outdir, "scripts", "client-post-processing")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove client-post-processing directory: %w", err)
	}
	return nil
}

// DefaultOutput derives an output path from a library name and a default
// output directory. Currently, this just assumes each library is a directory
// directly underneath the default output directory.
func DefaultOutput(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}

// DefaultLibraryName derives a library name from an API path by stripping
// the version suffix and replacing "/" with "-".
// For example: "google/cloud/secretmanager/v1" ->
// "google-cloud-secretmanager".
func DefaultLibraryName(api string) string {
	path := api
	if serviceconfig.ExtractVersion(api) != "" {
		// Strip version suffix (v1, v1beta2, v2alpha, etc.).
		path = filepath.Dir(api)
	}
	return strings.ReplaceAll(path, "/", "-")
}

// isPreview determines if the given output directory contains the canonical
// preview subdirectory segments as a means of identifying the library as a
// preview library.
func isPreview(output string) bool {
	return strings.Contains(output, "preview-packages")
}

// deriveGAPICNamespace derives the value to pass as python-gapic-namespace when
// it's not specified explicitly. This is the first two components of the API
// path (excluding any trailing version), dot-separated.
func deriveGAPICNamespace(path string) string {
	version := serviceconfig.ExtractVersion(path)
	if version != "" {
		path = strings.TrimSuffix(path, "/"+version)
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return path
	}
	return parts[0] + "." + parts[1]
}

// deriveGAPICName derives the value to pass as python-gapic-name when it's not
// specified explicitly. This is the path, without the leading namespace (after
// replacing dots with slashes), and without any version suffix, and then
// replacing slashes with underscores. Example:
// a path of google/cloud/foo/bar/v1 would have a GAPIC name of "foo_bar".
func deriveGAPICName(path string) string {
	version := serviceconfig.ExtractVersion(path)
	if version != "" {
		path = strings.TrimSuffix(path, version)
	}
	derivedNamespace := deriveGAPICNamespace(path)
	path = strings.TrimPrefix(path, strings.ReplaceAll(derivedNamespace, ".", "/"))
	path = strings.Trim(path, "/")
	return strings.ReplaceAll(path, "/", "_")
}

// findOption finds the value of a named option within a list of name=value
// strings. If the option isn't found, an empty string is returned. The second
// value indicates whether the option was found or not.
func findOption(options []string, name string) (string, bool) {
	prefix := name + "="
	for _, candidate := range options {
		if strings.HasPrefix(candidate, prefix) {
			return strings.TrimPrefix(candidate, prefix), true
		}
	}
	return "", false
}
