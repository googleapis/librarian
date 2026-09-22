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
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// sampleIntegerFieldValue generates a mock integer sample value matching the GAPIC
// generator algorithm (sum of unicode ordinals of field name characters).
func sampleIntegerFieldValue(fieldName string) string {
	sum := 0
	for _, r := range fieldName {
		sum += int(r)
	}
	return strconv.Itoa(sum)
}

// sampleFloatFieldValue generates a mock float sample value matching the GAPIC
// generator algorithm ("0." followed by the sum of unicode ordinals of field name characters).
func sampleFloatFieldValue(fieldName string) string {
	sum := 0
	for _, r := range fieldName {
		sum += int(r)
	}
	return "0." + strconv.Itoa(sum)
}

// sampleRequestArg represents an argument passed to the request constructor in a code sample.
type sampleRequestArg struct {
	Name  string
	Value string
}

func (c *codec) buildSampleCode(m *api.Method, versionSegment string) ([]string, []*sampleRequestArg) {
	inMsg := m.InputType
	if inMsg == nil && m.InputTypeID != "" && c.Model != nil {
		inMsg = c.resolveMessageType(m.InputTypeID)
	}
	if inMsg == nil {
		return nil, nil
	}

	var preInitLines []string
	var args []*sampleRequestArg

	// Order: selected_oneofs (first option of each oneof) + required_fields (not in oneof)
	var requestFields []*api.Field
	for _, o := range inMsg.OneOfs {
		if len(o.Fields) > 0 {
			requestFields = append(requestFields, o.Fields[0])
		}
	}
	for _, f := range inMsg.Fields {
		if f.IsOneOf {
			continue
		}
		if slices.Contains(f.Behavior, api.FieldBehaviorRequired) {
			requestFields = append(requestFields, f)
		}
	}

	for _, f := range requestFields {
		if f.Typez == api.TypezMessage {
			target := f.MessageType
			if target == nil && f.TypezID != "" && c.Model != nil {
				target = c.resolveMessageType(f.TypezID)
			}
			if target != nil {
				lines := c.sampleMessageInit(versionSegment, pythonIdentifier(snakeCase(f.Name)), target)
				if len(lines) > 0 {
					if len(preInitLines) > 0 {
						preInitLines = append(preInitLines, "")
					}
					preInitLines = append(preInitLines, lines...)
					args = append(args, &sampleRequestArg{
						Name:  pythonIdentifier(snakeCase(f.Name)),
						Value: pythonIdentifier(snakeCase(f.Name)),
					})
				}
			}
			continue
		}

		val := c.sampleFieldValue(f)
		if val != "" {
			args = append(args, &sampleRequestArg{
				Name:  pythonIdentifier(snakeCase(f.Name)),
				Value: val,
			})
		}
	}

	return preInitLines, args
}

const maxSampleRecursionDepth = 5

func (c *codec) sampleFieldInitLines(varPrefix string, f *api.Field) []string {
	return c.sampleFieldInitLinesWithDepth(varPrefix, f, 0)
}

func (c *codec) sampleFieldInitLinesWithDepth(varPrefix string, f *api.Field, depth int) []string {
	if depth >= maxSampleRecursionDepth {
		return nil
	}
	if f.Typez == api.TypezMessage {
		target := f.MessageType
		if target == nil && f.TypezID != "" && c.Model != nil {
			target = c.resolveMessageType(f.TypezID)
		}
		if target == nil {
			return nil
		}
		var lines []string
		for _, subF := range target.Fields {
			if subF.IsOneOf {
				continue
			}
			if slices.Contains(subF.Behavior, api.FieldBehaviorRequired) {
				lines = append(lines, c.sampleFieldInitLinesWithDepth(varPrefix+"."+f.Name, subF, depth+1)...)
			}
		}
		for _, subO := range target.OneOfs {
			if len(subO.Fields) > 0 {
				leafF := subO.Fields[0]
				lines = append(lines, c.sampleFieldInitLinesWithDepth(varPrefix+"."+f.Name, leafF, depth+1)...)
			}
		}
		return lines
	}
	val := c.sampleFieldValue(f)
	if val != "" {
		return []string{fmt.Sprintf("%s.%s = %s", varPrefix, f.Name, val)}
	}
	return nil
}

func (c *codec) sampleMessageInit(versionSegment, varName string, msg *api.Message) []string {
	var lines []string
	constructor := fmt.Sprintf("%s = %s.%s()", varName, versionSegment, msg.Name)
	lines = append(lines, constructor)

	for _, f := range msg.Fields {
		if f.IsOneOf {
			continue
		}
		if slices.Contains(f.Behavior, api.FieldBehaviorRequired) {
			lines = append(lines, c.sampleFieldInitLines(varName, f)...)
		}
	}

	for _, o := range msg.OneOfs {
		if len(o.Fields) > 0 {
			f := o.Fields[0]
			lines = append(lines, c.sampleFieldInitLines(varName, f)...)
		}
	}

	if len(lines) == 1 {
		return nil
	}
	return lines
}

func (c *codec) sampleFieldValue(f *api.Field) string {
	switch f.Typez {
	case api.TypezString:
		if f.Repeated {
			return fmt.Sprintf("['%s_value1', '%s_value2']", f.Name, f.Name)
		}
		if f.Name == "name" {
			return `"name_value"`
		}
		return fmt.Sprintf("%q", f.Name+"_value")
	case api.TypezBytes:
		return fmt.Sprintf("b'%s_blob'", f.Name)
	case api.TypezEnum:
		target := f.EnumType
		if target == nil && f.TypezID != "" && c.Model != nil {
			target = c.Model.Enum(f.TypezID)
			if target == nil && strings.HasPrefix(f.TypezID, ".") {
				target = c.Model.Enum(strings.TrimPrefix(f.TypezID, "."))
			}
		}
		if target != nil && len(target.Values) > 0 {
			return fmt.Sprintf("%q", target.Values[len(target.Values)-1].Name)
		}
		return `"ENUM_VALUE"`
	case api.TypezInt32, api.TypezInt64, api.TypezUint32, api.TypezUint64,
		api.TypezSint32, api.TypezSint64, api.TypezFixed32, api.TypezFixed64,
		api.TypezSfixed32, api.TypezSfixed64:
		return sampleIntegerFieldValue(f.Name)
	case api.TypezBool:
		return "True"
	case api.TypezFloat, api.TypezDouble:
		return sampleFloatFieldValue(f.Name)
	}
	return ""
}
