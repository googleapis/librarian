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

package provider

import (
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

const (
	ALPHA = "ALPHA"
	BETA  = "BETA"
	GA    = "GA"
)

// apiVersionFromPackage extracts the API version from the package name.
// e.g. "google.cloud.parallelstore.v1" -> "v1".
func apiVersionFromPackage(pkg string) string {
	parts := strings.Split(pkg, ".")
	return parts[len(parts)-1]
}

// APIVersionFromModel extracts the API version from the API model's package name.
func APIVersionFromModel(model *api.API) string {
	return apiVersionFromPackage(model.PackageName)
}

// Tracks infers the release tracks from the API version string.
// as mandated per AIP-185
// e.g. "v1beta" -> ["BETA"].
func Tracks(version string) []string {
	// AIP-191: The version component MUST follow the pattern `v[0-9]+...`.
	if !strings.HasPrefix(version, "v") {
		return []string{GA}
	}

	if strings.Contains(version, "alpha") {
		return []string{ALPHA}
	}
	if strings.Contains(version, "beta") {
		return []string{BETA}
	}
	return []string{GA}
}

// GetServiceTitle returns the service title for documentation.
// It tries to use the API title, falling back to a CamelCase version of the short service name.
func GetServiceTitle(model *api.API, shortServiceName string) string {
	if t := strings.TrimSuffix(model.Title, " API"); t != "" {
		return t
	}
	return strcase.ToCamel(shortServiceName)
}

// ResolveRootPackage extracts the service name from the package name (second to last element),
// falling back to the provided fallback if there are not enough segments.
func ResolveRootPackage(model *api.API) string {
	if parts := strings.Split(model.PackageName, "."); len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return model.Name
}
