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

package language

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func TestExtractCrossReferenceLinks(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name: "standard links",
			input: `
[Any][google.protobuf.Any]
[Message][test.v1.SomeMessage]
`,
			want: []string{"google.protobuf.Any", "test.v1.SomeMessage"},
		},
		{
			name: "implied links",
			input: `
implied service reference [SomeService][]
implied method reference [SomeService.CreateFoo][]
`,
			want: []string{"SomeService", "SomeService.CreateFoo"},
		},
		{
			name:  "no links",
			input: `Just some text without links.`,
			want:  nil,
		},
		{
			name:  "multiple links on one line",
			input: `[Service][test.v1.SomeService] [field][test.v1.SomeMessage.field]`,
			want:  []string{"test.v1.SomeMessage.field", "test.v1.SomeService"},
		},
		{
			name: "link definitions",
			input: `Link definitions should be added when collapsed links are used.
For example, [google][].
Second [example][].
[Third] example.
[google]: https://www.google.com
[example]: https://www.example.com
[Third]: https://www.third.com`,
			want: []string{"example", "google"},
		},
		{
			name: "explicit cross links",
			input: `
[Any][google.protobuf.Any]
[Message][test.v1.SomeMessage]
[Enum][test.v1.SomeMessage.SomeEnum]
[Message][test.v1.SomeMessage] repeated
[Service][test.v1.SomeService] [field][test.v1.SomeMessage.field]
[oneof group][test.v1.SomeMessage.result]
[oneof field][test.v1.SomeMessage.error]
[unmangled field][test.v1.SomeMessage.type] - normally r#type, but not in links
[SomeMessage.error][test.v1.SomeMessage.error]
[ExternalMessage][google.iam.v1.SetIamPolicyRequest]
[ExternalService][google.iam.v1.IAMPolicy]
[ENUM_VALUE][test.v1.SomeMessage.SomeEnum.ENUM_VALUE]
[SomeService.CreateFoo][test.v1.SomeService.CreateFoo]
[SomeService.CreateBar][test.v1.SomeService.CreateBar]
[a method][test.v1.YELL.CreateThing]
[the service name][test.v1.YELL]
[renamed service][test.v1.RenamedService]
[method of renamed service][test.v1.RenamedService.CreateFoo]
`,
			want: []string{
				"google.iam.v1.IAMPolicy",
				"google.iam.v1.SetIamPolicyRequest",
				"google.protobuf.Any",
				"test.v1.RenamedService",
				"test.v1.RenamedService.CreateFoo",
				"test.v1.SomeMessage",
				"test.v1.SomeMessage.SomeEnum",
				"test.v1.SomeMessage.SomeEnum.ENUM_VALUE",
				"test.v1.SomeMessage.error",
				"test.v1.SomeMessage.field",
				"test.v1.SomeMessage.result",
				"test.v1.SomeMessage.type",
				"test.v1.SomeService",
				"test.v1.SomeService.CreateBar",
				"test.v1.SomeService.CreateFoo",
				"test.v1.YELL",
				"test.v1.YELL.CreateThing",
			},
		},
		{
			name: "relative cross links",
			input: `
[relative link to service][SomeService]
[relative link to method][SomeService.CreateFoo]
[relative link to message][SomeMessage]
[relative link to message field][SomeMessage.field]
[relative link to message oneof group][SomeMessage.result]
[relative link to message oneof field][SomeMessage.error]
[relative link to unmangled field][SomeMessage.type]
[relative link to enum][SomeMessage.SomeEnum]
[relative link to enum value][SomeMessage.SomeEnum.ENUM_VALUE]
`,
			want: []string{
				"SomeMessage",
				"SomeMessage.SomeEnum",
				"SomeMessage.SomeEnum.ENUM_VALUE",
				"SomeMessage.error",
				"SomeMessage.field",
				"SomeMessage.result",
				"SomeMessage.type",
				"SomeService",
				"SomeService.CreateFoo",
			},
		},
		{
			name: "implied cross links",
			input: `
implied service reference [SomeService][]
implied method reference [SomeService.CreateFoo][]
implied message reference [SomeMessage][]
implied message field reference [SomeMessage.field][]
implied message oneof group reference [SomeMessage.result][]
implied message oneof field reference [SomeMessage.error][]
implied message unmangled field reference [SomeMessage.type][]
implied enum reference [SomeMessage.SomeEnum][]
implied enum value reference [SomeMessage.SomeEnum.ENUM_VALUE][]
`,
			want: []string{
				"SomeMessage",
				"SomeMessage.SomeEnum",
				"SomeMessage.SomeEnum.ENUM_VALUE",
				"SomeMessage.error",
				"SomeMessage.field",
				"SomeMessage.result",
				"SomeMessage.type",
				"SomeService",
				"SomeService.CreateFoo",
			},
		},
		{
			name:  "text block in list item",
			input: `- [ListMessage][test.v1.ListMessage]`,
			want:  []string{"test.v1.ListMessage"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			md := goldmark.New(
				goldmark.WithParserOptions(
					parser.WithAutoHeadingID(),
				),
			)
			doc := md.Parser().Parse(text.NewReader([]byte(test.input)))
			got := ExtractCrossReferenceLinks(doc, []byte(test.input))
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
