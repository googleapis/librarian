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

package config

import (
	"os"
	"path/filepath"
)

// Sources contains the directory paths for source repositories used by
// sidekick.
type Sources struct {
	Conformance string
	Discovery   string
	Googleapis  string
	ProtobufSrc string
	Showcase    string
}

// SourceConfig holds the configuration for source roots and path resolution.
type SourceConfig struct {
	Sources     Sources
	ActiveRoots []string
	IncludeList []string
	ExcludeList []string
}

// ModelRoot returns the model root for the given specification source path.
func (c SourceConfig) ModelRoot(path string) string {
	if path == "schema/google/showcase/v1beta1" {
		return c.Sources.Showcase
	}
	return c.Sources.Googleapis
}

// Root returns the directory path for the given root name.
func (c SourceConfig) Root(name string) string {
	switch name {
	case "googleapis":
		return c.Sources.Googleapis
	case "discovery":
		return c.Sources.Discovery
	case "showcase":
		return c.Sources.Showcase
	case "protobuf", "protobuf-src":
		return c.Sources.ProtobufSrc
	case "conformance":
		return c.Sources.Conformance
	default:
		return ""
	}
}

// Resolve returns an absolute path for the given relative path if it is found
// within the active source roots. Otherwise, it returns the original path.
func (c SourceConfig) Resolve(relPath string) string {
	for _, root := range c.ActiveRoots {
		rootPath := c.Root(root)
		if rootPath == "" {
			continue
		}
		fullName := filepath.Join(rootPath, relPath)
		if stat, err := os.Stat(fullName); err == nil && !stat.IsDir() {
			return fullName
		}
	}
	return relPath
}
