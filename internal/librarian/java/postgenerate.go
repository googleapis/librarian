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

const (
	// rootLibrary is the name of the monorepo library used to identify
	// the version for all libraries in the repository.
	rootLibrary = "google-cloud-java"
	// gapicBOM is the name of the directory and artifact ID for the
	// generated Bill of Materials (BOM) for all GAPIC libraries.
	gapicBOM = "gapic-libraries-bom"
)

var (
	errModuleDiscovery      = errors.New("failed to search for java modules")
	errRootPomGeneration    = errors.New("failed to generate root pom")
	errInvalidBOMArtifactID = errors.New("invalid BOM artifact ID")
	errMalformedBOM         = errors.New("malformed BOM")
)

// PostGenerate performs repository-level actions after all individual Java libraries have been generated.
func PostGenerate(ctx context.Context, repoPath string, cfg *config.Config) error {
	monorepoVersion := ""
	for _, lib := range cfg.Libraries {
		if lib.Name == rootLibrary {
			monorepoVersion = lib.Version
			break
		}
	}
	if monorepoVersion == "" {
		return fmt.Errorf("%s library not found in librarian.yaml", rootLibrary)
	}
	modules, err := searchForJavaModules(repoPath)
	if err != nil {
		return fmt.Errorf("%w: %w", errModuleDiscovery, err)
	}
	if err := generateRootPom(repoPath, modules); err != nil {
		return fmt.Errorf("%w: %w", errRootPomGeneration, err)
	}
	bomConfigs, err := searchForBOMArtifacts(repoPath)
	if err != nil {
		return fmt.Errorf("failed to search for BOM artifacts: %w", err)
	}
	if err := generateGapicLibrariesBOM(repoPath, monorepoVersion, bomConfigs); err != nil {
		return fmt.Errorf("failed to generate %s: %w", gapicBOM, err)
	}
	return nil
}

var ignoredDirs = map[string]bool{
	gapicBOM:                   true,
	"google-cloud-jar-parent":  true,
	"google-cloud-pom-parent":  true,
	"google-cloud-shared-deps": true,
}

// searchForJavaModules scans top-level subdirectories in the repoPath for those that
// contain a pom.xml file, excluding known non-library directories. Returns a sorted list of
// subdirectory names as module names.
func searchForJavaModules(repoPath string) ([]string, error) {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}
	var modules []string
	for _, entry := range entries {
		if !entry.IsDir() || ignoredDirs[entry.Name()] {
			continue
		}
		if _, err := os.Stat(filepath.Join(repoPath, entry.Name(), "pom.xml")); err == nil {
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

var groupInclusions = map[string]bool{
	"com.google.cloud":     true,
	"com.google.analytics": true,
	"com.google.area120":   true,
}

// searchForBOMArtifacts scans the repoPath for subdirectories that contain a -bom subdirectory
// with a pom.xml file. It also includes specific special-case modules like dns, notification, and grafeas.
// It returns a list of bomConfig objects sorted by ArtifactID.
func searchForBOMArtifacts(repoPath string) ([]bomConfig, error) {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}
	var configs []bomConfig
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == gapicBOM {
			continue
		}
		// Search for -bom subdirectories.
		subEntries, err := os.ReadDir(filepath.Join(repoPath, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", entry.Name(), err)
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() || !strings.HasSuffix(subEntry.Name(), "-bom") {
				continue
			}
			pomPath := filepath.Join(repoPath, entry.Name(), subEntry.Name(), "pom.xml")
			if _, err := os.Stat(pomPath); err != nil {
				continue
			}
			conf, err := extractBOMConfig(repoPath, entry.Name(), subEntry.Name())
			if err != nil {
				return nil, fmt.Errorf("failed to extract BOM config from %s: %w", pomPath, err)
			}
			if groupInclusions[conf.GroupID] {
				configs = append(configs, conf)
			}
		}
	}
	// Handle edge cases before sorting. These are older libraries that do not have a BOM module;
	// their clients are included directly in the GAPIC BOM.
	specialBOMs := []struct {
		module     string
		groupID    string
		artifactID string
	}{
		{"java-dns", "com.google.cloud", "google-cloud-dns"},
		{"java-notification", "com.google.cloud", "google-cloud-notification"},
	}
	for _, bom := range specialBOMs {
		conf, err := handleSpecialBOM(repoPath, bom.module, bom.groupID, bom.artifactID)
		if err != nil {
			return nil, fmt.Errorf("failed to handle special BOM for %s: %w", bom.module, err)
		}
		configs = append(configs, conf)
	}
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].ArtifactID < configs[j].ArtifactID
	})
	// Handle edge cases. This is done after sorting to match the current order in google-cloud-java.
	// It is without a BOM and included directly in the GAPIC BOM.
	// TODO(https://github.com/googleapis/librarian/issues/4706): Move this prior to sort.
	conf, err := handleSpecialBOM(repoPath, "java-grafeas", "io.grafeas", "grafeas")
	if err != nil {
		return nil, fmt.Errorf("failed to handle special BOM for java-grafeas: %w", err)
	}
	configs = append(configs, conf)
	return configs, nil
}

// handleSpecialBOM for special cases, such as java-dns, java-notification, and java-grafeas,
// only version is parsed from pom.xml in the module.
func handleSpecialBOM(repoPath, module, groupID, artifactID string) (bomConfig, error) {
	pomPath := filepath.Join(repoPath, module, "pom.xml")
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

// extractBOMConfig parses a pom.xml file within a library's -bom subdirectory to
// produce a bomConfig object.
func extractBOMConfig(repoPath, libraryDir, bomDir string) (bomConfig, error) {
	pomPath := filepath.Join(repoPath, libraryDir, bomDir, "pom.xml")
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return bomConfig{}, err
	}
	var p mavenProject
	if err := xml.Unmarshal(data, &p); err != nil {
		return bomConfig{}, fmt.Errorf("%w: %w", errMalformedBOM, err)
	}
	// Derive version annotation from artifactId.
	versionAnnotation, err := deriveVersionAnnotation(p.ArtifactID)
	if err != nil {
		return bomConfig{}, err
	}
	return bomConfig{
		GroupID:           p.GroupID,
		ArtifactID:        p.ArtifactID,
		Version:           p.Version,
		VersionAnnotation: versionAnnotation,
		IsImport:          true,
	}, nil
}

// deriveVersionAnnotation extracts the version annotation from a Maven artifact ID
// by removing the last segment (assumed to be -bom).
func deriveVersionAnnotation(artifactID string) (string, error) {
	bomSuffix := "-bom"
	if !strings.HasSuffix(artifactID, bomSuffix) {
		return "", fmt.Errorf("%w: %s", errInvalidBOMArtifactID, artifactID)
	}
	return strings.TrimSuffix(artifactID, bomSuffix), nil
}

// generateRootPom writes the aggregator pom.xml for the monorepo root, including
// all discovered Java modules.
func generateRootPom(repoPath string, modules []string) (err error) {
	f, err := os.Create(filepath.Join(repoPath, "pom.xml"))
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
func generateGapicLibrariesBOM(repoPath, version string, bomConfigs []bomConfig) (err error) {
	bomDir := filepath.Join(repoPath, gapicBOM)
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
	if terr := templates.ExecuteTemplate(f, "gapic-libraries-bom.xml.tmpl", data); terr != nil {
		return terr
	}
	return nil
}
