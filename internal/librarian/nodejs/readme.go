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
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
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
	partials = ".readme-partials.yaml"
)

var (
	//go:embed template/_README.md.txt
	readmeTmpl            string
	readmeTmplParsed      = template.Must(template.New("readme").Parse(readmeTmpl))
	errFindSampleMetadata = errors.New("error finding sample metadata")
	errReadPartials       = errors.New("error reading partials")
	samplePathPrefix      = filepath.Join("samples", "generated")
)

type sampleMetadata struct {
	Name     string
	FilePath string
}

func generateReadmeNew(cfg *config.Config, library *config.Library, googleapisDir, output string) (err error) {
	metadata, err := generateRepoMetadata(cfg, library, googleapisDir)
	if err != nil {
		return err
	}
	sampleMetadata, err := findSampleMetadata(output)
	if err != nil {
		return err
	}
	partialContent, err := readPartials(output)
	if err != nil {
		return err
	}
	readmePath := filepath.Join(output, "README.md")
	f, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return readmeTmplParsed.Execute(f, map[string]any{
		"APIID":            metadata.APIID,
		"ClientDoc":        metadata.ClientDocumentation,
		"DistributionName": metadata.DistributionName,
		"LibraryName":      library.Name,
		"Partials":         partialContent,
		"Name":             metadata.NamePretty,
		"ProductDoc":       metadata.ProductDocumentation,
		"ReleaseLevel":     releaseLevelMarkdown(metadata.ReleaseLevel),
		"Samples":          sampleMetadata,
	})
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
			Name:     extractSampleName(d.Name()),
			FilePath: repoURLPrefix + "/" + filepath.ToSlash(relPath),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFindSampleMetadata, err)
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

func readPartials(output string) (map[string]string, error) {
	part, err := yaml.Read[map[string]string](filepath.Join(output, partials))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("%w: %w", errReadPartials, err)
	}
	res := make(map[string]string)
	for k, v := range *part {
		res[k] = strings.TrimSpace(v)
	}
	return res, nil
}
