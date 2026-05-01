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

//go:build docgen

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseHelp(t *testing.T) {
	for _, test := range []struct {
		name string
		help string
		want CommandDoc
	}{
		{
			name: "full help text",
			help: `NAME:
   librarian generate - generate client library code

USAGE:
   librarian generate [options]

DESCRIPTION:
   Generate runs the configured generator to produce client
   library source code from API definitions.

OPTIONS:
   --api value   path to the API definition
   --help, -h    show help

GLOBAL OPTIONS:
   --verbose, -v   enable verbose logging
`,
			want: CommandDoc{
				Summary:     "generate client library code",
				Tagline:     "Generate client library code",
				Usage:       "librarian generate [options]",
				Description: "Generate runs the configured generator to produce client\nlibrary source code from API definitions.",
				Flags:       "\t--api value   path to the API definition",
			},
		},
		{
			name: "name and usage only",
			help: `NAME:
   librarian version - print the version

USAGE:
   librarian version
`,
			want: CommandDoc{
				Summary: "print the version",
				Tagline: "Print the version",
				Usage:   "librarian version",
			},
		},
		{
			name: "description with after-flags marker",
			help: `NAME:
   librarian release - cut a release

USAGE:
   librarian release [options]

DESCRIPTION:
   Release tags release commits and publishes artifacts.

   [after-flags]

   See https://example.com/release for the full release process.
`,
			want: CommandDoc{
				Summary:     "cut a release",
				Tagline:     "Cut a release",
				Usage:       "librarian release [options]",
				Description: "Release tags release commits and publishes artifacts.",
				AfterFlags:  "See https://example.com/release for the full release process.",
			},
		},
		{
			name: "description containing comment terminator",
			help: `NAME:
   librarian onboard - onboard a new API

USAGE:
   librarian onboard [options]

DESCRIPTION:
   Onboard reads API definitions from google/cloud/*/v1 and adds
   them to librarian.yaml.
`,
			want: CommandDoc{
				Summary:     "onboard a new API",
				Tagline:     "Onboard a new API",
				Usage:       "librarian onboard [options]",
				Description: "Onboard reads API definitions from google/cloud/* /v1 and adds\nthem to librarian.yaml.",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := parseHelp(test.help)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSplitSections(t *testing.T) {
	for _, test := range []struct {
		name string
		help string
		want map[string]string
	}{
		{
			name: "all sections",
			help: `NAME:
   librarian generate - generate client library code

USAGE:
   librarian generate [options]

DESCRIPTION:
   generates code from API definitions

OPTIONS:
   --api value  path to the API

GLOBAL OPTIONS:
   --verbose, -v  enable verbose logging
`,
			want: map[string]string{
				"NAME":           "\n   librarian generate - generate client library code\n\n",
				"USAGE":          "\n   librarian generate [options]\n\n",
				"DESCRIPTION":    "\n   generates code from API definitions\n\n",
				"OPTIONS":        "\n   --api value  path to the API\n\n",
				"GLOBAL OPTIONS": "\n   --verbose, -v  enable verbose logging\n",
			},
		},
		{
			name: "no headers",
			help: "just some text\nwith no headers\n",
			want: map[string]string{},
		},
		{
			name: "single section",
			help: `USAGE:
   librarian generate [options]
`,
			want: map[string]string{
				"USAGE": "\n   librarian generate [options]\n",
			},
		},
		{
			name: "non-consecutive sections",
			help: `USAGE:
   librarian generate [options]

OPTIONS:
   --api value  path to the API
`,
			want: map[string]string{
				"USAGE":   "\n   librarian generate [options]\n\n",
				"OPTIONS": "\n   --api value  path to the API\n",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := splitSections(test.help)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSplitAfterFlags(t *testing.T) {
	for _, test := range []struct {
		name       string
		desc       string
		wantBefore string
		wantAfter  string
	}{
		{
			name:       "marker present",
			desc:       "before text\n\n[after-flags]\n\nafter text",
			wantBefore: "before text",
			wantAfter:  "after text",
		},
		{
			name:       "marker absent",
			desc:       "just description text",
			wantBefore: "just description text",
			wantAfter:  "",
		},
		{
			name:       "marker with surrounding whitespace",
			desc:       "above\n   [after-flags]   \nbelow",
			wantBefore: "above",
			wantAfter:  "below",
		},
		{
			name:       "marker only",
			desc:       "[after-flags]",
			wantBefore: "",
			wantAfter:  "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotBefore, gotAfter := splitAfterFlags(test.desc)
			if diff := cmp.Diff(test.wantBefore, gotBefore); diff != "" {
				t.Errorf("before mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantAfter, gotAfter); diff != "" {
				t.Errorf("after mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDedent(t *testing.T) {
	for _, test := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "common indent stripped",
			in:   "    hello\n    world",
			want: "hello\nworld",
		},
		{
			name: "trims surrounding blank lines",
			in:   "\n\n   foo\n   bar\n\n",
			want: "foo\nbar",
		},
		{
			name: "all blank input",
			in:   "\n\n   \n\n",
			want: "",
		},
		{
			name: "no common indent",
			in:   "foo\n  bar",
			want: "foo\n  bar",
		},
		{
			name: "mixed indent uses smallest",
			in:   "  foo\n      bar\n    baz",
			want: "foo\n    bar\n  baz",
		},
		{
			name: "blank lines ignored when finding min",
			in:   "    foo\n\n    bar",
			want: "foo\n\nbar",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := dedent(test.in)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFilterFlags(t *testing.T) {
	for _, test := range []struct {
		name string
		opts string
		want string
	}{
		{
			name: "removes help and indents with tab",
			opts: "   --api value  path to the API\n   --help, -h   show help\n",
			want: "\t--api value  path to the API",
		},
		{
			name: "multiple non-help flags",
			opts: "   --api value  path to the API\n   --out value  output dir\n",
			want: "\t--api value  path to the API\n\t--out value  output dir",
		},
		{
			name: "only help flag",
			opts: "   --help, -h  show help\n",
			want: "",
		},
		{
			name: "empty input",
			opts: "",
			want: "",
		},
		{
			name: "skips blank lines",
			opts: "   --api value  do api\n\n   --out value  do out\n",
			want: "\t--api value  do api\n\t--out value  do out",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := filterFlags(test.opts)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSentenceCase(t *testing.T) {
	for _, test := range []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"lowercase", "hello", "Hello"},
		{"already capital", "Hello", "Hello"},
		{"single letter", "a", "A"},
		{"with spaces", "hello world", "Hello world"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := sentenceCase(test.in)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSanitize(t *testing.T) {
	for _, test := range []struct {
		name string
		in   string
		want string
	}{
		{"comment terminator", "see google/cloud/*/v1 for paths", "see google/cloud/* /v1 for paths"},
		{"multiple terminators", "google/cloud/*/v1 and google/cloud/*/v2", "google/cloud/* /v1 and google/cloud/* /v2"},
		{"no terminator", "no special chars", "no special chars"},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := sanitize(test.in)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractCommandNames(t *testing.T) {
	for _, test := range []struct {
		name     string
		helpText string
		want     []string
		wantErr  bool
	}{
		{
			name: "lowercase commands header",
			helpText: `Usage: foo [options]

Commands:

  generate   Generate code
  release    Cut a release
  help       Show help

Options:
  --verbose
`,
			want: []string{"generate", "release"},
		},
		{
			name: "uppercase commands header",
			helpText: `NAME:
   foo - do stuff

COMMANDS:
   generate  Generate code
   release   Cut a release
   h         alias for help

GLOBAL OPTIONS:
   --verbose
`,
			want: []string{"generate", "release"},
		},
		{
			name:     "no commands header",
			helpText: "Usage: foo\n\nOptions:\n  --verbose\n",
			wantErr:  true,
		},
		{
			name: "filters help and h",
			helpText: `COMMANDS:
   help   show help
   h      alias

`,
			want: nil,
		},
		{
			name: "trims trailing comma",
			helpText: `COMMANDS:
   generate, gen   Generate code

`,
			want: []string{"generate"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := extractCommandNames([]byte(test.helpText))
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
