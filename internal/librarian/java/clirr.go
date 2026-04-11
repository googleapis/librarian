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

package java

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

var (
	//go:embed template/*.tmpl
	templatesFS embed.FS
	templates   = template.Must(template.New("").ParseFS(templatesFS, "template/*.tmpl"))
)

const (
	// clirrIgnoreFile is the name of the Clirr ignore file to generate.
	clirrIgnoreFile = "clirr-ignored-differences.xml"
	// templateName is the name of the template used to generate Clirr ignore file.
	templateName = "clirr-ignored-differences.xml.tmpl"
)

// generateClirrIfMissing generates the clirr-ignored-differences.xml file in the protoModulePath
// if it doesn't already exist in the checkPath.
//
// It identifies Java packages containing Protobuf-generated code by searching for
// files ending in "OrBuilder.java" under "src/main/java". The "OrBuilder" suffix
// is used as a reliable marker because it is consistently generated for every
// Protobuf message.
//
// The generated file contains a set of whitelist rules that tell the Clirr tool
// to ignore specific changes (like method additions to interfaces) to
// prevent false-positive binary compatibility failures in the build.
func generateClirrIfMissing(protoModulePath, checkPath string) error {
	if checkPath != "" {
		repoFilePath := filepath.Join(checkPath, clirrIgnoreFile)
		_, err := os.Stat(repoFilePath)
		if err == nil {
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to check for %s: %w", repoFilePath, err)
		}
	}
	protoPaths, err := findProtoPackages(protoModulePath)
	if err != nil {
		return fmt.Errorf("failed to find proto packages in %s: %w", protoModulePath, err)
	}
	if len(protoPaths) == 0 {
		return nil
	}
	outputPath := filepath.Join(protoModulePath, clirrIgnoreFile)
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", outputPath, err)
	}
	var returnErr error
	defer func() {
		if cerr := f.Close(); cerr != nil {
			if returnErr == nil {
				returnErr = cerr
			} else {
				returnErr = fmt.Errorf("%w; close error: %w", returnErr, cerr)
			}
		}
	}()
	returnErr = templates.ExecuteTemplate(f, templateName, protoPaths)
	return returnErr
}

func findProtoPackages(protoModulePath string) ([]string, error) {
	srcDir := filepath.Join(protoModulePath, "src", "main", "java")
	if _, err := os.Stat(srcDir); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	packageSet := make(map[string]bool)
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), "OrBuilder.java") {
			return nil
		}
		rel, err := filepath.Rel(srcDir, filepath.Dir(path))
		if err != nil {
			return err
		}
		if pkgPath := filepath.ToSlash(rel); pkgPath != "." {
			packageSet[pkgPath] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk source directory %s: %w", srcDir, err)
	}
	packages := make([]string, 0, len(packageSet))
	for pkg := range packageSet {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	return packages, nil
}
