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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	protoPomTemplateName  = "module_proto_pom.xml.tmpl"
	grpcPomTemplateName   = "module_grpc_pom.xml.tmpl"
	clientPomTemplateName = "module_client_pom.xml.tmpl"
	parentPomTemplateName = "module_parent_pom.xml.tmpl"
	bomPomTemplateName    = "module_bom_pom.xml.tmpl"
	googleGroupID         = "com.google"
	protoGrpcSuffix       = ".api.grpc"
)

// grpcProtoPomData holds the data for rendering POM templates.
type grpcProtoPomData struct {
	Proto          coordinates
	Grpc           coordinates
	Parent         coordinates
	Version        string
	MainArtifactID string
}

type coordinates struct {
	GroupID    string
	ArtifactID string
	Version    string
}

// clientPomData holds the data for rendering the client library POM template.
type clientPomData struct {
	Client       coordinates
	Version      string
	Name         string
	Description  string
	Parent       coordinates
	ProtoModules []coordinates
	GrpcModules  []coordinates
}

// bomParentPomData holds the data for rendering the BOM and Parent library POM template.
type bomParentPomData struct {
	MainModule      coordinates
	Name            string
	MonorepoVersion string
	Modules         []coordinates
}

// javaModule represents a Maven module and its POM generation state.
type javaModule struct {
	artifactID   string
	dir          string
	isMissing    bool
	templateData any
	template     string
}

// generatePomsIfMissing generates missing proto-*, grpc-*, and client POMs.
func generatePomsIfMissing(library *config.Library, libraryDir, googleapisDir, monorepoVersion string, metadata *repoMetadata) error {
	modules, err := collectModules(library, libraryDir, googleapisDir, monorepoVersion, metadata)
	if err != nil {
		return err
	}
	var newModules []string
	for _, m := range modules {
		if !m.isMissing {
			continue
		}
		if err := writePom(filepath.Join(m.dir, "pom.xml"), m.template, m.templateData); err != nil {
			return fmt.Errorf("failed to generate %s: %w", m.artifactID, err)
		}
		newModules = append(newModules, m.artifactID)
	}
	if len(newModules) > 0 {
		if err := updateVersionsFile(libraryDir, newModules, library.Version); err != nil {
			return fmt.Errorf("failed to update versions.txt: %w", err)
		}
	}
	return nil
}

// collectModules identifies all expected proto-*, grpc-*, client, BOM and Parent modules
// for the given library based on its configuration and checks a pom.xml presence
// on the filesystem.
//
// All expected modules are collected (even if they exist) because the client
// module's POM requires a full list of all proto and gRPC dependencies
// to ensure its dependency list is fully synchronized.
func collectModules(library *config.Library, libraryDir, googleapisDir, monorepoVersion string, metadata *repoMetadata) ([]javaModule, error) {
	distName := deriveDistributionName(library)
	parts := strings.SplitN(distName, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid distribution name %q: expected format groupID:artifactID", distName)
	}
	gapicGroupID := parts[0]
	gapicArtifactID := parts[1]

	var modules []javaModule
	protoModules := make([]coordinates, 0, len(library.APIs))
	grpcModules := make([]coordinates, 0, len(library.APIs))
	for _, api := range library.APIs {
		version := serviceconfig.ExtractVersion(api.Path)
		if version == "" {
			return nil, fmt.Errorf("failed to extract version from API path %q", api.Path)
		}

		names := deriveModuleNames(gapicArtifactID, version)

		apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageJava)
		if err != nil {
			return nil, fmt.Errorf("failed to find api config for %s: %w", api.Path, err)
		}
		transport := apiCfg.Transport(config.LanguageJava)

		protoGrpcID := protoGroupID(gapicGroupID)
		data := grpcProtoPomData{
			Proto: coordinates{
				GroupID:    protoGrpcID,
				ArtifactID: names.proto,
				Version:    library.Version,
			},
			Grpc: coordinates{
				GroupID:    protoGrpcID,
				ArtifactID: names.grpc,
				Version:    library.Version,
			},
			Parent: coordinates{
				GroupID:    gapicGroupID,
				ArtifactID: fmt.Sprintf("%s-parent", gapicArtifactID),
				Version:    library.Version,
			},
			MainArtifactID: gapicArtifactID,
			Version:        library.Version,
		}

		// Proto module
		protoDir := filepath.Join(libraryDir, names.proto)
		isProtoMissing, err := isPomMissing(protoDir)
		if err != nil {
			return nil, err
		}
		modules = append(modules, javaModule{
			artifactID:   names.proto,
			dir:          protoDir,
			isMissing:    isProtoMissing,
			templateData: data,
			template:     protoPomTemplateName,
		})
		protoModules = append(protoModules, data.Proto)

		// gRPC module
		if transport == serviceconfig.GRPC || transport == serviceconfig.GRPCRest {
			grpcDir := filepath.Join(libraryDir, names.grpc)
			isGrpcMissing, err := isPomMissing(grpcDir)
			if err != nil {
				return nil, err
			}
			modules = append(modules, javaModule{
				artifactID:   names.grpc,
				dir:          grpcDir,
				isMissing:    isGrpcMissing,
				templateData: data,
				template:     grpcPomTemplateName,
			})
			grpcModules = append(grpcModules, data.Grpc)
		}
	}

	// Client module
	clientDir := filepath.Join(libraryDir, gapicArtifactID)
	isClientMissing, err := isPomMissing(clientDir)
	if err != nil {
		return nil, err
	}
	clientCoord := coordinates{GroupID: gapicGroupID, ArtifactID: gapicArtifactID, Version: library.Version}
	modules = append(modules, javaModule{
		artifactID: gapicArtifactID,
		dir:        clientDir,
		isMissing:  isClientMissing,
		templateData: clientPomData{
			Client:       clientCoord,
			Version:      library.Version,
			Name:         metadata.NamePretty,
			Description:  metadata.APIDescription,
			Parent:       coordinates{GroupID: gapicGroupID, ArtifactID: fmt.Sprintf("%s-parent", gapicArtifactID), Version: library.Version},
			ProtoModules: protoModules,
			GrpcModules:  grpcModules,
		},
		template: clientPomTemplateName,
	})

	allModules := []coordinates{clientCoord}
	allModules = append(allModules, grpcModules...)
	allModules = append(allModules, protoModules...)

	// BOM module
	bomArtifactID := fmt.Sprintf("%s-bom", gapicArtifactID)
	bomDir := filepath.Join(libraryDir, bomArtifactID)
	isBomMissing, err := isPomMissing(bomDir)
	if err != nil {
		return nil, err
	}
	modules = append(modules, javaModule{
		artifactID: bomArtifactID,
		dir:        bomDir,
		isMissing:  isBomMissing,
		templateData: bomParentPomData{
			MainModule:      clientCoord,
			Name:            metadata.NamePretty,
			MonorepoVersion: monorepoVersion,
			Modules:         allModules,
		},
		template: bomPomTemplateName,
	})

	// Parent module
	parentDir := libraryDir
	isParentMissing, err := isPomMissing(parentDir)
	if err != nil {
		return nil, err
	}
	modules = append(modules, javaModule{
		artifactID: fmt.Sprintf("%s-parent", gapicArtifactID),
		dir:        parentDir,
		isMissing:  isParentMissing,
		templateData: bomParentPomData{
			MainModule:      clientCoord,
			Name:            metadata.NamePretty,
			MonorepoVersion: monorepoVersion,
			Modules:         allModules,
		},
		template: parentPomTemplateName,
	})

	return modules, nil
}

