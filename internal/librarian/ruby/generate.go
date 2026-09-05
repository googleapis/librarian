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

// Package ruby provides Ruby specific functionality for librarian.
package ruby

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
	"github.com/googleapis/librarian/internal/proto"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/tool/protoc"
)

const commonResourcesProto = "google/cloud/common_resources.proto"

var (
	errNoAPIs        = errors.New("no apis configured for library")
	errInvalidPath   = errors.New("invalid path: must be a relative path within the directory")
	errEmptyToysTask = errors.New("toys task must not be empty")
)

// DefaultOutput derives an output path from a library name and a default
// output path.
func DefaultOutput(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}

// Generate generates a Ruby client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, srcs *sources.Sources) (err error) {
	if len(library.APIs) == 0 {
		return errNoAPIs
	}
	outDir, err := filepath.Abs(library.Output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(outDir, "librarian-ruby-")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			err = errors.Join(err, removeErr)
		}
	}()
	googleapisDir := srcs.Googleapis
	var pc *config.Protoc
	if cfg != nil && cfg.Tools != nil {
		pc = cfg.Tools.Protoc
	}

	if isMultiWrapper(library) {
		if err := generateMultiWrapper(ctx, library, pc, googleapisDir, tempDir); err != nil {
			return err
		}
	} else {
		for _, api := range library.APIs {
			if err := generateAPI(ctx, api, library, pc, googleapisDir, tempDir); err != nil {
				return err
			}
		}
	}
	if err := deleteLibraryAfterGeneration(library, tempDir, outDir); err != nil {
		return err
	}
	keepSet := buildKeepSet(library.Name, library.Keep)
	keepFunc := func(rel string) bool {
		return isKept(rel, keepSet)
	}
	if err := filesystem.MoveAndMergeWithKeep(tempDir, outDir, outDir, keepFunc); err != nil {
		return err
	}
	return runToysTasks(ctx, library, outDir)
}

func isWrapperLibrary(library *config.Library) bool {
	if library.Ruby != nil && len(library.Ruby.WrapperOf) > 0 {
		return true
	}
	for _, api := range library.APIs {
		if api.Ruby != nil && api.Ruby.RubyCloudOpts != nil && api.Ruby.RubyCloudOpts.WrapperOf != "" {
			return true
		}
	}
	return false
}

func isMultiWrapper(library *config.Library) bool {
	if !isWrapperLibrary(library) {
		return false
	}
	return len(library.APIs) > 1
}

func generateMultiWrapper(ctx context.Context, library *config.Library, pc *config.Protoc, googleapisDir, stagingDir string) error {
	var sourceGems []string
	for _, api := range library.APIs {
		gemName := apiGemName(library, api)
		sourceGems = append(sourceGems, gemName)
		apiStagingDir := filepath.Join(stagingDir, gemName)
		if err := os.MkdirAll(apiStagingDir, 0o755); err != nil {
			return err
		}
		if err := generateAPI(ctx, api, library, pc, googleapisDir, apiStagingDir); err != nil {
			return err
		}
	}
	return prepareMultiWrapper(stagingDir, sourceGems, library.TitleOverride, library.Name)
}

