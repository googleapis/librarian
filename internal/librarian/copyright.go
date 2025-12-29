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

package librarian

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var copyrightYearRegex = regexp.MustCompile(`^(?://|#)\s*Copyright\s+(\d{4})\s+Google\s+LLC`)

// extractCopyrightYear reads the copyright year from language-specific files.
// It returns the current year if the file does not exist or if no copyright
// year is found.
func extractCopyrightYear(dir, language string) (string, error) {
	var filePath string
	switch language {
	case languageRust:
		filePath = filepath.Join(dir, "Cargo.toml")
	case languagePython:
		filePath = filepath.Join(dir, "setup.py")
	default:
		return "", nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Sprintf("%d", time.Now().Year()), nil
		}
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := copyrightYearRegex.FindStringSubmatch(line); matches != nil {
			return matches[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", time.Now().Year()), nil
}
