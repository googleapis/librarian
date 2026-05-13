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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

const (
	protoPOMTemplateName  = "module_proto_pom.xml.tmpl"
	gRPCPOMTemplateName   = "module_grpc_pom.xml.tmpl"
	clientPOMTemplateName = "module_client_pom.xml.tmpl"
	parentPOMTemplateName = "module_parent_pom.xml.tmpl"
	bomPOMTemplateName    = "module_bom_pom.xml.tmpl"
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

// grpcProtoPOMData holds the data for rendering POM templates.
type gRPCProtoPOMData struct {
	Proto          Coordinate
	GRPC           Coordinate
	Parent         Coordinate
	Version        string
	MainArtifactID string
}

// clientPOMData holds the data for rendering the client library POM template.
type clientPOMData struct {
	Client       Coordinate
	Version      string
	Name         string
	Description  string
	Parent       Coordinate
	ProtoModules []Coordinate
	GRPCModules  []Coordinate
}

// bomParentPOMData holds the data for rendering the BOM and Parent library POM template.
type bomParentPOMData struct {
	MainModule      Coordinate
	Name            string
	MonorepoVersion string
	Modules         []Coordinate
}

// javaModule represents a Maven module and its POM generation state.
type javaModule struct {
	artifactID   string
	dir          string
	isMissing    bool
	templateData any
	template     string
}

type moduleKind int

const (
	kindProto moduleKind = iota
	kindGRPC
	kindClient
	kindBOM
	kindParent
)

type expectedModule struct {
	ArtifactID string
	Dir        string
	Kind       moduleKind
	IsMissing  bool
	Coordinate Coordinate
	APICoords  *APICoordinate
}

func loadTransports(library *config.Library, googleapisDir string) (map[string]serviceconfig.Transport, error) {
	transports := make(map[string]serviceconfig.Transport)
	for _, api := range library.APIs {
		apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageJava)
		if err != nil {
			return nil, fmt.Errorf("failed to find api config for %s: %w", api.Path, err)
		}
		transports[api.Path] = apiCfg.Transport(config.LanguageJava)
	}
	return transports, nil
}

func discoverModules(library *config.Library, libraryDir string, transports map[string]serviceconfig.Transport) ([]expectedModule, error) {
	if library.Java != nil && library.Java.SkipPOMUpdates {
		return nil, nil
	}
	var modules []expectedModule
	libCoord := DeriveLibraryCoordinates(library)
	for _, api := range library.APIs {
		apiBase := deriveAPIBase(library, api.Path)
		javaAPI := ResolveJavaAPI(library, api)
		apiCoord := DeriveAPICoordinates(libCoord, apiBase, javaAPI)
		transport := transports[api.Path]
		// Proto module
		protoDir := filepath.Join(libraryDir, apiCoord.Proto.ArtifactID)
		isProtoMissing, err := isPOMMissing(protoDir)
		if err != nil {
			return nil, err
		}
		modules = append(modules, expectedModule{
			ArtifactID: apiCoord.Proto.ArtifactID,
			Dir:        protoDir,
			Kind:       kindProto,
			IsMissing:  isProtoMissing,
			Coordinate: apiCoord.Proto,
			APICoords:  &apiCoord,
		})
		// gRPC module
		if transport != serviceconfig.Rest {
			gRPCDir := filepath.Join(libraryDir, apiCoord.GRPC.ArtifactID)
			isGRPCMissing, err := isPOMMissing(gRPCDir)
			if err != nil {
				return nil, err
			}
			modules = append(modules, expectedModule{
				ArtifactID: apiCoord.GRPC.ArtifactID,
				Dir:        gRPCDir,
				Kind:       kindGRPC,
				IsMissing:  isGRPCMissing,
				Coordinate: apiCoord.GRPC,
				APICoords:  &apiCoord,
			})
		}
	}
	// Client module
	clientDir := filepath.Join(libraryDir, libCoord.GAPIC.ArtifactID)
	isClientMissing, err := isPOMMissing(clientDir)
	if err != nil {
		return nil, err
	}
	modules = append(modules, expectedModule{
		ArtifactID: libCoord.GAPIC.ArtifactID,
		Dir:        clientDir,
		Kind:       kindClient,
		IsMissing:  isClientMissing,
		Coordinate: libCoord.GAPIC,
	})
	// BOM module
	bomDir := filepath.Join(libraryDir, libCoord.BOM.ArtifactID)
	isBOMMissing, err := isPOMMissing(bomDir)
	if err != nil {
		return nil, err
	}
	modules = append(modules, expectedModule{
		ArtifactID: libCoord.BOM.ArtifactID,
		Dir:        bomDir,
		Kind:       kindBOM,
		IsMissing:  isBOMMissing,
		Coordinate: libCoord.BOM,
	})
	// Parent module
	parentDir := libraryDir
	isParentMissing, err := isPOMMissing(parentDir)
	if err != nil {
		return nil, err
	}
	modules = append(modules, expectedModule{
		ArtifactID: libCoord.Parent.ArtifactID,
		Dir:        parentDir,
		Kind:       kindParent,
		IsMissing:  isParentMissing,
		Coordinate: libCoord.Parent,
	})
	return modules, nil
}

