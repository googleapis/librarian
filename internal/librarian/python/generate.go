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
	"github.com/googleapis/librarian/internal/tool/protoc"
)

const (
	generatorSidekick                   = "sidekick"
	cloudGoogleComDocumentationTemplate = "https://cloud.google.com/python/docs/reference/%s/latest"
	googleapisDevDocumentationTemplate  = "https://googleapis.dev/python/%s/latest"
	transportOption                     = "transport"
	warehousePackageNameOption          = "warehouse-package-name"
)

var (
	// ErrNilLibrary indicates that the library configuration is nil.
	ErrNilLibrary = errors.New("library cannot be nil")
	// ErrEmptyOutput indicates that the library output path is empty.
	ErrEmptyOutput = errors.New("library output cannot be empty")
	// ErrNilAPIConfig indicates that the API configuration is nil.
	ErrNilAPIConfig = errors.New("api configuration cannot be nil")
	// ErrNilSources indicates that the sources configuration is nil.
	ErrNilSources = errors.New("sources cannot be nil")

	errNoDefaultVersion        = errors.New("default version must be specified for every library with generated APIs")
	errExplicitTransportOption = errors.New("transport option is derived from sdk.yaml and must not be specified explicitly")
)

// Generate generates a Python client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, srcs *sources.Sources) error {
	if library == nil {
		return ErrNilLibrary
	}
	if isSidekickGenerator(library) {
		return generateSidekick(ctx, cfg, library, srcs)
	}
	return generateLegacy(ctx, cfg, library, srcs)
}

func generateLegacy(ctx context.Context, cfg *config.Config, library *config.Library, srcs *sources.Sources) error {
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
	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var pc *config.Protoc
	if cfg.Tools != nil {
		pc = cfg.Tools.Protoc
	}

	// Some aspects of generation currently require the repo root. Compute it
	// once here and pass it down.
	repoRoot := filepath.Dir(filepath.Dir(outdir))
	// The "generation root" is a tmp directory created within the package
	// directory, to isolate it from other generation operations which may
	// happen in parallel. It is deleted by cleanUpFilesAfterPostProcessing.
	var generationRoot string
	if len(library.APIs) > 0 {
		generationRoot, err = prepareGenerationRoot(outdir)
		if err != nil {
			return err
		}
	}
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
		if err := generateAPI(ctx, api, library, pc, googleapisDir, generationRoot); err != nil {
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
		if err := runPostProcessor(ctx, repoRoot, outdir, generationRoot); err != nil {
			return fmt.Errorf("failed to run post processor: %w", err)
		}
		if err := cleanUpFilesAfterPostProcessing(generationRoot, outdir); err != nil {
			return fmt.Errorf("failed to cleanup after post processing: %w", err)
		}
	}

	if err := copyReadmeToDocsDir(library, outdir); err != nil {
		return fmt.Errorf("failed to copy README to docs: %w", err)
	}

	if err := createChangelog(library.Name, outdir); err != nil {
		return fmt.Errorf("failed to create changelog: %w", err)
	}
	return nil
}

// prepareGenerationRoot creates a tmp directory underneath the package root.
// This is designed to "look like" the repo root as far as this package is
// concerned, such that packages/{xyz}/tmp/packages/{xyz} is a symlink back to
// packages/{xyz}, and we generate into packages/{xyz}/tmp/owl-bot-staging.
// This allows the post-processor to operate on packages/{xyz}/tmp (which in
// turn allows us to run everything in parallel) without changing the
// post-processor itself.
// See go/sdk:librarian-python-parallel-generation for more details.
func prepareGenerationRoot(packageRoot string) (string, error) {
	packageName := filepath.Base(packageRoot)
	generationRoot := filepath.Join(packageRoot, "tmp")
	if err := os.MkdirAll(filepath.Join(generationRoot, "packages"), 0o755); err != nil {
		return "", err
	}
	if err := os.Symlink("../..", filepath.Join(generationRoot, "packages", packageName)); err != nil {
		return "", err
	}
	return generationRoot, nil
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
	repoMetadata.ClientDocumentation = buildClientDocumentationURI(library.Name, repoMetadata.Name)
	// Even after migration oddities, just a few libraries don't fit into the
	// normal pattern for client documentation URI (e.g. the documentation is
	// in cloud.google.com when it would be expected to be in googleapis.dev).
	if packageOptions.ClientDocumentationOverride != "" {
		repoMetadata.ClientDocumentation = packageOptions.ClientDocumentationOverride
	}
	// TODO(https://github.com/googleapis/librarian/issues/4175): remove these.
	if packageOptions.IssueTrackerOverride != "" {
		repoMetadata.IssueTracker = packageOptions.IssueTrackerOverride
	}
	return repoMetadata, nil
}

// buildClientDocumentationURI builds the URI for the client documentation
// for the library.
func buildClientDocumentationURI(libraryName, repoMetadataName string) string {
	// Work out the right documentation URI based on whether this is a Cloud
	// or non-Cloud API.
	docTemplate := cloudGoogleComDocumentationTemplate
	if !strings.HasPrefix(libraryName, "google-cloud") {
		docTemplate = googleapisDevDocumentationTemplate
	}
	return fmt.Sprintf(docTemplate, repoMetadataName)
}

// generateAPI generates part of a library for a single api.
func generateAPI(ctx context.Context, api *config.API, library *config.Library, pc *config.Protoc, googleapisDir, generationRoot string) error {
	// Note: the Python Librarian container generates to a temporary directory,
	// then the results into owl-bot-staging. We generate straight into
	// owl-bot-staging instead. The post-processor then moves the files into
	// the correct final position in the repository.
	// TODO(https://github.com/googleapis/librarian/issues/3210): generate
	// directly in place.
	protoOnly := isProtoOnly(api, library)
	stagingChildDirectory := getStagingChildDirectory(api.Path, protoOnly)
	stagingDir := filepath.Join(generationRoot, "owl-bot-staging", library.Name, stagingChildDirectory)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
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

	protocCmd, err := protoc.BinaryPathOrSystem(pc)
	if err != nil {
		return fmt.Errorf("failed to find protoc: %w", err)
	}

	cmdArgs := append(protos, protocOptions...)
	if err := command.RunInDir(ctx, googleapisDir, protocCmd, cmdArgs...); err != nil {
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
		if err := os.MkdirAll(dir, 0o755); err != nil {
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
	}
	return versionCandidate + "-py"
}
