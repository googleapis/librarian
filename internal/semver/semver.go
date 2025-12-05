// Copyright 2025 Google LLC
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

// Package semver provides functionality for parsing, comparing, and manipulating
// semantic version strings according to the SemVer 1.0.0 and 2.0.0 spec.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

// version represents a semantic version.
type version struct {
	Major, Minor, Patch int
	// Prerelease is the non-numeric part of the prerelease string (e.g., "alpha", "beta").
	Prerelease string
	// PrereleaseSeparator is the separator between the pre-release string and
	// its version (e.g., "."). SemVer 1.0.0 versions do not have a prerelease
	// separator.
	PrereleaseSeparator string
	// PrereleaseNumber is the numeric part of the pre-release segment of the
	// version string (e.g., the 1 in "alpha.1"). Zero is a valid pre-release
	// number. If there is no numeric part in the pre-release segment, this
	// field is nil.
	PrereleaseNumber *int
}

// semverV1PrereleaseNumberRegexp extracts the prerelease number, if present, in
// the prerelease portion of the SemVer 1.0.0 version string. For example, a
// version string like "1.2.3-alpha01" is a SemVer 1.0.0. compliant, numbered
// prerelease - https://semver.org/spec/v1.0.0.html#spec-item-4.
var semverV1PrereleaseNumberRegexp = regexp.MustCompile(`^(.*?)(\d+)$`)

// parse deconstructs the SemVer 1.0.0 or 2.0.0 version string into a [version]
// struct.
func parse(versionString string) (version, error) {
	// Our client versions must not have a "v" prefix.
	if strings.HasPrefix(versionString, "v") {
		return version{}, fmt.Errorf("invalid version format: %s", versionString)
	}

	// Prepend "v" internally so that we can use various [semver] APIs.
	// Then canonicalize it to zero-fill any missing version segments.
	// Strips build metadata if present - we do not use build metadata suffixes.
	vPrefixedVersion := "v" + versionString
	if !semver.IsValid(vPrefixedVersion) {
		return version{}, fmt.Errorf("invalid version format: %s", versionString)
	}
	vPrefixedVersion = semver.Canonical(vPrefixedVersion)

	// Preemptively pull out the prerelease segment so that we can trim it off
	// of the Patch segment.
	prerelease := semver.Prerelease(vPrefixedVersion)

	versionCore := strings.TrimPrefix(vPrefixedVersion, "v")
	versionCore = strings.TrimSuffix(versionCore, prerelease)
	vParts := strings.Split(versionCore, ".")

	var v version
	var err error

	v.Major, err = strconv.Atoi(vParts[0])
	if err != nil {
		return version{}, fmt.Errorf("invalid major version: %w", err)
	}

	v.Minor, err = strconv.Atoi(vParts[1])
	if err != nil {
		return version{}, fmt.Errorf("invalid minor version: %w", err)
	}

	v.Patch, err = strconv.Atoi(vParts[2])
	if err != nil {
		return version{}, fmt.Errorf("invalid patch version: %w", err)
	}

	if prerelease == "" {
		return v, nil
	}

	prerelease = strings.TrimPrefix(prerelease, "-")
	var numStr string
	if i := strings.LastIndex(prerelease, "."); i != -1 {
		v.Prerelease = prerelease[:i]
		v.PrereleaseSeparator = "."
		numStr = prerelease[i+1:]
	} else if matches := semverV1PrereleaseNumberRegexp.FindStringSubmatch(prerelease); len(matches) == 3 {
		v.Prerelease = matches[1]
		numStr = matches[2]
	} else {
		v.Prerelease = prerelease
	}

	if numStr != "" {
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return version{}, fmt.Errorf("invalid prerelease number: %w", err)
		}
		v.PrereleaseNumber = &num
	}

	return v, nil
}

// String formats a [version] struct into a string.
func (v version) String() string {
	vStr := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		vStr += "-" + v.Prerelease
		if v.PrereleaseNumber != nil {
			vStr += v.PrereleaseSeparator + strconv.Itoa(*v.PrereleaseNumber)
		}
	}
	return vStr
}

// MaxVersion returns the largest semantic version string among the provided version strings.
func MaxVersion(versionStrings ...string) string {
	if len(versionStrings) == 0 {
		return ""
	}
	versions := make([]string, 0)
	for _, versionString := range versionStrings {
		// Our client versions must not have a "v" prefix.
		if strings.HasPrefix(versionString, "v") {
			continue
		}

		// Prepend "v" internally so that we can use [semver.IsValid] and
		// [semver.Sort].
		vPrefixedString := "v" + versionString
		if !semver.IsValid(vPrefixedString) {
			continue
		}
		versions = append(versions, vPrefixedString)
	}

	if len(versions) == 0 {
		return ""
	}

	semver.Sort(versions)

	// Trim the "v" we prepended to make use of [semver].
	return strings.TrimPrefix(versions[len(versions)-1], "v")
}

// ChangeLevel represents the level of change, corresponding to semantic versioning.
type ChangeLevel int

const (
	// None indicates no change.
	None ChangeLevel = iota
	// Patch is for backward-compatible bug fixes.
	Patch
	// Minor is for backward-compatible new features.
	Minor
	// Major is for incompatible API changes.
	Major
)

// String converts a ChangeLevel to its string representation.
func (c ChangeLevel) String() string {
	return [...]string{"none", "patch", "minor", "major"}[c]
}

// DeriveNextOptions contains options for controlling SemVer version derivation.
type DeriveNextOptions struct {
	// BumpVersionCore forces the version bump to occur in the version core,
	// as opposed to the prerelease number, if one was present. If true, and
	// the version has a prerelease number, that number will be reset to 1.
	//
	// Default behavior is to prefer bumping the prerelease number or adding one
	// when the version is a prerelease without a number.
	BumpVersionCore bool
}

// DeriveNext determines the appropriate SemVer version bump based on the
// provided [ChangeLevel] and the provided [DeriveNextOptions].
func (o DeriveNextOptions) DeriveNext(highestChange ChangeLevel, currentVersion string) (string, error) {
	if highestChange == None {
		return currentVersion, nil
	}

	v, err := parse(currentVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse current version: %w", err)
	}

	// Only bump the prerelease version number.
	if v.Prerelease != "" && !o.BumpVersionCore {
		// Append prerelease number if there isn't one.
		if v.PrereleaseNumber == nil {
			v.PrereleaseSeparator = "."

			// Initialize a new int pointer set to 0. Fallthrough to increment
			// to 1. We prefer the first prerelease to use 1 instead of 0.
			v.PrereleaseNumber = new(int)
		}

		*v.PrereleaseNumber++
		return v.String(), nil
	}

	// Reset prerelease number, if present, then fallthrough to bump version core.
	if v.PrereleaseNumber != nil && o.BumpVersionCore {
		*v.PrereleaseNumber = 1
	}

	// Breaking changes result in a minor bump for pre-1.0.0 versions.
	if highestChange == Major && v.Major == 0 {
		highestChange = Minor
	}

	// Bump the version core.
	switch highestChange {
	case Major:
		v.Major++
		v.Minor = 0
		v.Patch = 0
	case Minor:
		v.Minor++
		v.Patch = 0
	case Patch:
		v.Patch++
	}

	return v.String(), nil
}

// DeriveNext calculates the next version based on the highest change type and
// current version using the default [DeriveNextOptions]. This is a convenience
// method.
func DeriveNext(highestChange ChangeLevel, currentVersion string) (string, error) {
	return DeriveNextOptions{}.DeriveNext(highestChange, currentVersion)
}
