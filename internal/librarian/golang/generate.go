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

// Package golang provides Go specific functionality for librarian.
package golang

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// Generate generates a Go client library.
func Generate(ctx context.Context, library *config.Library, googleapisDir, language, repo, defaultVersion string) error {
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

	// Generate each channel separately.
	for _, channel := range library.Channels {
		if err := generateChannel(ctx, channel, library, googleapisDir, outdir); err != nil {
			return fmt.Errorf("failed to generate channel %s: %w", channel.Path, err)
		}
	}

	// Generate .repo-metadata.json from the service config in the first
	// channel.
	sc, err := serviceconfig.Find(googleapisDir, library.Channels[0].Path)
	if err != nil {
		return fmt.Errorf("failed to lookup service config: %w", err)
	}
	absoluteServiceConfig := filepath.Join(googleapisDir, sc.ServiceConfig)
	if err := repometadata.GenerateRepoMetadata(library, language, repo, absoluteServiceConfig, defaultVersion, outdir); err != nil {
		return fmt.Errorf("failed to generate .repo-metadata.json: %w", err)
	}
	return nil
}

// generateChannel generates part of a library for a single channel.
func generateChannel(ctx context.Context, channel *config.Channel, library *config.Library, googleapisDir, outdir string) error {
	// Find proto files in the channel directory.
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

	protocOptions, err := createProtocOptions(channel, library, googleapisDir, outdir)
	if err != nil {
		return err
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

// createProtocOptions creates the protoc command-line options for Go generation.
func createProtocOptions(ch *config.Channel, library *config.Library, googleapisDir, outdir string) ([]string, error) {
	var opts []string

	// Add transport option
	if library.Transport != "" {
		opts = append(opts, fmt.Sprintf("transport=%s", library.Transport))
	}

	// Add metadata option
	opts = append(opts, "metadata")

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
		opts = append(opts, fmt.Sprintf("grpc-service-config=%s", grpcConfigPath))
	}

	// Add service config YAML
	channel, err := serviceconfig.Find(googleapisDir, ch.Path)
	if err != nil {
		return nil, err
	}
	if channel != nil && channel.ServiceConfig != "" {
		opts = append(opts, fmt.Sprintf("api-service-config=%s", channel.ServiceConfig))
	}

	// Apply Go-specific options from library.Go.GoAPIs if available
	if library.Go != nil && len(library.Go.GoAPIs) > 0 {
		for _, goAPI := range library.Go.GoAPIs {
			if goAPI.Path == ch.Path {
				if goAPI.DisableGAPIC {
					opts = append(opts, "disable-gapic")
				}
				if goAPI.ProtoPackage != "" {
					opts = append(opts, fmt.Sprintf("proto-package=%s", goAPI.ProtoPackage))
				}
				if len(goAPI.NestedProtos) > 0 {
					opts = append(opts, fmt.Sprintf("nested-protos=%s", strings.Join(goAPI.NestedProtos, ",")))
				}
				break
			}
		}
	}

	// Build protoc plugin options
	var result []string

	// Add protobuf generation
	result = append(result, fmt.Sprintf("--go_out=%s", outdir))
	result = append(result, "--go_opt=paths=source_relative")

	// Add gRPC generation
	result = append(result, fmt.Sprintf("--go-grpc_out=%s", outdir))
	result = append(result, "--go-grpc_opt=paths=source_relative")

	// Add GAPIC generation
	result = append(result, fmt.Sprintf("--go_gapic_out=%s", outdir))
	result = append(result, fmt.Sprintf("--go_gapic_opt=%s", strings.Join(opts, ",")))

	return result, nil
}

// Format formats a generated Go library using goimports.
func Format(ctx context.Context, library *config.Library) error {
	if err := command.Run(ctx, "go", "tool", "goimports", "-w", library.Output); err != nil {
		return err
	}
	return command.Run(ctx, "go", "mod", "tidy")
}
