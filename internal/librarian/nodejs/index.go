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

package nodejs

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
)

var (
	versionRegex      = regexp.MustCompile(`^(v\d+(_\d+)*)`)
	clientExportRegex = regexp.MustCompile(`export\s*\{\s*([A-Za-z0-9_]+Client)\s*\}\s*from\s*['"][^'"]+['"]`)
	errClientNotFound = errors.New("client not found in index.ts")
)

type versionAndClient struct {
	Version string
	Client  string
}

// findVersionAndClient scans the package source directory to discover API versions
// and their corresponding exported service client names.
func findVersionAndClient(output string) ([]versionAndClient, error) {
	output = filepath.Clean(output)
	srcDir := filepath.Join(output, "src")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, err
	}
	var versions []versionAndClient
	for _, e := range entries {
		if !e.IsDir() || !versionRegex.MatchString(e.Name()) {
			continue
		}
		versionIndex := filepath.Join(srcDir, e.Name(), "index.ts")
		content, err := os.ReadFile(versionIndex)
		if err != nil {
			return nil, err
		}
		matches := clientExportRegex.FindAllSubmatch(content, -1)
		if len(matches) != 1 {
			return nil, errClientNotFound
		}
		versions = append(versions, versionAndClient{
			Version: e.Name(),
			Client:  string(matches[0][1]),
		})
	}
	return versions, nil
}
