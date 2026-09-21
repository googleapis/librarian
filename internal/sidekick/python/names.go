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
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// pythonKeywords contains keywords reserved in Python 3 and proto-plus.
var pythonKeywords = map[string]bool{
	"False":                 true,
	"None":                  true,
	"True":                  true,
	"__peg_parser__":        true,
	"all":                   true,
	"and":                   true,
	"any":                   true,
	"as":                    true,
	"assert":                true,
	"async":                 true,
	"await":                 true,
	"break":                 true,
	"breakpoint":            true,
	"class":                 true,
	"cls":                   true,
	"continue":              true,
	"def":                   true,
	"del":                   true,
	"dir":                   true,
	"elif":                  true,
	"else":                  true,
	"except":                true,
	"exec":                  true,
	"finally":               true,
	"for":                   true,
	"format":                true,
	"from":                  true,
	"global":                true,
	"hash":                  true,
	"help":                  true,
	"if":                    true,
	"ignore_unknown_fields": true,
	"import":                true,
	"in":                    true,
	"is":                    true,
	"lambda":                true,
	"license":               true,
	"list":                  true,
	"locals":                true,
	"mapping":               true,
	"max":                   true,
	"min":                   true,
	"next":                  true,
	"nonlocal":              true,
	"not":                   true,
	"object":                true,
	"open":                  true,
	"or":                    true,
	"pass":                  true,
	"raise":                 true,
	"range":                 true,
	"return":                true,
	"self":                  true,
	"slice":                 true,
	"try":                   true,
	"type":                  true,
	"while":                 true,
	"with":                  true,
	"yield":                 true,
	"zip":                   true,
}

// snakeCase converts a string to snake_case.
func snakeCase(s string) string {
	if strings.ToLower(s) == s && !strings.ContainsRune(s, '-') {
		return s
	}
	return strcase.ToSnake(s)
}

// pascalCase converts a string to PascalCase.
func pascalCase(s string) string {
	if s == "" {
		return ""
	}
	if !strings.ContainsAny(s, "_-") {
		r, size := utf8.DecodeRuneInString(s)
		return string(unicode.ToUpper(r)) + s[size:]
	}
	var sb strings.Builder
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' }) {
		if part == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(part)
		sb.WriteString(string(unicode.ToUpper(r)))
		sb.WriteString(part[size:])
	}
	return sb.String()
}

// pythonIdentifier escapes keywords with a trailing underscore.
func pythonIdentifier(s string) string {
	if pythonKeywords[s] {
		return s + "_"
	}
	return s
}

var versionSegmentRe = regexp.MustCompile(`^v\d+[a-z0-9]*$`)

func isVersionSegment(s string) bool {
	return versionSegmentRe.MatchString(s)
}

func packageInitials(pkg string) string {
	var initials strings.Builder
	for part := range strings.SplitSeq(pkg, ".") {
		if isVersionSegment(part) {
			continue
		}
		if part == "secretmanager" {
			initials.WriteString("sm")
			continue
		}
		for sp := range strings.SplitSeq(part, "_") {
			if isVersionSegment(sp) {
				continue
			}
			if sp == "secretmanager" {
				initials.WriteString("sm")
				continue
			}
			if len(sp) > 0 {
				initials.WriteByte(sp[0])
			}
		}
	}
	return initials.String()
}

func pythonModuleFromProto(protoPath string) string {
	trimmed := strings.TrimSuffix(protoPath, ".proto")
	dotted := strings.ReplaceAll(trimmed, "/", ".")
	return dotted + "_pb2"
}

func (c *codec) fileNeedsAlias(currentFile, targetStem string) bool {
	if c.Model == nil {
		return false
	}
	for _, msg := range c.Model.Messages {
		msgFile := ""
		if msg.SourceLocation != nil && msg.SourceLocation.File != "" {
			msgFile = msg.SourceLocation.File
		}
		if msgFile == currentFile {
			if checkMessageFieldsForName(msg, targetStem) {
				return true
			}
		}
	}
	return false
}

func checkMessageFieldsForName(msg *api.Message, name string) bool {
	for _, f := range msg.Fields {
		if pythonIdentifier(snakeCase(f.Name)) == name {
			return true
		}
	}
	for _, nested := range msg.Messages {
		if checkMessageFieldsForName(nested, name) {
			return true
		}
	}
	return false
}

func resolveProtoFile(loc *api.SourceLocation, pkg, name string) string {
	if loc != nil && loc.File != "" {
		return loc.File
	}
	if pkg != "" {
		return strings.ReplaceAll(pkg, ".", "/") + "/" + snakeCase(name) + ".proto"
	}
	return ""
}

// deriveGAPICNamespace derives the value to pass as python-gapic-namespace when
// it's not specified explicitly. This is the first two components of the API
// path (excluding any trailing version), dot-separated.
func deriveGAPICNamespace(apiPath string) string {
	version := serviceconfig.ExtractVersion(apiPath)
	if version != "" {
		apiPath = strings.TrimSuffix(apiPath, "/"+version)
	}
	parts := strings.Split(apiPath, "/")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return apiPath
}

// deriveGAPICName derives the value to pass as python-gapic-name when it's not
// specified explicitly. This is the path, without the leading namespace (after
// replacing dots with slashes), and without any version suffix, and then
// replacing slashes with underscores.
func deriveGAPICName(apiPath string) string {
	version := serviceconfig.ExtractVersion(apiPath)
	if version != "" {
		apiPath = strings.TrimSuffix(apiPath, "/"+version)
	}
	namespace := deriveGAPICNamespace(apiPath)
	apiPath = strings.TrimPrefix(apiPath, strings.ReplaceAll(namespace, ".", "/"))
	apiPath = strings.Trim(apiPath, "/")
	return strings.ReplaceAll(apiPath, "/", "_")
}

// typeNameFromID returns the unqualified type name from a fully-qualified proto ID.
func typeNameFromID(id string) string {
	if id == "" {
		return ""
	}
	parts := strings.Split(id, ".")
	return parts[len(parts)-1]
}

// pypiPackageName returns the PyPI distribution package name from the codec's
// PackageName or derives it from GAPICNamespace and GAPICName.
func (c *codec) pypiPackageName() string {
	if c.PackageName != "" {
		return c.PackageName
	}
	ns := strings.NewReplacer("/", "-", ".", "-").Replace(c.GAPICNamespace)
	switch {
	case ns != "" && c.GAPICName != "":
		return ns + "-" + c.GAPICName
	case ns != "":
		return ns
	default:
		return c.GAPICName
	}
}

// caseInsensitiveCompare compares two strings case-insensitively, using case-sensitive comparison as a tie-breaker.
func caseInsensitiveCompare(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// isIAMType checks if a protobuf type ID belongs to google.iam.v1 and returns the external
// pb2 module name, type name, and true if so.
func isIAMType(typeID string) (moduleName, typeName string, ok bool) {
	if strings.HasPrefix(typeID, ".google.iam.v1.") {
		tName := typeNameFromID(typeID)
		if tName == "Policy" {
			return "policy_pb2", "Policy", true
		}
		return "iam_policy_pb2", tName, true
	}
	return "", "", false
}
