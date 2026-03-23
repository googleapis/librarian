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
	// gapicBom is the name of the directory and artifact ID for the
	// generated Bill of Materials (BOM) for all GAPIC libraries.
	gapicBom  = "gapic-libraries-bom"
	bomSuffix = "-bom"
)

var (
	errModuleDiscovery      = errors.New("failed to search for java modules")
	errRootPomGeneration    = errors.New("failed to generate root pom")
	errInvalidBOMArtifactID = errors.New("invalid BOM artifact ID")
	errMalformedBOM         = errors.New("malformed BOM")
)

// legacyBOM represents a library that does not have a -bom module
// and included directly in the GAPIC BOM.
type legacyBOM struct {
	module     string
	groupID    string
	artifactID string
}

var (
	dnsBom          = legacyBOM{"java-dns", "com.google.cloud", "google-cloud-dns"}
	notificationBom = legacyBOM{"java-notification", "com.google.cloud", "google-cloud-notification"}
	grafeasBom      = legacyBOM{"java-grafeas", "io.grafeas", "grafeas"}
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
		return fmt.Errorf("failed to generate %s: %w", gapicBom, err)
	}
	return nil
}

var ignoredDirs = map[string]bool{
	gapicBom:                   true,
	"google-cloud-jar-parent":  true,
	"google-cloud-pom-parent":  true,
	"google-cloud-shared-deps": true,
}

// searchForJavaModules scans top-level subdirectories in the repoPath for those that
// contain a pom.xml file, excluding known non-library directories. Returns a sorted list of
// subdirectory names as module names.
func searchForJavaModules(repoPath string) ([]string, error) {
	modules, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, module := range modules {
		if !module.IsDir() || ignoredDirs[module.Name()] {
			continue
		}
		if _, err := os.Stat(filepath.Join(repoPath, module.Name(), "pom.xml")); err == nil {
			names = append(names, module.Name())
		}
	}
	sort.Strings(names)
	return names, nil
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
	modules, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}
	var configs []bomConfig
	for _, module := range modules {
		if !module.IsDir() || module.Name() == gapicBom {
			continue
		}
		moduleConfigs, err := searchModuleForBOM(repoPath, module.Name())
		if err != nil {
			return nil, err
		}
		configs = append(configs, moduleConfigs...)
	}

	legacies, err := collectLegacyBOMs(repoPath, dnsBom, notificationBom)
	if err != nil {
		return nil, err
	}
	configs = append(configs, legacies...)
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].ArtifactID < configs[j].ArtifactID
	})
	// Add Grafeas last. This is done after sorting to match the current order in google-cloud-java.
	// TODO(https://github.com/googleapis/librarian/issues/4706): Move this prior to sort.
	grafeas, err := collectLegacyBOMs(repoPath, grafeasBom)
	if err != nil {
		return nil, err
	}
	return append(configs, grafeas...), nil
}

// searchModuleForBOM scans a specific module's directory for submodules that end in "-bom"
// and contain a pom.xml file. Returns a list of bomConfig objects for any discovered BOMs.
func searchModuleForBOM(repoPath, moduleName string) ([]bomConfig, error) {
	submodules, err := os.ReadDir(filepath.Join(repoPath, moduleName))
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", moduleName, err)
	}
	var configs []bomConfig
	for _, submodule := range submodules {
		if !submodule.IsDir() || !strings.HasSuffix(submodule.Name(), bomSuffix) {
			continue
		}
		pomPath := filepath.Join(repoPath, moduleName, submodule.Name(), "pom.xml")
		if _, err := os.Stat(pomPath); err != nil {
			continue
		}
		conf, err := extractBOMConfig(repoPath, moduleName, submodule.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to extract BOM config from %s: %w", pomPath, err)
		}
		if groupInclusions[conf.GroupID] {
			configs = append(configs, conf)
		}
	}
	return configs, nil
}

// collectLegacyBOMs parses pom.xml files for legacy libraries that do not have
// -bom modules and returns their BOM configurations.
func collectLegacyBOMs(repoPath string, boms ...legacyBOM) ([]bomConfig, error) {
	var configs []bomConfig
	for _, b := range boms {
		pomPath := filepath.Join(repoPath, b.module, "pom.xml")
		data, err := os.ReadFile(pomPath)
		if err != nil {
			return nil, fmt.Errorf("read legacy pom %s: %w", pomPath, err)
		}
		var p mavenProject
		if err := xml.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("unmarshal legacy pom %s: %w", pomPath, err)
		}
		configs = append(configs, bomConfig{
			GroupID:           b.groupID,
			ArtifactID:        b.artifactID,
			Version:           p.Version,
			VersionAnnotation: b.artifactID,
			IsImport:          false,
		})
	}
	return configs, nil
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
	if !strings.HasSuffix(artifactID, bomSuffix) {
		return "", fmt.Errorf("%s: %w", artifactID, errInvalidBOMArtifactID)
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
	bomDir := filepath.Join(repoPath, gapicBom)
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
