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

package python

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// ErrNilModel is returned when a nil api.API model is supplied to newCodec or Generate.
var ErrNilModel = errors.New("model cannot be nil")

// codec holds the configuration and context for the Python sidekick generator.
type codec struct {
	Model             *api.API
	Library           *config.Library
	OutDir            string
	GenerationYear    string
	PackageName       string
	PackageVersion    string
	DefaultVersion    string
	CurrentVersion    string
	GAPICNamespace    string
	GAPICName         string
	protoOptionsCache map[string]*protoServiceOptions
}

// newCodec constructs a new Python codec instance from model, library config, and outdir.
func newCodec(model *api.API, library *config.Library, outdir string) (*codec, error) {
	if model == nil {
		return nil, ErrNilModel
	}

	year := fmt.Sprintf("%04d", time.Now().Year())
	if library != nil && library.CopyrightYear != "" {
		year = library.CopyrightYear
	}

	version := "0.0.0"
	if library != nil && library.Version != "" {
		version = library.Version
	}

	packageName := model.PackageName
	if model.Name != "" {
		packageName = model.Name
	}
	if library != nil && library.Name != "" {
		packageName = library.Name
	}

	defaultVersion := ""
	if library != nil && library.Python != nil {
		defaultVersion = library.Python.DefaultVersion
	}

	var apiPath string
	if library != nil {
		for _, apiCfg := range library.APIs {
			if strings.ReplaceAll(apiCfg.Path, "/", ".") == model.PackageName || apiCfg.Path == model.PackageName {
				apiPath = apiCfg.Path
				break
			}
		}
		if apiPath == "" && len(library.APIs) == 1 {
			apiPath = library.APIs[0].Path
		}
	}
	if apiPath == "" {
		apiPath = strings.ReplaceAll(model.PackageName, ".", "/")
	}

	var namespace, name, currentVersion string
	if apiPath != "" {
		namespace = deriveGAPICNamespace(apiPath)
		name = deriveGAPICName(apiPath)
		currentVersion = serviceconfig.ExtractVersion(apiPath)
		if defaultVersion == "" {
			defaultVersion = currentVersion
		}
	}

	return &codec{
		Model:             model,
		Library:           library,
		OutDir:            outdir,
		GenerationYear:    year,
		PackageName:       packageName,
		PackageVersion:    version,
		DefaultVersion:    defaultVersion,
		CurrentVersion:    currentVersion,
		GAPICNamespace:    namespace,
		GAPICName:         name,
		protoOptionsCache: make(map[string]*protoServiceOptions),
	}, nil
}

func (c *codec) packageDir() string {
	if c.GAPICNamespace == "" && c.GAPICName == "" {
		return ""
	}
	nsPath := strings.ReplaceAll(c.GAPICNamespace, ".", "/")
	version := c.CurrentVersion
	if version == "" {
		version = c.DefaultVersion
	}
	pkgName := c.GAPICName
	if version != "" {
		pkgName = fmt.Sprintf("%s_%s", c.GAPICName, version)
	}
	return filepath.Join(nsPath, pkgName)
}

func (c *codec) rootPackageDir() string {
	if c.GAPICNamespace == "" && c.GAPICName == "" {
		return ""
	}
	nsPath := strings.ReplaceAll(c.GAPICNamespace, ".", "/")
	return filepath.Join(nsPath, c.GAPICName)
}

func (c *codec) isDefaultVersion() bool {
	return c.DefaultVersion == "" || c.CurrentVersion == "" || c.CurrentVersion == c.DefaultVersion
}

func (c *codec) pythonPackage() string {
	return strings.ReplaceAll(filepath.ToSlash(c.packageDir()), "/", ".")
}

// resolveTypeModule returns the Python module name for the given protobuf type ID.
// If the type ID is associated with a message or enum in the model, its source file's base name is used.
// Otherwise, it falls back to the snake_case name of the service or "common".
func (c *codec) resolveTypeModule(typeID string, service *api.Service) string {
	if c.Model != nil {
		for curID := typeID; curID != ""; {
			if msg := c.Model.Message(curID); msg != nil && msg.SourceLocation != nil && msg.SourceLocation.File != "" {
				return strings.TrimSuffix(filepath.Base(msg.SourceLocation.File), ".proto")
			}
			if enum := c.Model.Enum(curID); enum != nil && enum.SourceLocation != nil && enum.SourceLocation.File != "" {
				return strings.TrimSuffix(filepath.Base(enum.SourceLocation.File), ".proto")
			}
			idx := strings.LastIndex(curID, ".")
			if idx <= 0 {
				break
			}
			curID = curID[:idx]
		}
	}
	if service != nil {
		return snakeCase(service.Name)
	}
	return "common"
}

// transport returns the configured transport ("grpc", "rest", or "grpc+rest")
// across OptArgsByAPI settings, defaulting to "grpc+rest".
func (c *codec) transport() string {
	if c.Library != nil && c.Library.Python != nil {
		keys := slices.Sorted(maps.Keys(c.Library.Python.OptArgsByAPI))
		for _, k := range keys {
			for _, arg := range c.Library.Python.OptArgsByAPI[k] {
				if val, ok := strings.CutPrefix(arg, "transport="); ok {
					return val
				}
			}
		}
	}
	return "grpc+rest"
}

// hasGRPCTransport reports whether gRPC transport is enabled.
func (c *codec) hasGRPCTransport() bool {
	t := c.transport()
	return t == "grpc" || t == "grpc+rest"
}

// hasRESTTransport reports whether REST transport is enabled.
func (c *codec) hasRESTTransport() bool {
	t := c.transport()
	return t == "rest" || t == "grpc+rest"
}

// hasAsyncClient reports whether async client generation is supported.
func (c *codec) hasAsyncClient() bool {
	return c.hasGRPCTransport()
}
