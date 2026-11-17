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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/proto"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	errAPIAlreadyExists           = errors.New("api already exists in library")
	errMissingRubyPackage         = errors.New("missing option ruby_package")
	errInvalidRubyPackage         = errors.New("invalid option ruby_package")
	errRubyPackageVersionMismatch = errors.New("ruby_package version mismatch")

	rubyPackageRe     = regexp.MustCompile(`ruby_package\)?\s*=\s*"([^"]+)"`)
	rubySegmentRe     = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	acronymBoundaryRe = regexp.MustCompile(`([A-Z0-9]+)([A-Z][a-z])`)
	camelBoundaryRe   = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

// DeriveGemNames returns the Ruby gem name or names that must be reserved for
// adding apiPath under googleapisDir, given the existing Ruby configuration cfg.
//
// For a net-new service it returns the wrapper gem name followed by the
// versioned gem name. For a new version of a service already configured in cfg
// it returns only the versioned gem name, reusing the existing base gem name.
func DeriveGemNames(cfg *config.Config, googleapisDir, apiPath string) ([]string, error) {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if existingLib := findLibraryWithAPI(cfg, apiPath); existingLib != "" {
		return nil, fmt.Errorf("%w: %s (in library %s)", errAPIAlreadyExists, apiPath, existingLib)
	}
	version := serviceconfig.ExtractVersion(apiPath)
	segments, err := inspectRubyPackage(googleapisDir, apiPath, version)
	if err != nil {
		return nil, err
	}
	if baseGem, ok := findExistingBaseGemName(cfg, path.Dir(apiPath)); ok {
		return []string{baseGem + "-" + version}, nil
	}
	return gemNamesFromSegments(segments), nil
}

func findLibraryWithAPI(cfg *config.Config, apiPath string) string {
	for _, lib := range cfg.Libraries {
		if slices.ContainsFunc(lib.APIs, func(api *config.API) bool { return api.Path == apiPath }) {
			return lib.Name
		}
	}
	return ""
}

func findExistingBaseGemName(cfg *config.Config, unversionedPath string) (string, bool) {
	// Prefer a non-wrapper versioned library so multi-wrapper APIs resolve their
	// individual sub-gem base name cleanly even if the wrapper name is broader.
	for _, lib := range cfg.Libraries {
		if isWrapperLibrary(lib) {
			continue
		}
		for _, api := range lib.APIs {
			if path.Dir(api.Path) != unversionedPath {
				continue
			}
			ver := serviceconfig.ExtractVersion(api.Path)
			if base, ok := strings.CutSuffix(lib.Name, "-"+ver); ver != "" && ok && base != "" {
				return base, true
			}
		}
	}
	for _, lib := range cfg.Libraries {
		if !isWrapperLibrary(lib) {
			continue
		}
		for _, api := range lib.APIs {
			if path.Dir(api.Path) == unversionedPath {
				return apiGemName(lib, api), true
			}
		}
	}
	return "", false
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

func gemNamesFromSegments(segments []string) []string {
	underscored := make([]string, len(segments))
	for i, seg := range segments {
		underscored[i] = underscoreSegment(seg)
	}
	n := len(underscored)
	wrapper := strings.Join(underscored[:n-1], "-")
	return []string{wrapper, wrapper + "-" + underscored[n-1]}
}

func underscoreSegment(s string) string {
	s = acronymBoundaryRe.ReplaceAllString(s, "${1}_${2}")
	s = camelBoundaryRe.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}
