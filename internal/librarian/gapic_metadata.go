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
)

type serviceVersion struct {
	Service string
	Version string
}

type libraryPackageAPIVersions struct {
	LibraryPackage  string
	ServiceVersions []serviceVersion
	Versions        map[string]bool
}

func formatAPIVersionReleaseNotes(lpv []*libraryPackageAPIVersions) (string, error) {
	if len(lpv) == 0 {
		return "", nil
	}

	// Optimization for homogenous API version used across service interfaces.
	// Only triggers if there are more than 5 service interfaces in the API.
	// If there are fewer, there is still value in listing them individually.
	for _, v := range lpv {
		// This optimization only applies if all services in a library package share the same version.
		if len(v.Versions) != 1 {
			continue
		}

		// There is only one key in v.VersionServices, so this loop runs once.
		for sharedVersion := range v.Versions {
			if len(v.ServiceVersions) >= serviceVersionOptimizationThreshold {
				v.ServiceVersions = []serviceVersion{{Service: "All", Version: sharedVersion}}
			}
		}
	}

	var out bytes.Buffer
	if err := apiVersionReleaseNotesTemplate.Execute(&out, struct {
		LibraryPackageAPIVersions []*libraryPackageAPIVersions
	}{lpv}); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return out.String(), nil
}

func extractAPIVersions(mds map[string]*gapic.GapicMetadata) []*libraryPackageAPIVersions {
	var result []*libraryPackageAPIVersions
	for _, md := range mds {
		lpav := &libraryPackageAPIVersions{
			LibraryPackage:  md.GetLibraryPackage(),
			ServiceVersions: []serviceVersion{},
			Versions:        make(map[string]bool),
		}
		for serviceName, s := range md.GetServices() {
			if s.GetApiVersion() == "" {
				continue
			}
			lpav.ServiceVersions = append(lpav.ServiceVersions, serviceVersion{Service: serviceName, Version: s.GetApiVersion()})
			lpav.Versions[s.GetApiVersion()] = true
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
			if filepath.Base(path) == gapicMetadataFile {
				content, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("failed to read %s: %w", path, err)
				}
				var metadata gapic.GapicMetadata
				if err := protojson.Unmarshal(content, &metadata); err != nil {
					return fmt.Errorf("failed to unmarshal %s: %w", path, err)
				}
				mds[metadata.LibraryPackage] = &metadata
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", root, err)
		}
	}
	return mds, nil
}
