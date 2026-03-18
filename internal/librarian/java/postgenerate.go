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
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

var (
	errModuleDiscovery   = errors.New("failed to search for java modules")
	errRootPomGeneration = errors.New("failed to generate root pom")
)

// PostGenerate performs repository-level actions after all individual Java libraries have been generated.
func PostGenerate(ctx context.Context, cfg *config.Config) error {
	monorepoVersion := ""
	for _, lib := range cfg.Libraries {
		if lib.Name == "google-cloud-java" {
			monorepoVersion = lib.Version
			break
		}
	}
	if monorepoVersion == "" {
		return fmt.Errorf("google-cloud-java library not found in librarian.yaml")
	}
	modules, err := searchForJavaModules()
	if err != nil {
		return fmt.Errorf("%w: %w", errModuleDiscovery, err)
	}
	if err := generateRootPom(modules); err != nil {
		return fmt.Errorf("%w: %w", errRootPomGeneration, err)
	}
	bomConfigs, err := searchForBOMArtifacts()
	if err != nil {
		return fmt.Errorf("failed to search for BOM artifacts: %w", err)
	}
	if err := generateGapicLibrariesBOM(monorepoVersion, bomConfigs); err != nil {
		return fmt.Errorf("failed to generate gapic-libraries-bom: %w", err)
	}
	return nil
}

var ignoredDirs = map[string]bool{
	"gapic-libraries-bom":      true,
	"google-cloud-jar-parent":  true,
	"google-cloud-pom-parent":  true,
	"google-cloud-shared-deps": true,
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
	for _, entry := range entries {
		if !entry.IsDir() || ignoredDirs[entry.Name()] {
			continue
		}
		if _, err := os.Stat(filepath.Join(entry.Name(), "pom.xml")); err == nil {
			modules = append(modules, entry.Name())
		}
	}
	sort.Strings(modules)
	return modules, nil
}

type bomConfig struct {
	GroupID           string
	ArtifactID        string
	Version           string
	VersionAnnotation string
	IsImport          bool
}

// mavenProject represents a minimal Maven pom.xml for discovery.
type mavenProject struct {
	XMLName    xml.Name `xml:"http://maven.apache.org/POM/4.0.0 project"`
	GroupID    string   `xml:"groupId"`
	ArtifactID string   `xml:"artifactId"`
	Version    string   `xml:"version"`
}

// searchForBOMArtifacts scans the current directory for subdirectories that contain a -bom subdirectory
// with a pom.xml. It also includes specific special-case modules like dns, notification, and grafeas.
// Returns a list of bomConfig objects sorted by ArtifactID.
func searchForBOMArtifacts() ([]bomConfig, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var configs []bomConfig
	groupInclusions := map[string]bool{
		"com.google.cloud":     true,
		"com.google.analytics": true,
		"com.google.area120":   true,
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "gapic-libraries-bom" {
			continue
		}
		// Search for -bom subdirectories
		subEntries, err := os.ReadDir(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", entry.Name(), err)
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() || !strings.HasSuffix(subEntry.Name(), "-bom") {
				continue
			}
			pomPath := filepath.Join(entry.Name(), subEntry.Name(), "pom.xml")
			if _, err := os.Stat(pomPath); err != nil {
				continue
			}
			conf, err := extractBOMConfig(entry.Name(), subEntry.Name())
			if err != nil {
				return nil, fmt.Errorf("failed to extract BOM config from %s: %w", pomPath, err)
			}
			if groupInclusions[conf.GroupID] {
				configs = append(configs, conf)
			}
		}
	}
	// handle edge case: java-dns
	conf, err := handleSpecialBOM("java-dns", "com.google.cloud", "google-cloud-dns")
	if err != nil {
		return nil, fmt.Errorf("failed to handle special BOM for java-dns: %w", err)
	}
	configs = append(configs, conf)

	// handle edge case: java-notification
	conf, err = handleSpecialBOM("java-notification", "com.google.cloud", "google-cloud-notification")
	if err != nil {
		return nil, fmt.Errorf("failed to handle special BOM for java-notification: %w", err)
	}
	configs = append(configs, conf)
	// Sort by ArtifactID for determinism
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].ArtifactID < configs[j].ArtifactID
	})
	// handle edge case: java-grafeas and add to the end
	conf, err = handleSpecialBOM("java-grafeas", "io.grafeas", "grafeas")
	if err != nil {
		return nil, fmt.Errorf("failed to handle special BOM for java-grafeas: %w", err)
	}
	configs = append(configs, conf)

	return configs, nil
}

func handleSpecialBOM(module, groupID, artifactID string) (bomConfig, error) {
	pomPath := filepath.Join(module, "pom.xml")
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return bomConfig{}, err
	}

	var p mavenProject
	if err := xml.Unmarshal(data, &p); err != nil {
		return bomConfig{}, err
	}

	return bomConfig{
		GroupID:           groupID,
		ArtifactID:        artifactID,
		Version:           p.Version,
		VersionAnnotation: artifactID,
		IsImport:          false,
	}, nil
}

func extractBOMConfig(libraryDir, bomDir string) (bomConfig, error) {
	pomPath := filepath.Join(libraryDir, bomDir, "pom.xml")
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return bomConfig{}, err
	}
	var p mavenProject
	if err := xml.Unmarshal(data, &p); err != nil {
		return bomConfig{}, err
	}

	// Calculate VersionAnnotation: artifactId without the last segment (assumed to be -bom)
	// Following hermetic_build logic: artifact_id[:artifact_id.rfind("-")]
	lastDash := strings.LastIndex(p.ArtifactID, "-")
	versionAnnotation := p.ArtifactID
	if lastDash != -1 {
		versionAnnotation = p.ArtifactID[:lastDash]
	}

	return bomConfig{
		GroupID:           p.GroupID,
		ArtifactID:        p.ArtifactID,
		Version:           p.Version,
		VersionAnnotation: versionAnnotation,
		IsImport:          true,
	}, nil
}

// generateRootPom writes the aggregator pom.xml for the monorepo root, including
// all discovered Java modules.
func generateRootPom(modules []string) (err error) {
	f, err := os.Create("pom.xml")
	if err != nil {
		return fmt.Errorf("failed to create root pom.xml: %w", err)
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
	if terr := templates.ExecuteTemplate(f, "root-pom.xml.tmpl", data); terr != nil {
		return fmt.Errorf("failed to execute root-pom template: %w", terr)
	}
	return nil
}

// generateGapicLibrariesBOM writes the gapic-libraries-bom/pom.xml file, which manages
// versions for all individual library BOMs in the monorepo.
func generateGapicLibrariesBOM(version string, bomConfigs []bomConfig) error {
	bomDir := "gapic-libraries-bom"
	if err := os.MkdirAll(bomDir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(bomDir, "pom.xml"))
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
		Version    string
		BOMConfigs []bomConfig
	}{
		Version:    version,
		BOMConfigs: bomConfigs,
	}
	return templates.ExecuteTemplate(f, "gapic-libraries-bom.xml.tmpl", data)
}
