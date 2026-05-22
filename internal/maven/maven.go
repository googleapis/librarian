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

// Package maven provides utility functions for unmarshalling and parsing Maven pom.xml files.
package maven

import (
	"encoding/xml"
	"fmt"
	"os"
)

// POMProject represents the target Maven metadata structured from pom.xml.
type POMProject struct {
	XMLName    xml.Name `xml:"project"`
	ArtifactID string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Parent     struct {
		Version string `xml:"version"`
	} `xml:"parent"`
}

// ParsePOM extracts the artifactId and version from the specified pom.xml path.
func ParsePOM(pomPath string) (string, string, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read pom.xml %q: %w", pomPath, err)
	}
	var proj POMProject
	if err := xml.Unmarshal(data, &proj); err != nil {
		return "", "", fmt.Errorf("failed to parse pom.xml %q: %w", pomPath, err)
	}
	version := proj.Version
	if version == "" {
		version = proj.Parent.Version
	}
	if proj.ArtifactID == "" || version == "" {
		return "", "", fmt.Errorf("missing artifactId or version in pom.xml %q", pomPath)
	}
	return proj.ArtifactID, version, nil
}
