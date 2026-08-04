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

package php

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	namespaceRe     = regexp.MustCompile(`php_namespace\)?\s*=\s*"([^"]+)"`)
	versionSuffixRe = regexp.MustCompile(`\\V\d+.*$`)
)

type initParams struct {
	apiShortName    string
	productDocs     string
	productHomepage string
}

func newInitParams(googleapisDir, apiPath string) (*initParams, error) {
	api, err := serviceconfig.Find(googleapisDir, apiPath, config.LanguagePhp)
	if err != nil {
		return nil, err
	}
	return &initParams{
		apiShortName:    api.ShortName,
		productDocs:     api.DocumentationURI,
		productHomepage: repometadata.ExtractBaseProductURL(api.DocumentationURI),
	}, nil
}

// namespace reads the php_namespace option from the first .proto file in the API directory.
// If the option is not found, it generates a fallback namespace from the API path.
func namespace(googleapisDir, apiPath string) (string, error) {
	file, err := searchForProto(googleapisDir, apiPath)
	if err != nil {
		return "", err
	}
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Ignore comments.
		if strings.HasPrefix(line, "//") {
			continue
		}
		if matches := namespaceRe.FindStringSubmatch(line); len(matches) > 1 {
			// Backslashes are escapping chars in protobuf string literals, php namespace
			// in proto need to use double slashes.
			ns := strings.ReplaceAll(matches[1], `\\`, `\`)
			// Stripe the version suffix.
			return versionSuffixRe.ReplaceAllString(ns, ""), nil
		}
	}
	if scanner.Err() != nil {
		return "", scanner.Err()
	}
	return backupNamespace(apiPath), nil
}

// componentName returns the component name from a namespace.
func componentName(namespace string) string {
	if comp, ok := strings.CutPrefix(namespace, `Google\Cloud\`); ok {
		return comp
	}
	comp := strings.TrimPrefix(namespace, `Google\`)
	return strings.ReplaceAll(comp, `\`, "")
}

// searchForProto finds the first .proto file in the API directory.
func searchForProto(googleapisDir, apiPath string) (string, error) {
	dir := filepath.Join(googleapisDir, apiPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".proto" {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", fs.ErrNotExist
}

// backupNamespace generates a fallback namespace from the API path.
func backupNamespace(apiPath string) string {
	parts := strings.Split(apiPath, "/")
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	ns := strings.Join(parts, `\`)
	// Stripe the version suffix.
	return versionSuffixRe.ReplaceAllString(ns, "")
}
