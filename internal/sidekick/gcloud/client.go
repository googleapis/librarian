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
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
)

// goClientInfo describes the Go client and proto-Go packages for a proto
// package like "google.cloud.parallelstore.v1".
type goClientInfo struct {
	// Alias is the short name used as the import alias for the client
	// package, for example "parallelstore". The proto-Go package is
	// imported as Alias+"pb" (e.g. "parallelstorepb").
	Alias string

	// ClientPath is the import path of the GAPIC Go client package, for
	// example "cloud.google.com/go/parallelstore/apiv1".
	ClientPath string

	// PbPath is the import path of the proto-Go package, for example
	// "cloud.google.com/go/parallelstore/apiv1/parallelstorepb".
	PbPath string
}

// goClientPackage maps a proto package name like "google.cloud.parallelstore.v1"
// to its Go client (apiv1) and proto-Go (apiv1/parallelstorepb) packages.
// It returns nil when the proto package does not have the shape
// google.cloud.<short>.v<N>. Beta and alpha suffixes (e.g. v1beta1) are
// intentionally excluded for now.
func goClientPackage(protoPkg string) *goClientInfo {
	rest, ok := strings.CutPrefix(protoPkg, "google.cloud.")
	if !ok {
		return nil
	}
	short, version, ok := strings.Cut(rest, ".")
	if !ok || !isLowerAlphanum(short) || !isStableVersion(version) {
		return nil
	}
	return &goClientInfo{
		Alias:      short,
		ClientPath: fmt.Sprintf("cloud.google.com/go/%s/api%s", short, version),
		PbPath:     fmt.Sprintf("cloud.google.com/go/%s/api%s/%spb", short, version, short),
	}
}

// buildClientCall returns a ClientCall for an AIP-131 Get method when the
// model maps to a standard GAPIC Go package and the command composes a
// resource path. It returns nil otherwise so the command keeps its
// print-only action.
func buildClientCall(method *api.Method, goClient *goClientInfo, hasPath bool) *ClientCall {
	if goClient == nil || !hasPath {
		return nil
	}
	if !provider.IsGet(method) {
		return nil
	}
	if method.InputType == nil {
		return nil
	}
	return &ClientCall{
		Method:      method.Name,
		NameField:   "Name",
		Package:     goClient.Alias,
		RequestType: goClient.Alias + "pb." + method.InputType.Name,
	}
}

// isLowerAlphanum reports whether s starts with a lowercase letter and
// contains only lowercase letters and digits.
func isLowerAlphanum(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// isStableVersion reports whether s is a stable proto version like "v1" or
// "v2": a "v" followed by one or more digits, with no alpha/beta suffix.
func isStableVersion(s string) bool {
	digits, ok := strings.CutPrefix(s, "v")
	if !ok || digits == "" {
		return false
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
	}
	return true
}
