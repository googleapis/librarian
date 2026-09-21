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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// clientResourcePath represents a custom resource path helper and parser method.
type clientResourcePath struct {
	Name              string
	FunctionName      string
	ParseFunctionName string
	Docstring         string
	ParseDocstring    string
	Signature         string
	HasSignature      bool
	FormatPattern     string
	FormatArgs        string
	RegexPattern      string
}

func (c *codec) annotateCustomResourcePaths() []*clientResourcePath {
	var paths []*clientResourcePath
	seen := make(map[string]bool)

	addRes := func(r *api.Resource) {
		if r == nil || isCommonResource(r.Type) {
			return
		}
		name := snakeCase(r.Singular)
		if name == "" {
			if lastSlash := strings.LastIndex(r.Type, "/"); lastSlash >= 0 {
				name = snakeCase(r.Type[lastSlash+1:])
			} else if r.Self != nil {
				name = snakeCase(r.Self.Name)
			} else if r.Plural != "" {
				name = snakeCase(r.Plural)
			}
		}
		if name == "" || seen[name] {
			return
		}
		seen[name] = true

		patternStr := "*"
		var vars []string
		if len(r.Patterns) > 0 {
			pattern := r.Patterns[0]
			if len(pattern) == 1 && pattern[0].Literal == "*" {
				patternStr = "*"
			} else if len(pattern) > 0 {
				var b strings.Builder
				sep := ""
				for _, seg := range pattern {
					b.WriteString(sep)
					if seg.Literal != "" {
						b.WriteString(seg.Literal)
					} else if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
						vName := seg.Variable.FieldPath[0]
						vars = append(vars, vName)
						b.WriteString("{" + vName + "}")
					}
					sep = "/"
				}
				patternStr = b.String()
			}
		}

		funcName := name + "_path"
		parseFuncName := "parse_" + name + "_path"
		docstring := fmt.Sprintf("Returns a fully-qualified %s string.", name)
		parseDocstring := fmt.Sprintf("Parses a %s path into its component segments.", name)

		var sig, formatArgs, regex string
		if patternStr == "*" || len(vars) == 0 {
			sig = ""
			formatArgs = ""
			regex = "^.*$"
		} else {
			var sigParts []string
			for _, v := range vars {
				sigParts = append(sigParts, v+": str")
			}
			sig = strings.Join(sigParts, ",") + ","

			var argParts []string
			for _, v := range vars {
				argParts = append(argParts, fmt.Sprintf("%s=%s", v, v))
			}
			formatArgs = strings.Join(argParts, ", ") + ", "

			rePattern := patternStr
			for _, v := range vars {
				rePattern = strings.ReplaceAll(rePattern, "{"+v+"}", fmt.Sprintf("(?P<%s>.+?)", v))
			}
			regex = "^" + rePattern + "$"
		}

		paths = append(paths, &clientResourcePath{
			Name:              name,
			FunctionName:      funcName,
			ParseFunctionName: parseFuncName,
			Docstring:         docstring,
			ParseDocstring:    parseDocstring,
			Signature:         sig,
			HasSignature:      sig != "",
			FormatPattern:     patternStr,
			FormatArgs:        formatArgs,
			RegexPattern:      regex,
		})
	}

	if c.Model != nil {
		for _, r := range c.Model.ResourceDefinitions {
			addRes(r)
		}
		for r := range c.Model.AllResources() {
			addRes(r)
		}
		for _, m := range c.Model.Messages {
			if m.Resource != nil {
				addRes(m.Resource)
			}
		}
	}

	slices.SortFunc(paths, func(a, b *clientResourcePath) int {
		return strings.Compare(a.Name, b.Name)
	})

	return paths
}

func isCommonResource(typ string) bool {
	switch typ {
	case "cloudresourcemanager.googleapis.com/Project",
		"cloudresourcemanager.googleapis.com/Folder",
		"cloudresourcemanager.googleapis.com/Organization",
		"cloudresourcemanager.googleapis.com/Location",
		"cloudbilling.googleapis.com/BillingAccount":
		return true
	default:
		return false
	}
}
