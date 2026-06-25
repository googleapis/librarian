// Copyright 2025 Google LLC
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

// Package dart implements a native Dart code generator.
package dart

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

const (
	httpImport             = "package:http/http.dart as http"
	serviceClientImport    = "package:google_cloud_rpc/service_client.dart"
	serviceExceptionImport = "package:google_cloud_rpc/exceptions.dart"
	encodingImport         = "package:google_cloud_protobuf/src/encoding.dart"
	protobufImport         = "package:google_cloud_protobuf/protobuf.dart"
)

var needsCtorValidation = map[string]string{
	".google.protobuf.Duration":  "",
	".google.protobuf.Timestamp": "",
}

// usesCustomEncoding needs to be kept in sync with
// generator/dart/generated/google_cloud_protobuf/lib/src/protobuf.p.dart.
var usesCustomEncoding = map[string]string{
	".google.protobuf.BoolValue":   "",
	".google.protobuf.BytesValue":  "",
	".google.protobuf.DoubleValue": "",
	".google.protobuf.Duration":    "",
	".google.protobuf.FieldMask":   "",
	".google.protobuf.FloatValue":  "",
	".google.protobuf.Int32Value":  "",
	".google.protobuf.Int64Value":  "",
	".google.protobuf.ListValue":   "",
	".google.protobuf.NullValue":   "",
	".google.protobuf.StringValue": "",
	".google.protobuf.Struct":      "",
	".google.protobuf.Timestamp":   "",
	".google.protobuf.UInt32Value": "",
	".google.protobuf.UInt64Value": "",
	".google.protobuf.Value":       "",
}

// canHaveNullJsonSerialization indicates whether the message type's serialized
// JSON can be `null`.
var canHaveNullJsonSerialization = map[string]bool{
	".google.protobuf.NullValue": true,
	".google.protobuf.Value":     true,
}

const (
	// nestedMessageChar is used to concatenate a message and a child message.
	nestedMessageChar = "_"

	// nestedEnumChar is used to concatenate a message and a child enum.
	nestedEnumChar = "_"

	// deconflictChar is appended to a name to avoid conflicting with a Dart identifier.
	deconflictChar = "$"
)

// reservedNames is a blocklist of Dart reserved words.
//
// This blocklist includes words that can never be used as an identifier as well
// as a few that could be used depending on context. We can add additional
// context keywords as we discover conflicts.
//
// See also https://dart.dev/language/keywords.
var reservedNames = map[string]string{
	"assert":   "",
	"await":    "",
	"break":    "",
	"case":     "",
	"catch":    "",
	"class":    "",
	"const":    "",
	"continue": "",
	"default":  "",
	"do":       "",
	"else":     "",
	"enum":     "",
	"extends":  "",
	"false":    "",
	"final":    "",
	"finally":  "",
	"for":      "",
	"Function": "",
	"if":       "",
	"in":       "",
	"is":       "",
	"new":      "",
	"null":     "",
	"rethrow":  "",
	"return":   "",
	"super":    "",
	"switch":   "",
	"this":     "",
	"throw":    "",
	"true":     "",
	"try":      "",
	"var":      "",
	"void":     "",
	"while":    "",
	"with":     "",
	"yield":    "",

	// Names from dart:core to avoid.
	"bool":   "",
	"double": "",
}

func messageName(m *api.Message) string {
	name := strcase.ToCamel(m.Name)

	if m.Parent == nil {
		// For top-most symbols, check for conflicts with reserved names.
		if _, hasConflict := reservedNames[name]; hasConflict {
			return name + deconflictChar
		} else {
			return name
		}
	} else {
		return messageName(m.Parent) + nestedMessageChar + name
	}
}

func qualifiedName(m *api.Message) string {
	// Convert '.google.protobuf.Duration' to 'google.protobuf.Duration'.
	return strings.TrimPrefix(m.ID, ".")
}

func fieldName(field *api.Field) string {
	name := strcase.ToLowerCamel(field.Name)
	if _, hasConflict := reservedNames[name]; hasConflict {
		name = name + deconflictChar
	}
	return name
}

func enumName(e *api.Enum) string {
	name := strcase.ToCamel(e.Name)
	if e.Parent != nil {
		name = messageName(e.Parent) + nestedEnumChar + name
	}
	return name
}

func httpPathFmt(pathInfo *api.PathInfo) string {
	var builder strings.Builder
	t := pathInfo.Bindings[0].PathTemplate
	for _, segment := range t.Segments {
		switch {
		case segment.Literal != "":
			builder.WriteString("/")
			builder.WriteString(segment.Literal)
		case segment.Variable != nil:
			// Form '${request.foo!.bar!.baz}'.
			builder.WriteString("/${request")
			deref := "."
			for _, f := range segment.Variable.FieldPath {
				builder.WriteString(deref)
				builder.WriteString(strcase.ToLowerCamel(f))
				deref = "!."
			}
			builder.WriteString("}")
		}
	}
	if t.Verb != "" {
		builder.WriteString(":")
		builder.WriteString(t.Verb)
	}

	return builder.String()
}

