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

// Package yaml provides generic YAML read and write operations.
package yaml

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/google/yamlfmt/formatters/basic"
	"gopkg.in/yaml.v3"
)

var yearRegex = regexp.MustCompile(`(?m)^# Copyright (\d{4}) Google LLC`)

// StringSlice is a custom slice of strings that allows for fine-grained control
// over YAML marshaling when used with the 'omitempty' tag.
//
// By implementing the yaml.IsZeroer interface, it ensures that:
//  1. A nil slice is considered "zero" and is omitted from the output.
//  2. An empty but initialized slice (e.g., []string{}) is NOT considered "zero"
//     and is explicitly marshaled as an empty YAML sequence ([]).
type StringSlice []string

// IsZero implements the yaml.IsZeroer interface, which determines whether a
// field should be considered "empty" when the 'omitempty' struct tag is used.
func (s StringSlice) IsZero() bool {
	// return true ONLY if nil (omit field)
	// return false if empty slice (keep field)
	return s == nil
}

// Unmarshal parses YAML data into a value of type T.
func Unmarshal[T any](data []byte) (*T, error) {
	var v T
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Marshal converts a value to formatted YAML.
func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return format(buf.Bytes())
}

// Read reads a YAML file and unmarshals it into a value of type T.
func Read[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Unmarshal[T](data)
}

// Write marshals a value to YAML, formats it with yamlfmt, adds a copyright header
// and writes it to a file. It keeps the original year, defaults to current if cannot find one.
func Write(path string, v any) error {
	year := time.Now().Year()
	if existing, err := os.ReadFile(path); err == nil {
		if match := yearRegex.FindSubmatch(existing); len(match) > 1 {
			if y, err := strconv.Atoi(string(match[1])); err == nil {
				year = y
			}
		}
	}

	data, err := Marshal(v)
	if err != nil {
		return err
	}
	data = append([]byte(Header(year)), data...)
	return os.WriteFile(path, data, 0644)
}

// format runs yamlfmt on the given YAML content and returns the formatted output.
func format(data []byte) ([]byte, error) {
	factory := &basic.BasicFormatterFactory{}
	formatter, err := factory.NewFormatter(nil)
	if err != nil {
		return nil, err
	}
	return formatter.Format(data)
}
