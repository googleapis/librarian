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
	"fmt"
	"strconv"
	"strings"
)

const (
	literalBlockPrefix = "\uE003LITERAL_BLOCK_"
	literalBlockSuffix = "\uE004"
)

// isIndentedLiteralLine reports whether a line begins with at least 3 spaces or a tab,
// which represents a CommonMark indented code block in normalized protobuf comments.
func isIndentedLiteralLine(line string) bool {
	trimmedRight := strings.TrimRight(line, " \t\r")
	if trimmedRight == "" {
		return false
	}
	trimmedLeft := strings.TrimLeft(trimmedRight, " \t")
	leadSpaces := len(trimmedRight) - len(trimmedLeft)
	return leadSpaces >= 3 || strings.HasPrefix(trimmedRight, "\t")
}

// extractIndentedLiteralBlocks finds contiguous blocks where all non-empty lines have
// at least 3 leading spaces (originating from proto comments with 4+ spaces after //),
// replacing each block with a sentinel token \uE003LITERAL_BLOCK_<i>\uE004.
//
// This isolates preformatted reST blocks so inline conversions (e.g. single backticks)
// and text reflowing do not mutate literal example code or bullet lists.
func extractIndentedLiteralBlocks(text string) (string, [][]string) {
	if text == "" {
		return "", nil
	}
	var blocks [][]string
	text = fencedCodeRegex.ReplaceAllStringFunc(text, func(m string) string {
		sub := fencedCodeRegex.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		rawCode := sub[1]
		lines := strings.Split(rawCode, "\n")
		var indented []string
		for _, l := range lines {
			indented = append(indented, "   "+strings.TrimRight(l, " \t\r"))
		}
		blockIdx := len(blocks)
		blocks = append(blocks, indented)
		return fmt.Sprintf("\n\n%s%d%s\n\n", literalBlockPrefix, blockIdx, literalBlockSuffix)
	})

	lines := strings.Split(text, "\n")
	var outLines []string

	i := 0
	for i < len(lines) {
		prevBlank := i == 0 || strings.TrimSpace(lines[i-1]) == ""
		if prevBlank && isIndentedLiteralLine(lines[i]) {
			// In CommonMark, a list item following a list item is a sub-list, not a code block.
			lastNonBlank := ""
			for k := i - 1; k >= 0; k-- {
				trimmed := strings.TrimSpace(lines[k])
				if trimmed != "" {
					lastNonBlank = trimmed
					break
				}
			}
			isSubList := (strings.HasPrefix(lastNonBlank, "- ") || strings.HasPrefix(lastNonBlank, "* ") || strings.HasPrefix(lastNonBlank, "+ ")) &&
				(strings.HasPrefix(strings.TrimSpace(lines[i]), "- ") || strings.HasPrefix(strings.TrimSpace(lines[i]), "* ") || strings.HasPrefix(strings.TrimSpace(lines[i]), "+ "))
			if isSubList {
				outLines = append(outLines, lines[i])
				i++
				continue
			}

			// Scan to find the extent of this indented literal block.
			lastIndented := i
			j := i + 1
			for j < len(lines) {
				if isIndentedLiteralLine(lines[j]) {
					lastIndented = j
					j++
				} else if strings.TrimSpace(lines[j]) == "" {
					j++
				} else {
					break
				}
			}

			// Extract block lines from i to lastIndented inclusive.
			block := make([]string, 0, lastIndented-i+1)
			for k := i; k <= lastIndented; k++ {
				block = append(block, strings.TrimRight(lines[k], " \t\r"))
			}
			blockIdx := len(blocks)
			blocks = append(blocks, block)

			token := fmt.Sprintf("%s%d%s", literalBlockPrefix, blockIdx, literalBlockSuffix)
			if len(outLines) > 0 && strings.TrimSpace(outLines[len(outLines)-1]) != "" {
				outLines = append(outLines, "")
			}
			outLines = append(outLines, token)
			if lastIndented+1 < len(lines) && strings.TrimSpace(lines[lastIndented+1]) != "" {
				outLines = append(outLines, "")
			}

			i = lastIndented + 1
			continue
		}

		outLines = append(outLines, lines[i])
		i++
	}

	return strings.Join(outLines, "\n"), blocks
}

// parseLiteralBlockIndex checks if a paragraph token is a literal block sentinel
// and returns its 0-based index.
func parseLiteralBlockIndex(p string) (int, bool) {
	if strings.HasPrefix(p, literalBlockPrefix) && strings.HasSuffix(p, literalBlockSuffix) {
		idxStr := p[len(literalBlockPrefix) : len(p)-len(literalBlockSuffix)]
		idx, err := strconv.Atoi(idxStr)
		if err == nil {
			return idx, true
		}
	}
	return -1, false
}
