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
	"log"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// knownPrefixes contains API path prefixes to be stripped when deriving a
// library name. The order matters: more specific prefixes must come before
// less specific ones (e.g., "google/cloud/" before "google/").
var knownPrefixes = []string{
	"google/cloud/",
	"google/api/",
	"google/devtools/",
	"google/",
}

const (
	defaultVersion         = "0.1.0-SNAPSHOT"
	defaultReleasedVersion = "0.0.0"
	fakeGroupID            = "please-configure-java-group-id"
)

// Add initializes a new Java library with default values.
func Add(lib *config.Library, googleapisDir string) *config.Library {
	lib.Version = defaultVersion
	// Java generation defaults to the system year for license headers,
	// so we reset it here to avoid redundancy in librarian.yaml.
	lib.CopyrightYear = ""
	if lib.Java == nil {
		lib.Java = &config.JavaModule{}
	}
	lib.Java.ReleasedVersion = defaultReleasedVersion
	// We use the first API to infer the group ID.
	// It is unrealistic for a single library to mix cloud and non-cloud APIs.
	apiPath := lib.APIs[0].Path
	switch {
	case strings.HasPrefix(apiPath, "google/shopping/"):
		return setNonCloudMavenDefaults(lib, "com.google.shopping")
	case strings.HasPrefix(apiPath, "google/maps/"):
		return setNonCloudMavenDefaults(lib, "com.google.maps")
	case strings.HasPrefix(apiPath, "google/ads/"):
		return setNonCloudMavenDefaults(lib, "com.google.api-ads")
	}
	if strings.HasPrefix(apiPath, "google/cloud/") {
		return lib
	}
	if googleapisDir != "" {
		if apiCfg, err := serviceconfig.Find(googleapisDir, apiPath, config.LanguageJava); err == nil && apiCfg.Organization == "CLOUD" {
			return lib
		}
	}
	log.Printf(
		"WARNING: unrecognized non-cloud API path %q. Setting fake GroupID %q. "+
			"Please manually configure java.group_id and java.distribution_name_override in librarian.yaml.",
		apiPath, fakeGroupID,
	)
	setNonCloudMavenDefaults(lib, fakeGroupID)
	return lib
}

func setNonCloudMavenDefaults(lib *config.Library, groupID string) *config.Library {
	lib.Java.ArtifactID = "google-" + lib.Name
	lib.Java.GroupID = groupID
	return lib
}

// DefaultLibraryName derives a default library name from an API path by stripping
// known prefixes (e.g., "google/cloud/", "google/api/") and returning all
// segments except the last one, joined by dashes.
func DefaultLibraryName(api string) string {
	path := api
	if idx := strings.LastIndex(api, "/"); idx != -1 {
		path = api[:idx]
	}
	for _, p := range knownPrefixes {
		if strings.HasPrefix(path, p) {
			path = strings.TrimPrefix(path, p)
			break
		}
	}
	return strings.ReplaceAll(path, "/", "-")
}
