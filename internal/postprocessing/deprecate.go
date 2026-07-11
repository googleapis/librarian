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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

var (
	errEmptyDeprecationMessage = errors.New("deprecation message is required and cannot be empty")
	errAmbiguousDeprecation    = errors.New("ambiguous deprecation")
	errMethodAlreadyDeprecated = errors.New("method is already deprecated")
)

type methodHeader struct {
	hasDeprecated      bool
	hasJavadoc         bool
	hasDeprecatedTag   bool
	javadocEndIdx      int
	firstAnnotationIdx int
}

// DeprecateMethod deprecates a Java method by adding the @Deprecated annotation
// and appending a @deprecated tag in its Javadoc block.
//
// Note: funcName must be the complete, single-line method signature declaration
// (including modifiers and return type, e.g., "public void foo()"). Matching is
// done via exact substring search, so any spacing or formatting mismatch will fail.
func DeprecateMethod(path, funcName, deprecationMessage, language string) error {
	if language != config.LanguageJava {
		return fmt.Errorf("%w: %s", errUnsupportedLanguage, language)
	}
	if !strings.Contains(funcName, "(") || !strings.Contains(funcName, ")") {
		return fmt.Errorf("%w: %q", errInvalidSignature, funcName)
	}
	if strings.TrimSpace(deprecationMessage) == "" {
		return fmt.Errorf("%w for method %q in %s", errEmptyDeprecationMessage, funcName, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cleaned := cleanJavaCode(content)
	boundsList, err := findMethodBounds(content, cleaned, funcName)
	if err != nil {
		return fmt.Errorf("deprecating method %s in %s: %w", funcName, path, err)
	}
	if len(boundsList) > 1 {
		return fmt.Errorf("%w: multiple methods found matching signature %q in %s", errAmbiguousDeprecation, funcName, path)
	}
	lines := bytes.Split(content, []byte("\n"))
	sigLineIdx := bytes.Count(content[:boundsList[0].start], []byte("\n"))
	header := analyzeMethodHeader(lines, sigLineIdx)
	if header.hasDeprecated && header.hasDeprecatedTag {
		return fmt.Errorf("%w: method %q in %s", errMethodAlreadyDeprecated, funcName, path)
	}
	indentation := trimLineIndentation(lines[sigLineIdx])
	lines = annotateMethod(lines, sigLineIdx, indentation, header)
	lines = addJavadocTag(lines, indentation, header, "deprecated", deprecationMessage)
	return os.WriteFile(path, bytes.Join(lines, []byte("\n")), 0644)
}

// analyzeMethodHeader scans the lines above a method signature to identify existing Javadoc and annotations.
func analyzeMethodHeader(lines [][]byte, sigLineIdx int) methodHeader {
	header := methodHeader{
		firstAnnotationIdx: sigLineIdx,
		javadocEndIdx:      -1,
	}
	idx := sigLineIdx - 1
	// Scan past annotations, checking if the method is deprecated.
	for idx >= 0 {
		line := bytes.TrimSpace(lines[idx])
		if bytes.HasPrefix(line, []byte("@")) {
			if bytes.Equal(line, []byte("@Deprecated")) || bytes.HasPrefix(line, []byte("@Deprecated(")) {
				header.hasDeprecated = true
			}
			header.firstAnnotationIdx = idx
			idx--
			continue
		}
		break
	}
	// Check if the line immediately above annotations is the end of Javadoc.
	if idx >= 0 {
		line := bytes.TrimSpace(lines[idx])
		if bytes.HasSuffix(line, []byte("*/")) {
			header.hasJavadoc = true
			header.javadocEndIdx = idx
			header.hasDeprecatedTag = hasTagInJavadoc(lines, idx, "deprecated")
		}
	}
	return header
}

// annotateMethod prepends the @Deprecated annotation above the method signature if not already present.
func annotateMethod(lines [][]byte, sigLineIdx int, indentation []byte, header methodHeader) [][]byte {
	if header.hasDeprecated {
		return lines
	}
	annotationLine := fmt.Appendf(nil, "%s@Deprecated", indentation)
	return slices.Insert(lines, sigLineIdx, annotationLine)
}

// addJavadocTag adds the @deprecated tag to the method's Javadoc block, creating one if it doesn't exist.
func addJavadocTag(lines [][]byte, indentation []byte, header methodHeader, tagName, tagMessage string) [][]byte {
	if header.hasDeprecatedTag {
		return lines
	}
	if !header.hasJavadoc {
		newJavadoc := makeNewJavadoc(indentation, tagName, tagMessage, nil)
		return slices.Insert(lines, header.firstAnnotationIdx, newJavadoc...)
	}
	line := bytes.TrimSpace(lines[header.javadocEndIdx])
	if after, ok := bytes.CutPrefix(line, []byte("/**")); ok {
		inner := bytes.TrimSpace(bytes.TrimSuffix(after, []byte("*/")))
		newJavadoc := makeNewJavadoc(indentation, tagName, tagMessage, inner)
		return slices.Replace(lines, header.javadocEndIdx, header.javadocEndIdx+1, newJavadoc...)
	}
	tagLine := fmt.Appendf(nil, "%s * @%s %s", indentation, tagName, tagMessage)
	return slices.Insert(lines, header.javadocEndIdx, tagLine)
}

func makeNewJavadoc(indentation []byte, tagName, tagMessage string, inner []byte) [][]byte {
	doc := [][]byte{
		fmt.Appendf(nil, "%s/**", indentation),
		fmt.Appendf(nil, "%s * @%s %s", indentation, tagName, tagMessage),
		fmt.Appendf(nil, "%s */", indentation),
	}
	if len(inner) > 0 {
		doc = slices.Insert(doc, 1, fmt.Appendf(nil, "%s * %s", indentation, inner))
	}
	return doc
}

func trimLineIndentation(line []byte) []byte {
	trimmed := bytes.TrimLeft(line, " \t")
	return line[:len(line)-len(trimmed)]
}

func hasTagInJavadoc(lines [][]byte, javadocEndIdx int, tagName string) bool {
	tagBytes := []byte("@" + tagName)
	for i := javadocEndIdx; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if bytes.Contains(line, tagBytes) {
			return true
		}
		if bytes.HasPrefix(line, []byte("/**")) {
			break
		}
	}
	return false
}
