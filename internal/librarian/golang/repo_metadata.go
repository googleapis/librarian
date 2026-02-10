package golang

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// metadata is used for JSON marshaling in manifest.
type metadata struct {
	APIShortname        string
	ClientDocumentation string
	ClientLibraryType   string
	Description         string
	DistributionName    string
	Language            string
	LibraryType         string
	ReleaseLevel        string
}

// generateRepoMetadata generates a .repo-metadata.json file for a given API.
func generateRepoMetadata(library *config.Library, apiPath, googleapisDir string) error {
	api, err := serviceconfig.Find(googleapisDir, apiPath, serviceconfig.LangGo)
	if err != nil {
		return err
	}
	goAPI := findGoAPI(library, apiPath)
	entry := metadata{
		APIShortname:        api.APIShortName,
		ClientDocumentation: api.DocumentationURI,
		ClientLibraryType:   "generated",
		Description:         api.Title,
		DistributionName:    goAPI.ImportPath,
		Language:            "go",
		LibraryType:         "GAPIC_AUTO",
		ReleaseLevel:        releaseLevel,
	}

	// Determine output path from the import path.
	outputPath := filepath.Join(library.Output, ".repo-metadata.json")

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	jsonData = append(jsonData, '\n')
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return err
	}
	return nil
}

// releaseLevel determines the release level of a library. It prioritizes the
// import path for "alpha" or "beta" suffixes. If not present, it falls back
// to checking the release_level specified in the BUILD.bazel file for "alpha"
// or "beta" , and finally defaults to returning "stable", per the behavior of
// the [go_gapic_opt protoc plugin option
// flag](https://github.com/googleapis/gapic-generator-go?tab=readme-ov-file#invocation):
// - `release-level`: the client library release level.
//   - Defaults to empty, which is essentially the GA release level.
//   - Acceptable values are `alpha` and `beta`.
func releaseLevel(importPath string, bazelConfig *bazel.Config) (string, error) {
	// 1. Scan import path
	i := strings.LastIndex(importPath, "/")
	lastElm := importPath[i+1:]
	if strings.Contains(lastElm, "alpha") {
		return "preview", nil
	} else if strings.Contains(lastElm, "beta") {
		return "preview", nil
	}

	// 2. Read release_level attribute, if present, from go_gapic_library rule in BUILD.bazel.
	if bazelConfig.ReleaseLevel() == "alpha" {
		return "preview", nil
	} else if bazelConfig.ReleaseLevel() == "beta" {
		return "preview", nil
	}

	// 3. If alpha or beta are not found in path or build file, default is `stable`.
	return "stable", nil
}
