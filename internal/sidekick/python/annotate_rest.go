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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// httpOptionAnnotation represents a single HTTP option mapping for a REST transport method.
type httpOptionAnnotation struct {
	Method  string
	URI     string
	Body    string
	HasBody bool
	First   bool
}

// requiredFieldDefaultAnnotation represents a default value mapping for a required request field.
type requiredFieldDefaultAnnotation struct {
	Key   string
	Value string
	Last  bool
}

// restMethodAnnotation contains metadata for rendering REST base method classes.
type restMethodAnnotation struct {
	Name                        string
	BaseClassName               string
	HTTPOptions                 []*httpOptionAnnotation
	HasRequiredFields           bool
	RequiredFieldsDefaultValues []*requiredFieldDefaultAnnotation
}

// formatPathTemplateURI formats an api.PathTemplate into a REST path template URI string.
func formatPathTemplateURI(template *api.PathTemplate) string {
	if template == nil || (len(template.Segments) == 0 && template.Verb == "") {
		return ""
	}
	var components []string
	for _, segment := range template.Segments {
		if segment.Literal != "" {
			components = append(components, segment.Literal)
			continue
		}
		if segment.Variable == nil {
			continue
		}
		fieldPath := strings.Join(segment.Variable.FieldPath, ".")
		if len(segment.Variable.Segments) > 0 {
			pattern := strings.Join(segment.Variable.Segments, "/")
			components = append(components, fmt.Sprintf("{%s=%s}", fieldPath, pattern))
			continue
		}
		components = append(components, fmt.Sprintf("{%s}", fieldPath))
	}
	uri := "/" + strings.Join(components, "/")
	if template.Verb != "" {
		uri += ":" + template.Verb
	}
	return uri
}

// annotateRestMethod generates the restMethodAnnotation for an api.Method.
func annotateRestMethod(m *api.Method) *restMethodAnnotation {
	if m == nil {
		return nil
	}
	name := pythonIdentifier(snakeCase(m.Name))
	baseClassName := "_Base" + pascalCase(m.Name)

	var httpOptions []*httpOptionAnnotation
	pathParams := make(map[string]bool)
	body := ""

	if m.PathInfo != nil {
		body = m.PathInfo.BodyFieldPath
		for _, b := range m.PathInfo.Bindings {
			if b == nil {
				continue
			}
			if b.Verb != "" || b.PathTemplate != nil {
				verb := strings.ToLower(b.Verb)
				uri := formatPathTemplateURI(b.PathTemplate)
				hasBody := body != "" && verb != "get" && verb != "delete"
				bodyVal := ""
				if hasBody {
					bodyVal = body
				}
				httpOptions = append(httpOptions, &httpOptionAnnotation{
					Method:  verb,
					URI:     uri,
					Body:    bodyVal,
					HasBody: hasBody,
				})
			}
			if b.PathTemplate != nil {
				for _, s := range b.PathTemplate.Segments {
					if s.Variable != nil {
						pathParams[strings.Join(s.Variable.FieldPath, ".")] = true
					}
				}
			}
		}
	}

	for i, opt := range httpOptions {
		opt.First = (i == 0)
	}

	var requiredFieldsDefaultValues []*requiredFieldDefaultAnnotation
	hasInputRequiredFields := false

	if m.InputType != nil {
		for _, f := range m.InputType.Fields {
			if !f.DocumentAsRequired() {
				continue
			}
			hasInputRequiredFields = true
			if body == "*" || (body != "" && (f.Name == body || f.JSONName == body)) {
				continue
			}
			if pathParams[f.Name] || (f.JSONName != "" && pathParams[f.JSONName]) {
				continue
			}
			key := f.JSONName
			if key == "" {
				key = f.Name
			}
			// In gapic-generator-python REST mappings, message and enum fields default to empty dict `{}`;
			// all other scalar types default to empty string `""`.
			valStr := `""`
			if f.Typez == api.TypezMessage || f.Typez == api.TypezEnum {
				valStr = "{}"
			}
			requiredFieldsDefaultValues = append(requiredFieldsDefaultValues, &requiredFieldDefaultAnnotation{
				Key:   key,
				Value: valStr,
			})
		}
		for i, field := range requiredFieldsDefaultValues {
			field.Last = (i == len(requiredFieldsDefaultValues)-1)
		}
	}

	return &restMethodAnnotation{
		Name:                        name,
		BaseClassName:               baseClassName,
		HTTPOptions:                 httpOptions,
		HasRequiredFields:           hasInputRequiredFields,
		RequiredFieldsDefaultValues: requiredFieldsDefaultValues,
	}
}

// docSummaryForRest splits a method docstring summary line into leading and wrapped segments.
func docSummaryForRest(methodName string) (lead, rest string, wrap bool) {
	humanized := strings.ReplaceAll(snakeCase(methodName), "_", " ")
	const (
		totalWidth, firstLineOffset = 70, 45
		firstLineAvail              = totalWidth - firstLineOffset // 25
	)
	if len(humanized) <= firstLineAvail {
		return humanized, "", false
	}
	words := strings.Fields(humanized)
	var (
		line1Words        []string
		currLen, splitIdx int
	)
	for i, w := range words {
		addedLen := len(w)
		if len(line1Words) > 0 {
			addedLen++
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
	lead = strings.Join(line1Words, " ")
	restWords := words[splitIdx:]
	if len(restWords) == 0 {
		return lead, "", false
	}
	return lead, strings.Join(restWords, " "), true
}

// methodArgsDoc formats method input or output documentation into first-line and subsequent lines.
func methodArgsDoc(doc string) (string, []string, bool) {
	lines := formatRstDoc(doc, 56, 16)
	if len(lines) == 0 {
		return "", nil, false
	}
	if len(lines) > 1 {
		return lines[0], lines[1:], true
	}
	return lines[0], nil, true
}
