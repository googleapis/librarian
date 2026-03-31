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
	protoPomTemplateName  = "proto_pom.xml.tmpl"
	grpcPomTemplateName   = "grpc_pom.xml.tmpl"
	clientPomTemplateName = "client_pom.xml.tmpl"
	grcpProtoGroupID      = "com.google.api.grpc"
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

// javaModule represents a Maven module and its POM generation state.
type javaModule struct {
	artifactID string
	dir        string
	isMissing  bool
	data       any
	template   string
}

// generatePomsIfMissing generates missing proto-*, grpc-*, and client POMs.
func generatePomsIfMissing(library *config.Library, libraryDir, googleapisDir string, metadata *repoMetadata) error {
	modules, err := collectModules(library, libraryDir, googleapisDir, metadata)
	if err != nil {
		return err
	}
	for _, m := range modules {
		if !m.isMissing {
			continue
		}
		if err := writePom(filepath.Join(m.dir, "pom.xml"), m.template, m.data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", m.artifactID, err)
		}
	}
	return nil
}

// collectModules identifies all expected proto-* and grpc-* modules
// for the given library based on its configuration and checks a pom.xml presence
// on the filesystem.
//
// All expected modules are collected (even if they exist) because the client
// module's POM requires a full list of all proto and gRPC dependencies
// to ensure its dependency list is fully synchronized.
func collectModules(library *config.Library, libraryDir, googleapisDir string, metadata *repoMetadata) ([]javaModule, error) {
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

		data := grpcProtoPomData{
			Proto: coordinates{
				GroupID:    grcpProtoGroupID,
				ArtifactID: names.proto,
			},
			Grpc: coordinates{
				GroupID:    grcpProtoGroupID,
				ArtifactID: names.grpc,
			},
			Parent: coordinates{
				GroupID:    gapicGroupID,
				ArtifactID: fmt.Sprintf("%s-parent", gapicArtifactID),
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
			artifactID: names.proto,
			dir:        protoDir,
			isMissing:  isProtoMissing,
			data:       data,
			template:   protoPomTemplateName,
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
				artifactID: names.grpc,
				dir:        grpcDir,
				isMissing:  isGrpcMissing,
				data:       data,
				template:   grpcPomTemplateName,
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
	modules = append(modules, javaModule{
		artifactID: gapicArtifactID,
		dir:        clientDir,
		isMissing:  isClientMissing,
		data: clientPomData{
			Client:       coordinates{GroupID: gapicGroupID, ArtifactID: gapicArtifactID},
			Version:      library.Version,
			Name:         metadata.NamePretty,
			Description:  metadata.APIDescription,
			Parent:       coordinates{GroupID: gapicGroupID, ArtifactID: fmt.Sprintf("%s-parent", gapicArtifactID)},
			ProtoModules: protoModules,
			GrpcModules:  grpcModules,
		},
		template: clientPomTemplateName,
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
