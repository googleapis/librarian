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
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestIndent(t *testing.T) {
	for _, test := range []struct {
		name   string
		spaces int
		input  string
		want   string
	}{
		{
			name:   "empty string",
			spaces: 4,
			input:  "",
			want:   "",
		},
		{
			name:   "zero spaces",
			spaces: 0,
			input:  "line1\nline2",
			want:   "line1\nline2",
		},
		{
			name:   "negative spaces",
			spaces: -2,
			input:  "line1\nline2",
			want:   "line1\nline2",
		},
		{
			name:   "multiline with empty lines",
			spaces: 4,
			input:  "\nline1\n\nline2\n",
			want:   "\n    line1\n\n    line2\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := indent(test.spaces, test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTemplatesExecuteAndGenerate(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/main.swift.gotmpl": &fstest.MapFile{
			Data: []byte("\nstruct {{ .Name }} {\n{{- include \"templates/inner\" . | indent 4 }}\n{{- include \"templates/empty\" . | indent 4 }}\n}\n"),
		},
		"templates/inner.gotmpl": &fstest.MapFile{
			Data: []byte("\nlet x = {{ .Description }}\n\nlet y = 2\n"),
		},
		"templates/empty.gotmpl": &fstest.MapFile{
			Data: []byte("\n{{- if .Title }}\nlet z = 3\n{{- end }}\n"),
		},
	}
	tmpl := MustParseTemplates(fsys)
	if err := tmpl.Validate(); err != nil {
		t.Fatal(err)
	}

	model := api.NewTestAPI(nil, nil, nil)
	model.Name = "Outer"
	model.Description = "1"
	model.Title = ""

	outDir := t.TempDir()
	files := WalkTemplatesDir(fsys, "templates")
	if err := tmpl.GenerateFromModel(outDir, model, files); err != nil {
		t.Fatal(err)
	}

	gotBytes, err := os.ReadFile(filepath.Join(outDir, "main.swift"))
	if err != nil {
		t.Fatal(err)
	}
	want := "struct Outer {\n    let x = 1\n\n    let y = 2\n}\n"
	if diff := cmp.Diff(want, string(gotBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestTemplatesExecute_Error(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/bad_include.gotmpl": &fstest.MapFile{
			Data: []byte("{{- include \"templates/missing\" . }}\n"),
		},
		"templates/missing_key.gotmpl": &fstest.MapFile{
			Data: []byte("{{ .NonExistentKey }}\n"),
		},
	}
	tmpl, err := ParseTemplates(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpl.Execute("templates/bad_include", nil); err == nil {
		t.Error("Execute(bad_include) = nil, want error")
	}
	if _, err := tmpl.Execute("templates/missing_key", map[string]string{}); err == nil {
		t.Error("Execute(missing_key) = nil, want error")
	}
	if _, err := ParseTemplates(fstest.MapFS{
		"templates/broken.gotmpl": &fstest.MapFile{Data: []byte("{{ unclosed")},
	}); err == nil {
		t.Error("ParseTemplates(broken) = nil, want error")
	}
}

func TestValidate_PermittedConstructs(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{
			name: "variable declaration and field access",
			body: "{{- $root := . }}{{ $root.Name }}{{ .Codec.Name }}",
		},
		{
			name: "not include and indent",
			body: "{{- if not .Deprecated }}{{- include \"templates/other\" . | indent 4 }}{{- end }}",
		},
		{
			name: "range and with blocks with else",
			body: "{{- range $i, $v := .Items }}{{ $v }}{{- else }}none{{- end }}{{- with .Sub }}{{ .Name }}{{- end }}",
		},
		{
			name: "template action",
			body: "{{ template \"templates/other\" . }}",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := ParseTemplates(fstest.MapFS{
				"templates/test.gotmpl": &fstest.MapFile{Data: []byte(test.body)},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := tmpl.Validate(); err != nil {
				t.Errorf("Validate(%q) = %v, want nil", test.body, err)
			}
		})
	}
}

func TestValidate_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		body    string
		wantErr error
	}{
		{
			name:    "variable mutation",
			body:    "{{- $x := .Name }}{{- $x = \"mutated\" }}{{ $x }}",
			wantErr: ErrVariableMutation,
		},
		{
			name:    "break in range",
			body:    "{{- range .Items }}{{ break }}{{- end }}",
			wantErr: ErrLoopControl,
		},
		{
			name:    "continue in range",
			body:    "{{- range .Items }}{{ continue }}{{- end }}",
			wantErr: ErrLoopControl,
		},
		{
			name:    "banned builtin printf",
			body:    "{{ printf \"%s\" .Name }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin print",
			body:    "{{ print .Name }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin println",
			body:    "{{ println .Name }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin len",
			body:    "{{ len .Items }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin index",
			body:    "{{ index .Items 0 }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin slice",
			body:    "{{ slice .Items 0 1 }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin call",
			body:    "{{ call .Fn }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin eq",
			body:    "{{- if eq .A .B }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin ne",
			body:    "{{- if ne .A .B }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin lt",
			body:    "{{- if lt .A .B }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin le",
			body:    "{{- if le .A .B }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin gt",
			body:    "{{- if gt .A .B }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin ge",
			body:    "{{- if ge .A .B }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin and",
			body:    "{{- if and .A .B }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin or",
			body:    "{{- if or .A .B }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "banned builtin inside parenthesized subpipe",
			body:    "{{- if not (eq .A .B) }}yes{{- end }}",
			wantErr: ErrBannedBuiltin,
		},
		{
			name:    "parameterized method call on FieldNode",
			body:    "{{ .Helper.WithArg \"foo\" }}",
			wantErr: ErrParameterizedMethod,
		},
		{
			name:    "parameterized method call on ChainNode",
			body:    "{{ (.Helper.Chain).WithArg \"foo\" }}",
			wantErr: ErrParameterizedMethod,
		},
		{
			name:    "parameterized method call on VariableNode",
			body:    "{{- $h := .Helper }}{{ $h.WithArg \"foo\" }}",
			wantErr: ErrParameterizedMethod,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := ParseTemplates(fstest.MapFS{
				"templates/test.gotmpl": &fstest.MapFile{Data: []byte(test.body)},
			})
			if err != nil {
				t.Fatal(err)
			}
			gotErr := tmpl.Validate()
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Validate(%q) error = %v, wantErr %v", test.body, gotErr, test.wantErr)
			}
		})
	}
}
