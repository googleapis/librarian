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

package parser

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func parseRoutingAnnotations(methodID string, m *descriptorpb.MethodDescriptorProto) ([]*api.RoutingInfo, error) {
	extensionId := eRouting
	if !proto.HasExtension(m.GetOptions(), extensionId) {
		return nil, nil
	}

	rule := proto.GetExtension(m.GetOptions(), extensionId).(*routingRule)
	var errs []error
	collect := map[string]*api.RoutingInfoBuilder{}
	collectVariants := map[string][]*api.RoutingInfoVariant{}
	for _, routing := range rule.GetRoutingParameters() {
		newInfo, err := parseRoutingInfo(methodID, routing)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		name := newInfo.Name()
		if _, ok := collect[name]; !ok {
			collect[name] = api.NewRoutingInfoBuilder().WithName(name)
		}
		collectVariants[name] = append(newInfo.Variants(), collectVariants[name]...)
	}
	if len(errs) != 0 {
		return nil, errors.Join(errs...)
	}
	info := []*api.RoutingInfo{}
	for _, k := range slices.Sorted(maps.Keys(collect)) {
		info = append(info, collect[k].WithVariants(collectVariants[k]...).Build())
	}
	return info, nil
}

func parseRoutingInfo(methodID string, routing *routingParameter) (*api.RoutingInfo, error) {
	pathTemplate := routing.GetPathTemplate()
	fieldName := routing.GetField()
	info, err := parseRoutingPathTemplate(fieldName, pathTemplate)
	if err != nil {
		return nil, fmt.Errorf("%w, method=%s", err, methodID)
	}
	return info, nil
}

func isValidRoutingIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		c := s[i]
		isAlnum := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
		if !isAlnum && c != '_' && c != '-' && c != '.' && c != '~' {
			return false
		}
	}
	return true
}

func isValidRoutingName(s string) bool {
	if !isValidRoutingIdent(s) {
		return false
	}
	for seg := range strings.SplitSeq(s, ".") {
		if seg == "" {
			return false
		}
	}
	return true
}

func parseRoutingPathTemplate(fieldName, pathTemplate string) (*api.RoutingInfo, error) {
	if fieldName == "" {
		return nil, fmt.Errorf("routing parameter field name cannot be empty, pathTemplate=%q", pathTemplate)
	}
	if !isValidRoutingName(fieldName) {
		return nil, fmt.Errorf("invalid routing parameter field name=%q", fieldName)
	}
	fieldPath := strings.Split(fieldName, ".")
	if pathTemplate == "" {
		info := api.NewRoutingInfoBuilder().
			WithName(fieldName).
			WithVariant(api.NewRoutingInfoVariantBuilder().
				WithFieldPath(fieldPath...).
				WithMatchingSegments(api.MultiSegmentWildcard).
				Build()).
			Build()
		return info, nil
	}
	// AIP-4222: "The last symbol in a path_template may be a delimiter - it will be ignored."
	if len(pathTemplate) > 1 && strings.HasSuffix(pathTemplate, "/") {
		pathTemplate = pathTemplate[:len(pathTemplate)-1]
	}
	if strings.Count(pathTemplate, api.MultiSegmentWildcard) > 1 {
		return nil, fmt.Errorf("too many `**` matchers in pathTemplate=%q", pathTemplate)
	}

	pos := 0
	prefix, width := parseRoutingPrefix(pathTemplate[pos:])
	pos += width
	if pos > 0 {
		if !strings.HasPrefix(pathTemplate[pos:], "/") {
			return nil, fmt.Errorf("expected '/' before '{', found=%s", pathTemplate[pos:])
		}
		pos += 1
	}
	if !strings.HasPrefix(pathTemplate[pos:], "{") {
		return nil, fmt.Errorf("expected '{', found=%s", pathTemplate[pos:])
	}
	pos += 1
	name, match, width, err := parseRoutingVariable(pathTemplate[pos:])
	if err != nil {
		return nil, err
	}
	pos += width
	if !strings.HasPrefix(pathTemplate[pos:], "}") {
		return nil, fmt.Errorf("expected '}', found=%s", pathTemplate[pos:])
	}
	pos += 1
	suffix := api.RoutingPathSpec{}
	if strings.HasPrefix(pathTemplate[pos:], "/") {
		pos += 1
		suffix, width = parseRoutingSuffix(pathTemplate[pos:])
		pos += width
		if len(suffix.Segments()) == 0 {
			return nil, fmt.Errorf("unexpected empty suffix after '/', template=%s", pathTemplate)
		}
	}
	if pathTemplate[pos:] != "" {
		return nil, fmt.Errorf("unexpected trailer in pathTemplate trailer=%s", pathTemplate[pos:])
	}
	all := slices.Concat(prefix.Segments(), match.Segments(), suffix.Segments())
	for _, seg := range all {
		if !isRoutingWildcard(seg) && !isValidRoutingIdent(seg) {
			return nil, fmt.Errorf("invalid routing path segment %q in template=%s", seg, pathTemplate)
		}
	}
	if index := slices.Index(all, api.MultiSegmentWildcard); index != -1 && index != len(all)-1 {
		return nil, fmt.Errorf("multi segment wildcards may only appear at the end of a path template, template=%s", pathTemplate)
	}
	info := api.NewRoutingInfoBuilder().
		WithName(name).
		WithVariant(api.NewRoutingInfoVariantBuilder().
			WithFieldPath(fieldPath...).
			WithPrefix(prefix).
			WithMatching(match).
			WithSuffix(suffix).
			Build()).
		Build()
	return info, nil
}

