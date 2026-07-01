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

	"github.com/googleapis/librarian/internal/config"
)

const (
	googleGroupID   = "com.google"
	protoGRPCSuffix = ".api.grpc"
	gRPCPrefix      = "grpc-"
	protoPrefix     = "proto-"
)

var groupInclusions = map[string]bool{
	"com.google.cloud":     true,
	"com.google.analytics": true,
	"com.google.area120":   true,
}

type coordinate struct {
	// GroupID is the Maven Group ID.
	GroupID string
	// ArtifactID is the Maven Artifact ID.
	ArtifactID string
	// Version is the Maven version.
	Version string
}

// libraryCoordinate contains Maven coordinates for the library modules (GAPIC,
// parent, and BOM).
type libraryCoordinate struct {
	// GAPIC is the Maven coordinate for the GAPIC module.
	GAPIC coordinate
	// Parent is the Maven coordinate for the parent module.
	Parent coordinate
	// BOM is the Maven coordinate for the BOM module.
	BOM coordinate
}

// apiCoordinate contains Maven coordinates for the library and its API-specific
// modules (proto and gRPC).
type apiCoordinate struct {
	libraryCoordinate
	// Proto is the Maven coordinate for the proto module.
	Proto coordinate
	// GRPC is the Maven coordinate for the gRPC module.
	GRPC coordinate
}

// distributionName returns the Maven distribution name (GroupID:ArtifactID)
// for the library.
func distributionName(library *config.Library) string {
	return fmt.Sprintf("%s:%s", library.Java.GroupID, library.Java.ArtifactID)
}

// deriveLibraryCoordinates calculates the Maven coordinates for the GAPIC library,
// its parent, and its BOM based on the library's configuration.
func deriveLibraryCoordinates(library *config.Library) libraryCoordinate {
	gapic := coordinate{
		GroupID:    library.Java.GroupID,
		ArtifactID: library.Java.ArtifactID,
		Version:    library.Version,
	}
	return libraryCoordinate{
		GAPIC: gapic,
		Parent: coordinate{
			GroupID:    gapic.GroupID,
			ArtifactID: fmt.Sprintf("%s-parent", gapic.ArtifactID),
			Version:    gapic.Version,
		},
		BOM: coordinate{
			GroupID:    gapic.GroupID,
			ArtifactID: fmt.Sprintf("%s-bom", gapic.ArtifactID),
			Version:    gapic.Version,
		},
	}
}

// deriveAPICoordinates returns the Maven coordinates for the proto and gRPC
// artifacts associated with a specific API version.
func deriveAPICoordinates(lc libraryCoordinate, version string, javaAPI *config.JavaAPI) apiCoordinate {
	if javaAPI.GAPICArtifactIDOverride != "" {
		lc.GAPIC.ArtifactID = javaAPI.GAPICArtifactIDOverride
	}
	protoGRPCGroupID := protoGroupID(lc.GAPIC.GroupID)
	protoArtifactID := javaAPI.ProtoArtifactIDOverride
	if protoArtifactID == "" {
		protoArtifactID = fmt.Sprintf("%s%s-%s", protoPrefix, lc.GAPIC.ArtifactID, version)
	}
	res := apiCoordinate{
		libraryCoordinate: lc,
		Proto: coordinate{
			GroupID:    protoGRPCGroupID,
			ArtifactID: protoArtifactID,
			Version:    lc.GAPIC.Version,
		},
	}
	grpcArtifactID := javaAPI.GRPCArtifactIDOverride
	if grpcArtifactID == "" {
		grpcArtifactID = fmt.Sprintf("%s%s-%s", gRPCPrefix, lc.GAPIC.ArtifactID, version)
	}
	res.GRPC = coordinate{
		GroupID:    protoGRPCGroupID,
		ArtifactID: grpcArtifactID,
		Version:    lc.GAPIC.Version,
	}
	return res
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