func generateAPI(ctx context.Context, api *config.API, library *config.Library, pc *config.Protoc, googleapisDir, stagingDir string) error {
	additionalProtos := []string{commonResourcesProto}
	if api.Ruby != nil {
		additionalProtos = append(additionalProtos, api.Ruby.AdditionalProtos...)
	}
	protoFiles, err := collectProtoFiles(googleapisDir, api.Path, additionalProtos)
	if err != nil {
		return err
	}
	serviceConfig, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageRuby)
	if err != nil {
		return err
	}
	gapicOpts, err := buildGAPICOpts(api, library, googleapisDir, serviceConfig)
	if err != nil {
		return err
	}
	installDir, err := InstallDir()
	if err != nil {
		return err
	}
	// Output --ruby_out and --grpc_out into lib/ so _pb.rb files land under lib/google/...
	// matching Bazel's ruby_gapic_assembly_pkg_impl:
	// https://github.com/googleapis/gapic-generator-ruby/blob/8fed6b7c1/rules_ruby_gapic/ruby_gapic_pkg.bzl#L39-L41
	libStagingDir := filepath.Join(stagingDir, "lib")
	if err := os.MkdirAll(libStagingDir, 0o755); err != nil {
		return err
	}
	// A main client is a wrapper of a versioned client
	isWrapper := isWrapperLibrary(library)
	args := buildProtocArgs(googleapisDir, stagingDir, libStagingDir, installDir, isWrapper, serviceConfig, gapicOpts, protoFiles)
	env, err := toolsEnv()
	if err != nil {
		return err
	}
	if err := protoc.RunOrSystem(ctx, env, pc, args...); err != nil {
		return err
	}
	if err := deleteAfterGeneration(api, stagingDir); err != nil {
		return err
	}
	// Remove google/cloud/common_resources_pb.rb from staging after generation.
	// Because librarian passes all protoFiles (including common_resources.proto) to protoc
	// in a single invocation, protoc outputs common_resources_pb.rb into the lib/ directory.
	// We delete it unconditionally so individual client gems do not bundle unused shared
	// protobuf definitions, which would cause class redefinition warnings and collisions.
	commonResourcesPB := filepath.Join(stagingDir, "lib", "google", "cloud", "common_resources_pb.rb")
	if err := os.Remove(commonResourcesPB); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func apiGemName(library *config.Library, api *config.API) string {
	if api.Ruby != nil && api.Ruby.RubyCloudOpts != nil && api.Ruby.RubyCloudOpts.GemName != "" {
		return api.Ruby.RubyCloudOpts.GemName
	}
	return library.Name
}

