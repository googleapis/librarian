// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package semver provides functionality for parsing, comparing, and manipulating
// semantic version strings according to the SemVer 2.0.0 spec.
package semver

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version.
type Version struct {
	Major, Minor, Patch int
	// Prerelease is the non-numeric part of the prerelease string (e.g., "alpha", "beta").
	Prerelease string
	// PrereleaseSeparator is the separator between the pre-release string and
	// its version (e.g., ".").
	PrereleaseSeparator string
	// PrereleaseNumber is the numeric part of the pre-release string (e.g., "1", "21").
	PrereleaseNumber string
}

// semverRegex defines format for semantic version.
// Regex from https://semver.org/, with buildmetadata part removed.
// It uses named capture groups for major, minor, patch, and prerelease.
var semverRegex = regexp.MustCompile(`^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|[a-zA-Z-][a-zA-Z0-9-]*)(?:\.(?:0|[1-9]\d*|[a-zA-Z-][a-zA-Z0-9-]*))*))?$`)

// Parse parses a version string into a Version struct.
func Parse(versionString string) (*Version, error) {
	matches := semverRegex.FindStringSubmatch(versionString)
	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %s", versionString)
	}

	// Create a map of capture group names to their values.
	result := make(map[string]string)
	for i, name := range semverRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = matches[i]
		}
	}

	v := &Version{}
	var err error
	v.Major, err = strconv.Atoi(result["major"])
	if err != nil {
		// This should not happen if the regex is correct.
		return nil, fmt.Errorf("invalid major version: %w", err)
	}
	v.Minor, err = strconv.Atoi(result["minor"])
	if err != nil {
		// This should not happen if the regex is correct.
		return nil, fmt.Errorf("invalid minor version: %w", err)
	}
	v.Patch, err = strconv.Atoi(result["patch"])
	if err != nil {
		// This should not happen if the regex is correct.
		return nil, fmt.Errorf("invalid patch version: %w", err)
	}

	if prerelease := result["prerelease"]; prerelease != "" {
		if i := strings.LastIndex(prerelease, "."); i != -1 {
			v.Prerelease = prerelease[:i]
			v.PrereleaseSeparator = "."
			v.PrereleaseNumber = prerelease[i+1:]
		} else {
			re := regexp.MustCompile(`^(.*?)(\d+)$`)
			matches := re.FindStringSubmatch(prerelease)
			if len(matches) == 3 {
				v.Prerelease = matches[1]
				v.PrereleaseNumber = matches[2]
			} else {
				v.Prerelease = prerelease
			}
		}
	}

	return v, nil
}

// Compare returns an integer comparing two versions.
// The result is -1, 0, or 1 depending on whether v is less than, equal to, or greater than other.
func (v *Version) Compare(other *Version) int {
	if v.Major < other.Major {
		return -1
	}
	if v.Major > other.Major {
		return 1
	}
	if v.Minor < other.Minor {
		return -1
	}
	if v.Minor > other.Minor {
		return 1
	}
	if v.Patch < other.Patch {
		return -1
	}
	if v.Patch > other.Patch {
		return 1
	}
	// a pre-release version is less than a non-pre-release version
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	// lexical comparison between prerelease type (e.g. "alpha" vs "beta")
	if v.Prerelease < other.Prerelease {
		return -1
	}
	if v.Prerelease > other.Prerelease {
		return 1
	}
	// prerelease number (e.g. "alpha1" vs "alpha2")
	if v.PrereleaseNumber < other.PrereleaseNumber {
		return -1
	}
	if v.PrereleaseNumber > other.PrereleaseNumber {
		return 1
	}
	return 0
}

// String formats a Version struct into a string.
func (v *Version) String() string {
	version := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		version += "-" + v.Prerelease + v.PrereleaseSeparator + v.PrereleaseNumber
	}
	return version
}

// incrementPrerelease increments the pre-release version number, or appends
// one if it doesn't exist.
func (v *Version) incrementPrerelease() error {
	if v.PrereleaseNumber == "" {
		v.PrereleaseSeparator = "."
		v.PrereleaseNumber = "1"
		return nil
	}
	num, err := strconv.Atoi(v.PrereleaseNumber)
	if err != nil {
		// This should not happen if Parse is correct.
		return fmt.Errorf("invalid prerelease version: %w", err)
	}
	v.PrereleaseNumber = strconv.Itoa(num + 1)
	return nil
}

// MaxVersion returns the largest semantic version string among the provided version strings.
func MaxVersion(versionStrings ...string) string {
	if len(versionStrings) == 0 {
		return ""
	}
	versions := make([]*Version, 0)
	for _, versionString := range versionStrings {
		v, err := Parse(versionString)
		if err != nil {
			slog.Warn("invalid version string", "version", v)
			continue
		}
		versions = append(versions, v)
	}
	largest := versions[0]
	for i := 1; i < len(versions); i++ {
		if largest.Compare(versions[i]) < 0 {
			largest = versions[i]
		}
	}
	return largest.String()
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

// DeriveNext calculates the next version based on the highest change type and current version.
func DeriveNext(highestChange ChangeLevel, currentVersion string) (string, error) {
	if highestChange == None {
		return currentVersion, nil
	}

	currentSemVer, err := Parse(currentVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse current version: %w", err)
	}

	// Handle prerelease versions
	if currentSemVer.Prerelease != "" {
		if err := currentSemVer.incrementPrerelease(); err != nil {
			return "", err
		}
		return currentSemVer.String(), nil
	}

	// Handle standard versions
	if currentSemVer.Major == 0 {
		// breaking change and feat result in minor bump for pre-1.0.0
		if highestChange == Major || highestChange == Minor {
			currentSemVer.Minor++
			currentSemVer.Patch = 0
		} else {
			currentSemVer.Patch++
		}
	} else {
		switch highestChange {
		case Major:
			currentSemVer.Major++
			currentSemVer.Minor = 0
			currentSemVer.Patch = 0
		case Minor:
			currentSemVer.Minor++
			currentSemVer.Patch = 0
		case Patch:
			currentSemVer.Patch++
		}
	}

	return currentSemVer.String(), nil
}
