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

package gcloud

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// shortServiceName extracts the short service name from a service's DefaultHost.
// For example, "parallelstore.googleapis.com" returns "parallelstore".
// It panics if DefaultHost does not contain a dot, since that indicates a
// programming error in the caller or corrupt input data.
func shortServiceName(service *api.Service) string {
	name, _, found := strings.Cut(service.DefaultHost, ".")
	if !found {
		panic(fmt.Sprintf("failed to determine short service name for service %q: default_host %q has no dot", service.Name, service.DefaultHost))
	}
	return name
}

// inferTrackFromPackage infers the release track from the proto package name.
// as mandated per AIP-185
// e.g. "google.cloud.parallelstore.v1beta" -> "beta".
func inferTrackFromPackage(pkg string) string {
	parts := strings.Split(pkg, ".")
	version := parts[len(parts)-1]

	// AIP-191: The version component MUST follow the pattern `v[0-9]+...`.
	if !strings.HasPrefix(version, "v") {
		return "ga"
	}

	if strings.Contains(version, "alpha") {
		return "alpha"
	}
	if strings.Contains(version, "beta") {
		return "beta"
	}
	return "ga"
}

// getServiceTitle returns the service title for documentation.
// It tries to use the API title, falling back to a CamelCase version of the short service name.
func getServiceTitle(model *api.API, shortServiceName string) string {
	if t := strings.TrimSuffix(model.Title, " API"); t != "" {
		return t
	}
	return strcase.ToCamel(shortServiceName)
}
