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
	"strconv"
	"strings"
	"time"

	"github.com/google/yamlfmt/formatters/basic"
	"github.com/googleapis/librarian/internal/license"
	"go.yaml.in/yaml/v3"
)

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

// MarshalWithComments converts a value to formatted YAML, retaining comments
// from the original YAML document where elements match.
func MarshalWithComments(v any, original []byte) ([]byte, error) {
	if len(original) == 0 {
		return Marshal(v)
	}
	var origNode yaml.Node
	if err := yaml.Unmarshal(original, &origNode); err != nil {
		return Marshal(v)
	}
	var newNode yaml.Node
	if n, ok := v.(*yaml.Node); ok {
		if n.Kind == yaml.DocumentNode {
			newNode = *n
		} else {
			newNode = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{n}}
		}
	} else if n, ok := v.(yaml.Node); ok {
		if n.Kind == yaml.DocumentNode {
			newNode = n
		} else {
			newNode = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{&n}}
		}
	} else {
		data, err := yaml.Marshal(v)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, &newNode); err != nil {
			return nil, err
		}
	}
	transferComments(&origNode, &newNode)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&newNode); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return format(buf.Bytes())
}

// Write marshals a value to YAML, formats it with yamlfmt, adds or preserves
// a copyright header and comments, and writes it to a file.
func Write(path string, v any) error {
	var data []byte
	existing, err := os.ReadFile(path)
	if err == nil && len(existing) > 0 {
		data, err = MarshalWithComments(v, existing)
		if err != nil {
			return err
		}
	} else {
		data, err = Marshal(v)
		if err != nil {
			return err
		}
	}
	if !hasLicenseHeader(data) {
		data = append([]byte(licenseHeader()), data...)
	}
	return os.WriteFile(path, data, 0o644)
}

func licenseHeader() string {
	var b strings.Builder
	year := time.Now().Year()
	for _, line := range license.Header(strconv.Itoa(year)) {
		b.WriteString("#")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func hasLicenseHeader(data []byte) bool {
	return bytes.Contains(data, []byte("Licensed under the Apache License, Version 2.0"))
}

func transferComments(src, dst *yaml.Node) {
	if src == nil || dst == nil {
		return
	}
	if src.Kind == yaml.DocumentNode && dst.Kind == yaml.DocumentNode {
		if dst.HeadComment == "" {
			dst.HeadComment = src.HeadComment
		}
		if dst.FootComment == "" {
			dst.FootComment = src.FootComment
		}
		if len(src.Content) > 0 && len(dst.Content) > 0 {
			transferComments(src.Content[0], dst.Content[0])
		}
		return
	}
	if dst.HeadComment == "" {
		dst.HeadComment = src.HeadComment
	}
	if dst.LineComment == "" {
		dst.LineComment = src.LineComment
	}
	if dst.FootComment == "" {
		dst.FootComment = src.FootComment
	}
	switch dst.Kind {
	case yaml.MappingNode:
		if src.Kind != yaml.MappingNode {
			return
		}
		type pair struct {
			key *yaml.Node
			val *yaml.Node
		}
		srcMap := make(map[string]pair, len(src.Content)/2)
		for i := 0; i+1 < len(src.Content); i += 2 {
			srcMap[src.Content[i].Value] = pair{key: src.Content[i], val: src.Content[i+1]}
		}
		for i := 0; i+1 < len(dst.Content); i += 2 {
			dk := dst.Content[i]
			dv := dst.Content[i+1]
			if sp, ok := srcMap[dk.Value]; ok {
				if dk.HeadComment == "" {
					dk.HeadComment = sp.key.HeadComment
				}
				if dk.LineComment == "" {
					dk.LineComment = sp.key.LineComment
				}
				if dk.FootComment == "" {
					dk.FootComment = sp.key.FootComment
				}
				transferComments(sp.val, dv)
			}
		}
	case yaml.SequenceNode:
		if src.Kind != yaml.SequenceNode {
			return
		}
		matchedSrc := make(map[int]bool)
		// First pass: match mappings by identifier.
		for _, ditem := range dst.Content {
			if ditem.Kind == yaml.MappingNode {
				ident := nodeIdentifier(ditem)
				if ident != "" {
					for si, sitem := range src.Content {
						if !matchedSrc[si] && sitem.Kind == yaml.MappingNode && nodeIdentifier(sitem) == ident {
							matchedSrc[si] = true
							transferComments(sitem, ditem)
							break
						}
					}
				}
			}
		}
		// Second pass: match scalars by value.
		for _, ditem := range dst.Content {
			if ditem.Kind == yaml.ScalarNode {
				for si, sitem := range src.Content {
					if !matchedSrc[si] && sitem.Kind == yaml.ScalarNode && sitem.Value == ditem.Value {
						matchedSrc[si] = true
						transferComments(sitem, ditem)
						break
					}
				}
			}
		}
		// Third pass: match remaining items that do not have an identifier by position.
		srcIdx := 0
		for _, ditem := range dst.Content {
			if ditem.Kind == yaml.MappingNode && nodeIdentifier(ditem) != "" {
				continue
			}
			if ditem.HeadComment != "" || ditem.LineComment != "" {
				continue
			}
			for srcIdx < len(src.Content) && (matchedSrc[srcIdx] || (src.Content[srcIdx].Kind == yaml.MappingNode && nodeIdentifier(src.Content[srcIdx]) != "")) {
				srcIdx++
			}
			if srcIdx < len(src.Content) {
				matchedSrc[srcIdx] = true
				transferComments(src.Content[srcIdx], ditem)
				srcIdx++
			}
		}
	}
}

func nodeIdentifier(n *yaml.Node) string {
	if n.Kind != yaml.MappingNode {
		return ""
	}
	for _, idKey := range []string{"name", "path", "id"} {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == idKey && n.Content[i+1].Kind == yaml.ScalarNode {
				return idKey + ":" + n.Content[i+1].Value
			}
		}
	}
	return ""
}

// Empty returns whether the given value serializes to an empty YAML object
// (i.e. "{}" with a line break).
func Empty(v any) (bool, error) {
	data, err := Marshal(v)
	if err != nil {
		return false, err
	}
	return string(data) == "{}\n", nil
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
