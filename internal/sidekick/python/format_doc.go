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
)

var (
	numberedListRe       = regexp.MustCompile(`^\d+[.)]\s+`)
	reNumberedListPrefix = regexp.MustCompile(`^\d+[.)]`)
	reNumberedListOnly   = regexp.MustCompile(`^\d+[.)]$`)
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
	return formatRstDocInternal(doc, fieldDocWidth, fieldDocIndent, true, true)
}

// formatRstDoc converts documentation text to reStructuredText format and wraps it to the target width.
func formatRstDoc(text string, width, indent int) []string {
	return formatRstDocInternal(text, width, indent, true, false)
}

// formatReturnRstDoc converts method return docstrings where Markdown links are not rewritten.
func formatReturnRstDoc(text string, width, indent int) []string {
	return formatRstDocInternal(text, width, indent, false, false)
}

// formatRstDocInternal converts text to RST, optionally rewriting markdown links, and wraps lines.
func formatRstDocInternal(text string, width, indent int, convertLinks, isFieldDoc bool) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	hasFormatting := strings.ContainsAny(text, "|*`_[]")
	var lines []string
	if hasFormatting {
		text, literalBlocks := extractIndentedLiteralBlocks(text)
		converted := convertMarkdownToRstWithOptions(text, convertLinks)
		lines = wrapCommonMark(converted, width, literalBlocks)
	} else {
		lines = wrapUnformatted(text, width, indent)
	}

	var result []string
	consecutiveBlanks := 0
	lastNonBlank := ""
	inLiteralBlock := false
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			consecutiveBlanks++
			if inLiteralBlock {
				result = append(result, line)
				continue
			}
			nextIsIndented := false
			for k := idx + 1; k < len(lines); k++ {
				if strings.TrimSpace(lines[k]) != "" {
					nextIsIndented = strings.HasPrefix(lines[k], " ")
					break
				}
			}
			if strings.HasSuffix(lastNonBlank, ":") || strings.HasSuffix(lastNonBlank, "::") || strings.HasSuffix(lastNonBlank, "as follows.") {
				if consecutiveBlanks <= 3 {
					result = append(result, line)
				}
			} else if nextIsIndented {
				if consecutiveBlanks <= 2 {
					result = append(result, line)
				}
			} else {
				if consecutiveBlanks <= 1 {
					result = append(result, line)
				}
			}
		} else {
			consecutiveBlanks = 0
			if inLiteralBlock && !strings.HasPrefix(line, " ") {
				inLiteralBlock = false
			}
			if trimmed == "::" {
				inLiteralBlock = true
			}
			lastNonBlank = trimmed
			result = append(result, line)
		}
	}
	for len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}
	if len(result) > 0 {
		lastIdx := len(result) - 1
		for lastIdx >= 0 && result[lastIdx] == "" {
			lastIdx--
		}
		if lastIdx >= 0 && strings.HasSuffix(result[lastIdx], `"`) {
			if isFieldDoc || len(result) == 1 {
				result[lastIdx] += "."
			}
		}
	}
	return result
}

func wrapCommonMark(text string, width int, literalBlocks [][]string) []string {
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

		pLeadSpaces := len(p) - len(strings.TrimLeft(p, " \t"))
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

		if idx, ok := parseLiteralBlockIndex(p); ok && idx < len(literalBlocks) {
			hasColonColon := false
			if len(result) >= 1 && result[len(result)-1] == "::" {
				result = append(result, "")
				hasColonColon = true
			} else if len(result) >= 2 && result[len(result)-2] == "::" && result[len(result)-1] == "" {
				hasColonColon = true
			}
			if !hasColonColon {
				result = append(result, "::")
				result = append(result, "")
			}
			result = append(result, literalBlocks[idx]...)
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

		for rlIdx, rl := range rawLines {
			trimmed := strings.TrimSpace(rl)
			if trimmed == "" {
				continue
			}
			leadSpaces := len(rl) - len(strings.TrimLeft(rl, " "))
			if rlIdx == 0 {
				leadSpaces = pLeadSpaces
			}

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
				if leadSpaces >= 4 || (leadSpaces >= 3 && (curKind == kindTopList || curKind == kindNestedList || (pIdx > 0 && strings.Contains(paragraphs[pIdx-1], "as in:")))) {
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

		isLooseList := strings.Contains(p, "as in:")
		var prevKind blockKind
		for bIdx, b := range blocks {
			if bIdx > 0 {
				if (prevKind == kindRegular && (b.kind == kindTopList || b.kind == kindNestedList)) ||
					((prevKind == kindTopList || prevKind == kindNestedList) && b.kind == kindRegular) ||
					(prevKind == kindTopList && b.kind == kindNestedList) ||
					(prevKind == kindNestedList && b.kind == kindTopList) ||
					(isLooseList && prevKind == kindTopList && b.kind == kindTopList) {
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
