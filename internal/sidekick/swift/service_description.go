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

package swift

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var reDocXref = regexp.MustCompile(`\[([^\]]+)\]\[[^\]]*\]`)

const (
	// maxSectionHeaderLength is the maximum character length for a paragraph ending in
	// a colon (":") to be classified as a structural section heading (e.g. "API Overview:"
	// or "Data Model:") and skipped. Across Google Cloud APIs, genuine section headings
	// are short labels (in the corpus, <= 13 characters), while actual introductory
	// descriptive sentences that end in a colon (e.g. "This service allows you to do the
	// following:") are at least 44 characters long. A threshold of 40 cleanly separates
	// headings from descriptive text.
	//
	// Having two distinct constants (maxSectionHeaderLength vs minFirstSentenceLength)
	// makes sense because they measure two fundamentally different grammatical entities:
	// maxSectionHeaderLength applies to an entire paragraph ending in a colon to detect
	// structural section labels, whereas minFirstSentenceLength applies to the first
	// sentence of a descriptive paragraph to detect title-like sentence fragments.
	maxSectionHeaderLength = 40

	// minFirstSentenceLength is the minimum character threshold for the first sentence of
	// a paragraph to stand on its own as a meaningful description. When an initial sentence
	// is shorter than this threshold (e.g. "Google Batch Service.", 21 characters), it is
	// usually just an entity title rather than an explanation of what the service does.
	// In such cases, the extractor includes the subsequent sentence (e.g. "The service manages
	// user submitted batch jobs...") to provide actionable context. Real standalone summary
	// sentences (such as "Manages Stackdriver dashboards.", 31 characters) exceed this threshold.
	minFirstSentenceLength = 30
)

// extractServiceDescription extracts a concise single-line description from
// service documentation comments using the CommonMark (goldmark) parser.
//
// Documentation comments in Google Cloud APIs are written in CommonMark. A naive
// strategy of extracting the first paragraph in full fails in several common cases:
//  1. Title-only first paragraphs: Many services (such as SecretManagerService
//     and EkmServiceClient) begin with a title line that lacks sentence-ending
//     punctuation (e.g. "Secret Manager Service" or "Google Cloud Key Management
//     EKM Service"). Using the first paragraph merely repeats the client name
//     without explaining what the service does. When this pattern is detected,
//     the function skips the title line and extracts from the following paragraph.
//  2. Multi-sentence or overly detailed paragraphs: Other services (such as
//     AutoKeyClient) start with a long paragraph detailing implementation
//     mechanics, architecture, and resource associations. Extracting the full
//     paragraph clutters README and landing page overviews. Extracting only the
//     first sentence yields a punchy, high-level summary.
//  3. Proto cross-reference links: Comments often embed reference-style links
//     such as `[CryptoKey][google.cloud.kms.v1.CryptoKey]`. Because their
//     definitions are omitted from overview pages, reference targets must be
//     stripped so only the display text remains.
//  4. Section headers: Some services start with generic section headers such
//     as "API Overview:" which should be skipped.
func extractServiceDescription(doc string, serviceName string) string {
	if strings.TrimSpace(doc) == "" {
		return fmt.Sprintf("Client for the %s.", serviceName)
	}

	source := []byte(doc)
	md := goldmark.New()
	root := md.Parser().Parse(text.NewReader(source))

	var candidates []string
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != ast.KindParagraph {
			continue
		}
		pText := extractNodeText(child, source)
		if pText == "" {
			continue
		}
		// Skip short section header paragraphs ending in ":" like "API Overview:"
		if strings.HasSuffix(pText, ":") && len(pText) < maxSectionHeaderLength && !strings.Contains(pText, ". ") {
			continue
		}
		candidates = append(candidates, pText)
	}

	if len(candidates) == 0 {
		return fmt.Sprintf("Client for the %s.", serviceName)
	}

	selected := candidates[0]
	// If the first paragraph is just a title without sentence-ending punctuation,
	// and a subsequent paragraph exists, prefer the subsequent paragraph.
	if !strings.HasSuffix(selected, ".") && !strings.HasSuffix(selected, "!") && !strings.HasSuffix(selected, "?") && len(candidates) > 1 {
		selected = candidates[1]
	}

	result := extractFirstSentence(selected)
	if result == "" || result == "." {
		return fmt.Sprintf("Client for the %s.", serviceName)
	}
	return result
}

// extractNodeText walks the inlines of an AST node and concatenates their text,
// converting soft line breaks to spaces and stripping proto cross-reference targets.
func extractNodeText(node ast.Node, source []byte) string {
	var b strings.Builder
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := n.(*ast.Text); ok {
			b.Write(t.Segment.Value(source))
			if t.SoftLineBreak() {
				b.WriteByte(' ')
			}
		}
		return ast.WalkContinue, nil
	})
	s := b.String()
	s = reDocXref.ReplaceAllString(s, "$1")
	return strings.Join(strings.Fields(s), " ")
}

// extractFirstSentence extracts the first sentence of a paragraph. If the first
// sentence is very short (< minFirstSentenceLength chars), it attempts to include the second sentence.
func extractFirstSentence(s string) string {
	findTerminator := func(text string) int {
		for i := 0; i < len(text)-1; i++ {
			if (text[i] == '.' || text[i] == '?' || text[i] == '!') && text[i+1] == ' ' {
				return i
			}
		}
		return -1
	}

	idx := findTerminator(s)
	if idx == -1 {
		if !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, "!") && !strings.HasSuffix(s, "?") {
			s = strings.TrimSuffix(s, ":")
			return strings.TrimSpace(s) + "."
		}
		return s
	}

	if idx < minFirstSentenceLength {
		rest := s[idx+2:]
		if secIdx := findTerminator(rest); secIdx != -1 {
			return s[:idx+2+secIdx+1]
		}
		if strings.HasSuffix(rest, ".") || strings.HasSuffix(rest, "!") || strings.HasSuffix(rest, "?") {
			return s
		}
	}
	return s[:idx+1]
}
