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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/tool/protoc"
)

var errNoAPIs = errors.New("no apis configured for library")

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
		return fmt.Errorf("failed to get absolute path of output directory: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	tempDir, err := os.MkdirTemp(outDir, "librarian-ruby-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
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

	// TODO(https://github.com/googleapis/librarian/issues/6885): Implement main client gem wrapper generation
	// for libraries configured with `ruby.wrapper_of`.
	for _, api := range library.APIs {
		if err := generateAPI(ctx, api, library.Name, pc, googleapisDir, tempDir); err != nil {
			return fmt.Errorf("api %q: %w", api.Path, err)
		}
	}
	keepSet := buildKeepSet(library.Keep)
	keepFunc := func(rel string) bool {
		return isKept(rel, keepSet)
	}
	if err := filesystem.MoveAndMergeWithKeep(tempDir, outDir, outDir, keepFunc); err != nil {
		return fmt.Errorf("failed to move generated files: %w", err)
	}
	return nil
}

func generateAPI(ctx context.Context, api *config.API, gemName string, pc *config.Protoc, googleapisDir, stagingDir string) error {
	protoFiles, err := collectProtoFiles(googleapisDir, api.Path)
	if err != nil {
		return err
	}
	gapicOpts, err := buildGAPICOpts(api, gemName, googleapisDir)
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
		return fmt.Errorf("failed to create lib staging directory: %w", err)
	}
	grpcPluginPath := filepath.Join(installDir, "bin", "grpc_tools_ruby_protoc_plugin")
	args := []string{
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
		"--ruby_out=" + libStagingDir,
		"--grpc_out=" + libStagingDir,
		"--plugin=protoc-gen-grpc=" + grpcPluginPath,
		"--ruby_cloud_out=" + stagingDir,
	}
	if len(gapicOpts) > 0 {
		args = append(args, "--ruby_cloud_opt="+strings.Join(gapicOpts, ","))
	}
	args = append(args, protoFiles...)
	env, err := toolsEnv()
	if err != nil {
		return err
	}
	return protoc.RunOrSystem(ctx, env, pc, args...)
}

func buildGAPICOpts(api *config.API, gemName, googleapisDir string) ([]string, error) {
	sc, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageRuby)
	if err != nil {
		return nil, err
	}
	gc, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, err
	}
	var opts []string
	if gemName != "" {
		opts = append(opts, "ruby-cloud-gem-name="+gemName)
	}
	if sc != nil && sc.ServiceConfig != "" {
		opts = append(opts, "service-yaml="+filepath.Join(googleapisDir, sc.ServiceConfig))
	}
	if gc != "" {
		opts = append(opts, "grpc-service-config="+filepath.Join(googleapisDir, gc))
	}
	if trans := transport(sc); trans != "" {
		opts = append(opts, fmt.Sprintf("transport=%s", trans))
	}
	if sc != nil && sc.HasRESTNumericEnums(config.LanguageRuby) {
		opts = append(opts, "ruby-cloud-rest-numeric-enums=true")
	}
	if api.Ruby != nil && api.Ruby.RubyCloudOpts != nil {
		if api.Ruby.RubyCloudOpts.EnvPrefix != "" {
			opts = append(opts, "ruby-cloud-env-prefix="+api.Ruby.RubyCloudOpts.EnvPrefix)
		}
		if api.Ruby.RubyCloudOpts.ExtraDependencies != "" {
			opts = append(opts, "ruby-cloud-extra-dependencies="+api.Ruby.RubyCloudOpts.ExtraDependencies)
		}
	}
	return opts, nil
}

func transport(sc *serviceconfig.API) serviceconfig.Transport {
	if sc != nil {
		return sc.Transport(config.LanguageRuby)
	}
	return serviceconfig.GRPCRest
}

func collectProtoFiles(googleapisDir, apiPath string) ([]string, error) {
	apiDir := filepath.Join(googleapisDir, apiPath)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read API directory %s: %w", apiDir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".proto" {
			files = append(files, filepath.Join(apiDir, entry.Name()))
		}
	}
	sort.Strings(files)
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
