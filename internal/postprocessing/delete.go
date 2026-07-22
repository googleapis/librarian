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

package postprocessing

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

var (
	errUnsupportedLanguage  = errors.New("unsupported language")
	errMethodNotFound       = errors.New("method not found")
	errOpeningBraceNotFound = errors.New("opening brace not found")
	errClosingBraceNotFound = errors.New("closing brace not found")
	errInvalidSignature     = errors.New("invalid method signature")
	// javaCleanRegex matches comments, text blocks, double-quoted strings, and char literals.
	javaCleanRegex = regexp.MustCompile(`"""[\s\S]*?"""|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|/\*[\s\S]*?\*/|//.*`)
)

// methodBounds represents the absolute physical byte range of a method in a file.
// The range is inclusive of the method's leading indentation and trailing newline.
type methodBounds struct {
	start int
	end   int
}

// DeleteMethod deletes all matching methods from a Java file.
// It handles brace counting to remove the entire method body.
//
// Note: funcName must be the complete, single-line method signature declaration
// (including modifiers and return type, e.g., "public void foo()"). Matching is
// done via exact substring search, so any spacing or formatting mismatch will fail.
func DeleteMethod(path, funcName, language string) error {
	if language != config.LanguageJava {
		return fmt.Errorf("%w: %s", errUnsupportedLanguage, language)
	}
	if !strings.Contains(funcName, "(") || !strings.Contains(funcName, ")") {
		return fmt.Errorf("%w: %q (must contain parameter list in parentheses)", errInvalidSignature, funcName)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	cleaned := cleanJavaCode(data)
	boundsList, err := findMethodBounds(data, cleaned, funcName)
	if err != nil {
		return fmt.Errorf("deleting method %s in %s: %w", funcName, path, err)
	}
	for _, bound := range slices.Backward(boundsList) {
		data = append(data[:bound.start], data[bound.end:]...)
	}
	return os.WriteFile(path, data, 0o644)
}

// findMethodBounds locates the physical byte boundaries of all valid matches for the signature.
func findMethodBounds(data, cleaned []byte, funcName string) ([]methodBounds, error) {
	var boundsList []methodBounds
	searchStart := 0
	for {
		idx := bytes.Index(cleaned[searchStart:], []byte(funcName))
		if idx == -1 {
			break
		}
		actualIdx := searchStart + idx
		openBraceIdx, err := findOpeningBrace(cleaned, actualIdx+len(funcName))
		if err != nil {
			return nil, err
		}
		closeBraceIdx, err := findClosingBrace(cleaned, openBraceIdx)
		if err != nil {
			return nil, err
		}
		start, end := adjustDeleteBounds(data, actualIdx, closeBraceIdx+1)
		boundsList = append(boundsList, methodBounds{start: start, end: end})
		searchStart = closeBraceIdx + 1
	}
	if len(boundsList) == 0 {
		return nil, errMethodNotFound
	}
	return boundsList, nil
}

// cleanJavaCode replaces comments, strings, and char literals with spaces.
func cleanJavaCode(content []byte) []byte {
	return javaCleanRegex.ReplaceAllFunc(content, func(match []byte) []byte {
		return bytes.Repeat([]byte(" "), len(match))
	})
}

// findOpeningBrace returns the index of the first '{' after start, or error if ';' is found first.
func findOpeningBrace(cleaned []byte, start int) (int, error) {
	for i, c := range cleaned[start:] {
		switch c {
		case '{':
			return start + i, nil
		case ';':
			return -1, errOpeningBraceNotFound
		}
	}
	return -1, errOpeningBraceNotFound
}

// findClosingBrace returns the index of the matching closing brace.
func findClosingBrace(cleaned []byte, openBraceIdx int) (int, error) {
	braceCount := 1
	start := openBraceIdx + 1
	for i, c := range cleaned[start:] {
		switch c {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				return start + i, nil
			}
		}
	}
	return -1, errClosingBraceNotFound
}

// adjustDeleteBounds adjusts the delete range [start, end) to include the entire line(s)
// if the method is the only content on those lines.
//
// Examples:
// - For a block method on its own lines:
//
//	class T {
//	    void m() {
//	    }
//	}
//
//	If deleting "void m() {\n    }", it returns bounds for "    void m() {\n    }\n"
//	to remove the leading indentation and trailing newline.
//
// - For an inline method:
//
//	class T { void m() {} void o() {} }
//
//	If deleting "void m() {}", it returns the exact bounds of "void m() {}"
//	to preserve "void o() {}".
func adjustDeleteBounds(content []byte, start, end int) (int, int) {
	startLineIdx := 0
	if lastNl := bytes.LastIndexByte(content[:start], '\n'); lastNl != -1 {
		startLineIdx = lastNl + 1
	}
	endLineIdx := len(content)
	if nextNl := bytes.IndexByte(content[end:], '\n'); nextNl != -1 {
		endLineIdx = end + nextNl
	}
	if len(bytes.TrimSpace(content[startLineIdx:start])) == 0 && len(bytes.TrimSpace(content[end:endLineIdx])) == 0 {
		finalEnd := endLineIdx
		if finalEnd < len(content) {
			finalEnd++
		}
		return startLineIdx, finalEnd
	}
	return start, end
}