func isPomMissing(dir string) (bool, error) {
	pomPath := filepath.Join(dir, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		return false, nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false, fmt.Errorf("target directory %s does not exist: %w", dir, err)
	}
	return true, nil
}

func writePom(pomPath, templateName string, data any) (err error) {
	f, err := os.Create(pomPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", pomPath, err)
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	if terr := templates.ExecuteTemplate(f, templateName, data); terr != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, terr)
	}
	return nil
}

// updateVersionsFile adds entries for new modules to versions.txt.
// The format is: module-name:released-version:snapshot-version
// For new modules, released-version is 0.0.0 and snapshot-version is library.Version-SNAPSHOT.
func updateVersionsFile(libraryDir string, newModules []string, version string) error {
	// Find the repository root (parent of library directory)
	repoRoot := filepath.Dir(libraryDir)
	versionsPath := filepath.Join(repoRoot, "versions.txt")
	
	// Read existing content if file exists
	existingContent, err := os.ReadFile(versionsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read versions.txt: %w", err)
	}
	
	// Parse existing entries to avoid duplicates
	existingModules := make(map[string]bool)
	if len(existingContent) > 0 {
		lines := strings.Split(string(existingContent), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Split(line, ":")
			if len(parts) >= 1 {
				existingModules[parts[0]] = true
			}
		}
	}
	
	// Prepare new entries
	var newEntries []string
	snapshotVersion := version + "-SNAPSHOT"
	for _, module := range newModules {
		if !existingModules[module] {
			// Format: module:released-version:snapshot-version
			// For new modules, released version is 0.0.0
			entry := fmt.Sprintf("%s:0.0.0:%s", module, snapshotVersion)
			newEntries = append(newEntries, entry)
		}
	}
	
	if len(newEntries) == 0 {
		return nil // No new entries to add
	}
	
	// Append new entries to the file
	f, err := os.OpenFile(versionsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open versions.txt: %w", err)
	}
	defer f.Close()
	
	// Add a newline before new entries if file doesn't end with one
	if len(existingContent) > 0 && !strings.HasSuffix(string(existingContent), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("write newline to versions.txt: %w", err)
		}
	}
	
	for _, entry := range newEntries {
		if _, err := f.WriteString(entry + "\n"); err != nil {
			return fmt.Errorf("write entry to versions.txt: %w", err)
		}
	}
	
	return nil
}