// commentRefsRegex matches Google API documentation reference links; it supports
// both regular references as well as implit references.
//
// - `[Code][google.rpc.Code]`
// - `[google.rpc.Code][]`.
var commentRefsRegex = regexp.MustCompile(`\[([^\]]+)\]\[([\w\d\._]*)\]`)

func (annotate *annotateModel) formatDocComments(documentation string) []string {
	lines := strings.Split(documentation, "\n")

	// Remove trailing whitespace.
	for i, line := range lines {
		lines[i] = strings.TrimRightFunc(line, unicode.IsSpace)
	}

	lines, codeBlocks := extractCodeBlocks(lines)
	lines = sanitizeUrls(lines)
	lines = annotate.resolveGoogleApiRefs(lines)
	lines = escapeHtml(lines)
	lines = annotate.processSingleBracketRefs(lines)
	lines = cleanupDoubleTicks(lines)
	lines = restoreCodeBlocks(lines, codeBlocks)

	return toDartDoc(lines)
}

// extractCodeBlocks detects code blocks (indented by at least 2 spaces or fenced)
// and replaces them with placeholders to avoid processing references inside them.
func extractCodeBlocks(lines []string) ([]string, [][]string) {
	var processedLines []string
	var codeBlocks [][]string

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Check for existing code fences.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			// Find the closing fence.
			j := i + 1
			for j < len(lines) {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
					break
				}
				j++
			}

			// If we found a closing fence.
			if j < len(lines) {
				// Valid fenced block!
				// Find minimum indentation of non-empty lines in the block (including fences).
				minIndent := 1000
				for k := i; k <= j; k++ {
					if lines[k] == "" {
						continue
					}
					indent := 0
					for indent < len(lines[k]) && lines[k][indent] == ' ' {
						indent++
					}
					if indent < minIndent {
						minIndent = indent
					}
				}

				var blockContent []string
				for k := i; k <= j; k++ {
					var lineContent string
					if lines[k] == "" {
						lineContent = ""
					} else if len(lines[k]) >= minIndent {
						lineContent = lines[k][minIndent:]
					} else {
						lineContent = lines[k]
					}

					// If it's the opening fence and it's just "```", add "text".
					if k == i && lineContent == "```" {
						lineContent = "```text"
					}

					blockContent = append(blockContent, lineContent)
				}

				codeBlocks = append(codeBlocks, blockContent)
				processedLines = append(processedLines, fmt.Sprintf("__CODE_BLOCK_%d__", len(codeBlocks)-1))

				i = j + 1
				continue
			}
		}

		// If line is indented by 4 spaces.
		if strings.HasPrefix(line, "    ") {
			// Find the end of the candidate block.
			j := i
			for j < len(lines) && (strings.HasPrefix(lines[j], "    ") || lines[j] == "") {
				j++
			}

			// Remove trailing empty lines from the candidate block.
			for j > i && lines[j-1] == "" {
				j--
			}

			// Candidate block is lines[i:j].
			// Check if it's a valid code block (preceded and followed by empty lines).
			validPre := i == 0 || lines[i-1] == ""
			validPost := j == len(lines) || lines[j] == ""

			if validPre && validPost && j > i {
				// Valid code block!
				// Find minimum indentation of non-empty lines in the block.
				minIndent := 1000 // Arbitrary large number.
				for k := i; k < j; k++ {
					if lines[k] == "" {
						continue
					}
					// Count leading spaces.
					indent := 0
					for indent < len(lines[k]) && lines[k][indent] == ' ' {
						indent++
					}
					if indent < minIndent {
						minIndent = indent
					}
				}

				var blockContent []string
				for k := i; k < j; k++ {
					if lines[k] == "" {
						blockContent = append(blockContent, "")
					} else {
						// Remove minIndent spaces.
						if len(lines[k]) >= minIndent {
							blockContent = append(blockContent, lines[k][minIndent:])
						} else {
							blockContent = append(blockContent, lines[k])
						}
					}
				}

				codeBlocks = append(codeBlocks, blockContent)
				processedLines = append(processedLines, fmt.Sprintf("__CODE_BLOCK_%d__", len(codeBlocks)-1))

				i = j
				continue
			}
		}

		processedLines = append(processedLines, line)
		i++
	}
	return processedLines, codeBlocks
}

