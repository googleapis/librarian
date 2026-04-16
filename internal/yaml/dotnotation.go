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

package yaml

import (
	"fmt"
	"strconv"
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
		switch c := current.(type) {
		case map[string]any:
			if val, exists := c[p]; exists {
				current = val
			} else {
				return nil, fmt.Errorf("key not found: %s", p)
			}
		case []any:
			idx, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("expected slice index, got %s", p)
			}
			if idx < 0 || idx >= len(c) {
				return nil, fmt.Errorf("slice index out of range: %d", idx)
			}
			current = c[idx]
		default:
			return nil, fmt.Errorf("cannot traverse %q: not a map or slice", p)
		}
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
		switch c := current.(type) {
		case map[string]any:
			next, exists := c[p]
			if !exists {
				next = make(map[string]any)
				c[p] = next
			}
			current = next
		case []any:
			idx, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("expected slice index, got %s", p)
			}
			if idx < 0 || idx >= len(c) {
				return nil, fmt.Errorf("slice index out of range: %d", idx)
			}
			current = c[idx]
		default:
			return nil, fmt.Errorf("cannot set path %s", path)
		}
	}

	lastPart := parts[len(parts)-1]
	if lastPart == "" {
		return nil, fmt.Errorf("cannot set path %q: last segment is empty", path)
	}

	switch c := current.(type) {
	case map[string]any:
		c[lastPart] = value
	case []any:
		idx, err := strconv.Atoi(lastPart)
		if err != nil {
			return nil, fmt.Errorf("expected slice index, got %s", lastPart)
		}
		if idx < 0 || idx >= len(c) {
			return nil, fmt.Errorf("slice index out of range: %d", idx)
		}
		c[idx] = value
	default:
		return nil, fmt.Errorf("cannot set path %s", path)
	}

	return m, nil
}
