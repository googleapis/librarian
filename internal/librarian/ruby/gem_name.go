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

package ruby

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/proto"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	errMissingRubyPackage         = errors.New("missing option ruby_package")
	errInvalidRubyPackage         = errors.New("invalid option ruby_package")
	errRubyPackageVersionMismatch = errors.New("ruby_package version mismatch")

	rubyPackageRe     = regexp.MustCompile(`ruby_package\)?\s*=\s*"([^"]+)"`)
	rubySegmentRe     = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	acronymBoundaryRe = regexp.MustCompile(`([A-Z0-9]+)([A-Z][a-z])`)
	camelBoundaryRe   = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

// GemNamesToReserve returns the Ruby gem names to reserve on RubyGems.org
// before adding the versioned apiPath, given the existing libraries (for Ruby
// the librarian.yaml library name is the gem name):
//
//   - For a net-new service, the wrapper and versioned gems derived from option
//     ruby_package in the protos under googleapisDir. For example,
//     Google::Cloud::SecretManager::V1 gives google-cloud-secret_manager and
//     google-cloud-secret_manager-v1.
//   - For a service that already has another version in libraries, only the
//     versioned gem, reusing that service's base gem name, such as
//     google-cloud-backupdr-v1beta.
//
// option ruby_package is always read and checked, so a missing or invalid
// value is reported even for an existing service.
func GemNamesToReserve(libraries []*config.Library, googleapisDir, apiPath string) ([]string, error) {
	version := serviceconfig.ExtractVersion(apiPath)
	segments, err := inspectRubyPackage(googleapisDir, apiPath, version)
	if err != nil {
		return nil, err
	}
	if base := existingBaseGemName(libraries, path.Dir(apiPath)); base != "" {
		return []string{base + "-" + version}, nil
	}
	wrapper, versioned := gemNamesFromSegments(segments)
	return []string{wrapper, versioned}, nil
}

// existingBaseGemName returns the base gem name (without a version suffix) of
// the service at unversionedPath, from a library for one of its versions, or
// "" if libraries has none. A versioned library is preferred over a wrapper, so
// a sub-service of a multi-wrapper library resolves to its own gem name rather
// than the broader wrapper name.
func existingBaseGemName(libraries []*config.Library, unversionedPath string) string {
	for _, lib := range libraries {
		if isWrapperLibrary(lib) {
			continue
		}
		for _, api := range lib.APIs {
			if path.Dir(api.Path) != unversionedPath {
				continue
			}
			if base := versionedBase(lib, api); base != "" {
				return base
			}
		}
	}
	for _, lib := range libraries {
		if !isWrapperLibrary(lib) {
			continue
		}
		for _, api := range lib.APIs {
			if path.Dir(api.Path) == unversionedPath {
				return apiGemName(lib, api)
			}
		}
	}
	return ""
}

// versionedBase returns lib's name without the "-<version>" suffix of api, or
// "" if the name does not end with that suffix.
func versionedBase(lib *config.Library, api *config.API) string {
	ver := serviceconfig.ExtractVersion(api.Path)
	if ver == "" {
		return ""
	}
	base, ok := strings.CutSuffix(lib.Name, "-"+ver)
	if !ok {
		return ""
	}
	return base
}

// inspectRubyPackage reads the ruby_package option from the first proto file
// in apiPath and returns its validated :: segments.
func inspectRubyPackage(googleapisDir, apiPath, version string) ([]string, error) {
	pkg, found, err := proto.Search(googleapisDir, apiPath, rubyPackageRe)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%w: %s", errMissingRubyPackage, apiPath)
	}
	return validateRubyPackage(pkg, version)
}

// validateRubyPackage checks rubyPkg against version and returns its ::
// segments.
func validateRubyPackage(rubyPkg, version string) ([]string, error) {
	segments := strings.Split(rubyPkg, "::")
	if len(segments) < 2 {
		return nil, fmt.Errorf("%w: %q must have at least two :: segments", errInvalidRubyPackage, rubyPkg)
	}
	for _, seg := range segments {
		if !rubySegmentRe.MatchString(seg) {
			return nil, fmt.Errorf("%w: invalid segment %q in %q", errInvalidRubyPackage, seg, rubyPkg)
		}
	}
	// An unversioned API path has no version, so it never matches.
	var want string
	if version != "" {
		want = "V" + version[1:]
	}
	if got := segments[len(segments)-1]; got != want {
		return nil, fmt.Errorf("%w: got %q, want %q in %q", errRubyPackageVersionMismatch, got, want, rubyPkg)
	}
	return segments, nil
}

func gemNamesFromSegments(segments []string) (wrapper, versioned string) {
	underscored := make([]string, len(segments))
	for i, seg := range segments {
		underscored[i] = underscoreSegment(seg)
	}
	n := len(underscored)
	wrapper = strings.Join(underscored[:n-1], "-")
	return wrapper, wrapper + "-" + underscored[n-1]
}

func underscoreSegment(s string) string {
	s = acronymBoundaryRe.ReplaceAllString(s, "${1}_${2}")
	s = camelBoundaryRe.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}
