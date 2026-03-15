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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

// Generate generates a Node.js client library.
func Generate(ctx context.Context, library *config.Library, googleapisDir string) error {
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
	if err := runPostProcessor(ctx, library, repoRoot, outdir); err != nil {
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
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		return nil, fmt.Errorf("failed to find protoc: %w", err)
	}

	args := []string{
		"gapic-generator-typescript",
		"--protoc=" + protocPath,
		"--common-proto-path=" + googleapisDir,
		"-I", googleapisDir,
		"--output-dir", stagingDir,
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
	transport := serviceconfig.GRPCRest
	if apiMetadata != nil {
		transport = apiMetadata.Transport(config.LanguageNodejs)
	}
	if transport != serviceconfig.GRPCRest {
		args = append(args, "--transport", string(transport))
	}

	if library.Nodejs != nil {
		if library.Nodejs.BundleConfig != "" {
			args = append(args, "--bundle-config", filepath.Join(googleapisDir, library.Nodejs.BundleConfig))
		}
		for _, param := range library.Nodejs.ExtraProtocParameters {
			if param == "metadata" {
				continue
			}
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

// runPostProcessor combines versioned API outputs from owl-bot-staging/ into
// the output directory using gapic-node-processing, then compiles protos.
func runPostProcessor(ctx context.Context, library *config.Library, repoRoot, outDir string) error {
	owlbotPath := filepath.Join(outDir, "owlbot.py")
	if _, err := os.Stat(owlbotPath); err == nil {
		// Old way: use synthtool
		if err := command.RunInDir(ctx, outDir, "python3", "owlbot.py"); err != nil {
			return fmt.Errorf("owlbot.py failed: %w", err)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check for owlbot.py: %w", err)
	}

	// Template generation and exclusions are handled at the generator level.
	// Synthtool is only used for post-processing handled by standalone scripts
	// like librarian.js and owlbot.py. (Note: librarian.js is unrelated to the
	// Librarian CLI tool).

	// combine-library wipes the destination directory before writing generated
	// files (src/, protos/). Save the non-generated files it would delete, then
	// restore them afterward.
	preserveFiles := []string{"librarian.js", ".readme-partials.yaml", "README.md"}
	backupDir, err := os.MkdirTemp("", "librarian-backup-*")
	if err != nil {
		return fmt.Errorf("failed to create backup dir: %w", err)
	}
	defer os.RemoveAll(backupDir)
	for _, name := range preserveFiles {
		src := filepath.Join(outDir, name)
		if _, err := os.Stat(src); err != nil {
			continue // file doesn't exist, nothing to save
		}
		if err := os.Rename(src, filepath.Join(backupDir, name)); err != nil {
			return fmt.Errorf("failed to save %s: %w", name, err)
		}
	}

	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name)
	if err := command.Run(ctx, "gapic-node-processing",
		"combine-library",
		"--source-path", stagingDir,
		"--destination-path", outDir,
	); err != nil {
		return fmt.Errorf("combine-library: %w", err)
	}

	// Restore non-generated files.
	for _, name := range preserveFiles {
		src := filepath.Join(backupDir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := os.Rename(src, filepath.Join(outDir, name)); err != nil {
			return fmt.Errorf("failed to restore %s: %w", name, err)
		}
	}

	if err := command.RunInDir(ctx, outDir, "compileProtos", "src"); err != nil {
		return fmt.Errorf("compileProtos: %w", err)
	}

	// librarian.js is a custom script some libraries use for post-processing.
	// It has nothing to do with the Librarian CLI tool.
	librarianScript := filepath.Join(outDir, "librarian.js")
	if _, err := os.Stat(librarianScript); err == nil {
		if err := command.RunInDir(ctx, outDir, "node", "librarian.js"); err != nil {
			return fmt.Errorf("librarian.js failed: %w", err)
		}
	}

	readmePartials := filepath.Join(outDir, ".readme-partials.yaml")
	if _, err := os.Stat(readmePartials); err == nil {
		type partials struct {
			Introduction string `yaml:"introduction"`
			Body         string `yaml:"body"`
		}
		p, err := yaml.Read[partials](readmePartials)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", readmePartials, err)
		}
		for name, replacement := range map[string]string{
			"introduction": p.Introduction,
			"body":         p.Body,
		} {
			if replacement == "" {
				continue
			}
			if err := command.RunInDir(ctx, outDir, "npx", "gapic-node-processing", "generate-readme",
				fmt.Sprintf("--source-path=%s", outDir),
				fmt.Sprintf("--string-to-replace=[//]: # \"partials.%s\"", name),
				fmt.Sprintf("--replacement-string=%s", replacement),
			); err != nil {
				return fmt.Errorf("generate-readme (%s) failed: %w", name, err)
			}
		}
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
