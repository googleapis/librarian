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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const repoURLPrefix = "https://github.com/googleapis/google-cloud-node/blob/main/packages"

var samplePathPrefix = filepath.Join("samples", "generated")

type sampleMetadata struct {
	name     string
	filePath string
}

func findSampleMetadata(output string) ([]sampleMetadata, error) {
	samplesPath := filepath.Join(output, samplePathPrefix)
	var metadata []sampleMetadata
	if _, err := os.Stat(samplesPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return metadata, nil
		}
		return nil, err
	}
	err := filepath.WalkDir(samplesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".js" {
			return nil
		}
		metadata = append(metadata, sampleMetadata{
			name:     extractSampleName(d.Name()),
			filePath: filepath.Join(repoURLPrefix, path),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

func extractSampleName(name string) string {
	name = strings.TrimSuffix(name, ".js")
	idx := strings.Index(name, ".")
	if idx != -1 {
		name = name[idx+1:]
	}
	return strings.ReplaceAll(name, "_", " ")
}
