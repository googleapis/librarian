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
	protoPomTemplateName = "proto_pom.xml.tmpl"
	grpcPomTemplateName  = "grpc_pom.xml.tmpl"
	grcpProtoGroupID     = "com.google.api.grpc"
)

// grpcProtoPomData holds the data for rendering POM templates.
type grpcProtoPomData struct {
	GroupID          string
	ProtoArtifactID  string
	GrpcArtifactID   string
	Version          string
	MainArtifactID   string
	ParentGroupID    string
	ParentArtifactID string
	ParentVersion    string
	ProtoGroupID     string
}

// generatePomsIfMissing generates missing proto-* and grpc-* POMs.
func generatePomsIfMissing(library *config.Library, libraryDir, googleapisDir string) error {
	distName := deriveDistributionName(library)
	parts := strings.Split(distName, ":")
	gapicGroupID := parts[0]
	gapicArtifactID := parts[1]
	for _, api := range library.APIs {
		if err := creatLeafPomsIfMissing(library, api, gapicGroupID, gapicArtifactID, libraryDir, googleapisDir); err != nil {
			return fmt.Errorf("failed to generate leaf POMs for %s: %w", api.Path, err)
		}
	}
	return nil
}

func creatLeafPomsIfMissing(library *config.Library, api *config.API, gapicGroupID, gapicArtifactID, libraryDir, googleapisDir string) error {
	version := serviceconfig.ExtractVersion(api.Path)
	if version == "" {
		return fmt.Errorf("failed to extract version from API path %q", api.Path)
	}
	protoArtifactID := fmt.Sprintf("proto-%s-%s", gapicArtifactID, version)
	grpcArtifactID := fmt.Sprintf("grpc-%s-%s", gapicArtifactID, version)
	apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageJava)
	if err != nil {
		return fmt.Errorf("failed to find api config for %s: %w", api.Path, err)
	}
	transport := apiCfg.Transport(config.LanguageJava)
	data := grpcProtoPomData{
		GroupID:          grcpProtoGroupID,
		ProtoArtifactID:  protoArtifactID,
		GrpcArtifactID:   grpcArtifactID,
		ParentGroupID:    gapicGroupID,
		MainArtifactID:   gapicArtifactID,
		ParentArtifactID: fmt.Sprintf("%s-parent", gapicArtifactID),
		Version:          library.Version,
	}
	// Sync Proto POM
	protoDir := filepath.Join(libraryDir, protoArtifactID)
	if err := writePomIfMissing(protoDir, protoPomTemplateName, data); err != nil {
		return fmt.Errorf("failed to write proto POM: %w", err)
	}
	// Sync gRPC POM if needed
	if transport == serviceconfig.GRPC || transport == serviceconfig.GRPCRest {
		grpcDir := filepath.Join(libraryDir, grpcArtifactID)
		if err := writePomIfMissing(grpcDir, grpcPomTemplateName, data); err != nil {
			return fmt.Errorf("failed to write gRPC POM: %w", err)
		}
	}
	return nil
}

func writePomIfMissing(dir, templateName string, data any) error {
	pomPath := filepath.Join(dir, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		// File exists, skip.
		return nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("target directory %s does not exist: %w", dir, err)
	}
	if err := writePom(pomPath, templateName, data); err != nil {
		return fmt.Errorf("failed to write %s: %w", pomPath, err)
	}
	return nil
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
