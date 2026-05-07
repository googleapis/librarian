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

// goClientInfo describes the Go client and proto-Go packages for a proto
// package like "google.cloud.parallelstore.v1".
type goClientInfo struct {
	// Alias is the short name used as the import alias for the client
	// package, for example "parallelstore". The proto-Go package is
	// imported as Alias+"pb" (e.g. "parallelstorepb").
	Alias string

	// ClientPath is the import path of the GAPIC Go client package, for
	// example "cloud.google.com/go/parallelstore/apiv1".
	ClientPath string

	// PbPath is the import path of the proto-Go package, for example
	// "cloud.google.com/go/parallelstore/apiv1/parallelstorepb".
	PbPath string
}

// goClientPackage maps a proto package name like "google.cloud.parallelstore.v1"
// to its Go client (apiv1) and proto-Go (apiv1/parallelstorepb) packages.
// It returns nil when the proto package does not have the shape
// google.cloud.<short>.v<N>. Beta and alpha suffixes (e.g. v1beta1) are
// intentionally excluded for now. If clientImportPath is non-empty it
// overrides the proto-derived path.
func goClientPackage(protoPkg, clientImportPath string) *goClientInfo {
	if clientImportPath != "" {
		return clientInfoFromPath(clientImportPath)
	}
	rest, ok := strings.CutPrefix(protoPkg, "google.cloud.")
	if !ok {
		return nil
	}
	short, version, ok := strings.Cut(rest, ".")
	if !ok || !isLowerAlphanum(short) || !isStableVersion(version) {
		return nil
	}
	return &goClientInfo{
		Alias:      short,
		ClientPath: fmt.Sprintf("cloud.google.com/go/%s/api%s", short, version),
		PbPath:     fmt.Sprintf("cloud.google.com/go/%s/api%s/%spb", short, version, short),
	}
}

// clientInfoFromPath derives a goClientInfo from an explicit GAPIC Go client
// import path. It returns nil when the path's final segment is not an
// "api<version>" segment or when no usable alias can be found.
func clientInfoFromPath(clientImportPath string) *goClientInfo {
	segments := strings.Split(clientImportPath, "/")
	last := segments[len(segments)-1]
	if !strings.HasPrefix(last, "api") || len(last) == len("api") {
		return nil
	}

	var alias string
	for i := len(segments) - 2; i >= 0; i-- {
		s := segments[i]
		// Skip any segment that looks like a version (e.g., v1, v2, v1beta1).
		if strings.HasPrefix(s, "v") && len(s) > 1 && s[1] >= '0' && s[1] <= '9' {
			continue
		}
		alias = s
		break
	}
	if !isLowerAlphanum(alias) {
		return nil
	}
	return &goClientInfo{
		Alias:      alias,
		ClientPath: clientImportPath,
		PbPath:     clientImportPath + "/" + alias + "pb",
	}
}

// buildClientCall returns a ClientCall for an AIP-131 Get, AIP-132 List,
// AIP-133 Create, or AIP-135 Delete method when the model maps to a
// standard GAPIC Go package and the command composes a resource path.
// Create may produce additional CLI flags for the resource id and the
// body's scalar fields; those are returned as the second result. It
// returns (nil, nil) otherwise so the command keeps its print-only
// action.
func buildClientCall(method *api.Method, model *api.API, goClient *goClientInfo, hasPath bool) (*ClientCall, []Flag) {
	if goClient == nil || !hasPath {
		return nil, nil
	}
	if method.InputType == nil {
		return nil, nil
	}

	switch {
	case provider.IsGet(method):
		return &ClientCall{
			Method:      method.Name,
			NameField:   "Name",
			Package:     goClient.Alias,
			RequestType: goClient.Alias + "pb." + method.InputType.Name,
		}, nil
	case provider.IsList(method):
		return &ClientCall{
			Method:      method.Name,
			NameField:   "Parent",
			Package:     goClient.Alias,
			RequestType: goClient.Alias + "pb." + method.InputType.Name,
			Paged:       true,
		}, nil
	case provider.IsCreate(method):
		return buildCreateClientCall(method, model, goClient)
	case provider.IsDelete(method):
		return &ClientCall{
			Method:      method.Name,
			NameField:   "Name",
			Package:     goClient.Alias,
			RequestType: goClient.Alias + "pb." + method.InputType.Name,
			IsDelete:    true,
			IsLRO:       method.IsLRO,
		}, nil
	default:
		return nil, nil
	}
}

