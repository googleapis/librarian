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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	protoPomTemplateName  = "module_proto_pom.xml.tmpl"
	gRPCPomTemplateName   = "module_grpc_pom.xml.tmpl"
	clientPomTemplateName = "module_client_pom.xml.tmpl"
	parentPomTemplateName = "module_parent_pom.xml.tmpl"
	bomPomTemplateName    = "module_bom_pom.xml.tmpl"
	// Template markers for client pom.xml.
	managedProtoStartMarker = "<!-- {x-generated-proto-dependencies-start} -->"
	managedProtoEndMarker   = "<!-- {x-generated-proto-dependencies-end} -->"
	managedGRPCStartMarker  = "<!-- {x-generated-grpc-dependencies-start} -->"
	managedGRPCEndMarker    = "<!-- {x-generated-grpc-dependencies-end} -->"
	// Template markers for BOM and parent pom.xml.
	managedDependenciesStartMarker = "<!-- {x-generated-dependencies-start} -->"
	managedDependenciesEndMarker   = "<!-- {x-generated-dependencies-end} -->"
	managedModulesStartMarker      = "<!-- {x-generated-modules-start} -->"
	managedModulesEndMarker        = "<!-- {x-generated-modules-end} -->"
)

// gRPCProtoPomData holds the data for rendering POM templates.
type gRPCProtoPomData struct {
	Proto          coordinate
	GRPC           coordinate
	Parent         coordinate
	Version        string
	MainArtifactID string
}

// clientPomData holds the data for rendering the client library POM template.
type clientPomData struct {
	Client       coordinate
	Version      string
	Name         string
	Description  string
	Parent       coordinate
	ProtoModules []coordinate
	GRPCModules  []coordinate
}

// bomParentPomData holds the data for rendering the BOM and Parent library POM template.
type bomParentPomData struct {
	MainModule      coordinate
	Name            string
	MonorepoVersion string
	Modules         []coordinate
}

// javaModule represents a Maven module and its POM generation state.
type javaModule struct {
	artifactID   string
	dir          string
	isMissing    bool
	templateData any
	template     string
}

// syncPoms generates missing proto-*, grpc-*, and client POMs, and surgically updates
// existing client library POMs to include new dependencies.
func syncPoms(library *config.Library, libraryDir, monorepoVersion string, metadata *repoMetadata, transports map[string]serviceconfig.Transport) error {
	modules, err := collectModules(library, libraryDir, monorepoVersion, metadata, transports)
	if err != nil {
		return err
	}
	for _, m := range modules {
		pomPath := filepath.Join(m.dir, "pom.xml")
		if m.isMissing {
			if err := writePom(pomPath, m.template, m.templateData); err != nil {
				return fmt.Errorf("failed to generate pom for %s: %w", m.artifactID, err)
			}
			continue
		}
		switch m.template {
		case clientPomTemplateName:
			if err := updateClientPom(pomPath, m.templateData.(clientPomData)); err != nil {
				return fmt.Errorf("failed to update client pom %s: %w", m.artifactID, err)
			}
		case bomPomTemplateName:
			if err := updateBomPom(pomPath, m.templateData.(bomParentPomData)); err != nil {
				return fmt.Errorf("failed to update bom pom %s: %w", m.artifactID, err)
			}
		case parentPomTemplateName:
			if err := updateParentPom(pomPath, m.templateData.(bomParentPomData)); err != nil {
				return fmt.Errorf("failed to update parent pom %s: %w", m.artifactID, err)
			}
		}
	}
	return nil
}

// updateClientPom surgicially updates the client POM using template markers
// to inject missing proto- and grpc- dependencies while preserving existing
// formatting and metadata comments.
func updateClientPom(pomPath string, data clientPomData) error {
	content, err := os.ReadFile(pomPath)
	if err != nil {
		return err
	}
	updated := string(content)
	if updated, err = updateManagedBlock(updated, "managed_proto_dependencies", managedProtoStartMarker, managedProtoEndMarker, data); err != nil {
		return err
	}
	if updated, err = updateManagedBlock(updated, "managed_grpc_dependencies", managedGRPCStartMarker, managedGRPCEndMarker, data); err != nil {
		return err
	}
	// compare to avoid unnecessary I/O
	if updated != string(content) {
		return os.WriteFile(pomPath, []byte(updated), 0644)
	}
	return nil
}

// updateBomPom surgically updates the BOM POM using template markers to inject
// the dependencyManagement section while preserving existing formatting and
// metadata comments.
func updateBomPom(pomPath string, data bomParentPomData) error {
	content, err := os.ReadFile(pomPath)
	if err != nil {
		return err
	}
	updated, err := updateManagedBlock(string(content), "managed_dependencies", managedDependenciesStartMarker, managedDependenciesEndMarker, data)
	if err != nil {
		return err
	}
	// compare to avoid unnecessary I/O
	if updated != string(content) {
		return os.WriteFile(pomPath, []byte(updated), 0644)
	}
	return nil
}

// updateParentPom surgically updates the Parent POM using template markers to inject
// the modules and dependencyManagement sections while preserving existing formatting
// and metadata comments.
func updateParentPom(pomPath string, data bomParentPomData) error {
	content, err := os.ReadFile(pomPath)
	if err != nil {
		return err
	}
	updated := string(content)
	if updated, err = updateManagedBlock(updated, "managed_modules", managedModulesStartMarker, managedModulesEndMarker, data); err != nil {
		return err
	}
	if updated, err = updateManagedBlock(updated, "managed_dependencies", managedDependenciesStartMarker, managedDependenciesEndMarker, data); err != nil {
		return err
	}
	// compare to avoid unnecessary I/O
	if updated != string(content) {
		return os.WriteFile(pomPath, []byte(updated), 0644)
	}
	return nil
}