func buildGAPICOpts(api *config.API, library *config.Library, googleapisDir string, serviceConfig *serviceconfig.API) ([]string, error) {
	gc, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, err
	}
	gemName := apiGemName(library, api)
	opts := []string{
		"ruby-cloud-gem-name=" + gemName,
	}
	if serviceConfig != nil && serviceConfig.ServiceConfig != "" {
		opts = append(opts, "service-yaml="+filepath.Join(googleapisDir, serviceConfig.ServiceConfig))
	}
	if serviceConfig != nil && serviceConfig.Description != "" {
		desc := escapeRubyCloudOptValue(strings.Join(strings.Fields(serviceConfig.Description), " "))
		opts = append(opts, "ruby-cloud-description="+desc, "ruby-cloud-summary="+desc)
	}
	if gc != "" {
		opts = append(opts, "grpc-service-config="+filepath.Join(googleapisDir, gc))
	}
	if trans := transport(serviceConfig); trans != "" {
		transports := strings.ReplaceAll(string(trans), "+", ";")
		opts = append(opts, "ruby-cloud-generate-transports="+transports)
	}
	if serviceConfig != nil && serviceConfig.HasRESTNumericEnums(config.LanguageRuby) {
		opts = append(opts, "ruby-cloud-rest-numeric-enums=true")
	}
	if api.Ruby != nil && api.Ruby.RubyCloudOpts != nil {
		if api.Ruby.RubyCloudOpts.EnvPrefix != "" {
			opts = append(opts, "ruby-cloud-env-prefix="+api.Ruby.RubyCloudOpts.EnvPrefix)
		}
		if api.Ruby.RubyCloudOpts.ExtraDependencies != "" {
			opts = append(opts, "ruby-cloud-extra-dependencies="+api.Ruby.RubyCloudOpts.ExtraDependencies)
		}
		if api.Ruby.RubyCloudOpts.FactoryMethodSuffix != "" {
			opts = append(opts, "ruby-cloud-factory-method-suffix="+api.Ruby.RubyCloudOpts.FactoryMethodSuffix)
		}
		if api.Ruby.RubyCloudOpts.GemNamespace != "" {
			opts = append(opts, "ruby-cloud-gem-namespace="+api.Ruby.RubyCloudOpts.GemNamespace)
		}
		if api.Ruby.RubyCloudOpts.GenericEndpoint {
			opts = append(opts, "ruby-cloud-generic-endpoint=true")
		}
		if api.Ruby.RubyCloudOpts.MigrationVersion != "" {
			opts = append(opts, "ruby-cloud-migration-version="+api.Ruby.RubyCloudOpts.MigrationVersion)
		}
		if api.Ruby.RubyCloudOpts.NamespaceOverride != "" {
			opts = append(opts, "ruby-cloud-namespace-override="+api.Ruby.RubyCloudOpts.NamespaceOverride)
		}
		if api.Ruby.RubyCloudOpts.PathOverride != "" {
			opts = append(opts, "ruby-cloud-path-override="+api.Ruby.RubyCloudOpts.PathOverride)
		}
		if api.Ruby.RubyCloudOpts.RenamedFrom != "" {
			opts = append(opts, "ruby-cloud-renamed-from="+api.Ruby.RubyCloudOpts.RenamedFrom)
		}
		if api.Ruby.RubyCloudOpts.ServiceOverride != "" {
			opts = append(opts, "ruby-cloud-service-override="+api.Ruby.RubyCloudOpts.ServiceOverride)
		}
		if api.Ruby.RubyCloudOpts.Title != "" {
			title := escapeRubyCloudOptValue(api.Ruby.RubyCloudOpts.Title)
			opts = append(opts, "ruby-cloud-title="+title)
		}
		if api.Ruby.RubyCloudOpts.WrapperGemOverride != "" {
			opts = append(opts, "ruby-cloud-wrapper-gem-override="+api.Ruby.RubyCloudOpts.WrapperGemOverride)
		}
		if api.Ruby.RubyCloudOpts.YardStrict != "" {
			opts = append(opts, "ruby-cloud-yard-strict="+api.Ruby.RubyCloudOpts.YardStrict)
		}
	}
	if api.Ruby != nil && api.Ruby.RubyCloudOpts != nil && api.Ruby.RubyCloudOpts.WrapperOf != "" {
		opts = append(opts, "ruby-cloud-wrapper-of="+api.Ruby.RubyCloudOpts.WrapperOf)
	} else if library.Ruby != nil && len(library.Ruby.WrapperOf) > 0 {
		// This controls the dependency range declaration in the gemspec file.
		opts = append(opts, "ruby-cloud-wrapper-of="+strings.Join(library.Ruby.WrapperOf, ";"))
	}
	return opts, nil
}

func buildProtocArgs(googleapisDir, stagingDir, libStagingDir, installDir string, isWrapper bool, serviceConfig *serviceconfig.API, gapicOpts, protoFiles []string) []string {
	args := []string{
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		"--ruby_cloud_out=" + stagingDir,
	}
	if !isWrapper {
		args = append(args, "--ruby_out="+libStagingDir)
		if trans := transport(serviceConfig); trans != serviceconfig.Rest {
			grpcPluginPath := filepath.Join(installDir, "bin", "grpc_tools_ruby_protoc_plugin")
			args = append(args,
				"--grpc_out="+libStagingDir,
				"--plugin=protoc-gen-grpc="+grpcPluginPath,
			)
		}
	}
	if len(gapicOpts) > 0 {
		args = append(args, "--ruby_cloud_opt="+strings.Join(gapicOpts, ","))
	}
	args = append(args, protoFiles...)
	return args
}

func transport(serviceConfig *serviceconfig.API) serviceconfig.Transport {
	if serviceConfig != nil {
		return serviceConfig.Transport(config.LanguageRuby)
	}
	return serviceconfig.GRPCRest
}

