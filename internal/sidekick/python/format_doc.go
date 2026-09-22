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

package python

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	numberedListRe       = regexp.MustCompile(`^\d+\.\s+`)
	reNumberedListPrefix = regexp.MustCompile(`^\d+\.`)
	reNumberedListOnly   = regexp.MustCompile(`^\d+\.$`)
	reColonNewline       = regexp.MustCompile(`:\n([^\n])`)
)

type blockKind int

const (
	kindRegular blockKind = iota
	kindTopList
	kindNestedList
)

type docBlock struct {
	kind blockKind
	text string
}

// formatDocLines splits documentation into lines, trimming trailing whitespace.
func formatDocLines(doc string) []string {
	if doc == "" {
		return nil
	}
	lines := strings.Split(doc, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, strings.TrimRight(line, " \t\r"))
	}
	return result
}

const (
	// messageDocWidth is the max wrap width for top-level message docstrings indented by messageDocIndent spaces.
	messageDocWidth  = 68
	messageDocIndent = 4
	// fieldDocWidth is the max wrap width for field attribute docstrings indented by fieldDocIndent spaces.
	fieldDocWidth  = 60
	fieldDocIndent = 12
)

// formatMessageDocLines converts a message's documentation into wrapped reStructuredText lines.
func formatMessageDocLines(doc string) []string {
	if strings.TrimSpace(doc) == "" {
		return nil
	}
	return formatRstDoc(doc, messageDocWidth, messageDocIndent)
}

// formatFieldDocLines converts a field attribute's documentation into wrapped reStructuredText lines.
func formatFieldDocLines(doc string) []string {
	if strings.TrimSpace(doc) == "" {
		return nil
	}
	return formatRstDoc(doc, fieldDocWidth, fieldDocIndent)
}

// formatRstDoc converts documentation text to reStructuredText format and wraps it to the target width.
func formatRstDoc(text string, width, indent int) []string {
	return formatRstDocInternal(text, width, indent, true)
}

// formatReturnRstDoc converts method return docstrings where Markdown links are not rewritten.
func formatReturnRstDoc(text string, width, indent int) []string {
	return formatRstDocInternal(text, width, indent, false)
}

// formatRstDocInternal converts text to RST, optionally rewriting markdown links, and wraps lines.
func formatRstDocInternal(text string, width, indent int, convertLinks bool) []string {
	hasFormatting := strings.ContainsAny(text, "|*`_[]")
	converted := convertMarkdownToRstWithOptions(text, convertLinks)
	var lines []string
	if hasFormatting {
		lines = wrapCommonMark(converted, width)
	} else {
		lines = wrapUnformatted(converted, width, indent)
	}

	var result []string
	for _, line := range lines {
		if line == "" {
			result = append(result, "")
			continue
		}
		result = append(result, line)
	}
	if len(result) > 0 {
		lastIdx := len(result) - 1
		for lastIdx >= 0 && result[lastIdx] == "" {
			lastIdx--
		}
		if lastIdx >= 0 && strings.HasSuffix(result[lastIdx], `"`) {
			result[lastIdx] += "."
		}
	}
	return result
}

