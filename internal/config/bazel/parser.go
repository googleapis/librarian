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

// Package bazel parses BUILD.bazel files to extract GAPIC configuration.
package bazel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"
)

// Config holds configuration extracted from googleapis BUILD.bazel files.
type Config struct {
	// DIREGAPIC indicates whether DIREGAPIC (Discovery REST GAPICs) is used.
	DIREGAPIC bool

	// GAPICImportPath is the import path for the GAPIC library.
	GAPICImportPath string

	// GRPCServiceConfig is the gRPC service config JSON file.
	GRPCServiceConfig string

	// HasGAPIC indicates whether the GAPIC generator should be run.
	HasGAPIC bool

	// HasGoGRPC indicates whether go_grpc_library is used.
	//
	// TODO(https://github.com/googleapis/librarian/issues/1021): Remove this field once
	// the googleapis migration from go_proto_library to go_grpc_library is complete.
	HasGoGRPC bool

	// HasLegacyGRPC indicates whether go_proto_library uses the legacy gRPC compiler.
	HasLegacyGRPC bool

	// Metadata indicates whether gapic_metadata.json should be generated.
	Metadata bool

	// ReleaseLevel is the API maturity level (e.g., "beta", "ga").
	ReleaseLevel string

	// RESTNumericEnums indicates whether numeric enums are supported in REST clients.
	RESTNumericEnums bool

	// ServiceYAML is the service configuration file.
	ServiceYAML string

	// Transport specifies the transport protocol (e.g., "grpc", "rest", "grpc+rest").
	Transport string
}

// Parse reads a BUILD.bazel file and extracts configuration from Bazel rules.
// The path can be a directory containing BUILD.bazel or a direct file path.
func Parse(path string) (*Config, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}
	if info.IsDir() {
		path = filepath.Join(path, "BUILD.bazel")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read BUILD.bazel file %s: %w", path, err)
	}
	f, err := build.ParseBuild(path, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BUILD.bazel file %s: %w", path, err)
	}

	cfg := &Config{}
	if rules := f.Rules("go_gapic_library"); len(rules) > 0 {
		rule := rules[0]
		cfg = &Config{
			HasGAPIC:          true,
			GRPCServiceConfig: rule.AttrString("grpc_service_config"),
			GAPICImportPath:   rule.AttrString("importpath"),
			ReleaseLevel:      rule.AttrString("release_level"),
			ServiceYAML:       strings.TrimPrefix(rule.AttrString("service_yaml"), ":"),
			Transport:         rule.AttrString("transport"),
			Metadata:          rule.AttrLiteral("metadata") == "True",
			RESTNumericEnums:  rule.AttrLiteral("rest_numeric_enums") == "True",
			DIREGAPIC:         rule.AttrLiteral("diregapic") == "True",
		}
	}
	if len(f.Rules("go_grpc_library")) > 0 {
		cfg.HasGoGRPC = true
	}
	if rules := f.Rules("go_proto_library"); len(rules) > 0 {
		if cfg.HasGoGRPC {
			return nil, fmt.Errorf("BUILD.bazel cannot have both go_grpc_library and go_proto_library: %s", path)
		}
		compilers := rules[0].AttrStrings("compilers")
		for _, compiler := range compilers {
			if strings.Contains(compiler, "@io_bazel_rules_go//proto:go_grpc") {
				cfg.HasLegacyGRPC = true
				break
			}
		}
	}
	if cfg.HasGAPIC {
		if cfg.GAPICImportPath == "" {
			return nil, fmt.Errorf("GAPICImportPath not set: %s", path)
		}
		if cfg.ServiceYAML == "" {
			return nil, fmt.Errorf("ServiceYAML not set: %s", path)
		}
	}
	return cfg, nil
}