func updateManagedBlock(content, templateName, startMarker, endMarker string, data any) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", err
	}
	return replaceBlock(content, startMarker, endMarker, buf.String())
}

// replaceBlock surgically replaces the content between startMarker and endMarker.
// It detects the indentation of the line where startMarker is placed and
// ensures the endMarker follows the same indentation.
func replaceBlock(content, startMarker, endMarker, newContent string) (string, error) {
	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return "", fmt.Errorf("missing start marker %q", startMarker)
	}
	endIdx := strings.Index(content, endMarker)
	if endIdx == -1 {
		return "", fmt.Errorf("found start marker %q but no end marker %q", startMarker, endMarker)
	}

	// Detect indentation of the start marker by looking at the content before
	// it on the same line.
	// TODO(https://github.com/googleapis/librarian/issues/5039):
	// Remove when formatter for pom.xml is used
	indent := detectIndentation(content, startIdx)

	// Calculate the content strictly between the markers. We preserve the
	// markers themselves and the indentation of the start marker is used for
	// the end marker as well.
	return content[:startIdx+len(startMarker)] + "\n" + strings.Trim(newContent, "\n") + "\n" + indent + content[endIdx:], nil
}

func detectIndentation(content string, index int) string {
	lineStart := strings.LastIndex(content[:index], "\n")
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++ // skip the newline
	}
	return content[lineStart:index]
}

// collectModules identifies all expected proto-*, grpc-*, client, BOM and Parent modules
// for the given library based on its configuration and checks a pom.xml presence
// on the filesystem.
//
// All expected modules are collected (even if they exist) because the client
// module's POM requires a full list of all proto and gRPC dependencies
// to ensure its dependency list is fully synchronized.
func collectModules(library *config.Library, libraryDir, monorepoVersion string, metadata *repoMetadata, transports map[string]serviceconfig.Transport) ([]javaModule, error) {
	var modules []javaModule
	libCoord := deriveLibCoord(library)

	protoModules := make([]coordinate, 0, len(library.APIs))
	gRPCModules := make([]coordinate, 0, len(library.APIs))
	for _, api := range library.APIs {
		version := serviceconfig.ExtractVersion(api.Path)
		if version == "" {
			return nil, fmt.Errorf("failed to extract version from API path %q", api.Path)
		}

		apiCoord := deriveAPICoord(libCoord, version)

		transport := transports[api.Path]
		data := gRPCProtoPomData{
			Proto:          apiCoord.proto,
			GRPC:           apiCoord.grpc,
			Parent:         libCoord.parent,
			MainArtifactID: libCoord.gapic.ArtifactID,
			Version:        library.Version,
		}

		// Proto module
		protoDir := filepath.Join(libraryDir, apiCoord.proto.ArtifactID)
		isProtoMissing, err := isPomMissing(protoDir)
		if err != nil {
			return nil, err
		}
		modules = append(modules, javaModule{
			artifactID:   apiCoord.proto.ArtifactID,
			dir:          protoDir,
			isMissing:    isProtoMissing,
			templateData: data,
			template:     protoPomTemplateName,
		})
		protoModules = append(protoModules, data.Proto)

		// gRPC module
		if transport == serviceconfig.GRPC || transport == serviceconfig.GRPCRest {
			gRPCDir := filepath.Join(libraryDir, apiCoord.grpc.ArtifactID)
			isGRPCMissing, err := isPomMissing(gRPCDir)
			if err != nil {
				return nil, err
			}
			modules = append(modules, javaModule{
				artifactID:   apiCoord.grpc.ArtifactID,
				dir:          gRPCDir,
				isMissing:    isGRPCMissing,
				templateData: data,
				template:     gRPCPomTemplateName,
			})
			gRPCModules = append(gRPCModules, data.GRPC)
		}
	}

	// Client module
	clientDir := filepath.Join(libraryDir, libCoord.gapic.ArtifactID)
	isClientMissing, err := isPomMissing(clientDir)
	if err != nil {
		return nil, err
	}
	modules = append(modules, javaModule{
		artifactID: libCoord.gapic.ArtifactID,
		dir:        clientDir,
		isMissing:  isClientMissing,
		templateData: clientPomData{
			Client:       libCoord.gapic,
			Version:      library.Version,
			Name:         metadata.NamePretty,
			Description:  metadata.APIDescription,
			Parent:       libCoord.parent,
			ProtoModules: protoModules,
			GRPCModules:  gRPCModules,
		},
		template: clientPomTemplateName,
	})

	allModules := []coordinate{libCoord.gapic}
	allModules = append(allModules, gRPCModules...)
	allModules = append(allModules, protoModules...)

	// BOM module
	bomDir := filepath.Join(libraryDir, libCoord.bom.ArtifactID)
	isBomMissing, err := isPomMissing(bomDir)
	if err != nil {
		return nil, err
	}
	modules = append(modules, javaModule{
		artifactID: libCoord.bom.ArtifactID,
		dir:        bomDir,
		isMissing:  isBomMissing,
		templateData: bomParentPomData{
			MainModule:      libCoord.gapic,
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
		artifactID: libCoord.parent.ArtifactID,
		dir:        parentDir,
		isMissing:  isParentMissing,
		templateData: bomParentPomData{
			MainModule:      libCoord.gapic,
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

func findMonorepoVersion(cfg *config.Config) (string, error) {
	for _, lib := range cfg.Libraries {
		if lib.Name == rootLibrary {
			return lib.Version, nil
		}
	}
	return "", errMonorepoVersion
}
