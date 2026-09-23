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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestConvertMarkdownToRst(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "fenced code block",
			input: "Example code:\n```python\nx = 42\nprint(x)\n```",
			want:  "Example code:\n\n\n::\n\n   x = 42\n   print(x)\n\n",
		},
		{
			name:  "markdown link conversion",
			input: "See the [Cloud Storage documentation](https://cloud.google.com/storage).",
			want:  "See the `Cloud Storage documentation\uE000<https://cloud.google.com/storage>`__.",
		},
		{
			name:  "bullet normalization",
			input: "* item one\n* item two\n+ item three\n- item four",
			want:  "- item one\n- item two\n- item three\n- item four",
		},
		{
			name:  "strip raw HTML tags while preserving placeholder with underscore",
			input: "Path: <canonical service name>/<type> with <asset type> and <service_account_email>",
			want:  "Path: / with  and <service_account_email>",
		},
		{
			name:  "strip raw HTML closing tags",
			input: `<a href="https://cloud.google.com">link</a>`,
			want:  "link",
		},
		{
			name:  "escape glob asterisk in prose",
			input: "Wildcard characters (such as * and ?) are supported.",
			want:  "Wildcard characters (such as \\* and ?) are supported.",
		},
		{
			name:  "escape domain glob asterisk",
			input: `"compute.googleapis.com.*" snapshots all compute resources.`,
			want:  `"compute.googleapis.com.\*" snapshots all compute resources.`,
		},
		{
			name:  "escape paired regex asterisk",
			input: `Pattern ".*Instance.*" matches instances.`,
			want:  `Pattern ".\ *Instance.*" matches instances.`,
		},
		{
			name:  "escape standalone underscore in parens",
			input: "Letters (A-Z), numbers (0-9), or underscores (_).",
			want:  "Letters (A-Z), numbers (0-9), or underscores (\\_).",
		},
		{
			name:  "in-word braced template emphasis",
			input: "Format is {SourceType}_{ACTION}_{DestType}.",
			want:  "Format is {SourceType}\\ *{ACTION}*\\ {DestType}.",
		},
		{
			name:  "quoted underscore replaced with asterisk",
			input: `Concatenated with "_" and separated by "_".`,
			want:  `Concatenated with "*" and separated by "*".`,
		},
		{
			name:  "code spans with single backticks become double",
			input: "Use `SecretPayload` to store data.",
			want:  "Use ``SecretPayload`` to store data.",
		},
		{
			name:  "quoted wildcard asterisk preserved",
			input: "Permissions with wildcards (such as '*' or 'storage.*') are not allowed.",
			want:  "Permissions with wildcards (such as '*' or 'storage.*') are not allowed.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := convertMarkdownToRst(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatMethodDocSummary(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		want       methodDocSummary
	}{
		{
			name:       "short method name",
			methodName: "CreateFoo",
			want: methodDocSummary{
				Lead: "create foo",
				Wrap: false,
			},
		},
		{
			name:       "long method name wrapping to second line",
			methodName: "AnalyzeOrgPolicyGovernedAssets",
			want: methodDocSummary{
				Lead: "analyze org policy governed",
				Rest: "assets",
				Wrap: true,
			},
		},
		{
			name:       "long method name with multiple words on rest",
			methodName: "AnalyzeOrgPolicyGovernedAssetsResponse",
			want: methodDocSummary{
				Lead: "analyze org policy governed",
				Rest: "assets response",
				Wrap: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatMethodDocSummary(test.methodName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWrapWords(t *testing.T) {
	for _, test := range []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{
			name:  "empty",
			text:  "",
			width: 20,
			want:  nil,
		},
		{
			name:  "single word fits",
			text:  "hello",
			width: 10,
			want:  []string{"hello"},
		},
		{
			name:  "multiple words wrapping",
			text:  "one two three four five six",
			width: 13,
			want:  []string{"one two three", "four five six"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := wrapWords(test.text, test.width)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatRstDocLines(t *testing.T) {
	for _, test := range []struct {
		name   string
		doc    string
		width  int
		indent int
		want   []string
	}{
		{
			name:   "empty",
			doc:    "",
			width:  72,
			indent: 4,
			want:   nil,
		},
		{
			name:   "whitespace only",
			doc:    "   \n  \t ",
			width:  72,
			indent: 4,
			want:   nil,
		},
		{
			name:   "single line",
			doc:    "Foo service documentation.",
			width:  72,
			indent: 4,
			want:   []string{"Foo service documentation."},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatRstDocLines(test.doc, test.width, test.indent)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatMethodReturnDoc(t *testing.T) {
	for _, test := range []struct {
		name string
		doc  string
		want []string
	}{
		{
			name: "empty",
			doc:  "",
			want: nil,
		},
		{
			name: "single line return doc",
			doc:  "The Secret object.",
			want: []string{"The Secret object."},
		},
		{
			name: "definition list return doc",
			doc:  "A [Secret][google.cloud.secretmanager.v1.Secret] object.\n\nDescription of the returned secret.",
			want: []string{
				"A [Secret][google.cloud.secretmanager.v1.Secret] object.",
				"   Description of the returned secret.",
			},
		},
		{
			name: "definition list return doc with multi-line definition",
			doc:  "A method returning:\n\n[ListWorkflows][google.cloud.workflows.v1.Workflows.ListWorkflows]\nmethod result.",
			want: []string{
				"A method returning:",
				"",
				"   [ListWorkflows][google.cloud.workflows.v1.Workflows.ListWorkflows]",
				"   method result.",
			},
		},
		{
			name: "definition list return doc for RetiredResource",
			doc: "A RetiredResource resource represents the record of a deleted\n" +
				"[CryptoKey][google.cloud.kms.v1.CryptoKey]. Its purpose is to provide\n" +
				"visibility into retained user data and to prevent reuse of these names for\n" +
				"new [CryptoKeys][google.cloud.kms.v1.CryptoKey].",
			want: []string{
				"A RetiredResource resource represents the record of a deleted",
				"   [CryptoKey][google.cloud.kms.v1.CryptoKey]. Its",
				"   purpose is to provide visibility into retained user",
				"   data and to prevent reuse of these names for new",
				"   [CryptoKeys][google.cloud.kms.v1.CryptoKey].",
			},
		},
		{
			name: "preserves markdown link in return doc without converting to rst link",
			doc: "An [ImportJob][google.cloud.kms.v1.ImportJob] can be used.\n\n" +
				"For more information, see [Importing a key](https://cloud.google.com/kms/docs/importing-a-key).",
			want: []string{
				"An [ImportJob][google.cloud.kms.v1.ImportJob] can be used.",
				"   For more information, see [Importing a",
				"   key](https://cloud.google.com/kms/docs/importing-a-key).",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatMethodReturnDoc(test.doc)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
