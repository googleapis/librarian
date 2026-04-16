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

// Package yaml provides utilities to interact with librarian configurations.
package yaml

import (
	"fmt"
	"strings"
)

// Get parses a dot-separated path within a map representation of the configuration
// and returns the value. Returns an error if the key is not found.
func Get(m map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = m

	for _, p := range parts {
		if p == "" {
			continue
		}
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot traverse %q: %q is not a map", p, p)
		}
		val, exists := currentMap[p]
		if !exists {
			return nil, fmt.Errorf("key not found: %s", p)
		}
		current = val
	}
	return current, nil
}

// Set sets a value at a dot-separated path within a map representation of the configuration.
// If intermediate keys do not exist, it creates them automatically. It returns the updated map
// to make the mutation explicit.
func Set(m map[string]any, path string, value any) (map[string]any, error) {
	parts := strings.Split(path, ".")
	var current any = m

	for i := 0; i < len(parts)-1; i++ {
		p := parts[i]
		if p == "" {
			continue
		}
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot set path %s", path)
		}

		next, exists := currentMap[p]
		if !exists {
			next = make(map[string]any)
			currentMap[p] = next
		}
		nextMap, ok := next.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot set path %q: %q is not a map", path, p)
		}
		current = nextMap
	}

	currentMap, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot set path %s", path)
	}

	lastPart := parts[len(parts)-1]
	if lastPart == "" {
		return nil, fmt.Errorf("cannot set path %q: last segment is empty", path)
	}
	currentMap[lastPart] = value
	return m, nil
}
