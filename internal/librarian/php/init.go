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

package php

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var (
	namespaceRe = regexp.MustCompile(`php_namespace\)?\s*=\s*"([^"]+)"`)
)

func namespace(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := namespaceRe.FindStringSubmatch(line); len(matches) > 1 {

			return strings.ReplaceAll(matches[1], `\\`, `\`), nil
		}
	}
	return "", scanner.Err()
}
