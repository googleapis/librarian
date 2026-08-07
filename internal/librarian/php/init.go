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
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	namespaceRe     = regexp.MustCompile(`php_namespace\)?\s*=\s*"([^"]+)"`)
	versionSuffixRe = regexp.MustCompile(`\\V\d+.*$`)
)

const devTool = "dev/google-cloud"

type initParams struct {
	componentName   string
	phpNamespace    string
	protoPackage    string
	apiShortName    string
	apiVersion      string
	productDocs     string
	productHomepage string
}

func newInitParams(googleapisDir string, api *config.API) (*initParams, error) {
	svcAPI, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguagePhp)
	if err != nil {
		return nil, err
	}
	ns, err := namespace(googleapisDir, api.Path)
	if err != nil {
		return nil, err
	}
	apiVersion := serviceconfig.ExtractVersion(api.Path)
	// TODO(https://github.com/googleapis/librarian/issues/7226):
	// Refactor namespace resolution to avoid stripping and re-appending the version suffix.
	phpNS := ns
	if apiVersion != "" {
		phpNS = ns + `\` + strings.ToUpper(apiVersion[:1]) + apiVersion[1:]
	}
	return &initParams{
		phpNamespace:    phpNS,
		protoPackage:    protoPackage(api),
		apiShortName:    svcAPI.ShortName,
		apiVersion:      apiVersion,
		productDocs:     svcAPI.DocumentationURI,
		productHomepage: repometadata.ExtractBaseProductURL(svcAPI.DocumentationURI),
	}, nil
}

func initComponent(ctx context.Context, params *initParams) error {
	args := []string{
		"component:new",
		"--no-update",
		"--component-name=" + params.componentName,
		"--php-namespace=" + params.phpNamespace,
		"--proto-package=" + params.protoPackage,
		"--api-short-name=" + params.apiShortName,
		"--api-version=" + params.apiVersion,
		"--product-docs=" + params.productDocs,
		"--product-homepage=" + params.productHomepage,
	}
	if err := command.Run(ctx, devTool, args...); err != nil {
		return fmt.Errorf("failed to init new PHP component %s: %w", params.componentName, err)
	}
	return nil
}

// ComponentNameForLibrary resolves the component name for a PHP library.
// If library.PHP.ComponentName is set, it is returned as an explicit override.
// Otherwise, the component name is derived on the fly from the php_namespace of the library's primary API.
// TODO(https://github.com/googleapis/librarian/issues/7213):
// Unexport ComponentNameForLibrary when the migrate tool code is removed.
func ComponentNameForLibrary(googleapisDir string, library *config.Library) (string, error) {
	if library.PHP != nil && library.PHP.ComponentName != "" {
		return library.PHP.ComponentName, nil
	}
	if len(library.APIs) == 0 {
		return "", fmt.Errorf("no apis configured for library %q", library.Name)
	}
	ns, err := namespace(googleapisDir, library.APIs[0].Path)
	if err != nil {
		return "", err
	}
	return componentName(ns), nil
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

// protoPackage returns the unversioned proto package from the API configuration or derived from the path.
func protoPackage(api *config.API) string {
	if api.PHP != nil && api.PHP.ProtoPackage != "" {
		return api.PHP.ProtoPackage
	}
	apiPath := api.Path
	if serviceconfig.ExtractVersion(apiPath) != "" {
		apiPath = path.Dir(apiPath)
	}
	return strings.ReplaceAll(apiPath, "/", ".")
}
