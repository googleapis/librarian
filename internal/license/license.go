// Copyright 2024 Google LLC
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

// Package license provides functions for generating license header text.
package license

import (
	"bytes"
	"fmt"
	"regexp"
)

// HasHeader returns true if the content contains the Apache 2.0 license header.
// It checks for the presence of a specific, static line from the license text.
func HasHeader(content []byte) bool {
	return bytes.Contains(content, []byte("Licensed under the Apache License, Version 2.0"))
}

// Header returns the license header with the given year.
func Header(year string) []string {
	full := []string{fmt.Sprintf(" Copyright %s Google LLC", year)}
	full = append(full, HeaderBulk()...)
	return full
}

// HeaderBulk returns the bulk of the license header.
func HeaderBulk() []string {
	return []string{
		"",
		" Licensed under the Apache License, Version 2.0 (the \"License\");",
		" you may not use this file except in compliance with the License.",
		" You may obtain a copy of the License at",
		"",
		"     https://www.apache.org/licenses/LICENSE-2.0",
		"",
		" Unless required by applicable law or agreed to in writing, software",
		" distributed under the License is distributed on an \"AS IS\" BASIS,",
		" WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.",
		" See the License for the specific language governing permissions and",
		" limitations under the License.",
	}
}

var copyrightYearRegexp = regexp.MustCompile(`Copyright (\d{4})`)

// OldestYear returns the oldest copyright year found in the content.
// It returns an empty string if no copyright year is found.
func OldestYear(content []byte) string {
	var oldest string
	for _, match := range copyrightYearRegexp.FindAllSubmatch(content, -1) {
		if len(match) > 1 {
			year := string(match[1])
			if oldest == "" || year < oldest {
				oldest = year
			}
		}
	}
	return oldest
}
