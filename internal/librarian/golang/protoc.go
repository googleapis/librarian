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

// Package golang provides functionality for generating Go client libraries.
package golang

import (
	"fmt"
	"os"
	"path/filepath"
)

// protocConfig holds the configuration needed to build a protoc command.
//
// The GAPIC options correspond to protoc-gen-go_gapic flags documented at:
// https://github.com/googleapis/gapic-generator-go.
type protocConfig struct {
	// APIPath is the API path within GoogleapisDir (e.g., "google/cloud/secretmanager/v1").
	APIPath string

	// DIREGAPIC enables DIREGAPIC (Discovery REST GAPICs) generation.
	DIREGAPIC bool

	// GAPICImportPath is the Go import path for the generated GAPIC library (go-gapic-package).
	GAPICImportPath string

	// GoogleapisDir is the root directory containing proto files (a googleapis checkout).
	GoogleapisDir string

	// GRPCServiceConfig is the gRPC service config JSON file (grpc-service-config).
	GRPCServiceConfig string

	// HasGAPIC indicates whether to run the GAPIC generator.
	HasGAPIC bool

	// HasGoGRPC indicates whether go_grpc_library is used.
	HasGoGRPC bool

	// Metadata enables generation of gapic_metadata.json (metadata).
	Metadata bool

	// NestedProtos lists additional proto files in subdirectories to include.
	NestedProtos []string

	// OutputDir is where generated files are written.
	OutputDir string

	// RESTNumericEnums enables numeric enum encoding in REST clients (rest-numeric-enums).
	RESTNumericEnums bool

	// ReleaseLevel is the API maturity level: "alpha", "beta", or "" for GA (release-level).
	ReleaseLevel string

	// ServiceYAML is the service configuration file (api-service-config).
	ServiceYAML string

	// Transport specifies the transport protocol: "grpc", "rest", or "grpc+rest" (transport).
	Transport string
}

// buildProtocCommand constructs the protoc command arguments for a Go API.
func buildProtocCommand(cfg *protocConfig) ([]string, error) {
	protoFiles, err := collectProtoFiles(cfg.GoogleapisDir, cfg.APIPath, cfg.NestedProtos)
	if err != nil {
		return nil, err
	}

	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"--go_out=" + cfg.OutputDir,
		"-I=" + cfg.GoogleapisDir,
	}

	if cfg.HasGoGRPC {
		args = append(args,
			"--go-grpc_out="+cfg.OutputDir,
			"--go-grpc_opt=require_unimplemented_servers=false",
		)
	}

	if cfg.HasGAPIC {
		args = append(args, "--go_gapic_out="+cfg.OutputDir)
		for _, opt := range buildGAPICOpts(cfg) {
			args = append(args, "--go_gapic_opt="+opt)
		}
	}

	args = append(args, protoFiles...)
	return args, nil
}

// collectProtoFiles gathers proto files from the API directory.
func collectProtoFiles(googleapisDir, apiPath string, nestedProtos []string) ([]string, error) {
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

	for _, nested := range nestedProtos {
		files = append(files, filepath.Join(apiDir, nested))
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .proto files found in %s", apiDir)
	}
	return files, nil
}

// buildGAPICOpts constructs the GAPIC generator options.
func buildGAPICOpts(cfg *protocConfig) []string {
	opts := []string{"go-gapic-package=" + cfg.GAPICImportPath}
	if cfg.ServiceYAML != "" {
		opts = append(opts, "api-service-config="+cfg.ServiceYAML)
	}
	if cfg.GRPCServiceConfig != "" {
		opts = append(opts, "grpc-service-config="+cfg.GRPCServiceConfig)
	}
	if cfg.Transport != "" {
		opts = append(opts, "transport="+cfg.Transport)
	}
	if cfg.ReleaseLevel != "" {
		opts = append(opts, "release-level="+cfg.ReleaseLevel)
	}
	if cfg.Metadata {
		opts = append(opts, "metadata")
	}
	if cfg.DIREGAPIC {
		opts = append(opts, "diregapic")
	}
	if cfg.RESTNumericEnums {
		opts = append(opts, "rest-numeric-enums")
	}
	return opts
}