// buildCreateClientCall returns the ClientCall and extra flags for an
// AIP-133 Create method. Body-field walking descends into the request's
// resource body field; scalar fields become CLI flags; everything else
// is recorded as a TODO so the generated code documents what's missing.
func buildCreateClientCall(method *api.Method, model *api.API, goClient *goClientInfo) (*ClientCall, []Flag) {
	resource := provider.GetResourceForMethod(method, model)
	if resource == nil || resource.Self == nil {
		return nil, nil
	}

	var bodyField *api.Field
	for _, f := range method.InputType.Fields {
		if f.TypezID == resource.Self.ID {
			bodyField = f
			break
		}
	}
	if bodyField == nil || bodyField.MessageType == nil {
		return nil, nil
	}

	idFieldName := resource.Singular + "_id"
	var idField *api.Field
	for _, f := range method.InputType.Fields {
		if f.Name == idFieldName && f.Typez == api.TypezString {
			idField = f
			break
		}
	}

	call := &ClientCall{
		IsCreate:    true,
		IsLRO:       method.IsLRO,
		Method:      method.Name,
		NameField:   "Parent",
		Package:     goClient.Alias,
		RequestType: goClient.Alias + "pb." + method.InputType.Name,
		BodyField:   strcase.ToCamel(bodyField.Name),
		BodyType:    goClient.Alias + "pb." + bodyField.MessageType.Name,
	}
	if idField != nil {
		call.IDField = strcase.ToCamel(idField.Name)
		call.IDFlag = strcase.ToKebab(idField.Name)
	}

	var extraFlags []Flag
	if idField != nil {
		extraFlags = append(extraFlags, pathFlag(call.IDFlag))
	}

	for _, f := range bodyField.MessageType.Fields {
		if hasBehavior(f, api.FieldBehaviorOutputOnly) {
			continue
		}
		if hasBehavior(f, api.FieldBehaviorIdentifier) {
			continue
		}
		if f.Map {
			call.BodySkippedFields = append(call.BodySkippedFields,
				fmt.Sprintf("map field %q", f.Name))
			continue
		}
		if f.Repeated {
			call.BodySkippedFields = append(call.BodySkippedFields,
				fmt.Sprintf("repeated field %q", f.Name))
			continue
		}
		switch f.Typez {
		case api.TypezEnum:
			call.BodySkippedFields = append(call.BodySkippedFields,
				fmt.Sprintf("enum field %q", f.Name))
			continue
		case api.TypezMessage:
			call.BodySkippedFields = append(call.BodySkippedFields,
				fmt.Sprintf("message field %q", f.Name))
			continue
		}
		kind, ok := scalarKind(f.Typez)
		if !ok {
			call.BodySkippedFields = append(call.BodySkippedFields,
				fmt.Sprintf("unsupported scalar field %q", f.Name))
			continue
		}
		call.BodyAssignments = append(call.BodyAssignments, BodyAssignment{
			Name: strcase.ToCamel(f.Name),
			Flag: strcase.ToKebab(f.Name),
			Kind: kind,
		})
		extraFlags = append(extraFlags, flag(strcase.ToKebab(f.Name), kind, hasBehavior(f, api.FieldBehaviorRequired)))
	}
	return call, extraFlags
}

// hasBehavior reports whether f's Behavior list contains b.
func hasBehavior(f *api.Field, b api.FieldBehavior) bool {
	for _, x := range f.Behavior {
		if x == b {
			return true
		}
	}
	return false
}

// scalarKind maps a scalar Typez to the urfave/cli flag accessor name.
// It returns ("", false) for non-scalar or unsupported types.
func scalarKind(t api.Typez) (string, bool) {
	switch t {
	case api.TypezString:
		return "String", true
	case api.TypezInt32:
		return "Int32", true
	case api.TypezInt64:
		return "Int64", true
	case api.TypezBool:
		return "Bool", true
	default:
		return "", false
	}
}

// isLowerAlphanum reports whether s starts with a lowercase letter and
// contains only lowercase letters and digits.
func isLowerAlphanum(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// isStableVersion reports whether s is a stable proto version like "v1" or
// "v2": a "v" followed by one or more digits, with no alpha/beta suffix.
func isStableVersion(s string) bool {
	digits, ok := strings.CutPrefix(s, "v")
	if !ok || digits == "" {
		return false
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
	}
	return true
}
