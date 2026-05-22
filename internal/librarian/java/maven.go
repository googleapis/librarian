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
	"encoding/xml"
	"errors"
	"fmt"
	"os"
)

var (
	errReadPOM    = errors.New("failed to read pom.xml")
	errParsePOM   = errors.New("failed to parse pom.xml")
	errInvalidPOM = errors.New("invalid pom.xml metadata")
)

// pomProject represents the target Maven metadata structured from pom.xml.
type pomProject struct {
	XMLName    xml.Name `xml:"project"`
	ArtifactID string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Parent     struct {
		Version string `xml:"version"`
	} `xml:"parent"`
}

// parsePOM extracts the Maven metadata from the specified pom.xml path.
func parsePOM(pomPath string) (*pomProject, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", errReadPOM, pomPath, err)
	}
	var proj pomProject
	if err := xml.Unmarshal(data, &proj); err != nil {
		return nil, fmt.Errorf("%w %q: %w", errParsePOM, pomPath, err)
	}
	if proj.Version == "" {
		proj.Version = proj.Parent.Version
	}
	if proj.ArtifactID == "" || proj.Version == "" {
		return nil, fmt.Errorf("%w %q: missing artifactId or version", errInvalidPOM, pomPath)
	}
	return &proj, nil
}