// sanitizeUrls converts URLs containing brackets to ticked URLs and strips ref targets.
func sanitizeUrls(lines []string) []string {
	refTargetRegex := regexp.MustCompile(`\[([^\]]+)\]\[([^\]]+)\]`)
	quotedUrlWithBracketsRegex := regexp.MustCompile(`"(https?://[^"]*\[[^"]*)"`)
	nonQuotedUrlWithBracketsRegex := regexp.MustCompile(`(^|\s)(https?://[^\s\[]*\[[^\s]*)`)

	result := make([]string, len(lines))
	copy(result, lines)

	for i, line := range result {
		// Quoted URLs.
		result[i] = quotedUrlWithBracketsRegex.ReplaceAllStringFunc(line, func(m string) string {
			submatches := quotedUrlWithBracketsRegex.FindStringSubmatch(m)
			if len(submatches) == 2 {
				url := submatches[1]
				url = refTargetRegex.ReplaceAllStringFunc(url, func(rm string) string {
					rSubmatches := refTargetRegex.FindStringSubmatch(rm)
					return "[" + rSubmatches[1] + "]"
				})
				return "`" + url + "`"
			}
			return m
		})

		// Non-quoted URLs.
		result[i] = nonQuotedUrlWithBracketsRegex.ReplaceAllStringFunc(result[i], func(m string) string {
			submatches := nonQuotedUrlWithBracketsRegex.FindStringSubmatch(m)
			if len(submatches) == 3 {
				prefix := submatches[1]
				url := submatches[2]
				url = refTargetRegex.ReplaceAllStringFunc(url, func(rm string) string {
					rSubmatches := refTargetRegex.FindStringSubmatch(rm)
					return "[" + rSubmatches[1] + "]"
				})
				return prefix + "`" + url + "`"
			}
			return m
		})
	}
	return result
}

func (annotate *annotateModel) findMessageByName(name string) *api.Message {
	if annotate == nil {
		return nil
	}
	return annotate.messageShortNames[name]
}

func (annotate *annotateModel) findEnumByName(name string) *api.Enum {
	if annotate == nil {
		return nil
	}
	return annotate.enumShortNames[name]
}

func (annotate *annotateModel) classExists(name string) bool {
	return annotate.findMessageByName(name) != nil || annotate.findEnumByName(name) != nil
}

// resolveGoogleApiRefs rewrites Google API doc references to code formatted text.
func (annotate *annotateModel) resolveGoogleApiRefs(lines []string) []string {
	result := make([]string, len(lines))
	copy(result, lines)

	for i := 0; i < len(result); i++ {
		line := result[i]
		matches := commentRefsRegex.FindAllStringSubmatchIndex(line, -1)
		for j := len(matches) - 1; j >= 0; j-- {
			start, end := matches[j][0], matches[j][1]
			// Skip if inside backticks
			if strings.Count(line[:start], "`")%2 != 0 {
				continue
			}

			textStart, textEnd := matches[j][2], matches[j][3]
			refStart, refEnd := matches[j][4], matches[j][5]

			text := line[textStart:textEnd]
			ref := ""
			if refStart != -1 {
				ref = line[refStart:refEnd]
			}

			// If ref is a valid symbol, leave it as is.
			if ref != "" && annotate.classExists(ref) {
				continue
			}

			replacement := ""
			if ref == "" {
				replacement = "`" + text + "`"
			} else {
				if !strings.Contains(text, " ") {
					replacement = "`" + text + "`"
				} else {
					replacement = text
				}
			}

			line = line[:start] + replacement + line[end:]
		}
		result[i] = line
	}
	return result
}

// escapeHtml replaces < and > with HTML entities to avoid analyzer warnings.
func escapeHtml(lines []string) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		line = strings.ReplaceAll(line, "<", "&lt;")
		line = strings.ReplaceAll(line, ">", "&gt;")
		result[i] = line
	}
	return result
}

