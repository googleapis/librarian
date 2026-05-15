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

package language

import (
	"bytes"
	"regexp"
	"sort"

	"github.com/yuin/goldmark/ast"
)

// ExtractCrossReferenceLinks returns the cross-reference links found in `doc` and `source`.
//
// Google Cloud documentation comments include cross-reference links:
//
// https://google.aip.dev/192#cross-references
//
// That is, markdown reference-style links in the form `[Title][Definition]` where `Title` is the text that should
// appear in the documentation and `Definition` is the name of a Protobuf entity, e.g., `google.longrunning.Operation`.
//
// The cross reference links can be of the form `[Title][]` when both the title and definitions match. And may also
// be relative to the current entity (e.g. `Operation` instead of the fully qualified `google.longrunning.Operation`).
func ExtractCrossReferenceLinks(doc ast.Node, source []byte) []string {
	protobufLinks := map[string]bool{}
	ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node.Kind() {
		case ast.KindParagraph:
			text := node.Lines().Value(source)
			extractProtoLinks(text, protobufLinks)
			return ast.WalkContinue, nil
		case ast.KindTextBlock:
			text := node.Lines().Value(source)
			extractProtoLinks(text, protobufLinks)
			return ast.WalkContinue, nil
		default:
			return ast.WalkContinue, nil
		}
	})
	var sortedLinks []string
	for link := range protobufLinks {
		sortedLinks = append(sortedLinks, link)
	}
	sort.Strings(sortedLinks)
	return sortedLinks
}

var (
	explicitLink = regexp.MustCompile(`\]\[([^.][^]]*)\]`)
	impliedLink  = regexp.MustCompile(`\[([^.][^]]*)\]\[\]`)
)

func extractProtoLinks(p []byte, links map[string]bool) {
	for _, m := range explicitLink.FindAllSubmatch(p, -1) {
		if validProtoName(m[1]) {
			links[string(m[1])] = true
		}
	}
	for _, m := range impliedLink.FindAllSubmatch(p, -1) {
		if validProtoName(m[1]) {
			links[string(m[1])] = true
		}
	}
}

func validProtoName(b []byte) bool {
	parts := bytes.Split(b, []byte("."))
	for _, p := range parts {
		if !isIdent(p) {
			return false
		}
	}
	return len(parts) != 0
}

// isProtoIdent returns true if the id looks like a valid identifier.
func isIdent(id []byte) bool {
	if len(id) == 0 {
		return false
	}
	if !isIdStartingChar(id[0]) {
		return false
	}
	for _, b := range id[1:] {
		if !isIdChar(b) {
			return false
		}
	}
	return true
}

func isIdStartingChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isIdChar(b byte) bool {
	return isIdStartingChar(b) || (b >= '0' && b <= '9') || b == '_'
}
