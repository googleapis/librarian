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
	mdLinkRegex            = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	numberedListRe         = regexp.MustCompile(`^\d+\.\s+`)
	fencedCodeRegex        = regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\n(.*?)\n```")
	reColonNewline         = regexp.MustCompile(`:\n([^\n])`)
	rawHTMLTagRegex        = regexp.MustCompile(`<[a-zA-Z][a-zA-Z0-9-]*(?:\s+[^>]*)?>`)
	reStandaloneAsterisk   = regexp.MustCompile(`([(\s])\*([)\s,?])`)
	reStandaloneUnderscore = regexp.MustCompile(`([(\s])_([)\s,?])`)
	reDomainGlob           = regexp.MustCompile(`([a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+)\.\*`)
	reDotAsteriskWord      = regexp.MustCompile(`\.\*([A-Z][a-zA-Z0-9]+)\.\*`)
	reBracedEmphasis       = regexp.MustCompile(`\}_([a-zA-Z0-9_{}]+)_\{`)
	reQuotedUnderscore     = regexp.MustCompile(`"_"`)
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

func formatMessageDocLines(doc string) []string {
	if strings.TrimSpace(doc) == "" {
		return nil
	}
	return formatRstDoc(doc, messageDocWidth, messageDocIndent)
}

func formatFieldDocLines(doc string) []string {
	if strings.TrimSpace(doc) == "" {
		return nil
	}
	return formatRstDoc(doc, fieldDocWidth, fieldDocIndent)
}

func formatRstDoc(text string, width, indent int) []string {
	hasFormatting := strings.ContainsAny(text, "|*`_[]")
	converted := convertMarkdownToRst(text)

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
		if strings.HasPrefix(p, "     ") {
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
				subsequentIndent = 2
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
				isBulletOnly := trimmedCur == "-" || trimmedCur == "*" || trimmedCur == "+" || numberedListRe.MatchString(trimmedCur+" ")

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
	text = strings.ReplaceAll(text, "\n ", "\n")

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
		tok = strings.TrimSpace(tok)
		if tok == "" {
			if len(result) > 0 {
				result = append(result, "")
			}
			continue
		}
		listIndent := getSubsequentLineIndentLevel(tok)
		words := strings.Fields(tok)
		if len(words) == 0 {
			continue
		}
		curWidth := availableWidth
		var curLine strings.Builder
		for _, w := range words {
			if curLine.Len() == 0 {
				curLine.WriteString(w)
			} else if curLine.Len()+1+len(w) <= curWidth {
				curLine.WriteByte(' ')
				curLine.WriteString(w)
			} else {
				result = append(result, curLine.String())
				curLine.Reset()
				if listIndent > 0 {
					curLine.WriteString(strings.Repeat(" ", listIndent))
				}
				curLine.WriteString(w)
				curWidth = availableWidth
			}
		}
		if curLine.Len() > 0 {
			result = append(result, curLine.String())
		}
	}
	return result
}

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

	shortThreshold := int(float64(width) * 0.75)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			tokens = append(tokens, "")
			continue
		}
		if isListItem(trimmed) && current.Len() > 0 {
			flush()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(trimmed)
		if len(trimmed) < shortThreshold || strings.HasSuffix(trimmed, ":") {
			flush()
		}
	}
	flush()
	return tokens
}

