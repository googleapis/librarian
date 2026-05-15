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

package swift

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// formatDocumentation converts a documentation string from the source (typically Protobuf
// comments) into a sequence of lines suitable for Swift documentation comments.
//
// Both Swift and the Protobuf comments use markdown, but our markdown includes cross-reference
// links and sometimes needs cleaning up work correctly on a different markdown engine.
func (c *codec) formatDocumentation(doc string, scopes []string) ([]string, error) {
	if doc == "" {
		return nil, nil
	}
	results := strings.Split(doc, "\n")

	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	docBytes := []byte(doc)
	node := md.Parser().Parse(text.NewReader(docBytes))
	links := language.ExtractCrossReferenceLinks(node, docBytes)

	var definitions []string
	for _, link := range links {
		resolved, err := c.linkDefinition(link, scopes)
		if err != nil {
			return nil, err
		}
		if resolved != "" {
			definitions = append(definitions, fmt.Sprintf("[%s]: %s", link, resolved))
		}
	}
	if len(definitions) != 0 {
		results = append(results, "") // Add a blank line before link definitions, if any
		results = append(results, definitions...)
	}

	return results, nil
}
