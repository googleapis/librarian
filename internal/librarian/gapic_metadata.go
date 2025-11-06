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
	"maps"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	gapic "google.golang.org/genproto/googleapis/gapic/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	gapicMetadataFile = "gapic_metadata.json"
)

var (
	apiVersionReleaseNotesTemplate = template.Must(template.New("apiVersionReleaseNotes").Parse(`### API Versions
{{- range .LibraryPackageAPIVersions }}

<details><summary>{{.LibraryPackage}}</summary>
{{ range $service, $version := .ServiceVersion }}
* {{$service}}: {{$version}}{{ end }}

</details>
{{ end }}`))
)

type libraryPackageAPIVersions struct {
	LibraryPackage  string
	ServiceVersion  map[string]string
	VersionServices map[string][]string
}

func generateAPIVersionReleaseNotes(state *config.LibrarianState, repoDir string) (map[string]string, error) {
	notes := make(map[string]string)

	for _, library := range state.Libraries {
		if !library.ReleaseTriggered {
			continue
		}
		mds, err := readGapicMetadata(repoDir, library)
		if err != nil {
			return nil, fmt.Errorf("error reading gapic_metadata: %w", err)
		}

		v := extractAPIVersions(mds)
		if len(v) == 0 {
			continue
		}
		n, err := formatAPIVersionReleaseNotes(v)
		if err != nil {
			return nil, err
		}
		notes[library.ID] = n
	}

	return notes, nil
}

func formatAPIVersionReleaseNotes(lpv []*libraryPackageAPIVersions) (string, error) {
	if len(lpv) == 0 {
		return "", nil
	}

	// Optimization for homogenous API version used across service interfaces.
	// Only triggers if there are more than 5 service interfaces in the API.
	// If there are fewer, there is still value in listing them individually.
	for _, v := range lpv {
		if len(v.VersionServices) > 1 {
			continue
		}
		for sharedVersion := range maps.Keys(v.VersionServices) {
			if len(v.VersionServices[sharedVersion]) < 5 {
				break
			}
			v.ServiceVersion = map[string]string{
				"All": sharedVersion,
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
			ServiceVersion:  make(map[string]string),
			VersionServices: make(map[string][]string),
		}
		for serviceName, s := range md.GetServices() {
			if s.GetApiVersion() == "" {
				continue
			}
			lpav.ServiceVersion[serviceName] = s.GetApiVersion()
			lpav.VersionServices[s.GetApiVersion()] = append(lpav.VersionServices[s.GetApiVersion()], serviceName)
		}
		result = append(result, lpav)
	}

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