func convertMarkdownToRst(text string) string {
	// Normalize bullet items: lines starting with * or + become -
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		leadSpaces := line[:len(line)-len(trimmed)]
		if strings.HasPrefix(trimmed, "* ") {
			lines[idx] = leadSpaces + "- " + trimmed[2:]
		} else if strings.HasPrefix(trimmed, "+ ") {
			lines[idx] = leadSpaces + "- " + trimmed[2:]
		}
	}
	text = strings.Join(lines, "\n")

	text = rawHTMLTagRegex.ReplaceAllString(text, "")
	text = reQuotedUnderscore.ReplaceAllString(text, `"*"`)
	text = reDomainGlob.ReplaceAllString(text, `$1.\*`)
	text = reDotAsteriskWord.ReplaceAllString(text, `.\ *$1.*`)
	text = reBracedEmphasis.ReplaceAllString(text, `}\ *$1*\ {`)
	text = reStandaloneAsterisk.ReplaceAllString(text, `$1\*$2`)
	text = reStandaloneUnderscore.ReplaceAllString(text, `$1\_$2`)

	text = fencedCodeRegex.ReplaceAllStringFunc(text, func(m string) string {
		sub := fencedCodeRegex.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		rawCode := sub[1]
		lines := strings.Split(rawCode, "\n")
		var indented []string
		for _, l := range lines {
			indented = append(indented, "   "+l)
		}
		return "\n\n::\n\n" + strings.Join(indented, "\n") + "\n\n"
	})

	// 1. Convert single backticks to double backticks: `code` -> ``code``
	// preserving `oneof`_, `...`__, and existing ``...``.
	var sb strings.Builder
	runes := []rune(text)
	n := len(runes)
	i := 0
	for i < n {
		if runes[i] == '`' {
			if i+1 < n && runes[i+1] == '`' {
				// Double backtick: keep as-is until closing double backtick.
				sb.WriteString("``")
				i += 2
				for i < n {
					if runes[i] == '`' && i+1 < n && runes[i+1] == '`' {
						sb.WriteString("``")
						i += 2
						break
					}
					if runes[i] == ' ' {
						sb.WriteRune('\uE000')
					} else {
						sb.WriteRune(runes[i])
					}
					i++
				}
				continue
			}
			// Single backtick. Find matching single backtick, possibly across newlines.
			j := i + 1
			for j < n && runes[j] != '`' {
				if runes[j] == '\n' && j+1 < n && runes[j+1] == '\n' {
					break
				}
				j++
			}
			if j < n && runes[j] == '`' {
				// Check if followed by '_' or '__'
				isRef := false
				if j+1 < n && runes[j+1] == '_' {
					isRef = true
				}
				inner := string(runes[i+1 : j])
				inner = strings.Join(strings.Fields(inner), " ")
				if isRef {
					sb.WriteRune('`')
					sb.WriteString(inner)
					sb.WriteRune('`')
					i = j + 1
				} else {
					sb.WriteString("``")
					innerProtected := strings.ReplaceAll(inner, " ", "\uE000")
					sb.WriteString(innerProtected)
					sb.WriteString("``")
					i = j + 1
				}
				continue
			}
			sb.WriteRune(runes[i])
			i++
		} else {
			sb.WriteRune(runes[i])
			i++
		}
	}
	res := sb.String()

	// 2. Convert markdown links: [text](url) -> `text <url>`__
	res = mdLinkRegex.ReplaceAllStringFunc(res, func(m string) string {
		sub := mdLinkRegex.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		rawText := sub[1]
		url := sub[2]
		if strings.HasPrefix(rawText, "``") && strings.HasSuffix(rawText, "``") {
			return "```" + strings.Trim(rawText, "`") + "``\uE000<" + url + ">`__"
		}
		linkText := strings.Trim(rawText, "`")
		words := strings.Fields(linkText)
		if len(words) == 0 {
			return "`<" + url + ">`__"
		}
		if len(words) == 1 {
			return "`" + words[0] + "\uE000<" + url + ">`__"
		}
		return "`" + strings.Join(words[:len(words)-1], " ") + " " + words[len(words)-1] + "\uE000<" + url + ">`__"
	})

	return res
}

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

// methodDocSummary contains the lead and optional rest of a method docstring summary.
// For short method names, Lead contains the entire humanized name and Wrap is false.
// For longer method names that exceed the first line budget (30 chars = 70 total - 40 prefix),
// Lead contains the first line and Rest contains the subsequent wrapped line(s).
type methodDocSummary struct {
	Lead string
	Rest string
	Wrap bool
}

func formatMethodDocSummary(methodName string) methodDocSummary {
	humanized := strings.ReplaceAll(snakeCase(methodName), "_", " ")
	const (
		totalWidth         = 70
		summaryIndent      = 8
		firstLinePrefixLen = 40 // 8 spaces indent + len(`r"""Return a callable for the `)
		suffixLen          = 18 // len(" method over gRPC.")
	)
	firstLineAvail := totalWidth - firstLinePrefixLen // 30 chars
	if len(humanized) <= firstLineAvail {
		return methodDocSummary{
			Lead: humanized,
			Wrap: false,
		}
	}

	words := strings.Fields(humanized)
	var line1Words []string
	currLen := 0
	splitIdx := 0
	for i, w := range words {
		addedLen := len(w)
		if len(line1Words) > 0 {
			addedLen++ // space separator
		}
		if len(line1Words) > 0 && currLen+addedLen > firstLineAvail {
			splitIdx = i
			break
		}
		line1Words = append(line1Words, w)
		currLen += addedLen
	}
	if splitIdx == 0 && len(words) > 0 {
		line1Words = []string{words[0]}
		splitIdx = 1
	}

	lead := strings.Join(line1Words, " ")
	restWords := words[splitIdx:]
	if len(restWords) == 0 {
		return methodDocSummary{
			Lead: lead,
			Wrap: false,
		}
	}

	restLines := wrapWords(strings.Join(restWords, " "), totalWidth-summaryIndent-suffixLen)
	rest := strings.Join(restLines, "\n"+strings.Repeat(" ", summaryIndent))

	return methodDocSummary{
		Lead: lead,
		Rest: rest,
		Wrap: true,
	}
}

func wrapWords(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var current strings.Builder
	for _, w := range words {
		wLen := len(w)
		if current.Len() == 0 {
			current.WriteString(w)
		} else if current.Len()+1+wLen <= width {
			current.WriteByte(' ')
			current.WriteString(w)
		} else {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(w)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

// formatRstDocLines converts documentation into a sequence of wrapped lines for Python docstrings.
// The returned lines are not indented with leading spaces, allowing Mustache templates to control
// horizontal indentation.
func formatRstDocLines(doc string, width, indent int) []string {
	if strings.TrimSpace(doc) == "" {
		return nil
	}
	return formatRstDoc(doc, width-indent, indent)
}
