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
)

var (
	namespaceRe     = regexp.MustCompile(`php_namespace\)?\s*=\s*"([^"]+)"`)
	versionSuffixRe = regexp.MustCompile(`[\\.][vV]\d+.*$`)
	packageRe       = regexp.MustCompile(`^package\s+([^\s;]+);`)
)

type initParams struct {
	componentName string
	namespace     string
	protoPackage  string
}

func newInitParams(googleapisDir, apiPath string) (*initParams, error) {
	ns, err := namespace(googleapisDir, apiPath)
	if err != nil {
		return nil, err
	}
	pkg, err := protoPackage(googleapisDir, apiPath)
	if err != nil {
		return nil, err
	}
	return &initParams{
		componentName: componentName(ns),
		namespace:     ns,
		protoPackage:  pkg,
	}, nil
}

// namespace reads the php_namespace option from the first .proto file in the API directory.
// If the option is not found, it generates a fallback namespace from the API path.
func namespace(googleapisDir, apiPath string) (string, error) {
	match, err := findInProto(googleapisDir, apiPath, namespaceRe)
	if err != nil {
		return "", err
	}
	if match == "" {
		return backupNamespace(apiPath), nil
	}
	// Backslashes are escapping chars in protobuf string literals, php namespace
	// in proto need to use double slashes.
	ns := strings.ReplaceAll(match, `\\`, `\`)
	// Stripe the version suffix.
	return versionSuffixRe.ReplaceAllString(ns, ""), nil
}

// componentName returns the component name from a namespace.
func componentName(namespace string) string {
	if comp, ok := strings.CutPrefix(namespace, `Google\Cloud\`); ok {
		return comp
	}
	comp := strings.TrimPrefix(namespace, `Google\`)
	return strings.ReplaceAll(comp, `\`, "")
}

func protoPackage(googleapisDir, apiPath string) (string, error) {
	match, err := findInProto(googleapisDir, apiPath, packageRe)
	if err != nil || match == "" {
		return match, err
	}
	return versionSuffixRe.ReplaceAllLiteralString(match, ""), nil
}

func findInProto(googleapisDir, apiPath string, re *regexp.Regexp) (string, error) {
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
		if strings.HasPrefix(line, "//") {
			continue
		}
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1], nil
		}
	}
	return "", scanner.Err()
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
