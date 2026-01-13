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

package discovery

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

const (
	beginExpression = '{'
	endExpression   = '}'
	slash           = '/'
)

var (
	identifierRe = regexp.MustCompile("^[A-Za-z][A-Za-z0-9_]*$")
)

// ParseUriTemplate parses a [RFC 6570] URI template as an `api.PathTemplate`.
//
// In sidekick we need to capture the structure of the URI template for the
// codec(s) to emit good templates with them.
func ParseUriTemplate(uriTemplate string) (*api.PathTemplate, error) {
	template := &api.PathTemplate{}
	var pos int
	for {
		var err error
		var segment *api.PathSegment
		var width int

		if pos == len(uriTemplate) {
			return nil, fmt.Errorf("expected a segment, found eof: %s", uriTemplate)
		}
		if uriTemplate[pos] == beginExpression {
			segment, width, err = parseExpression(uriTemplate[pos:])
		} else {
			segment, width, err = parseLiteral(uriTemplate[pos:])
		}
		if err != nil {
			return nil, err
		}
		template.Segments = append(template.Segments, *segment)
		pos += width
		if pos == len(uriTemplate) || uriTemplate[pos] != slash {
			break
		}
		pos++ // Skip slash
	}
	if pos != len(uriTemplate) {
		return nil, fmt.Errorf("trailing data (%q) cannot be parsed as a URI template", uriTemplate[pos:])
	}
	return template, nil
}

func parseExpression(input string) (*api.PathSegment, int, error) {
	if input == "" || input[0] != beginExpression {
		return nil, 0, fmt.Errorf("missing `{` character in expression %q", input)
	}
	tail := input[1:]
	end := strings.IndexRune(tail, endExpression)
	if end == -1 {
		return nil, 0, fmt.Errorf("missing `}` character at the end of the expression %q", input)
	}
	tail = tail[:end]
	if tail == "" {
		return nil, 0, fmt.Errorf("empty expression %q", input)
	}

	if strings.IndexAny(tail, "./?&") == 0 {
		return nil, 0, fmt.Errorf("level 3 expressions unsupported input=%q", input)
	}
	if strings.IndexAny(tail, "=,!@|") == 0 {
		return nil, 0, fmt.Errorf("reserved character on expression %q", input)
	}

	allowReserved := false
	switch tail[0] {
	case '+':
		allowReserved = true
		tail = tail[1:]
	case '#':
		return nil, 0, fmt.Errorf("level 2 fragment expressions unsupported input=%q", input)
	}

	match := identifierRe.MatchString(tail)
	if !match {
		return nil, 0, fmt.Errorf("invalid varname found on expression %q", input)
	}
	variable := api.NewPathVariable(tail).WithMatch()
	if allowReserved {
		variable.WithAllowReserved()
	}
	return &api.PathSegment{Variable: variable}, end + 2, nil
}

// parseLiteral() extracts a literal value from `input`.
//
// The format for literals is defined in:
//
//	https://www.rfc-editor.org/rfc/rfc6570.html#section-2.1
//
// We simplify the parsing a bit assuming most discovery docs contain valid
// URI templates.
func parseLiteral(input string) (*api.PathSegment, int, error) {
	index := strings.IndexAny(input, " \"'<>\\^`{|}/")
	var literal string
	var tail string
	var width int
	if index == -1 {
		literal = input
		tail = ""
		width = len(input)
	} else {
		literal = input[:index]
		tail = input[index:]
		width = index
	}
	if literal == "" {
		return nil, 0, fmt.Errorf("invalid empty literal with input=%q", input)
	}
	if tail != "" && tail[0] != slash {
		return nil, index, fmt.Errorf("found unexpected character %v in literal %q, stopped at position %v", tail[0], input, index)
	}
	return &api.PathSegment{Literal: &literal}, width, nil
}