func wrapCommonMark(text string, width int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var result []string

	for pIdx, p := range paragraphs {
		if strings.HasPrefix(p, "    ") {
			if pIdx > 0 && len(result) > 0 {
				result = append(result, "")
			}
			result = append(result, strings.Split(p, "\n")...)
			continue
		}

		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if pIdx > 0 && len(result) > 0 {
			result = append(result, "")
		}

		if p == "::" {
			result = append(result, "::")
			continue
		}

		rawLines := strings.Split(p, "\n")
		var blocks []docBlock
		var curBlock strings.Builder
		curKind := kindRegular

		flushBlock := func() {
			if curBlock.Len() > 0 {
				blocks = append(blocks, docBlock{
					kind: curKind,
					text: strings.TrimSpace(curBlock.String()),
				})
				curBlock.Reset()
			}
		}

		for _, rl := range rawLines {
			trimmed := strings.TrimSpace(rl)
			if trimmed == "" {
				continue
			}
			leadSpaces := len(rl) - len(strings.TrimLeft(rl, " "))

			if (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ")) &&
				reNumberedListPrefix.MatchString(trimmed[2:]) {
				flushBlock()
				blocks = append(blocks, docBlock{
					kind: kindTopList,
					text: "-",
				})
				curKind = kindNestedList
				curBlock.WriteString(trimmed[2:])
				continue
			}

			isList := false
			if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
				isList = true
			} else if numberedListRe.MatchString(trimmed) {
				isList = true
			}

			if isList {
				lineKind := kindTopList
				if leadSpaces >= 4 {
					lineKind = kindNestedList
				}
				flushBlock()
				curKind = lineKind
				curBlock.WriteString(trimmed)
			} else {
				if curBlock.Len() == 0 {
					curKind = kindRegular
					curBlock.WriteString(trimmed)
				} else {
					curBlock.WriteByte(' ')
					curBlock.WriteString(trimmed)
				}
			}
		}
		flushBlock()

		var prevKind blockKind
		for bIdx, b := range blocks {
			if bIdx > 0 {
				if (prevKind == kindRegular && (b.kind == kindTopList || b.kind == kindNestedList)) ||
					((prevKind == kindTopList || prevKind == kindNestedList) && b.kind == kindRegular) ||
					(prevKind == kindTopList && b.kind == kindNestedList) ||
					(prevKind == kindNestedList && b.kind == kindTopList) {
					result = append(result, "")
				}
			}
			prevKind = b.kind

			initialIndent := 0
			subsequentIndent := 0
			switch b.kind {
			case kindTopList:
				if m := numberedListRe.FindString(b.text); m != "" {
					subsequentIndent = len(m)
				} else {
					subsequentIndent = 2
				}
			case kindNestedList:
				initialIndent = 2
				subsequentIndent = 4
			}

			words := strings.Fields(b.text)
			if len(words) == 0 {
				continue
			}

			var curLine strings.Builder
			if initialIndent > 0 {
				curLine.WriteString(strings.Repeat(" ", initialIndent))
			}

			for _, w := range words {
				wDisplay := strings.ReplaceAll(w, "\uE000", " ")
				wLen := len([]rune(wDisplay))

				trimmedCur := strings.TrimSpace(curLine.String())
				isBulletOnly := trimmedCur == "-" || trimmedCur == "*" || trimmedCur == "+" || reNumberedListOnly.MatchString(trimmedCur)

				if curLine.Len() == initialIndent || curLine.Len() == 0 {
					curLine.WriteString(wDisplay)
				} else if isBulletOnly || len([]rune(curLine.String()))+1+wLen <= width {
					curLine.WriteByte(' ')
					curLine.WriteString(wDisplay)
				} else {
					result = append(result, curLine.String())
					curLine.Reset()
					if subsequentIndent > 0 {
						curLine.WriteString(strings.Repeat(" ", subsequentIndent))
					}
					curLine.WriteString(wDisplay)
				}
			}
			if curLine.Len() > 0 {
				result = append(result, curLine.String())
			}
		}
	}
	return result
}