// syncPOMs generates missing POMs and surgically updates existing client, BOM,
// and parent POMs when new proto or gRPC modules are added. It returns a list
// of newly created artifact version entries to be added to versions.txt.
// TODO(https://github.com/googleapis/librarian/issues/5529): remove returning version entries.
func syncPOMs(library *config.Library, libraryDir, monorepoVersion string, metadata *repoMetadata, transports map[string]serviceconfig.Transport) error {
	modules, err := collectModules(library, libraryDir, monorepoVersion, metadata, transports)
	if err != nil {
		return err
	}

	var anyMissingProtoGRPC bool
	for _, m := range modules {
		if m.isMissing && (m.template == protoPOMTemplateName || m.template == gRPCPOMTemplateName) {
			anyMissingProtoGRPC = true
			break
		}
	}

	for _, m := range modules {
		pomPath := filepath.Join(m.dir, "pom.xml")
		if m.isMissing {
			if err := writePOM(pomPath, m.template, m.templateData); err != nil {
				return fmt.Errorf("failed to generate pom.xml for %s: %w", m.artifactID, err)
			}
			continue
		}

		if !anyMissingProtoGRPC {
			continue
		}
		switch m.template {
		case clientPOMTemplateName:
			if err := updateClientPOM(pomPath, m.templateData.(clientPOMData)); err != nil {
				return fmt.Errorf("failed to update client pom.xml %s: %w", m.artifactID, err)
			}
		case bomPOMTemplateName:
			if err := updateBOMPOM(pomPath, m.templateData.(bomParentPOMData)); err != nil {
				return fmt.Errorf("failed to update BOM pom.xml %s: %w", m.artifactID, err)
			}
		case parentPOMTemplateName:
			if err := updateParentPOM(pomPath, m.templateData.(bomParentPOMData)); err != nil {
				return fmt.Errorf("failed to update parent pom.xml %s: %w", m.artifactID, err)
			}
		}
	}
	return nil
}

// IdentifyMissingModules identifies all expected proto-*, grpc-*, client, BOM and Parent modules
// for the given library based on its configuration and checks for pom.xml presence
// on the filesystem. It returns a list of artifact IDs for the missing modules.
func IdentifyMissingModules(library *config.Library, libraryDir string, srcs *sources.Sources) ([]string, error) {
	srcCfg := sources.NewSourceConfig(srcs, library.Roots)
	primaryDir := srcCfg.Root(srcCfg.ActiveRoots[0])
	transports, err := loadTransports(library, primaryDir)
	if err != nil {
		return nil, err
	}
	expectedModules, err := discoverModules(library, libraryDir, transports)
	if err != nil {
		return nil, err
	}
	var missingModules []string
	for _, m := range expectedModules {
		if m.IsMissing {
			missingModules = append(missingModules, m.ArtifactID)
		}
	}
	return missingModules, nil
}

