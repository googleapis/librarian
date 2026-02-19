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
	_ "embed"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/license"
)

var (
	//go:embed template/_version.go.txt
	clientVersionTmpl string

	//go:embed template/_internal_version.go.txt
	internalVersionTmpl string
)

func generateInternalVersionFile(moduleDir, version string) (err error) {
	if version == "" {
		version = "0.0.0"
	}
	internalDir := filepath.Join(moduleDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(internalDir, "version.go"))
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	if err := writeLicenseHeader(f); err != nil {
		return err
	}
	t := template.Must(template.New("version").Parse(internalVersionTmpl))
	return t.Execute(f, map[string]any{
		"Version": version,
	})
}

func generateClientVersionFile(library *config.Library, apiPath string) (err error) {
	dir, clientDir := resolveClientPath(library, apiPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "version.go"))
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	if err := writeLicenseHeader(f); err != nil {
		return err
	}
	t := template.Must(template.New("version").Parse(clientVersionTmpl))
	pkg := library.Name
	if clientDir != "" {
		pkg = clientDir
	}
	return t.Execute(f, map[string]any{
		"Package":    pkg,
		"ModulePath": modulePath(library),
	})
}

// resolveClientPath constructs the full path for the API version and determines the client directory.
func resolveClientPath(library *config.Library, apiPath string) (string, string) {
	version := filepath.Base(apiPath)
	clientDir := clientDirectory(library, apiPath)
	return filepath.Join(library.Output, library.Name, clientDir, "api"+version), clientDir
}

func clientDirectory(library *config.Library, apiPath string) string {
	goAPI := findGoAPI(library, apiPath)
	if goAPI != nil {
		return goAPI.ClientDirectory
	}
	// Return an empty client directory if we can't find one.
	return ""
}

// writeLicenseHeader writes the license header as Go comments to the given file.
func writeLicenseHeader(f *os.File) error {
	year := time.Now().Format("2006")
	for _, line := range license.Header(year) {
		if _, err := f.WriteString("//" + line + "\n"); err != nil {
			return err
		}
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}
	return nil
}
