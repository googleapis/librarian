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
	reColonNewline            = regexp.MustCompile(`:\n([^\n])`)
	reUnformattedNumberedList = regexp.MustCompile(`^\d+\.\s+`)
)

// wrapUnformatted formats docstring paragraphs without Markdown formatting using Python textwrap rules.
func wrapUnformatted(text string, width, indent int) []string {
	if text == "" {
		return nil
	}
	offset := indent + 3

	firstRaw := strings.Split(text, "\n")[0] + "\n"
	if strings.HasSuffix(firstRaw, ":\n") {
		firstRaw += "\n"
	}
	first := firstRaw

	if len(firstRaw) > width-offset {
		initChunks := textwrapFill(firstRaw, width-offset, "", "")
		if len(initChunks) > 0 {
			if strings.Contains(text, "\n") {
				parts := strings.SplitN(text, "\n", 2)
				if len(parts) > 1 && !isListItem(strings.TrimSpace(parts[1])) {
					text = strings.Replace(text, "\n", " ", 1)
				}
			}
			first = initChunks[0] + "\n"
		}
	}

	text = reColonNewline.ReplaceAllString(text, ":\n\n$1")
	if len(text) > len(first) {
		text = text[len(first):]
	} else {
		text = ""
	}
	if strings.TrimSpace(text) == "" {
		firstClean := strings.TrimSpace(first)
		if firstClean == "" {
			return nil
		}
		return []string{firstClean}
	}

	newLine := ""
	if strings.HasPrefix(text, "\n") {
		newLine = "\n"
	}
	text = newLine + strings.TrimSpace(text)

	var tokens []string
	var token strings.Builder
	lines := strings.Split(text, "\n")
	threshold := int(float64(width) * 0.75)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if (isListItem(trimmed) || len(line) == 0) && token.Len() > 0 {
			tokens = append(tokens, token.String())
			token.Reset()
		}
		token.WriteString(line)
		token.WriteByte('\n')

		if len(line) < threshold || strings.HasSuffix(line, ":") {
			tokens = append(tokens, token.String())
			token.Reset()
		}
	}
	if token.Len() > 0 {
		tokens = append(tokens, token.String())
	}

	firstTrimmed := strings.TrimRight(first, "\n")
	result := []string{firstTrimmed}
	for i := 1; i < len(first)-len(firstTrimmed); i++ {
		result = append(result, "")
	}
	initialIndent := strings.Repeat(" ", indent)
	for _, tok := range tokens {
		subIndent := initialIndent + strings.Repeat(" ", getSubsequentLineIndentLevel(strings.TrimSpace(tok)))
		filled := textwrapFill(tok, width, initialIndent, subIndent)
		if len(filled) == 0 {
			result = append(result, "")
		} else {
			for _, fl := range filled {
				result = append(result, strings.TrimPrefix(fl, initialIndent))
			}
		}
	}

	return result
}

// textwrapFill emulates Python's textwrap.fill algorithm with initial and subsequent indentation.
func textwrapFill(text string, width int, initialIndent, subsequentIndent string) []string {
	text = strings.ReplaceAll(text, "\t", "    ")
	var sb strings.Builder
	for _, r := range text {
		if unicode.IsSpace(r) {
			sb.WriteByte(' ')
		} else {
			sb.WriteRune(r)
		}
	}
	chunks := splitWordsAndSpaces(sb.String())
	if len(chunks) == 0 || (len(chunks) == 1 && strings.TrimSpace(chunks[0]) == "") {
		return nil
	}

	var lines []string
	i := 0
	n := len(chunks)
	for i < n {
		indent := initialIndent
		if len(lines) > 0 {
			indent = subsequentIndent
		}
		avail := width - len(indent)

		if len(lines) > 0 && strings.TrimSpace(chunks[i]) == "" {
			i++
			continue
		}

		var curLine []string
		curLen := 0
		for i < n {
			c := chunks[i]
			cl := len(c)
			if curLen+cl <= avail {
				curLine = append(curLine, c)
				curLen += cl
				i++
			} else {
				break
			}
		}

		if len(curLine) == 0 && i < n {
			curLine = append(curLine, chunks[i])
			i++
		}

		for len(curLine) > 0 && strings.TrimSpace(curLine[len(curLine)-1]) == "" {
			curLine = curLine[:len(curLine)-1]
		}

		if len(curLine) > 0 {
			lines = append(lines, indent+strings.Join(curLine, ""))
		}
	}
	return lines
}

// splitWordsAndSpaces splits string s into alternating word and whitespace chunks.
func splitWordsAndSpaces(s string) []string {
	var chunks []string
	runes := []rune(s)
	n := len(runes)
	i := 0
	for i < n {
		isSpace := unicode.IsSpace(runes[i])
		start := i
		for i < n && unicode.IsSpace(runes[i]) == isSpace {
			i++
		}
		chunks = append(chunks, string(runes[start:i]))
	}
	return chunks
}

// isListItem reports whether string s starts with a list bullet or numbered list prefix.
func isListItem(s string) bool {
	if len(s) < 3 {
		return false
	}
	return strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "+ ") || reUnformattedNumberedList.MatchString(s)
}

// getSubsequentLineIndentLevel returns the indentation width for subsequent lines of a list item.
func getSubsequentLineIndentLevel(s string) int {
	if len(s) >= 2 && (s[0:2] == "- " || s[0:2] == "+ ") {
		return 2
	}
	if len(s) >= 4 && reUnformattedNumberedList.MatchString(s) {
		return 4
	}
	return 0
}
