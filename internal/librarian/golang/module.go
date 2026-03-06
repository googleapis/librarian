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

package golang

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	errGoAPINotFound = errors.New("go API not found")
)

// Fill populates empty Go-specific fields from the api path.
// Library configurations takes precedence.
func Fill(library *config.Library) *config.Library {
	if library.Go == nil {
		library.Go = &config.GoModule{}
	}
	var goAPIs []*config.GoAPI
	for _, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if goAPI == nil {
			goAPI = &config.GoAPI{
				Path: api.Path,
			}
		}
		importPath, clientPkg := defaultImportPathAndClientPkg(api.Path)
		if goAPI.ImportPath == "" {
			goAPI.ImportPath = importPath
		}
		if goAPI.ClientPackage == "" {
			goAPI.ClientPackage = clientPkg
		}
		goAPIs = append(goAPIs, goAPI)
	}
	library.Go.GoAPIs = goAPIs

	return library
}

func findGoAPI(library *config.Library, apiPath string) *config.GoAPI {
	if library.Go == nil {
		return nil
	}
	for _, ga := range library.Go.GoAPIs {
		if ga.Path == apiPath {
			return ga
		}
	}
	return nil
}

// modulePath returns the Go module path for the library. ModulePathVersion is
// set for modules at v2+, e.g. "cloud.google.com/go/pubsub/v2".
func modulePath(library *config.Library) string {
	path := "cloud.google.com/go/" + library.Name
	if library.Go != nil && library.Go.ModulePathVersion != "" {
		path += "/" + library.Go.ModulePathVersion
	}
	return path
}

// initModule initializes and tidies a Go module in the given directory.
func initModule(ctx context.Context, dir, modPath string) error {
	if err := command.RunInDir(ctx, dir, "go", "mod", "init", modPath); err != nil {
		return err
	}
	return command.RunInDir(ctx, dir, "go", "mod", "tidy")
}

// defaultImportPathAndClientPkg returns the default Go import path and client package name
// based on the provided API path.
//
// The API path is expected to be google/cloud/{dir}/{0 or more nested directories}/{version}.
func defaultImportPathAndClientPkg(apiPath string) (string, string) {
	apiPath = strings.TrimPrefix(apiPath, "google/cloud/")
	apiPath = strings.TrimPrefix(apiPath, "google/")
	idx := strings.LastIndex(apiPath, "/")
	version := serviceconfig.ExtractVersion(apiPath)
	if idx == -1 || version == "" {
		// Do not guess non-versioned APIs, define the import path and
		// client package name in Go API configuration.
		return "", ""
	}
	importPath, version := apiPath[:idx], apiPath[idx+1:]
	idx = strings.LastIndex(importPath, "/")
	pkg := importPath[idx+1:]
	return fmt.Sprintf("%s/api%s", importPath, version), pkg
}

// clientPathFromLibraryRoot returns the relative path from the module root to the client directory.
// It strips any module path version from the import path to get the correct filesystem path.
func clientPathFromLibraryRoot(library *config.Library, goAPI *config.GoAPI) string {
	importPath := goAPI.ImportPath
	if library.Go != nil && library.Go.ModulePathVersion != "" {
		modulePathVersion := filepath.Join(string(filepath.Separator), library.Go.ModulePathVersion)
		importPath = strings.Replace(importPath, modulePathVersion, "", 1)
	}
	return importPath
}

// snippetDirectory returns the path to the directory where Go snippets are generated
// for the given library output directory and Go import path.
func snippetDirectory(output, importPath string) string {
	return filepath.Join(output, "internal", "generated", "snippets", importPath)
}
