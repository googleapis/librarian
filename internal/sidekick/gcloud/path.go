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

package gcloud

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
)

// subgroupName returns the kebab-cased last literal segment of the method's
// primary binding path. It returns ("", false) when the method has no
// primary binding or the binding has no literal segments.
func subgroupName(method *api.Method) (string, bool) {
	binding := provider.PrimaryBinding(method)
	if binding == nil {
		return "", false
	}
	segments := provider.GetLiteralSegments(binding.PathTemplate.Segments)
	if len(segments) == 0 {
		return "", false
	}
	return strcase.ToKebab(segments[len(segments)-1]), true
}

// buildCommand constructs a Command for a method. The command's flags name
// each component of the resource the method operates on, and (when the
// resource has any variables) the path is composed at runtime via
// [fmt.Sprintf].
func buildCommand(method *api.Method, model *api.API, commandName, subgroupName string) Command {
	segments := resourceSegments(method, model)
	cmd := Command{
		Name:  commandName,
		Usage: fmt.Sprintf("%s %s", commandName, subgroupName),
		Flags: pathFlagsFromSegments(segments),
	}
	if format := pathFormatFromSegments(segments); format != "" {
		cmd.PathFormat = format
		cmd.Args = pathArgsFromSegments(segments)
		cmd.PathLabel = pathLabel(method)
	}
	return cmd
}

// resourceSegments returns the resource pattern segments for a method, or
// nil when the method's resource cannot be resolved or has no pattern. For
// collection methods (List, Create, custom collection) the pattern is
// trimmed to the parent.
func resourceSegments(method *api.Method, model *api.API) []api.PathSegment {
	resource := provider.GetResourceForMethod(method, model)
	if resource == nil || len(resource.Patterns) == 0 {
		return nil
	}
	segments := resource.Patterns[0]
	if provider.IsCollectionMethod(method) {
		if parent := provider.GetParentFromSegments(segments); parent != nil {
			segments = parent
		}
	}
	return segments
}

// pathFlagsFromSegments returns one required string flag for each variable
// segment in the pattern, named after the variable's last FieldPath
// component. Duplicates (same FieldPath) are skipped.
func pathFlagsFromSegments(segments []api.PathSegment) []Flag {
	var flags []Flag
	seen := map[string]bool{}
	for _, seg := range segments {
		if seg.Variable == nil || len(seg.Variable.FieldPath) == 0 {
			continue
		}
		name := seg.Variable.FieldPath[len(seg.Variable.FieldPath)-1]
		if seen[name] {
			continue
		}
		seen[name] = true
		flags = append(flags, pathFlag(name))
	}
	return flags
}

// pathFormatFromSegments returns a "/"-joined format string with literals
// as themselves and variables as "%s", or "" if there are no variables.
func pathFormatFromSegments(segments []api.PathSegment) string {
	hasVar := false
	var parts []string
	for _, seg := range segments {
		switch {
		case seg.Literal != nil:
			parts = append(parts, *seg.Literal)
		case seg.Variable != nil && len(seg.Variable.FieldPath) > 0:
			parts = append(parts, "%s")
			hasVar = true
		}
	}
	if !hasVar {
		return ""
	}
	return strings.Join(parts, "/")
}

// pathArgsFromSegments returns the variable names in segment order, one
// per "%s" position in the format string from pathFormatFromSegments.
func pathArgsFromSegments(segments []api.PathSegment) []string {
	var args []string
	for _, seg := range segments {
		if seg.Variable == nil || len(seg.Variable.FieldPath) == 0 {
			continue
		}
		args = append(args, seg.Variable.FieldPath[len(seg.Variable.FieldPath)-1])
	}
	return args
}

// pathLabel returns the local variable name used in the generated action
// to hold the composed path. Collection methods compose the parent path,
// so the label is "parent"; resource methods compose the resource name.
func pathLabel(method *api.Method) string {
	if provider.IsCollectionMethod(method) {
		return "parent"
	}
	return "name"
}
