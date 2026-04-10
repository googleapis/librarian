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
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

const (
	googleGroupID   = "com.google"
	protoGRPCSuffix = ".api.grpc"
	cloudPrefix     = "google-cloud-"
	gRPCPrefix      = "grpc-"
	protoPrefix     = "proto-"
)

var groupInclusions = map[string]bool{
	"com.google.cloud":     true,
	"com.google.analytics": true,
	"com.google.area120":   true,
}

// TODO(https://github.com/googleapis/librarian/issues/5050):
// Exported selected functions and fields to use in migrate tool.
// Unexport after migrate is done.

// Coordinate represents a Maven Coordinate, uniquely identifies a project
// artifact using its GroupID, ArtifactID, and Version.
type Coordinate struct {
	// GroupID is the Maven Group ID.
	GroupID string
	// ArtifactID is the Maven Artifact ID.
	ArtifactID string
	// Version is the Maven version.
	Version string
}

// LibraryCoordinate contains Maven coordinates for the library modules (GAPIC,
// parent, and BOM).
type LibraryCoordinate struct {
	// GAPIC is the Maven coordinate for the GAPIC module.
	GAPIC Coordinate
	// Parent is the Maven coordinate for the parent module.
	Parent Coordinate
	// BOM is the Maven coordinate for the BOM module.
	BOM Coordinate
}

// APICoordinate contains Maven coordinates for the library and its API-specific
// modules (proto and gRPC).
type APICoordinate struct {
	LibraryCoordinate
	// Proto is the Maven coordinate for the proto module.
	Proto Coordinate
	// GRPC is the Maven coordinate for the gRPC module.
	GRPC Coordinate
}

// DeriveLibraryCoordinates calculates the Maven coordinates for the GAPIC library,
// its parent, and its BOM based on the library's configuration.
func DeriveLibraryCoordinates(library *config.Library) LibraryCoordinate {
	distName := DeriveDistributionName(library)
	parts := strings.SplitN(distName, ":", 2)
	groupID := parts[0]
	artifactID := groupID
	if len(parts) == 2 {
		artifactID = parts[1]
	}
	gapic := Coordinate{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    library.Version,
	}
	return LibraryCoordinate{
		GAPIC: gapic,
		Parent: Coordinate{
			GroupID:    gapic.GroupID,
			ArtifactID: fmt.Sprintf("%s-parent", gapic.ArtifactID),
			Version:    gapic.Version,
		},
		BOM: Coordinate{
			GroupID:    gapic.GroupID,
			ArtifactID: fmt.Sprintf("%s-bom", gapic.ArtifactID),
			Version:    gapic.Version,
		},
	}
}

// DeriveAPICoordinates returns the Maven coordinates for the proto and gRPC
// artifacts associated with a specific API version.
func DeriveAPICoordinates(lc LibraryCoordinate, version string, javaAPI *config.JavaAPI) APICoordinate {
	protoGRPCGroupID := protoGroupID(lc.GAPIC.GroupID)
	protoArtifactID := javaAPI.ProtoArtifactIDOverride
	if protoArtifactID == "" {
		protoArtifactID = fmt.Sprintf("%s%s-%s", protoPrefix, lc.GAPIC.ArtifactID, version)
	}
	grpcArtifactID := javaAPI.GRPCArtifactIDOverride
	if grpcArtifactID == "" {
		grpcArtifactID = fmt.Sprintf("%s%s-%s", gRPCPrefix, lc.GAPIC.ArtifactID, version)
	}
	return APICoordinate{
		LibraryCoordinate: lc,
		Proto: Coordinate{
			GroupID:    protoGRPCGroupID,
			ArtifactID: protoArtifactID,
			Version:    lc.GAPIC.Version,
		},
		GRPC: Coordinate{
			GroupID:    protoGRPCGroupID,
			ArtifactID: grpcArtifactID,
			Version:    lc.GAPIC.Version,
		},
	}
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
	return prefix + protoGRPCSuffix
}

// ensureCloudPrefix returns name with the "google-cloud-" prefix,
// adding it if not already present.
func ensureCloudPrefix(name string) string {
	if !strings.HasPrefix(name, cloudPrefix) {
		return cloudPrefix + name
	}
	return name
}

// DeriveDistributionName returns the Maven distribution name (GroupID:ArtifactID)
// for the library, applying overrides and defaults as necessary.
func DeriveDistributionName(library *config.Library) string {
	if library.Java != nil && library.Java.DistributionNameOverride != "" {
		return library.Java.DistributionNameOverride
	}
	groupID := "com.google.cloud"
	if library.Java != nil && library.Java.GroupID != "" {
		groupID = library.Java.GroupID
	}
	artifactID := ensureCloudPrefix(library.Name)
	return fmt.Sprintf("%s:%s", groupID, artifactID)
}
