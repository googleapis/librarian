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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/googleapis/librarian/internal/config"
)

// PostGenerate performs repository-level actions after all individual Java libraries have been generated.
func PostGenerate(ctx context.Context, cfg *config.Config) error {
	modules, err := searchForJavaModules()
	if err != nil {
		return fmt.Errorf("failed to search for java modules: %w", err)
	}
	if err := generateRootPom(modules); err != nil {
		return fmt.Errorf("failed to generate root pom: %w", err)
	}
	return nil
}

// searchForJavaModules scans top-level subdirectories in the current directory for those that
// contain a pom.xml file, excluding known non-library directories. Returns a sorted list of
// subdirectory names as module names.
func searchForJavaModules() ([]string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var modules []string
	exclusions := map[string]bool{
		"gapic-libraries-bom":      true,
		"google-cloud-jar-parent":  true,
		"google-cloud-pom-parent":  true,
		"google-cloud-shared-deps": true,
	}
	for _, entry := range entries {
		if !entry.IsDir() || exclusions[entry.Name()] {
			continue
		}
		if _, err := os.Stat(filepath.Join(entry.Name(), "pom.xml")); err == nil {
			modules = append(modules, entry.Name())
		}
	}
	sort.Strings(modules)
	return modules, nil
}

// generateRootPom writes the aggregator pom.xml for the monorepo root, including
// all discovered Java modules.
func generateRootPom(modules []string) error {
	f, err := os.Create("pom.xml")
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	data := struct {
		Modules []string
	}{
		Modules: modules,
	}
	return templates.ExecuteTemplate(f, "root-pom.xml.tmpl", data)
}
