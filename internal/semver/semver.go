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

package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version.
type Version struct {
	Major, Minor, Patch int
	// Prerelease is the non-numeric part of the pre-release string (e.g., "alpha", "beta").
	Prerelease string
	// PrereleaseSeparator is the separator between the pre-release string and
	// its version (e.g., ".").
	PrereleaseSeparator string
	// PrereleaseNumber is the numeric part of the pre-release string (e.g., "1", "21").
	PrereleaseNumber string
}

// Parse parses a version string into a Version struct.
func Parse(versionString string) (*Version, error) {
	parts := strings.SplitN(versionString, "-", 2)
	versionPart := parts[0]

	versionParts := strings.Split(versionPart, ".")
	if len(versionParts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s", versionString)
	}

	major, err := strconv.Atoi(versionParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.Atoi(versionParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err := strconv.Atoi(versionParts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %w", err)
	}

	v := &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}

	if len(parts) > 1 {
		prerelease := parts[1]
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

// DeriveNext calculates the next version based on the highest change type and current version.
func DeriveNext(highestChange, currentVersion string) (string, error) {
	if highestChange == "none" {
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
		if highestChange == "major" {
			currentSemVer.Major++
			currentSemVer.Minor = 0
			currentSemVer.Patch = 0
		} else { // feat and fix result in a patch bump for pre-1.0.0
			currentSemVer.Patch++
		}
	} else {
		switch highestChange {
		case "major":
			currentSemVer.Major++
			currentSemVer.Minor = 0
			currentSemVer.Patch = 0
		case "minor":
			currentSemVer.Minor++
			currentSemVer.Patch = 0
		case "patch":
			currentSemVer.Patch++
		}
	}

	return currentSemVer.String(), nil
}
