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
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// formatDocumentation converts a documentation string from the source (typically Protobuf
// comments) into a sequence of lines suitable for Swift documentation comments.
//
// Both Swift and the Protobuf comments use markdown, but our markdown includes cross-reference
// links and sometimes needs cleaning up work correctly on a different markdown engine.
func (c *codec) formatDocumentation(doc string, scopes []string) ([]string, error) {
	if doc == "" {
		return nil, nil
	}
	results := strings.Split(doc, "\n")

	docBytes := []byte(doc)
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	node := md.Parser().Parse(text.NewReader(docBytes))
	definitions, _, err := c.resolveLinkDefinitions(node, docBytes, scopes, nil)
	if err != nil {
		return nil, err
	}
	return appendLinkDefinitions(results, definitions), nil
}

func (c *codec) resolveLinkDefinitions(node ast.Node, docBytes []byte, scopes []string, rewrite func(string) string) ([]string, map[string]bool, error) {
	links := language.ExtractCrossReferenceLinks(node, docBytes)
	var definitions []string
	resolvedLinks := make(map[string]bool, len(links))
	for _, link := range links {
		resolved, err := c.linkDefinition(link, scopes)
		if err != nil {
			return nil, nil, err
		}
		if resolved == "" {
			continue
		}
		resolvedLinks[link] = true
		if rewrite != nil {
			resolved = rewrite(resolved)
		}
		definitions = append(definitions, fmt.Sprintf("[%s]: %s", link, resolved))
	}
	return definitions, resolvedLinks, nil
}

func appendLinkDefinitions(lines, definitions []string) []string {
	if len(definitions) == 0 {
		return lines
	}
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		lines = append(lines, "") // Add a blank line before link definitions, if any
	}
	return append(lines, definitions...)
}

type textRange struct {
	start int
	stop  int
}

// formatReadmeDocumentation converts a documentation string into a sequence of
// lines suitable for GitHub-flavored Markdown README files.
//
// Resolved proto cross-reference links receive Swift Package Index DocC URL
// definitions, while unresolved proto cross-reference links without Markdown
// link definitions are replaced in place with their display text.
func (c *codec) formatReadmeDocumentation(doc string, scopes []string) ([]string, error) {
	if doc == "" {
		return nil, nil
	}
	docBytes := []byte(doc)
	md := goldmark.New()
	ctx := parser.NewContext()
	root := md.Parser().Parse(text.NewReader(docBytes), parser.WithContext(ctx))

	definitions, resolvedLinks, err := c.resolveLinkDefinitions(root, docBytes, scopes, c.swiftPackageIndexDocURL)
	if err != nil {
		return nil, err
	}

	var ranges []textRange
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node.Kind() {
		case ast.KindParagraph, ast.KindTextBlock, ast.KindHeading:
			lines := node.Lines()
			if lines.Len() > 0 {
				ranges = append(ranges, textRange{
					start: lines.At(0).Start,
					stop:  lines.At(lines.Len() - 1).Stop,
				})
			}
		}
		return ast.WalkContinue, nil
	})
	if len(ranges) == 0 {
		return appendLinkDefinitions(strings.Split(doc, "\n"), definitions), nil
	}
	slices.SortFunc(ranges, func(a, b textRange) int {
		return cmp.Compare(a.start, b.start)
	})

	var out strings.Builder
	cursor := 0
	for _, r := range ranges {
		if r.start < cursor {
			continue
		}
		out.Write(docBytes[cursor:r.start])
		out.WriteString(stripUnresolvedProtoCrossRefs(string(docBytes[r.start:r.stop]), ctx, resolvedLinks))
		cursor = r.stop
	}
	out.Write(docBytes[cursor:])
	return appendLinkDefinitions(strings.Split(out.String(), "\n"), definitions), nil
}

func (c *codec) swiftPackageIndexDocURL(target string) string {
	if !strings.HasPrefix(target, "<doc:") || !strings.HasSuffix(target, ">") {
		return target
	}
	path := strings.TrimSuffix(strings.TrimPrefix(target, "<doc:"), ">")
	return fmt.Sprintf(
		"https://swiftpackageindex.com/googleapis/%s/%s/documentation/%s/%s",
		c.PackageRepoName,
		c.PackageVersion,
		strings.ToLower(c.LibraryName),
		strings.ToLower(path),
	)
}

func stripUnresolvedProtoCrossRefs(s string, ctx parser.Context, resolvedLinks map[string]bool) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		switch s[i] {
		case '\\':
			if i+1 < len(s) {
				out.WriteString(s[i : i+2])
				i += 2
				continue
			}
		case '`':
			end := skipCodeSpan(s, i)
			out.WriteString(s[i:end])
			i = end
			continue
		case '[':
			if display, end, ok := matchUnresolvedProtoCrossRef(s[i:], ctx, resolvedLinks); ok {
				out.WriteString(display)
				i += end
				continue
			}
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

func skipCodeSpan(s string, start int) int {
	tickLen := 0
	for start+tickLen < len(s) && s[start+tickLen] == '`' {
		tickLen++
	}
	for i := start + tickLen; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		closeLen := 0
		for i+closeLen < len(s) && s[i+closeLen] == '`' {
			closeLen++
		}
		if closeLen == tickLen {
			return i + closeLen
		}
		i += closeLen
	}
	return start + tickLen
}

func matchUnresolvedProtoCrossRef(s string, ctx parser.Context, resolvedLinks map[string]bool) (string, int, bool) {
	pos := 1
	displayEnd := -1
	for pos < len(s) {
		switch s[pos] {
		case '\\':
			if pos+1 < len(s) {
				pos += 2
				continue
			}
		case '`':
			pos = skipCodeSpan(s, pos)
			continue
		case '[':
			return "", 0, false
		case ']':
			displayEnd = pos
		}
		if displayEnd != -1 {
			break
		}
		pos++
	}
	if displayEnd <= 1 {
		return "", 0, false
	}
	display := s[1:displayEnd]
	targetStart := displayEnd + 1
	if targetStart >= len(s) || s[targetStart] != '[' {
		return "", 0, false
	}
	targetEnd := -1
	for i := targetStart + 1; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			continue
		}
		if s[i] == '[' {
			return "", 0, false
		}
		if s[i] == ']' {
			targetEnd = i
			break
		}
	}
	if targetEnd == -1 {
		return "", 0, false
	}
	if targetEnd+1 < len(s) && s[targetEnd+1] == '(' {
		return "", 0, false
	}
	ref := s[targetStart+1 : targetEnd]
	if ref == "" {
		ref = display
	}
	if _, defined := ctx.Reference(util.ToLinkReference([]byte(ref))); defined {
		return "", 0, false
	}
	norm := strings.Join(strings.Fields(ref), " ")
	if resolvedLinks[norm] || !isProtoCrossRef(norm) {
		return "", 0, false
	}
	return display, targetEnd + 1, true
}

func isProtoCrossRef(s string) bool {
	return protoreflect.FullName(strings.TrimPrefix(s, ".")).IsValid()
}
