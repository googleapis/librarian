// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	gapic "google.golang.org/genproto/googleapis/gapic/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	gapicMetadataFile                   = "gapic_metadata.json"
	serviceVersionOptimizationThreshold = 5
)

var (
	apiVersionReleaseNotesTemplate = template.Must(template.New("apiVersionReleaseNotes").Parse(`### API Versions
{{- range .LibraryPackageAPIVersions }}

<details><summary>{{.LibraryPackage}}</summary>
{{ range .ServiceVersions }}
* {{ .Service }}: {{ .Version }}{{ end }}

</details>
{{ end }}`))

	// Common places that may contain a gapic_metadata.json file as test data
	// that should be ignored when preparing a release e.g. for client
	// generators.
	testdataRegex = regexp.MustCompile(`(?i)/(baselines|goldens|java-showcase|output|prototests|test|tests|testdata)/`)
)

// serviceVersion represents a pairing of an API service interface e.g. Protobuf
// service and API service interface version.
type serviceVersion struct {
	// Service is the name of the API service interface. This appears as the
	// key in [gapic.GapicMetadata] Services map. For example, "LibraryService".
	Service string

	// Version is the API service interface version identifier. This is sourced
	// from the [gapic.GapicMetadata_ServiceForTransport] ApiVersion property.
	// For example, "2025-09-14".
	Version string
}

// libraryPackageAPIVersions contains the API service interface version
// information for a generated library package. For example, the DotNet library
// package "Google.Example.Deltav.V1" has Services "SpaceService" and
// "SpaceShipService" with ApiVersion values therein of "2025-09-14" and
// "2025-04-04" respectively.
//
// There can be zero or more libraryPackageAPIVersions per
// [config.LibraryState]. They constructed by [extractAPIVersions] following a
// [readGapicMetadata] call.
type libraryPackageAPIVersions struct {
	// LibraryPackage is the language-specific, generated library package name
	// as identified by the [gapic.GapicMetadata] LibraryPackage property. For
	// example, a DotNet library package could be "Google.Example.Deltav.V1".
	LibraryPackage string

	// ServiceVersions is the pairing of API service interface name to API
	// version identifier found in the [gapic.GapicMetadata] Services mapping.
	ServiceVersions []serviceVersion

	// Versions is the set of unique API version identifiers that appear in the
	// [gapic.GapicMetadata] Services mapping.
	Versions map[string]bool
}

// formatAPIVersionReleaseNotes accepts the library's per-package API version
// information, and produces a change log/release note ready markdown
// "API Versions" section. It does leverage HTML for some formatting, so its
// output should be coerced to a [html/template.HTML] before being used in
// another [html/template.Template] as input data.
func formatAPIVersionReleaseNotes(lpv []*libraryPackageAPIVersions) (string, error) {
	if len(lpv) == 0 {
		return "", nil
	}

	// Optimization for homogenous API version used across service interfaces.
	// Only triggers if there are more than 5 service interfaces in the API.
	// If there are fewer, there is still value in listing them individually.
	for _, v := range lpv {
		if len(v.Versions) != 1 {
			continue
		}
		if len(v.ServiceVersions) < serviceVersionOptimizationThreshold {
			continue
		}

		v.ServiceVersions = []serviceVersion{{Service: "All", Version: v.ServiceVersions[0].Version}}
	}

	var out bytes.Buffer
	if err := apiVersionReleaseNotesTemplate.Execute(&out, struct {
		LibraryPackageAPIVersions []*libraryPackageAPIVersions
	}{lpv}); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return out.String(), nil
}

// extractAPIVersions constructs a set of per-library package entities from the
// given [gapic.GapicMetadata] documents loaded from [readGapicMetadata]. If the
// document contains an API version identifier associated with a Services entry
// therein, it will get a [libraryPackageAPIVersions] item.
func extractAPIVersions(metadataDocuments map[string]*gapic.GapicMetadata) []*libraryPackageAPIVersions {
	var result []*libraryPackageAPIVersions
	for _, md := range metadataDocuments {
		lpav := &libraryPackageAPIVersions{
			LibraryPackage:  md.GetLibraryPackage(),
			ServiceVersions: []serviceVersion{},
			Versions:        make(map[string]bool),
		}
		for serviceName, service := range md.GetServices() {
			if service.GetApiVersion() == "" {
				continue
			}
			lpav.ServiceVersions = append(lpav.ServiceVersions, serviceVersion{Service: serviceName, Version: service.GetApiVersion()})
			lpav.Versions[service.GetApiVersion()] = true
		}
		if len(lpav.Versions) == 0 {
			continue
		}
		slices.SortStableFunc(lpav.ServiceVersions, func(a, b serviceVersion) int {
			return strings.Compare(a.Service, b.Service)
		})
		result = append(result, lpav)
	}

	slices.SortStableFunc(result, func(a, b *libraryPackageAPIVersions) int {
		return strings.Compare(a.LibraryPackage, b.LibraryPackage)
	})

	return result
}

// readGapicMetadata traverses the [config.LibraryState] SourceRoots under the
// provided directory, parses any "gapic_metadata.json" file it finds, and
// stores it in a map keyed by the [gapic.GapicMetadata] LibraryPackage.
// There should be at most one "gapic_metadta.json" file per generated library
// package under SourceRoots, typically one per APIs entry, each with a unique
// library package name.
func readGapicMetadata(dir string, library *config.LibraryState) (map[string]*gapic.GapicMetadata, error) {
	mds := make(map[string]*gapic.GapicMetadata)
	for _, root := range library.SourceRoots {
		sr := filepath.Join(dir, root)
		err := filepath.WalkDir(sr, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if testdataRegex.MatchString(path) || filepath.Base(path) != gapicMetadataFile {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", path, err)
			}
			var metadata gapic.GapicMetadata
			if err := protojson.Unmarshal(content, &metadata); err != nil {
				return fmt.Errorf("failed to unmarshal %s: %w", path, err)
			}
			mds[metadata.LibraryPackage] = &metadata

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", root, err)
		}
	}
	return mds, nil
}
