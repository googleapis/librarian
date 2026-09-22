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
	fencedCodeRegex        = regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\n(.*?)\n```")
	rawHTMLTagRegex        = regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9-]*(?:\s+[^>]*)?>`)
	reStandaloneAsterisk   = regexp.MustCompile(`([(\s'])\*([)'\s,?])`)
	reStandaloneUnderscore = regexp.MustCompile(`([(\s'])_([)'\s,?])`)
	reDomainGlob           = regexp.MustCompile(`([a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+)\.\*`)
	reDotAsteriskWord      = regexp.MustCompile(`\.\*([A-Z][a-zA-Z0-9]+)\.\*`)
	reBracedEmphasis       = regexp.MustCompile(`\}_([a-zA-Z0-9_{}]+)_\{`)
	reQuotedUnderscore     = regexp.MustCompile(`"_"`)
)

// convertMarkdownToRst converts CommonMark Markdown syntax in proto comments to
// reStructuredText syntax used in Python docstrings.
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

// methodDocSummary contains the lead and optional rest of a method docstring summary.
// For short method names, Lead contains the entire humanized name and Wrap is false.
// For longer method names that exceed the first line budget (30 chars = 70 total - 40 prefix),
// Lead contains the first line and Rest contains the subsequent wrapped line(s).
type methodDocSummary struct {
	Lead string
	Rest string
	Wrap bool
}

// formatMethodDocSummary computes the lead line and wrapped continuation lines
// for a method docstring summary.
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

// wrapWords wraps space-delimited text into lines not exceeding the specified character width.
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

// definitionIndent defines the 3-space indentation required for Sphinx reST definition list bodies.
const definitionIndent = "   "

// formatMethodReturnDoc formats docstrings for method return values, handling Sphinx definition list
// structures where line 1 is the term and subsequent lines are indented definitions.
func formatMethodReturnDoc(doc string) []string {
	if strings.TrimSpace(doc) == "" {
		return nil
	}
	lines := strings.Split(doc, "\n")
	if len(lines) > 1 && strings.HasPrefix(lines[0], "A [") && strings.Contains(doc, "\n\n") {
		term := strings.TrimSpace(lines[0])
		rest := strings.TrimSpace(strings.Join(lines[1:], "\n"))
		wrappedRest := formatRstDoc(rest, 56-len(definitionIndent), 0)
		result := []string{term}
		for _, rl := range wrappedRest {
			if rl == "" {
				result = append(result, "")
			} else {
				result = append(result, definitionIndent+rl)
			}
		}
		return result
	}

	res := formatRstDoc(doc, 56, 16)
	// In Sphinx reST definition lists, once a definition entry begins with a cross-reference
	// link ([...]), subsequent lines in that definition block must be indented by 3 spaces
	// until a blank line separating entries.
	if len(res) > 1 && (strings.HasPrefix(res[0], "A ") || strings.HasPrefix(res[0], "The ") || strings.Contains(doc, "\n[")) {
		indentRest := false
		for i := 1; i < len(res); i++ {
			if res[i] == "" {
				indentRest = false
				continue
			}
			if strings.HasPrefix(res[i], "[") {
				indentRest = true
			}
			if indentRest {
				res[i] = definitionIndent + res[i]
			}
		}
	}
	return res
}
