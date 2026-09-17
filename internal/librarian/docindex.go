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

package librarian

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/librarian/swift"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// docIndexFilename is the filename for the DevSite Cloud Reference Documentation index.
const docIndexFilename = "_libraries.json"

// DocIndexEntry represents an individual library entry in _libraries.json.
type DocIndexEntry struct {
	PkgName      string `json:"PkgName"`
	Language     string `json:"Language"`
	DocsURL      string `json:"DocsURL"`
	APIShortname string `json:"APIShortname"`
	Product      string `json:"Product"`
}

// generateDocIndex generates and writes _libraries.json to cfg.Default.Output.
func generateDocIndex(cfg *config.Config, googleapisDir string) error {
	if cfg.Default == nil || cfg.Default.Output == "" {
		return nil
	}
	if cfg.Language != config.LanguageRust && cfg.Language != config.LanguageSwift {
		return nil
	}
	data, err := GenerateDocIndex(cfg, googleapisDir)
	if err != nil {
		return fmt.Errorf("generating doc index: %w", err)
	}
	if err := os.MkdirAll(cfg.Default.Output, 0o755); err != nil {
		return fmt.Errorf("creating doc index directory %s: %w", cfg.Default.Output, err)
	}
	targetPath := filepath.Join(cfg.Default.Output, docIndexFilename)
	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("writing doc index file %s: %w", targetPath, err)
	}
	return nil
}

// GenerateDocIndex produces the formatted JSON byte slice for _libraries.json
// from the workspace configuration and googleapis directory.
func GenerateDocIndex(cfg *config.Config, googleapisDir string) ([]byte, error) {
	index := make(map[string][]DocIndexEntry)
	for _, rawLib := range cfg.Libraries {
		libCopy := *rawLib
		libCopy.APIs = append([]*config.API(nil), rawLib.APIs...)
		libCopy.Keep = append([]string(nil), rawLib.Keep...)
		lib, err := applyDefaults(cfg.Language, &libCopy, cfg.Default)
		if err != nil {
			return nil, err
		}
		if lib.SkipRelease {
			continue
		}
		apiPath := primaryAPIPath(lib)
		if apiPath == "" || apiPath == repometadata.ShowcasePath {
			continue
		}
		api, err := serviceconfig.Find(googleapisDir, apiPath, cfg.Language)
		if err != nil {
			return nil, fmt.Errorf("failed to find API for library %s (path %s): %w", lib.Name, apiPath, err)
		}
		if api == nil || api.Title == "" || api.ShortName == "" {
			slog.Warn("Skipping library in docindex due to missing API service configuration or fields", "library", lib.Name, "apiPath", apiPath)
			continue
		}
		pkgName := resolvePackageName(cfg.Language, lib)
		entry := DocIndexEntry{
			PkgName:      pkgName,
			Language:     formatLanguage(cfg.Language),
			DocsURL:      resolveDocsURL(cfg.Language, lib),
			APIShortname: api.ShortName,
			Product:      api.Title,
		}
		index[pkgName] = append(index[pkgName], entry)
	}
	content, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal docindex JSON: %w", err)
	}
	content = append(content, '\n')
	return content, nil
}

// primaryAPIPath returns the primary API path for a library. For libraries with
// explicitly configured APIs, this is the primary sorted API path. For veneer libraries
// that define modules instead of top-level APIs, it selects the primary non-embedded API path
// using serviceconfig.SortAPIs.
func primaryAPIPath(lib *config.Library) string {
	apiPaths := candidateAPIPaths(lib)
	if len(apiPaths) == 0 {
		return ""
	}
	apis := make([]*config.API, len(apiPaths))
	for i, p := range apiPaths {
		apis[i] = &config.API{Path: p}
	}
	serviceconfig.SortAPIs(apis)
	return apis[0].Path
}

func candidateAPIPaths(lib *config.Library) []string {
	if len(lib.APIs) > 0 {
		var apiPaths []string
		for _, api := range lib.APIs {
			if api.Path != "" {
				apiPaths = append(apiPaths, api.Path)
			}
		}
		return apiPaths
	}
	var apiPaths []string
	if lib.Rust != nil {
		for _, m := range lib.Rust.Modules {
			if m.APIPath != "" && !isEmbeddedAPIPath(m.APIPath) {
				apiPaths = append(apiPaths, m.APIPath)
			}
		}
	}
	if lib.Swift != nil {
		for _, m := range lib.Swift.Modules {
			if m.APIPath != "" && !isEmbeddedAPIPath(m.APIPath) {
				apiPaths = append(apiPaths, m.APIPath)
			}
		}
	}
	return apiPaths
}

// isEmbeddedAPIPath reports whether the given API path represents an auxiliary or
// mixin API (such as IAM policies, long-running operations, or common protobuf types)
// that is embedded into veneer libraries alongside their primary service API.
func isEmbeddedAPIPath(path string) bool {
	for _, prefix := range []string{
		"google/iam",
		"google/longrunning",
		"google/type",
		"google/rpc",
		"google/protobuf",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// resolvePackageName returns the distribution package name for a library in the target language.
func resolvePackageName(lang string, lib *config.Library) string {
	if lang == config.LanguageSwift {
		return swift.PackageName(lib)
	}
	return lib.Name
}

// resolveDocsURL returns the documentation URL for a library in the target language.
func resolveDocsURL(lang string, lib *config.Library) string {
	switch strings.ToLower(lang) {
	case config.LanguageSwift:
		return swift.DocumentationURL(lib)
	case config.LanguageRust:
		return rust.DocumentationURL(lib)
	default:
		return ""
	}
}

// formatLanguage returns the canonical display name for a language (e.g. "Rust", "Swift", "Node.js").
func formatLanguage(lang string) string {
	switch strings.ToLower(lang) {
	case "rust":
		return "Rust"
	case "swift":
		return "Swift"
	case "go":
		return "Go"
	case "cpp":
		return "C++"
	case "java":
		return "Java"
	case "python":
		return "Python"
	case "ruby":
		return "Ruby"
	case "php":
		return "PHP"
	case "nodejs":
		return "Node.js"
	case "dart":
		return "Dart"
	case "dotnet":
		return ".NET"
	default:
		if len(lang) == 0 {
			return ""
		}
		return strings.ToUpper(lang[:1]) + strings.ToLower(lang[1:])
	}
}
