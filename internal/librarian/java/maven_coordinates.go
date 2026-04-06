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
	protoGrpcSuffix = ".api.grpc"
	cloudPrefix     = "google-cloud-"
	grpcPrefix      = "grpc-"
	protoPrefix     = "proto-"
)

var groupInclusions = map[string]bool{
	"com.google.cloud":     true,
	"com.google.analytics": true,
	"com.google.area120":   true,
}

type coordinates struct {
	GroupID    string
	ArtifactID string
	Version    string
}

type libCoords struct {
	gapic  coordinates
	parent coordinates
	bom    coordinates
}

type apiCoords struct {
	libCoords
	proto coordinates
	grpc  coordinates
}

func deriveLibCoords(library *config.Library) libCoords {
	distName := deriveDistributionName(library)
	parts := strings.SplitN(distName, ":", 2)
	groupID := parts[0]
	artifactID := groupID
	if len(parts) == 2 {
		artifactID = parts[1]
	}
	gapic := coordinates{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    library.Version,
	}
	return libCoords{
		gapic: gapic,
		parent: coordinates{
			GroupID:    gapic.GroupID,
			ArtifactID: fmt.Sprintf("%s-parent", gapic.ArtifactID),
			Version:    gapic.Version,
		},
		bom: coordinates{
			GroupID:    gapic.GroupID,
			ArtifactID: fmt.Sprintf("%s-bom", gapic.ArtifactID),
			Version:    gapic.Version,
		},
	}
}

func deriveAPICoords(lc libCoords, version string) apiCoords {
	protoGrpcGroupID := protoGroupID(lc.gapic.GroupID)
	return apiCoords{
		libCoords: lc,
		proto: coordinates{
			GroupID:    protoGrpcGroupID,
			ArtifactID: fmt.Sprintf("%s%s-%s", protoPrefix, lc.gapic.ArtifactID, version),
			Version:    lc.gapic.Version,
		},
		grpc: coordinates{
			GroupID:    protoGrpcGroupID,
			ArtifactID: fmt.Sprintf("%s%s-%s", grpcPrefix, lc.gapic.ArtifactID, version),
			Version:    lc.gapic.Version,
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
	return prefix + protoGrpcSuffix
}

// ensureCloudPrefix returns name with the "google-cloud-" prefix,
// adding it if not already present.
func ensureCloudPrefix(name string) string {
	if !strings.HasPrefix(name, cloudPrefix) {
		return cloudPrefix + name
	}
	return name
}

func deriveDistributionName(library *config.Library) string {
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
