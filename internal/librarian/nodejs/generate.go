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

// Package nodejs provides Node.js-specific functionality for librarian.
package nodejs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// Generate generates all given libraries in sequence.
func Generate(ctx context.Context, libraries []*config.Library, googleapisDir string) error {
	for _, library := range libraries {
		if err := generateLibrary(ctx, library, googleapisDir); err != nil {
			return err
		}
	}
	return nil
}

func generateLibrary(ctx context.Context, library *config.Library, googleapisDir string) error {
	if len(library.APIs) == 0 {
		return nil
	}

	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}

	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	repoRoot := filepath.Dir(filepath.Dir(outdir))

	for _, api := range library.APIs {
		if err := generateAPI(ctx, api, library, googleapisDir, repoRoot); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}

	if err := runPostProcessor(ctx, repoRoot, outdir); err != nil {
		return fmt.Errorf("failed to run post processor: %w", err)
	}
	return nil
}

func generateAPI(ctx context.Context, api *config.API, library *config.Library, googleapisDir, repoRoot string) error {
	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return err
	}

	googleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		return fmt.Errorf("failed to resolve googleapis directory path: %w", err)
	}

	apiDir := filepath.Join(googleapisDir, api.Path)
	protos, err := filepath.Glob(apiDir + "/*.proto")
	if err != nil {
		return fmt.Errorf("failed to find protos: %w", err)
	}
	if len(protos) == 0 {
		return fmt.Errorf("no protos found in api %q", api.Path)
	}

	args, err := buildGeneratorArgs(api, library, googleapisDir, stagingDir)
	if err != nil {
		return err
	}
	cmdArgs := append(args[1:], protos...)
	return command.Run(ctx, args[0], cmdArgs...)
}

// buildGeneratorArgs constructs the gapic-generator-typescript arguments,
// excluding proto files.
func buildGeneratorArgs(api *config.API, library *config.Library, googleapisDir, stagingDir string) ([]string, error) {
	args := []string{
		"gapic-generator-typescript",
		"-I", googleapisDir,
		"--output_dir", stagingDir,
	}

	grpcConfigPath, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, err
	}
	if grpcConfigPath != "" {
		args = append(args, "--grpc-service-config", filepath.Join(googleapisDir, grpcConfigPath))
	}

	apiMetadata, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageNodejs)
	if err != nil {
		return nil, err
	}
	if apiMetadata != nil && apiMetadata.ServiceConfig != "" {
		args = append(args, "--service-yaml", filepath.Join(googleapisDir, apiMetadata.ServiceConfig))
	}

	args = append(args, "--package-name", DerivePackageName(library))
	args = append(args, "--metadata")

	// Only pass --transport for non-default values (default is grpc+rest).
	transport := library.Transport
	if transport == "" {
		transport = "grpc+rest"
	}
	if transport != "grpc+rest" {
		args = append(args, "--transport", transport)
	}

	if library.Nodejs != nil {
		if library.Nodejs.BundleConfig != "" {
			args = append(args, "--bundle-config", filepath.Join(googleapisDir, library.Nodejs.BundleConfig))
		}
		for _, param := range library.Nodejs.ExtraProtocParameters {
			args = append(args, "--"+param)
		}
		if library.Nodejs.HandwrittenLayer {
			args = append(args, "--handwritten-layer")
		}
		if library.Nodejs.MainService != "" {
			args = append(args, "--main-service", library.Nodejs.MainService)
		}
		if library.Nodejs.Mixins != "" {
			args = append(args, "--mixins", library.Nodejs.Mixins)
		}
	}

	return args, nil
}

// runPostProcessor runs synthtool to move files from owl-bot-staging/ to the
// output directory.
func runPostProcessor(ctx context.Context, repoRoot, outDir string) error {
	pythonCode := fmt.Sprintf(`
from synthtool.languages import node_mono_repo
node_mono_repo.owlbot_main(%q)
`, outDir)
	if err := command.RunInDir(ctx, repoRoot, "python3", "-c", pythonCode); err != nil {
		return err
	}

	if err := os.RemoveAll(filepath.Join(repoRoot, "owl-bot-staging")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove owl-bot-staging: %w", err)
	}
	return nil
}

// Format runs gts (npm run fix) on the library directory.
func Format(ctx context.Context, library *config.Library) error {
	return command.RunInDir(ctx, library.Output, "npm", "run", "fix")
}

// DerivePackageName returns the npm package name for a library. It uses
// nodejs.package_name if set, otherwise derives it by splitting the library
// name on the second dash (e.g. "google-cloud-batch" → "@google-cloud/batch").
func DerivePackageName(library *config.Library) string {
	if library.Nodejs != nil && library.Nodejs.PackageName != "" {
		return library.Nodejs.PackageName
	}
	return derivePackageNameFromLibraryName(library.Name)
}

func derivePackageNameFromLibraryName(name string) string {
	firstDash := strings.Index(name, "-")
	if firstDash < 0 {
		return name
	}
	secondDash := strings.Index(name[firstDash+1:], "-")
	if secondDash < 0 {
		return name
	}
	secondDash += firstDash + 1
	scope := name[:secondDash]
	pkg := name[secondDash+1:]
	return fmt.Sprintf("@%s/%s", scope, pkg)
}

// DefaultOutput returns the output path for a library.
func DefaultOutput(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}