func parseRoutingPrefix(pathTemplate string) (api.RoutingPathSpec, int) {
	return parseRoutingPathSpec(pathTemplate)
}

func isRoutingWildcard(segment string) bool {
	return segment == api.SingleSegmentWildcard || segment == api.MultiSegmentWildcard
}

func parseRoutingVariable(pathTemplate string) (string, api.RoutingPathSpec, int, error) {
	spec, width := parseRoutingPathSpec(pathTemplate)
	segments := spec.Segments()
	if strings.HasPrefix(pathTemplate[width:], "=") {
		pos := width + 1
		// The initial spec must be a single valid variable name.
		if len(segments) != 1 || isRoutingWildcard(segments[0]) || !isValidRoutingName(segments[0]) {
			return "", api.RoutingPathSpec{}, 0, fmt.Errorf("expected name=pathspec, but the name format is invalid name=%v", segments)
		}
		name := segments[0]
		spec, width := parseRoutingPathSpec(pathTemplate[pos:])
		if len(spec.Segments()) == 0 {
			return "", api.RoutingPathSpec{}, 0, fmt.Errorf("expected non-empty pathspec in variable %q", name)
		}
		return name, spec, pos + width, nil
	}
	if len(segments) == 1 && !isRoutingWildcard(segments[0]) && isValidRoutingName(segments[0]) {
		// AIP-4222: It is acceptable to omit the pattern in the resource ID
		// segment, `{parent}` for example, is equivalent to `{parent=*}`.
		return segments[0], api.NewRoutingPathSpec(api.SingleSegmentWildcard), width, nil
	}
	return "", api.RoutingPathSpec{}, 0, fmt.Errorf("expected variable name or name=pathspec, got=%v", segments)
}

func parseRoutingSuffix(pathTemplate string) (api.RoutingPathSpec, int) {
	return parseRoutingPathSpec(pathTemplate)
}

func parseRoutingPathSpec(pathTemplate string) (api.RoutingPathSpec, int) {
	var segments []string
	pos := 0
	for {
		segment, width := parseRoutingSegment(pathTemplate[pos:])
		if segment == "" {
			break
		}
		segments = append(segments, segment)
		pos += width
		if !strings.HasPrefix(pathTemplate[pos:], "/") {
			break
		}
		nextSeg, _ := parseRoutingSegment(pathTemplate[pos+1:])
		if nextSeg == "" {
			break
		}
		pos += 1
	}
	return api.NewRoutingPathSpec(segments...), pos
}

func parseRoutingSegment(pathTemplate string) (string, int) {
	index := strings.IndexAny(pathTemplate, "=/{}")
	if index == -1 {
		return pathTemplate, len(pathTemplate)
	}
	return pathTemplate[:index], index
}
