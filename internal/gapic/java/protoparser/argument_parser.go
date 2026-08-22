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

// Package protoparser parses protobuf descriptors and GAPIC configurations into generator models.
package protoparser

import (
	"strings"

	"google.golang.org/protobuf/types/pluginpb"
)

// Option keys for GAPIC generator plugin arguments.
const (
	// KeyGrpcServiceConfig is the option key for the gRPC service config JSON path.
	KeyGrpcServiceConfig = "grpc-service-config"
	// KeyGapicConfig is the option key for the GAPIC YAML config path.
	KeyGapicConfig = "gapic-config"
	// KeyApiServiceConfig is the option key for the API service config YAML path.
	KeyApiServiceConfig = "api-service-config"
	// KeyTransport is the option key for the transport type (grpc, rest, grpc+rest).
	KeyTransport = "transport"
	// KeyRepo is the option key for the repository name.
	KeyRepo = "repo"
	// KeyArtifact is the option key for the artifact name.
	KeyArtifact = "artifact"
	// KeyMetadata is the option key to generate gapic_metadata.json.
	KeyMetadata = "metadata"
	// KeyNumericEnum is the option key to enable REST numeric enum representation.
	KeyNumericEnum = "rest-numeric-enums"
	// KeyGenerateVersionJava is the option key to generate Version.java.
	KeyGenerateVersionJava = "generate-version-java"

	jsonFileEnding        = "grpc_service_config.json"
	gapicYamlFileEnding   = "gapic.yaml"
	serviceYamlFileEnding = ".yaml"
)

// PluginArguments contains the parsed command line / protoc plugin parameters.
type PluginArguments struct {
	GrpcServiceConfigPath  string
	GapicYamlConfigPath    string
	ServiceYamlConfigPath  string
	Transport              string
	Repo                   string
	Artifact               string
	HasMetadata            bool
	HasNumericEnum         bool
	HasGenerateVersionJava bool
}

// ParsePluginArguments parses the parameter string from protoc.
func ParsePluginArguments(req *pluginpb.CodeGeneratorRequest) *PluginArguments {
	if req == nil {
		return &PluginArguments{}
	}
	return ParsePluginArgumentsString(req.GetParameter())
}

// ParsePluginArgumentsString parses comma-separated key=value or flag options.
func ParsePluginArgumentsString(param string) *PluginArguments {
	args := &PluginArguments{}
	if param == "" {
		return args
	}

	var parts []string
	var cur strings.Builder
	escaped := false
	for i := 0; i < len(param); i++ {
		ch := param[i]
		if escaped {
			cur.WriteByte(ch)
			escaped = false
		} else if ch == '\\' {
			escaped = true
		} else if ch == ',' {
			parts = append(parts, cur.String())
			cur.Reset()
		} else {
			cur.WriteByte(ch)
		}
	}
	parts = append(parts, cur.String())

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == KeyMetadata {
			args.HasMetadata = true
			continue
		}
		if part == KeyNumericEnum {
			args.HasNumericEnum = true
			continue
		}
		if part == KeyGenerateVersionJava {
			args.HasGenerateVersionJava = true
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		k := strings.TrimSpace(kv[0])
		v := ""
		if len(kv) > 1 {
			v = strings.TrimSpace(kv[1])
		}

		switch k {
		case KeyGrpcServiceConfig:
			if args.GrpcServiceConfigPath == "" {
				args.GrpcServiceConfigPath = v
			}
		case KeyGapicConfig:
			if args.GapicYamlConfigPath == "" {
				args.GapicYamlConfigPath = v
			}
		case KeyApiServiceConfig:
			if args.ServiceYamlConfigPath == "" {
				args.ServiceYamlConfigPath = v
			}
		case KeyTransport:
			if args.Transport == "" {
				args.Transport = v
			}
		case KeyRepo:
			if args.Repo == "" {
				args.Repo = v
			}
		case KeyArtifact:
			if args.Artifact == "" {
				args.Artifact = v
			}
		default:
			// Fallbacks for positional / file ending based detection
			if args.GrpcServiceConfigPath == "" && strings.HasSuffix(k, jsonFileEnding) {
				args.GrpcServiceConfigPath = k
			} else if args.GapicYamlConfigPath == "" && strings.HasSuffix(k, gapicYamlFileEnding) {
				args.GapicYamlConfigPath = k
			} else if args.ServiceYamlConfigPath == "" && strings.HasSuffix(k, serviceYamlFileEnding) {
				args.ServiceYamlConfigPath = k
			}
		}
	}
	return args
}
