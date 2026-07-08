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
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

var errMethodAlreadyExists = errors.New("method already exists")
var errAmbiguousDuplication = errors.New("ambiguous duplication")

// DuplicateMethod extracts a method block, renames it, and appends it immediately after the original method.
//
// Note: funcName must be the complete, single-line method signature declaration
// (including modifiers and return type, e.g., "public void foo()"). Matching is
// done via exact substring search, so any spacing or formatting mismatch will fail.
func DuplicateMethod(path, funcName, newName, language string) error {
	if language != config.LanguageJava {
		return fmt.Errorf("%w: %s", errUnsupportedLanguage, language)
	}
	if !strings.Contains(funcName, "(") || !strings.Contains(funcName, ")") {
		return fmt.Errorf("%w: %q (must contain parameter list in parentheses)", errInvalidSignature, funcName)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cleaned := cleanJavaCode(content)
	parenIdx := strings.Index(funcName, "(")
	beforeParen := funcName[:parenIdx]
	trimmedBeforeParen := strings.TrimRight(beforeParen, " \t")
	spaceIdx := strings.LastIndexAny(trimmedBeforeParen, " \t")
	var newSignature string
	if spaceIdx == -1 {
		newSignature = newName + funcName[parenIdx:]
	} else {
		newSignature = beforeParen[:spaceIdx+1] + newName + funcName[parenIdx:]
	}
	if bytes.Contains(cleaned, []byte(newSignature)) {
		return fmt.Errorf("%w: %s; duplication rule is redundant", errMethodAlreadyExists, newSignature)
	}

	boundsList, err := findMethodBounds(content, cleaned, funcName)
	if err != nil {
		return fmt.Errorf("duplicating method %s in %s: %w", funcName, path, err)
	}
	if len(boundsList) > 1 {
		return fmt.Errorf("%w: multiple methods found matching signature %q in %s", errAmbiguousDuplication, funcName, path)
	}
	b := boundsList[0]
	methodBlock := content[b.start:b.end]
	renamedBlock := bytes.Replace(methodBlock, []byte(funcName), []byte(newSignature), 1)
	newContent := make([]byte, 0, len(content)+1+len(renamedBlock))
	newContent = append(newContent, content[:b.end]...)
	newContent = append(newContent, '\n')
	newContent = append(newContent, renamedBlock...)
	newContent = append(newContent, content[b.end:]...)
	return os.WriteFile(path, newContent, 0644)
}
