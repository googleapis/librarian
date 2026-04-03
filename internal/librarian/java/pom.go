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
	protoPomTemplateName    = "module_proto_pom.xml.tmpl"
	grpcPomTemplateName     = "module_grpc_pom.xml.tmpl"
	clientPomTemplateName   = "module_client_pom.xml.tmpl"
	parentPomTemplateName   = "module_parent_pom.xml.tmpl"
	bomPomTemplateName      = "module_bom_pom.xml.tmpl"
	googleGroupID           = "com.google"
	protoGrpcSuffix         = ".api.grpc"
	managedProtoStartMarker = "<!-- {x-generated-proto-dependencies-start} -->"
	managedProtoEndMarker   = "<!-- {x-generated-proto-dependencies-end} -->"
	managedGrpcStartMarker  = "<!-- {x-generated-grpc-dependencies-start} -->"
	managedGrpcEndMarker    = "<!-- {x-generated-grpc-dependencies-end} -->"
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

// syncPoms generates missing proto-*, grpc-*, and client POMs, and surgically updates
// existing client library POMs to include new dependencies.
func syncPoms(library *config.Library, libraryDir, googleapisDir, monorepoVersion string, metadata *repoMetadata) error {
	modules, err := collectModules(library, libraryDir, googleapisDir, monorepoVersion, metadata)
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
		if m.template == clientPomTemplateName {
			if err := updateClientPom(pomPath, m.templateData.(clientPomData)); err != nil {
				return fmt.Errorf("failed to update client pom %s: %w", m.artifactID, err)
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
	if updated, err = updateManagedBlock(updated, "managed_grpc_dependencies", managedGrpcStartMarker, managedGrpcEndMarker, data); err != nil {
		return err
	}
	if updated != string(content) {
		return os.WriteFile(pomPath, []byte(updated), 0644)
	}
	return nil
}

func updateManagedBlock(content, templateName, startMarker, endMarker string, data clientPomData) (string, error) {
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

func findMonorepoVersion(cfg *config.Config) (string, error) {
	for _, lib := range cfg.Libraries {
		if lib.Name == rootLibrary {
			return lib.Version, nil
		}
	}
	return "", fmt.Errorf("could not find monorepo version for %s in config", rootLibrary)
}

// protoGroupID returns the Maven Group ID for the generated proto and gRPC
// artifacts. It maps the GAPIC library's Group ID to a standard format and
// checks for special cases in groupInclusions (e.g., mapping
// "com.google.cloud" to "com.google.api.grpc").
func protoGroupID(mainArtifactGroupID string) string {
	prefix := mainArtifactGroupID
	if groupInclusions[mainArtifactGroupID] {
		prefix = googleGroupID
	}
	return prefix + protoGrpcSuffix
}
