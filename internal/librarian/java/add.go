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
	"log"
	"os"
	"slices"
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
	versionsFileName       = "versions.txt"
)

// Add initializes a new Java library with default values, or extends an
// existing library with a new API path, and registers the appropriate
// modules in versions.txt.
func Add(lib *config.Library, addedAPI *config.API) (*config.Library, error) {
	if lib.Version == "" {
		lib.Version = defaultVersion
	}
	// Java generation defaults to the system year for license headers,
	// so we reset it here to avoid redundancy in librarian.yaml.
	lib.CopyrightYear = ""

	if lib.Java == nil {
		lib.Java = &config.JavaModule{}
	}
	if lib.Java.ReleasedVersion == "" && addedAPI == nil {
		lib.Java.ReleasedVersion = defaultReleasedVersion
	}

	// We use the first API to infer the group ID.
	// It is unrealistic for a single library to mix cloud and non-cloud APIs.
	apiPath := lib.APIs[0].Path
	switch {
	case strings.HasPrefix(apiPath, "google/shopping/"):
		lib = setNonCloudMavenDefaults(lib, "com.google.shopping")
	case strings.HasPrefix(apiPath, "google/maps/"):
		lib = setNonCloudMavenDefaults(lib, "com.google.maps")
	case strings.HasPrefix(apiPath, "google/ads/"):
		lib = setNonCloudMavenDefaults(lib, "com.google.api-ads")
	default:
		if !strings.HasPrefix(apiPath, "google/cloud/") {
			log.Printf(
				"WARNING: unrecognized non-cloud API path %q. Setting fake GroupID %q. "+
					"Please manually configure java.group_id and java.distribution_name_override in librarian.yaml.",
				apiPath, fakeGroupID,
			)
			lib = setNonCloudMavenDefaults(lib, fakeGroupID)
		}
	}

	// Fill the library entry in order to derive the appropriate artifacts to add.
	lib, err := Fill(lib)
	if err != nil {
		return nil, err
	}

	newArtifactIDs, err := deriveAddedArtifactIDs(lib, addedAPI)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, id := range newArtifactIDs {
		versions = append(versions, fmt.Sprintf("%s:%s:%s", id, lib.Java.ReleasedVersion, lib.Version))
	}
	if err := appendVersions(versions); err != nil {
		return nil, err
	}

	lib, err = Tidy(lib)
	if err != nil {
		return nil, err
	}

	return lib, nil
}

func deriveAddedArtifactIDs(lib *config.Library, addedAPI *config.API) ([]string, error) {
	libCoord := deriveLibraryCoordinates(lib)
	var modules []expectedModule

	if addedAPI != nil {
		transport, err := serviceconfig.FindTransport(addedAPI.Path, config.LanguageJava)
		if err != nil {
			return nil, err
		}
		modules = expectedAPIModules(addedAPI, libCoord, transport)
	} else {
		var err error
		modules, err = expectedNewLibraryModules(lib)
		if err != nil {
			return nil, err
		}
	}

	var artifacts []string
	for _, m := range modules {
		artifacts = append(artifacts, m.ArtifactID)
	}

	if lib.Java != nil && len(lib.Java.ExcludedPOMs) > 0 {
		var filtered []string
		for _, art := range artifacts {
			if !slices.Contains(lib.Java.ExcludedPOMs, art) {
				filtered = append(filtered, art)
			}
		}
		artifacts = filtered
	}

	return artifacts, nil
}

// expectedNewLibraryModules returns all expected modules for a new library,
// ordered to match the versions.txt expectation: Parent, BOM, APIs, Client.
func expectedNewLibraryModules(lib *config.Library) ([]expectedModule, error) {
	transports, err := loadTransports(lib)
	if err != nil {
		return nil, err
	}
	modules := expectedModules(lib, transports)

	// Reorder modules to match versions.txt expectation: Parent, BOM, APIs, Client.
	var parent *expectedModule
	var bom *expectedModule
	var client *expectedModule
	var apis []expectedModule

	for _, m := range modules {
		switch m.Kind {
		case kindParent:
			parent = &m
		case kindBOM:
			bom = &m
		case kindClient:
			client = &m
		default: // kindProto, kindGRPC
			apis = append(apis, m)
		}
	}

	var ordered []expectedModule
	if parent != nil {
		ordered = append(ordered, *parent)
	}
	if bom != nil {
		ordered = append(ordered, *bom)
	}
	ordered = append(ordered, apis...)
	if client != nil {
		ordered = append(ordered, *client)
	}
	return ordered, nil
}

// readExistingModules reads and parses the versions file at the given path,
// returning a map of existing module artifact IDs to true.
func readExistingModules(path string) (map[string]bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	modules := make(map[string]bool)
	lines := bytes.SplitSeq(content, []byte("\n"))
	for line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || bytes.HasPrefix(line, []byte("#")) {
			continue
		}
		parts := bytes.Split(line, []byte(":"))
		if len(parts) > 0 && len(parts[0]) > 0 {
			modules[string(parts[0])] = true
		}
	}
	return modules, nil
}

func appendVersions(versions []string) error {
	existing, err := readExistingModules(versionsFileName)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", versionsFileName, err)
	}
	var newVersions []string
	for _, line := range versions {
		parts := strings.Split(line, ":")
		if len(parts) > 0 && !existing[parts[0]] {
			newVersions = append(newVersions, line)
		}
	}
	if err := appendLines(versionsFileName, newVersions); err != nil {
		return fmt.Errorf("failed to update %s: %w", versionsFileName, err)
	}
	return nil
}

// appendLines appends the given lines to an existing file, ensuring that it
// ends with a newline character before appending. It returns an error if the
// file does not exist.
func appendLines(path string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.Write(existing)
	// Ensure the file ends with a newline before appending so that we
	// do not concatenate lines instead of appending them.
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		buf.WriteByte('\n')
	}
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
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
		if after, ok := strings.CutPrefix(path, p); ok {
			path = after
			break
		}
	}
	return strings.ReplaceAll(path, "/", "-")
}
