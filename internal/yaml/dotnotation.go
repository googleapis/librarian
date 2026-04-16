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

<<<<<<< HEAD
=======
// Package yaml provides utilities to interact with librarian configurations.
>>>>>>> 8dc1d627 (feat(yaml): add generic dotnotation parser and tests)
package yaml

import (
	"fmt"
	"strings"
)

<<<<<<< HEAD
// Get parses a dot-separated path within a map and returns the value.
func Get(m map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = m
	// Traverse the map hierarchy step-by-step based on the path components
=======
// Get parses a dot-separated path within a map representation of the configuration
// and returns the value. Returns an error if the key is not found.
func Get(m map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = m

>>>>>>> 8dc1d627 (feat(yaml): add generic dotnotation parser and tests)
	for _, p := range parts {
		if currentMap, ok := current.(map[string]any); ok {
			if val, exists := currentMap[p]; exists {
				current = val
			} else {
				return nil, fmt.Errorf("key not found: %s", p)
			}
		} else {
<<<<<<< HEAD
			// Fails if intermediate keys are not maps
=======
>>>>>>> 8dc1d627 (feat(yaml): add generic dotnotation parser and tests)
			return nil, fmt.Errorf("key not found: %s", p)
		}
	}
	return current, nil
}

<<<<<<< HEAD
// Set sets a value at a dot-separated path within a map.
// It returns the updated map to make the mutation explicit.
func Set(m map[string]any, path string, value any) (map[string]any, error) {
	parts := strings.Split(path, ".")
	var current any = m
=======
// Set sets a value at a dot-separated path within a map representation of the configuration.
// If intermediate keys do not exist, it creates them automatically. It returns the updated map
// to make the mutation explicit.
func Set(m map[string]any, path string, value any) (map[string]any, error) {
	parts := strings.Split(path, ".")
	var current any = m

>>>>>>> 8dc1d627 (feat(yaml): add generic dotnotation parser and tests)
	for i := 0; i < len(parts)-1; i++ {
		p := parts[i]
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot set path %s", path)
		}
<<<<<<< HEAD
=======

>>>>>>> 8dc1d627 (feat(yaml): add generic dotnotation parser and tests)
		if next, exists := currentMap[p]; exists {
			if _, ok := next.(map[string]any); !ok {
				nextMap := make(map[string]any)
				currentMap[p] = nextMap
				current = nextMap
			} else {
				current = next
			}
		} else {
			nextMap := make(map[string]any)
			currentMap[p] = nextMap
			current = nextMap
		}
	}
<<<<<<< HEAD
=======

>>>>>>> 8dc1d627 (feat(yaml): add generic dotnotation parser and tests)
	currentMap, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot set path %s", path)
	}
<<<<<<< HEAD
=======

>>>>>>> 8dc1d627 (feat(yaml): add generic dotnotation parser and tests)
	currentMap[parts[len(parts)-1]] = value
	return m, nil
}
