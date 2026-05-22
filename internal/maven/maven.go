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
	"errors"
	"fmt"
	"os"
)

var (
	// ErrReadPOM indicates a failure to read the pom.xml file.
	ErrReadPOM = errors.New("failed to read pom.xml")

	// ErrParsePOM indicates a failure to parse the pom.xml file XML syntax.
	ErrParsePOM = errors.New("failed to parse pom.xml")

	// ErrInvalidPOM indicates that the pom.xml is missing required metadata fields.
	ErrInvalidPOM = errors.New("invalid pom.xml metadata")
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

// ParsePOM extracts the Maven metadata from the specified pom.xml path.
func ParsePOM(pomPath string) (*POMProject, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrReadPOM, pomPath, err)
	}
	var proj POMProject
	if err := xml.Unmarshal(data, &proj); err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrParsePOM, pomPath, err)
	}
	if proj.Version == "" {
		proj.Version = proj.Parent.Version
	}
	if proj.ArtifactID == "" || proj.Version == "" {
		return nil, fmt.Errorf("%w %q: missing artifactId or version", ErrInvalidPOM, pomPath)
	}
	return &proj, nil
}