func collectProtoFiles(googleapisDir, apiPath string, additionalProtos []string) ([]string, error) {
	apiDir := filepath.Join(googleapisDir, apiPath)
	files, err := proto.Gather(apiDir, apiPath)
	if err != nil {
		return nil, err
	}
	for _, add := range additionalProtos {
		files = append(files, filepath.Join(googleapisDir, add))
	}
	slices.Sort(files)
	files = slices.Compact(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no .proto files found in %s", apiDir)
	}
	return files, nil
}

func toolsEnv() (map[string]string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return nil, err
	}
	binDir := filepath.Join(installDir, "bin")
	path := binDir
	if currentPath := os.Getenv("PATH"); currentPath != "" {
		path = binDir + string(os.PathListSeparator) + currentPath
	}
	env := map[string]string{
		"PATH":     path,
		"GEM_HOME": installDir,
	}
	if gemPath := os.Getenv("GEM_PATH"); gemPath != "" {
		env["GEM_PATH"] = installDir + string(os.PathListSeparator) + gemPath
	} else {
		env["GEM_PATH"] = installDir
	}
	return env, nil
}

// escapeRubyCloudOptValue escapes backslashes and commas in generator option values
// (such as ruby-cloud-description) so that protoc and gapic-generator-ruby parameter parsers
// do not incorrectly split option strings when options are joined with commas.
func escapeRubyCloudOptValue(val string) string {
	// This follows the same escaping convention as Bazel's _escape_config_value in rules_ruby_gapic
	// (rules_ruby_gapic/private/ruby_gapic_library_internal.bzl#L120-L121 in gapic-generator-ruby
	// at commit 8fed6b7c117c7cebaeb5aa5c45eb3f866164eb75) and gapic-generator-ruby's unescaping
	// logic in RequestParamParser (gapic-generator/lib/gapic/schema/request_param_parser.rb#L30-L32).
	val = strings.ReplaceAll(val, "\\", "\\\\")
	return strings.ReplaceAll(val, ",", "\\,")
}

func cleanRelativePath(path string) (string, error) {
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") || clean == "." {
		return "", fmt.Errorf("%w: %q", errInvalidPath, path)
	}
	return clean, nil
}

// deleteAfterGeneration removes files from the staging directory after generation.
func deleteAfterGeneration(api *config.API, stagingDir string) error {
	if api.Ruby == nil {
		return nil
	}
	for _, path := range api.Ruby.DeleteGenerationOutputPaths {
		cleanPath, err := cleanRelativePath(path)
		if err != nil {
			return err
		}
		target := filepath.Join(stagingDir, "lib", cleanPath)
		// Return an error for non-existent paths to keep the configurations
		// up to date.
		if _, err := os.Stat(target); err != nil {
			return err
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}
	return nil
}

// deleteLibraryAfterGeneration removes configured paths from the staging and
// output directories after generation.
func deleteLibraryAfterGeneration(library *config.Library, stagingDir, outDir string) error {
	if library.Ruby == nil {
		return nil
	}
	for _, path := range library.Ruby.DeleteGenerationOutputPaths {
		cleanPath, err := cleanRelativePath(path)
		if err != nil {
			return err
		}
		target := filepath.Join(stagingDir, cleanPath)
		if _, err := os.Stat(target); err != nil {
			return err
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(outDir, cleanPath)); err != nil {
			return err
		}
	}
	return nil
}

func runToysTasks(ctx context.Context, library *config.Library, outDir string) error {
	if library.Ruby == nil || len(library.Ruby.ToysTasks) == 0 {
		return nil
	}
	env, err := toolsEnv()
	if err != nil {
		return err
	}
	for _, task := range library.Ruby.ToysTasks {
		task = strings.TrimSpace(task)
		if task == "" {
			return errEmptyToysTask
		}
		args := strings.Fields(task)
		if err := command.RunInDirWithEnv(ctx, outDir, env, "toys", args...); err != nil {
			return fmt.Errorf("failed to run toys %s: %w", task, err)
		}
	}
	return nil
}
