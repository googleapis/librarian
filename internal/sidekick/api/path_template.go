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

package api

import (
	"fmt"
	"strings"
)

// PathTemplate is a template for a path.
type PathTemplate struct {
	Segments []PathSegment
	Verb     string
}

// FlatPath returns a simplified representation of the path template as a string.
//
// In the context of discovery LROs it is useful to get the path template as a
// simplified string, such as "compute/v1/projects/{project}/zones/{zone}/instances".
// The path can be matched against LRO prefixes and then mapped to the correct
// poller RPC.
func (template *PathTemplate) FlatPath() string {
	var buffer strings.Builder
	sep := ""
	for _, segment := range template.Segments {
		buffer.WriteString(sep)
		if segment.Literal != "" {
			buffer.WriteString(segment.Literal)
		} else if segment.Variable != nil {
			fmt.Fprintf(&buffer, "{%s}", strings.Join(segment.Variable.FieldPath, "."))
		}
		sep = "/"
	}
	return buffer.String()
}

// WithLiteral adds a literal to the path template.
func (p *PathTemplate) WithLiteral(l string) *PathTemplate {
	p.Segments = append(p.Segments, PathSegment{Literal: l})
	return p
}

// WithVariable adds a variable to the path template.
func (p *PathTemplate) WithVariable(v *PathVariable) *PathTemplate {
	p.Segments = append(p.Segments, PathSegment{Variable: v})
	return p
}

// WithVariableNamed adds a variable with the given name to the path template.
func (p *PathTemplate) WithVariableNamed(fields ...string) *PathTemplate {
	v := PathVariable{FieldPath: fields}
	p.Segments = append(p.Segments, PathSegment{Variable: v.WithMatch()})
	return p
}

// WithVerb adds a verb to the path template.
func (p *PathTemplate) WithVerb(v string) *PathTemplate {
	p.Verb = v
	return p
}