// processSingleBracketRefs handles array access, literals, and symbol resolution.
func (annotate *annotateModel) processSingleBracketRefs(lines []string) []string {
	arrayAccessRegex := regexp.MustCompile(`[\w\d_\.]+\[\d+\](?:\.[\w\d_]+(?:\[\d+\])?)*`)
	arrayLiteralRegex := regexp.MustCompile(`\[\d+(?:,\d+)*\]`)
	singleRefRegex := regexp.MustCompile(`\[([\w\d\._]+)\]`)

	result := make([]string, len(lines))
	copy(result, lines)

	for i, line := range result {
		// Process array access.
		matches := arrayAccessRegex.FindAllStringIndex(line, -1)
		for j := len(matches) - 1; j >= 0; j-- {
			match := matches[j]
			start := match[0]
			end := match[1]
			if strings.Count(line[:start], "`")%2 != 0 {
				continue
			}
			line = line[:start] + "`" + line[start:end] + "`" + line[end:]
		}

		// Process array literals.
		matches = arrayLiteralRegex.FindAllStringIndex(line, -1)
		for j := len(matches) - 1; j >= 0; j-- {
			match := matches[j]
			start := match[0]
			end := match[1]
			if strings.Count(line[:start], "`")%2 != 0 {
				continue
			}
			line = line[:start] + "`" + line[start:end] + "`" + line[end:]
		}

		// Process single brackets.
		matches = singleRefRegex.FindAllStringSubmatchIndex(line, -1)
		for j := len(matches) - 1; j >= 0; j-- {
			match := matches[j]
			start := match[0]
			end := match[1]
			contentStart := match[2]
			contentEnd := match[3]

			content := line[contentStart:contentEnd]

			if strings.Count(line[:start], "`")%2 != 0 {
				continue
			}

			if end < len(line) && (line[end] == '(' || line[end] == '[') {
				continue
			}

			replacement := ""

			if !strings.Contains(content, ".") {
				if annotate.classExists(content) {
					continue
				}
				replacement = "`[" + content + "]`"
			} else {
				parts := strings.Split(content, ".")
				if len(parts) == 2 {
					msgName := parts[0]
					targetFieldName := parts[1]

					found := false
					mappedFieldName := targetFieldName
					var targetMsg *api.Message

					if annotate != nil {
						if msg, ok := annotate.messageShortNames[msgName]; ok {
							for _, f := range msg.Fields {
								if f.Name == targetFieldName || f.JSONName == targetFieldName {
									mappedFieldName = fieldName(f)
									found = true
									targetMsg = msg
									break
								}
							}
						}
					}

					if found {
						flattenedName := messageName(targetMsg)
						replacement = "[" + flattenedName + "." + mappedFieldName + "]"
					}
				}

				if replacement == "" {
					replacement = "`[" + content + "]`"
				}
			}

			if replacement != "" {
				line = line[:start] + replacement + line[end:]
			}
		}
		result[i] = line
	}
	return result
}

// cleanupDoubleTicks removes double backticks around simple words.
func cleanupDoubleTicks(lines []string) []string {
	doubleTicksRegex := regexp.MustCompile("``([\\w\\d_]+)``")
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = doubleTicksRegex.ReplaceAllString(line, "`$1`")
	}
	return result
}

// restoreCodeBlocks replaces placeholders with original code blocks.
func restoreCodeBlocks(lines []string, codeBlocks [][]string) []string {
	var finalLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "__CODE_BLOCK_") && strings.HasSuffix(line, "__") {
			var index int
			fmt.Sscanf(line, "__CODE_BLOCK_%d__", &index)
			if index >= 0 && index < len(codeBlocks) {
				block := codeBlocks[index]
				if len(block) > 0 && strings.HasPrefix(strings.TrimSpace(block[0]), "```") {
					finalLines = append(finalLines, block...)
				} else {
					finalLines = append(finalLines, "```text")
					finalLines = append(finalLines, block...)
					finalLines = append(finalLines, "```")
				}
				continue
			}
		}
		finalLines = append(finalLines, line)
	}
	return finalLines
}

// toDartDoc removes trailing blank lines and converts to dartdoc format.
func toDartDoc(lines []string) []string {
	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		if len(line) == 0 {
			result[i] = "///"
		} else {
			result[i] = "/// " + line
		}
	}
	return result
}

func packageName(api *api.API, packageNameOverride string) string {
	if len(packageNameOverride) > 0 {
		return packageNameOverride
	}

	// Convert 'google.protobuf' to 'google_cloud_protobuf' and
	// 'google.cloud.language.v2' to 'google_cloud_language_v2.
	packageName := api.PackageName
	packageName = strings.TrimPrefix(packageName, "google.cloud.")
	packageName = strings.TrimPrefix(packageName, "google.")
	return "google_cloud_" + strings.ReplaceAll(packageName, ".", "_")
}

func shouldGenerateMethod(m *api.Method) bool {
	// Ignore methods without HTTP annotations; we cannot generate working RPCs
	// for them.
	// TODO(#499) Switch to explicitly excluding such functions.
	if m.ClientSideStreaming || m.PathInfo == nil {
		return false
	}
	if len(m.PathInfo.Bindings) == 0 {
		return false
	}
	return m.PathInfo.Bindings[0].PathTemplate != nil
}

func formatDirectory(ctx context.Context, dir string) error {
	if err := command.Run(ctx, "dart", "format", dir); err != nil {
		return fmt.Errorf("got an error trying to run `dart format`; perhaps try https://dart.dev/get-dart (%w)", err)
	}
	return nil
}