func wrapUnformatted(text string, width, indent int) []string {
	text = reColonNewline.ReplaceAllString(text, ":\n\n$1")
	offset := indent + 3

	// Protocol buffers preserves single initial spaces after line breaks
	// when parsing comments. Re-wrapping causes these to be two spaces;
	// correct for this, but preserve lines starting with two or more spaces (examples/code).
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "  ") {
			lines[i] = line[1:]
		}
	}
	text = strings.Join(lines, "\n")

	// Break off first line
	rawLines := strings.Split(text, "\n")
	firstRaw := rawLines[0]
	var firstLine string
	var remainder string

	firstLimit := width - offset
	if len(firstRaw) > firstLimit {
		words := strings.Fields(firstRaw)
		var cur strings.Builder
		wIdx := 0
		for wIdx < len(words) {
			w := words[wIdx]
			if cur.Len() == 0 {
				cur.WriteString(w)
			} else if cur.Len()+1+len(w) <= firstLimit {
				cur.WriteByte(' ')
				cur.WriteString(w)
			} else {
				break
			}
			wIdx++
		}
		var restOfFirst string
		if wIdx > 0 {
			searchPos := 0
			for i := 0; i < wIdx; i++ {
				pos := strings.Index(firstRaw[searchPos:], words[i])
				if pos != -1 {
					searchPos += pos + len(words[i])
				}
			}
			firstLine = strings.TrimSpace(firstRaw[:searchPos])
			restOfFirst = strings.TrimSpace(firstRaw[searchPos:])
		} else {
			firstLine = cur.String()
			restOfFirst = strings.Join(words[wIdx:], " ")
		}
		restOfText := strings.Join(rawLines[1:], "\n")
		switch {
		case restOfFirst != "" && restOfText != "":
			remainder = restOfFirst + " " + restOfText
		case restOfFirst != "":
			remainder = restOfFirst
		default:
			remainder = restOfText
		}
	} else {
		firstLine = firstRaw
		remainder = strings.Join(rawLines[1:], "\n")
	}

	var result []string
	if strings.TrimSpace(firstLine) != "" {
		result = append(result, strings.TrimSpace(firstLine))
	}

	if strings.TrimSpace(remainder) == "" {
		return result
	}

	availableWidth := width - indent
	tokens := tokenizeParagraph(remainder, availableWidth)
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "  ") {
			result = append(result, tok)
			continue
		}
		tok = strings.TrimSpace(tok)
		if tok == "" {
			if len(result) > 0 {
				result = append(result, "")
			}
			continue
		}
		listIndent := getSubsequentLineIndentLevel(tok)
		toks := parseTokensWithSpace(tok)
		if len(toks) == 0 {
			continue
		}
		curWidth := availableWidth
		var curLine strings.Builder
		for _, t := range toks {
			space := t.lead
			if space == "" {
				space = " "
			}
			wLen := len([]rune(t.text))
			sLen := len([]rune(space))
			if curLine.Len() == 0 {
				curLine.WriteString(t.text)
			} else if len([]rune(curLine.String()))+sLen+wLen <= curWidth {
				curLine.WriteString(space)
				curLine.WriteString(t.text)
			} else {
				result = append(result, curLine.String())
				curLine.Reset()
				if listIndent > 0 {
					curLine.WriteString(strings.Repeat(" ", listIndent))
				}
				curLine.WriteString(t.text)
				curWidth = availableWidth
			}
		}
		if curLine.Len() > 0 {
			result = append(result, curLine.String())
		}
	}
	return result
}

// tokenWithSpace holds a parsed token along with its leading whitespace.
type tokenWithSpace struct {
	lead string
	text string
}

// parseTokensWithSpace decomposes a string into tokens while preserving interstitial whitespace.
func parseTokensWithSpace(s string) []tokenWithSpace {
	var tokens []tokenWithSpace
	runes := []rune(s)
	n := len(runes)
	i := 0
	for i < n && unicode.IsSpace(runes[i]) {
		i++
	}
	lead := ""
	for i < n {
		start := i
		for i < n && !unicode.IsSpace(runes[i]) {
			i++
		}
		word := string(runes[start:i])
		tokens = append(tokens, tokenWithSpace{lead: lead, text: word})
		startSpace := i
		for i < n && unicode.IsSpace(runes[i]) {
			i++
		}
		lead = string(runes[startSpace:i])
	}
	return tokens
}

// shortLineThresholdRatio defines the fraction of line wrap width below which
// a line is treated as terminal and flushed, preventing distinct sentences from being
// merged inappropriately during docstring paragraph tokenization.
const shortLineThresholdRatio = 0.80

func tokenizeParagraph(text string, width int) []string {
	lines := strings.Split(text, "\n")
	var tokens []string
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}

	shortThreshold := int(float64(width) * shortLineThresholdRatio)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			tokens = append(tokens, "")
			continue
		}
		if strings.HasPrefix(line, "  ") {
			flush()
			tokens = append(tokens, line)
			continue
		}
		if isListItem(trimmed) && current.Len() > 0 {
			flush()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(trimmed)
		if len(trimmed) <= shortThreshold || strings.HasSuffix(trimmed, ":") {
			flush()
		}
	}
	flush()
	return tokens
}

// isListItem reports whether a line begins with a bullet or numbered list marker.
func isListItem(s string) bool {
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "+ ") || strings.HasPrefix(s, "* ") {
		return true
	}
	return numberedListRe.MatchString(s)
}

func getSubsequentLineIndentLevel(s string) int {
	if len(s) >= 2 && (s[0:2] == "- " || s[0:2] == "+ " || s[0:2] == "* ") {
		return 2
	}
	if len(s) >= 4 && numberedListRe.MatchString(s) {
		return 4
	}
	return 0
}
