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
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/googleapis/librarian/internal/config"
)

const (
	repoURLPrefix      = "https://github.com/googleapis/google-cloud-node/blob/main"
	releaseLevelStable = `This library is considered to be **stable**. The code surface will not change in backwards-incompatible ways
unless absolutely necessary (e.g. because of critical security issues) or with
an extensive deprecation period. Issues and requests against **stable** libraries
are addressed with the highest priority.`

	releaseLevelPreview = `This library is considered to be in **preview**. This means it is still a
work-in-progress and under active development. Any release is subject to
backwards-incompatible changes at any time.`
)

var (
	//go:embed template/_README.md.txt
	readmeTmpl       string
	readmeTmplParsed = template.Must(template.New("readme").Parse(readmeTmpl))
	errorFindSampleMetadata = errors.New("error finding sample metadata")
	samplePathPrefix        = filepath.Join("samples", "generated")
)

type sampleMetadata struct {
	name     string
	filePath string
}

func generateReadme(library *config.Library, output string) error {
	sampleMetadata, err := findSampleMetadata(output)
	if err != nil {
		return err
	}
	readmePath := filepath.Join(output, "README.md")
	f, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	err = readmeTmplParsed.Execute(f, map[string]interface{}{
		"Name":       title,
		"ModulePath": modulePath(library),
		"Samples":  sampleMetadata,
	})
	cerr := f.Close()
	if err != nil {
		return err
	}
	return cerr
}

func findSampleMetadata(output string) ([]sampleMetadata, error) {
	output = filepath.Clean(output)
	samplesPath := filepath.Join(output, samplePathPrefix)
	var metadata []sampleMetadata
	if _, err := os.Stat(samplesPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return metadata, nil
		}
		return nil, err
	}
	repoRoot := filepath.Dir(filepath.Dir(output))
	err := filepath.WalkDir(samplesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".js" {
			return nil
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		metadata = append(metadata, sampleMetadata{
			name:     extractSampleName(d.Name()),
			filePath: repoURLPrefix + "/" + filepath.ToSlash(relPath),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errorFindSampleMetadata, err)
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

func releaseLevelMarkdown(rl string) string {
	if rl == "stable" {
		return releaseLevelStable
	}
	return releaseLevelPreview
}
