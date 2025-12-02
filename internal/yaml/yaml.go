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

	"github.com/google/yamlfmt/formatters/basic"
	"gopkg.in/yaml.v3"
)

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

// Write marshals a value to YAML, formats it with yamlfmt, and writes it to a
// file.
func Write(path string, v any) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}
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
