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

var commentCrossReferenceLink = regexp.MustCompile(
	`` + // `go fmt` is annoying
		`\]` + // The closing bracket for the `[Thing]`
		`\[` + // The opening bracket for the code element.
		`[A-Za-z][A-Za-z0-9_]*` + // A thing that looks like a Protobuf identifier
		`(\.` + // Followed by (maybe a dot)
		`[A-Za-z][A-Za-z0-9_]*` + // A thing that looks like a Protobuf identifier
		`)*` + // zero or more times
		`\]`) // The closing bracket

var commentImpliedCrossReferenceLink = regexp.MustCompile(
	`` + // `go fmt` is annoying
		`\[` +
		`[A-Za-z][A-Za-z0-9_]*` + // A thing that looks like a Protobuf identifier
		`(\.[A-Za-z][A-Za-z0-9_]*)*` + // Followed by more identifiers
		`\]\[\]`) // The closing bracket followed by an empty link label

func extractProtoLinks(paragraph []byte, links map[string]bool) {
	for _, match := range commentCrossReferenceLink.FindAll(paragraph, -1) {
		match = bytes.TrimSuffix(bytes.TrimPrefix(match, []byte("][")), []byte("]"))
		links[string(match)] = true
	}
	for _, match := range commentImpliedCrossReferenceLink.FindAll(paragraph, -1) {
		match = bytes.TrimSuffix(bytes.TrimPrefix(match, []byte("[")), []byte("][]"))
		links[string(match)] = true
	}
}
