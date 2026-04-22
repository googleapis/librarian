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
	gapicBOM  = "gapic-libraries-bom"
	bomSuffix = "-bom"
)

var (
	errModuleDiscovery      = errors.New("failed to search for java modules")
	errRootPOMGeneration    = errors.New("failed to generate root pom.xml")
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
	dnsBOM          = legacyBOM{"java-dns", "com.google.cloud", "google-cloud-dns"}
	notificationBOM = legacyBOM{"java-notification", "com.google.cloud", "google-cloud-notification"}
	grafeasBOM      = legacyBOM{"java-grafeas", "io.grafeas", "grafeas"}
)

// PostGenerate performs repository-level actions after all individual Java libraries have been generated.
// TODO(https://github.com/googleapis/librarian/issues/5529): remove appending to versions.txt
func PostGenerate(ctx context.Context, repoPath string, cfg *config.Config, newVersions []string) error {
	monorepoVersion, err := findMonorepoVersion(cfg)
	if err != nil {
		return err
	}
	if monorepoVersion == "" {
		return fmt.Errorf("%s library not found in librarian.yaml", rootLibrary)
	}

	if err := appendVersions(repoPath, newVersions); err != nil {
		return err
	}

	modules, err := searchForJavaModules(repoPath)
	if err != nil {
		return fmt.Errorf("%w: %w", errModuleDiscovery, err)
	}
	if err := generateRootPOM(repoPath, modules); err != nil {
		return fmt.Errorf("%w: %w", errRootPOMGeneration, err)
	}
	bomConfigs, err := searchForBOMArtifacts(repoPath)
	if err != nil {
		return fmt.Errorf("failed to search for BOM artifacts: %w", err)
	}
	if err := generateGAPICLibrariesBOM(repoPath, monorepoVersion, bomConfigs); err != nil {
		return fmt.Errorf("failed to generate %s: %w", gapicBOM, err)
	}
	return nil
}

func appendVersions(repoPath string, versions []string) (err error) {
	if len(versions) == 0 {
		return nil
	}
	versionsPath := filepath.Join(repoPath, "versions.txt")
	f, err := os.OpenFile(versionsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open versions.txt: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close versions.txt: %w", cerr)
		}
	}()

	for _, v := range versions {
		if _, err := fmt.Fprintf(f, "%s\n", v); err != nil {
			return fmt.Errorf("failed to write to versions.txt: %w", err)
		}
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

// searchForBOMArtifacts scans the repoPath for subdirectories that contain a -bom subdirectory
// with a pom.xml file. It also includes specific special-case modules like dns, notification, and grafeas.
// It returns a list of bomConfig objects sorted by ArtifactID.
func searchForBOMArtifacts(repoPath string) ([]*bomConfig, error) {
	modules, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}
	configs := make([]*bomConfig, 0, len(modules)+3)
	for _, module := range modules {
		if !module.IsDir() || module.Name() == gapicBOM {
			continue
		}
		moduleConfigs, err := searchModuleForBOM(repoPath, module.Name())
		if err != nil {
			return nil, err
		}
		configs = append(configs, moduleConfigs...)
	}

	legacies, err := collectLegacyBOMs(repoPath, dnsBOM, notificationBOM)
	if err != nil {
		return nil, err
	}
	configs = append(configs, legacies...)
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].ArtifactID < configs[j].ArtifactID
	})
	// Add Grafeas last. This is done after sorting to match the current order in google-cloud-java.
	// TODO(https://github.com/googleapis/librarian/issues/4706): Move this prior to sort.
	grafeas, err := collectLegacyBOMs(repoPath, grafeasBOM)
	if err != nil {
		return nil, err
	}
	return append(configs, grafeas...), nil
}

// searchModuleForBOM scans a specific module's directory for submodules that end in "-bom"
// and contain a pom.xml file. Returns a list of bomConfig objects for any discovered BOMs.
func searchModuleForBOM(repoPath, moduleName string) ([]*bomConfig, error) {
	submodules, err := os.ReadDir(filepath.Join(repoPath, moduleName))
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", moduleName, err)
	}
	configs := make([]*bomConfig, 0, len(submodules))
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
func collectLegacyBOMs(repoPath string, boms ...legacyBOM) ([]*bomConfig, error) {
	configs := make([]*bomConfig, 0, len(boms))
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
		configs = append(configs, &bomConfig{
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
func extractBOMConfig(repoPath, libraryDir, bomDir string) (*bomConfig, error) {
	pomPath := filepath.Join(repoPath, libraryDir, bomDir, "pom.xml")
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, err
	}
	var p mavenProject
	if err := xml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("%w: %w", errMalformedBOM, err)
	}
	versionAnnotation, err := deriveVersionAnnotation(p.ArtifactID)
	if err != nil {
		return nil, err
	}
	return &bomConfig{
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

// generateRootPOM writes the aggregator pom.xml for the monorepo root, including
// all discovered Java modules.
func generateRootPOM(repoPath string, modules []string) error {
	data := struct {
		Modules []string
	}{
		Modules: modules,
	}
	return writePOM(filepath.Join(repoPath, "pom.xml"), "root-pom.xml.tmpl", data)
}

// generateGAPICLibrariesBOM writes the gapic-libraries-bom/pom.xml file, which manages
// versions for all individual library BOMs in the monorepo.
func generateGAPICLibrariesBOM(repoPath, version string, bomConfigs []*bomConfig) error {
	bomDir := filepath.Join(repoPath, gapicBOM)
	if err := os.MkdirAll(bomDir, 0755); err != nil {
		return err
	}
	data := struct {
		Version    string
		BOMConfigs []*bomConfig
	}{
		Version:    version,
		BOMConfigs: bomConfigs,
	}
	return writePOM(filepath.Join(bomDir, "pom.xml"), "gapic-libraries-bom.xml.tmpl", data)
}
