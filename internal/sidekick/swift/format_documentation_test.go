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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func TestFormatDocumentation(t *testing.T) {

	codec := newTestCodec(t, api.NewTestAPI(nil, nil, nil).WithPackageName("test"), nil)

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
			name: "single line",
			doc:  "Hello world",
			want: []string{"Hello world"},
		},
		{
			name: "multiple lines",
			doc:  "Line 1\nLine 2",
			want: []string{"Line 1", "Line 2"},
		},
		{
			name: "trailing newline",
			doc:  "Line 1\n",
			want: []string{"Line 1", ""},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codec.formatDocumentation(test.doc, []string{})
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatDocumentationWithLinks(t *testing.T) {
	someMessage := api.NewTestMessage("SomeMessage").WithPackage("test.v1")
	model := api.NewTestAPI([]*api.Message{someMessage}, nil, nil)
	c := newTestCodec(t, model, nil)

	for _, test := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "no trailing newline",
			input: `Refer to [SomeMessage][] for details.`,
			want: []string{
				"Refer to [SomeMessage][] for details.",
				"",
				"[SomeMessage]: <doc:SomeMessage>",
			},
		},
		{
			name:  "trailing newline",
			input: "Refer to [SomeMessage][] for details.\n",
			want: []string{
				"Refer to [SomeMessage][] for details.",
				"",
				"[SomeMessage]: <doc:SomeMessage>",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.formatDocumentation(test.input, []string{"test.v1"})
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatReadmeDocumentation(t *testing.T) {
	pkg := "google.cloud.policysimulator.v1"
	replay := api.NewTestMessage("Replay").WithPackage(pkg)
	timestamp := api.NewTestMessage("Timestamp").WithPackage(wellKnownProtobufPackage)
	model := api.NewTestAPI([]*api.Message{replay, timestamp}, nil, nil).WithPackageName(pkg)
	c := newTestCodec(t, model, &config.Library{Version: "0.7.0"})

	for _, test := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "plain multi-paragraph with special characters",
			input: "Manages user's & group's data.\n\nSecond paragraph.",
			want: []string{
				"Manages user's & group's data.",
				"",
				"Second paragraph.",
			},
		},
		{
			name:  "resolved explicit cross-reference",
			input: "Viewing a [Replay][google.cloud.policysimulator.v1.Replay].",
			want: []string{
				"Viewing a [Replay][google.cloud.policysimulator.v1.Replay].",
				"",
				"[google.cloud.policysimulator.v1.Replay]: https://swiftpackageindex.com/googleapis/swift-google-cloud-policysimulator-v1/0.7.0/documentation/googlecloudpolicysimulatorv1/replay",
			},
		},
		{
			name:  "resolved implied cross-reference",
			input: "Viewing a [Replay][].",
			want: []string{
				"Viewing a [Replay][].",
				"",
				"[Replay]: https://swiftpackageindex.com/googleapis/swift-google-cloud-policysimulator-v1/0.7.0/documentation/googlecloudpolicysimulatorv1/replay",
			},
		},
		{
			name:  "mixed resolved and unresolved links",
			input: "Uses [Replay][google.cloud.policysimulator.v1.Replay] and [Unknown][].",
			want: []string{
				"Uses [Replay][google.cloud.policysimulator.v1.Replay] and Unknown.",
				"",
				"[google.cloud.policysimulator.v1.Replay]: https://swiftpackageindex.com/googleapis/swift-google-cloud-policysimulator-v1/0.7.0/documentation/googlecloudpolicysimulatorv1/replay",
			},
		},
		{
			name:  "external package cross-reference",
			input: "Uses [Timestamp][google.protobuf.Timestamp].",
			want: []string{
				"Uses [Timestamp][google.protobuf.Timestamp].",
				"",
				"[google.protobuf.Timestamp]: https://www.google.com/search?q=Swift+google.protobuf+GoogleWKT.WKTTimestamp",
			},
		},
		{
			name:  "inline code inside unresolved display text",
			input: "Viewing a [`Unknown`][google.cloud.policysimulator.v1.Unknown].",
			want:  []string{"Viewing a `Unknown`."},
		},
		{
			name:  "inline code span untouched",
			input: "Use `matrix[a][b]` or `[Foo][Bar]` literally.",
			want:  []string{"Use `matrix[a][b]` or `[Foo][Bar]` literally."},
		},
		{
			name:  "fenced code block untouched",
			input: "Example:\n\n```md\n[Unknown][google.cloud.policysimulator.v1.Unknown]\n```",
			want: []string{
				"Example:",
				"",
				"```md",
				"[Unknown][google.cloud.policysimulator.v1.Unknown]",
				"```",
			},
		},
		{
			name:  "inline url link untouched",
			input: "Read the [Cloud Monitoring documentation](https://cloud.google.com/monitoring/docs).",
			want:  []string{"Read the [Cloud Monitoring documentation](https://cloud.google.com/monitoring/docs)."},
		},
		{
			name:  "defined reference link untouched",
			input: "See [Google][g].\n\n[g]: https://google.com",
			want: []string{
				"See [Google][g].",
				"",
				"[g]: https://google.com",
			},
		},
		{
			name:  "non-proto brackets untouched",
			input: "Matches pattern [A-Za-z][a-z].",
			want:  []string{"Matches pattern [A-Za-z][a-z]."},
		},
		{
			name:  "brackets before inline link untouched",
			input: "Prefix [Foo][bar](https://example.com).",
			want:  []string{"Prefix [Foo][bar](https://example.com)."},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.formatReadmeDocumentation(test.input, []string{pkg})
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMatchUnresolvedProtoCrossRef(t *testing.T) {
	for _, test := range []struct {
		name          string
		input         string
		docWithRef    string
		resolvedLinks map[string]bool
		wantDisplay   string
		wantEnd       int
		wantOk        bool
	}{
		{
			name:        "unresolved explicit target",
			input:       "[Replay][google.cloud.policysimulator.v1.Replay] trailing",
			wantDisplay: "Replay",
			wantEnd:     len("[Replay][google.cloud.policysimulator.v1.Replay]"),
			wantOk:      true,
		},
		{
			name:        "unresolved implied target",
			input:       "[Replay][] trailing",
			wantDisplay: "Replay",
			wantEnd:     len("[Replay][]"),
			wantOk:      true,
		},
		{
			name:        "display with code span",
			input:       "[`Replay`][google.cloud.policysimulator.v1.Replay]",
			wantDisplay: "`Replay`",
			wantEnd:     len("[`Replay`][google.cloud.policysimulator.v1.Replay]"),
			wantOk:      true,
		},
		{
			name:        "escaped bracket inside display",
			input:       `[Foo\]Bar][pkg.Target]`,
			wantDisplay: `Foo\]Bar`,
			wantEnd:     len(`[Foo\]Bar][pkg.Target]`),
			wantOk:      true,
		},
		{
			name:        "escaped bracket inside target",
			input:       `[Foo][pkg.Bar\]Baz]`,
			wantDisplay: "Foo",
			wantEnd:     len(`[Foo][pkg.Bar\]Baz]`),
			wantOk:      false,
		},
		{
			name:          "already resolved link",
			input:         "[Replay][google.cloud.policysimulator.v1.Replay]",
			resolvedLinks: map[string]bool{"google.cloud.policysimulator.v1.Replay": true},
			wantOk:        false,
		},
		{
			name:       "already defined markdown reference link",
			input:      "[Google][g]",
			docWithRef: "[Google][g]\n\n[g]: https://google.com",
			wantOk:     false,
		},
		{
			name:   "followed by inline link url",
			input:  "[Foo][bar](https://example.com)",
			wantOk: false,
		},
		{
			name:   "non proto target identifier",
			input:  "[Foo][a-b-c]",
			wantOk: false,
		},
		{
			name:   "non proto implied target identifier",
			input:  "[a-b-c][]",
			wantOk: false,
		},
		{
			name:   "nested opening bracket in display",
			input:  "[[nested]][target]",
			wantOk: false,
		},
		{
			name:   "nested opening bracket in target",
			input:  "[foo][[bar]]",
			wantOk: false,
		},
		{
			name:   "unclosed display bracket",
			input:  "[foo",
			wantOk: false,
		},
		{
			name:   "empty display bracket",
			input:  "[][target]",
			wantOk: false,
		},
		{
			name:   "missing second bracket",
			input:  "[foo] not a link",
			wantOk: false,
		},
		{
			name:   "unclosed target bracket",
			input:  "[foo][bar",
			wantOk: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := parser.NewContext()
			if test.docWithRef != "" {
				md := goldmark.New()
				docBytes := []byte(test.docWithRef)
				_ = md.Parser().Parse(text.NewReader(docBytes), parser.WithContext(ctx))
			}
			gotDisplay, gotEnd, gotOk := matchUnresolvedProtoCrossRef(test.input, ctx, test.resolvedLinks)
			if gotOk != test.wantOk {
				t.Fatalf("ok mismatch: got %v, want %v", gotOk, test.wantOk)
			}
			if !test.wantOk {
				return
			}
			if gotDisplay != test.wantDisplay {
				t.Errorf("display mismatch: got %q, want %q", gotDisplay, test.wantDisplay)
			}
			if gotEnd != test.wantEnd {
				t.Errorf("end mismatch: got %d, want %d", gotEnd, test.wantEnd)
			}
		})
	}
}
