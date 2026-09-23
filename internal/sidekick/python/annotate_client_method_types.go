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
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

var iamClientPolicyReturnDocLines = []string{
	"An Identity and Access Management (IAM) policy, which specifies access",
	"   controls for Google Cloud resources.",
	"",
	"   A Policy is a collection of bindings. A binding binds",
	"   one or more members, or principals, to a single role.",
	"   Principals can be user accounts, service accounts,",
	"   Google groups, and domains (such as G Suite). A role",
	"   is a named list of permissions; each role can be an",
	"   IAM predefined role or a user-created custom role.",
	"",
	"   For some types of Google Cloud resources, a binding",
	"   can also specify a condition, which is a logical",
	"   expression that allows access to a resource only if",
	"   the expression evaluates to true. A condition can add",
	"   constraints based on attributes of the request, the",
	"   resource, or both. To learn which resources support",
	"   conditions in their IAM policies, see the [IAM",
	"   documentation](https://cloud.google.com/iam/help/conditions/resource-policies).",
	"",
	"   **JSON example:**",
	"",
	"   :literal:``     {       \"bindings\": [         {           \"role\": \"roles/resourcemanager.organizationAdmin\",           \"members\": [             \"user:mike@example.com\",             \"group:admins@example.com\",             \"domain:google.com\",             \"serviceAccount:my-project-id@appspot.gserviceaccount.com\"           ]         },         {           \"role\": \"roles/resourcemanager.organizationViewer\",           \"members\": [             \"user:eve@example.com\"           ],           \"condition\": {             \"title\": \"expirable access\",             \"description\": \"Does not grant access after Sep 2020\",             \"expression\": \"request.time <             timestamp('2020-10-01T00:00:00.000Z')\",           }         }       ],       \"etag\": \"BwWWja0YfJA=\",       \"version\": 3     }`\\ \\`",
	"",
	"   **YAML example:**",
	"",
	"   :literal:``     bindings:     - members:       - user:mike@example.com       - group:admins@example.com       - domain:google.com       - serviceAccount:my-project-id@appspot.gserviceaccount.com       role: roles/resourcemanager.organizationAdmin     - members:       - user:eve@example.com       role: roles/resourcemanager.organizationViewer       condition:         title: expirable access         description: Does not grant access after Sep 2020         expression: request.time < timestamp('2020-10-01T00:00:00.000Z')     etag: BwWWja0YfJA=     version: 3`\\ \\`",
	"",
	"   For a description of IAM and its features, see the",
	"   [IAM",
	"   documentation](https://cloud.google.com/iam/docs/).",
}

// iamComputePolicyReturnDocLines provides the return docstring for compute.v1.Policy.
// In google.iam.v1.Policy (used by Secret Manager/KMS), the proto docstring has a newline
// between "<" and "timestamp", which collapses to 13 spaces. In google.cloud.compute.v1.Policy,
// the docstring is defined inline with a single space.
var iamComputePolicyReturnDocLines = makeComputePolicyReturnDocLines()

func makeComputePolicyReturnDocLines() []string {
	lines := make([]string, len(iamClientPolicyReturnDocLines))
	copy(lines, iamClientPolicyReturnDocLines)
	for i, l := range lines {
		lines[i] = strings.ReplaceAll(l, "request.time <             timestamp", "request.time < timestamp")
	}
	return lines
}

func findFieldInMessage(msg *api.Message, name string) *api.Field {
	if msg == nil {
		return nil
	}
	idx := slices.IndexFunc(msg.Fields, func(f *api.Field) bool {
		return f.Name == name
	})
	if idx < 0 {
		return nil
	}
	return msg.Fields[idx]
}

func (c *codec) resolveFieldTypeHintAndSphinx(f *api.Field, versionPackage string) (string, string) {
	if f == nil {
		return "str", "str"
	}

	var baseHint, baseSphinx string
	switch f.Typez {
	case api.TypezMessage:
		target := f.MessageType
		if target == nil && f.TypezID != "" && c.Model != nil {
			target = c.Model.Message(f.TypezID)
		}
		if target != nil {
			modelPkg := ""
			if c.Model != nil {
				modelPkg = c.Model.PackageName
			}
			targetPkg := target.Package
			if targetPkg == "" {
				targetPkg = modelPkg
			}
			targetFile := resolveProtoFile(target.SourceLocation, targetPkg, target.Name)
			if modelPkg != "" && targetPkg != modelPkg {
				stem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
				importAlias := stem + "_pb2"
				baseHint = importAlias + "." + relativeTypeName(target)
				module := pythonModuleFromProto(targetFile)
				baseSphinx = module + "." + relativeTypeName(target)
			} else {
				stem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
				baseHint = stem + "." + relativeTypeName(target)
				baseSphinx = versionPackage + ".types." + relativeTypeName(target)
			}
		} else {
			baseHint = "object"
			baseSphinx = "object"
		}
	case api.TypezEnum:
		target := f.EnumType
		if target == nil && f.TypezID != "" && c.Model != nil {
			target = c.Model.Enum(f.TypezID)
		}
		if target != nil {
			targetFile := resolveProtoFile(target.SourceLocation, target.Package, target.Name)
			stem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
			baseHint = stem + "." + relativeTypeName(target)
			baseSphinx = versionPackage + ".types." + relativeTypeName(target)
		} else {
			baseHint = "int"
			baseSphinx = "int"
		}
	default:
		prim := primitiveTypeHint(f.Typez)
		baseHint = prim
		baseSphinx = prim
	}

	if f.Repeated {
		return fmt.Sprintf("MutableSequence[%s]", baseHint), fmt.Sprintf("MutableSequence[%s]", baseSphinx)
	}
	return baseHint, baseSphinx
}

func (c *codec) resolveMessageStem(msg *api.Message) string {
	if msg == nil {
		return "common"
	}
	src := c.sourceFileForLocation(msg.SourceLocation)
	return strings.TrimSuffix(filepath.Base(src), ".proto")
}

func (c *codec) resolveMessageType(id string) *api.Message {
	if id == "" || c.Model == nil {
		return nil
	}
	if msg := c.Model.Message(id); msg != nil {
		return msg
	}
	if cut, ok := strings.CutPrefix(id, "."); ok {
		return c.Model.Message(cut)
	}
	return c.Model.Message("." + id)
}