// updateClientPOM surgically updates the client POM using template markers
// to inject missing proto- and grpc- dependencies while preserving existing
// formatting and metadata comments.
func updateClientPOM(pomPath string, data clientPOMData) error {
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

// updateBOMPOM surgically updates the BOM POM using template markers to inject
// the dependencyManagement section while preserving existing formatting and
// metadata comments.
func updateBOMPOM(pomPath string, data bomParentPOMData) error {
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

// updateParentPOM surgically updates the Parent POM using template markers to inject
// the modules and dependencyManagement sections while preserving existing formatting
// and metadata comments.
func updateParentPOM(pomPath string, data bomParentPOMData) error {
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
	expectedModules, err := discoverModules(library, libraryDir, transports)
	if err != nil {
		return nil, err
	}
	libCoord := DeriveLibraryCoordinates(library)
	protoModules := make([]Coordinate, 0, len(library.APIs))
	gRPCModules := make([]Coordinate, 0, len(library.APIs))
	for _, m := range expectedModules {
		switch m.Kind {
		case kindProto:
			protoModules = append(protoModules, m.Coordinate)
		case kindGRPC:
			gRPCModules = append(gRPCModules, m.Coordinate)
		}
	}

	allModules := []Coordinate{libCoord.GAPIC}
	allModules = append(allModules, gRPCModules...)
	allModules = append(allModules, protoModules...)

	var modules []javaModule
	for _, m := range expectedModules {
		var templateData any
		var template string
		switch m.Kind {
		case kindProto, kindGRPC:
			templateData = gRPCProtoPOMData{
				Proto:          m.APICoords.Proto,
				GRPC:           m.APICoords.GRPC,
				Parent:         libCoord.Parent,
				MainArtifactID: libCoord.GAPIC.ArtifactID,
				Version:        library.Version,
			}
			if m.Kind == kindProto {
				template = protoPOMTemplateName
			} else {
				template = gRPCPOMTemplateName
			}
		case kindClient:
			templateData = clientPOMData{
				Client:       libCoord.GAPIC,
				Version:      library.Version,
				Name:         metadata.NamePretty,
				Description:  metadata.APIDescription,
				Parent:       libCoord.Parent,
				ProtoModules: protoModules,
				GRPCModules:  gRPCModules,
			}
			template = clientPOMTemplateName
		case kindBOM:
			templateData = bomParentPOMData{
				MainModule:      libCoord.GAPIC,
				Name:            metadata.NamePretty,
				MonorepoVersion: monorepoVersion,
				Modules:         allModules,
			}
			template = bomPOMTemplateName
		case kindParent:
			templateData = bomParentPOMData{
				MainModule:      libCoord.GAPIC,
				Name:            metadata.NamePretty,
				MonorepoVersion: monorepoVersion,
				Modules:         allModules,
			}
			template = parentPOMTemplateName
		}
		modules = append(modules, javaModule{
			artifactID:   m.ArtifactID,
			dir:          m.Dir,
			isMissing:    m.IsMissing,
			templateData: templateData,
			template:     template,
		})
	}
	return modules, nil
}

// isPOMMissing checks if a pom.xml exists in the given directory.
// It returns true if the file is confirmed to be missing (fs.ErrNotExist).
// It returns an error if the check fails for unexpected reasons (e.g., permission issues).
func isPOMMissing(dir string) (bool, error) {
	pomPath := filepath.Join(dir, "pom.xml")
	_, err := os.Stat(pomPath)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	return false, fmt.Errorf("failed to check %s: %w", pomPath, err)
}

func writePOM(pomPath, templateName string, data any) (err error) {
	if err := os.MkdirAll(filepath.Dir(pomPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", pomPath, err)
	}
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
