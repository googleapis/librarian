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
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

var (
	// ErrVariableMutation indicates a template mutates an existing variable ($x = ...).
	ErrVariableMutation = errors.New("variable mutation is banned")
	// ErrLoopControl indicates a template uses break or continue inside a range loop.
	ErrLoopControl = errors.New("loop control statement is banned")
	// ErrBannedBuiltin indicates a template invokes a forbidden computational built-in.
	ErrBannedBuiltin = errors.New("computational built-in is banned")
	// ErrParameterizedMethod indicates a template calls a method with arguments.
	ErrParameterizedMethod = errors.New("parameterized method call is banned")

	bannedIdentifiers = map[string]bool{
		"printf":  true,
		"print":   true,
		"println": true,
		"len":     true,
		"index":   true,
		"slice":   true,
		"call":    true,
		"eq":      true,
		"ne":      true,
		"lt":      true,
		"le":      true,
		"gt":      true,
		"ge":      true,
		"and":     true,
		"or":      true,
	}
)

// Templates holds a parsed tree of Go templates (.gotmpl).
type Templates struct {
	tmpl *template.Template
}

// MustParseTemplates parses all .gotmpl files in fsys under "templates" and
// panics if parsing fails.
func MustParseTemplates(fsys fs.FS) *Templates {
	t, err := ParseTemplates(fsys)
	if err != nil {
		panic(fmt.Sprintf("failed to parse templates: %v", err))
	}
	return t
}

// ParseTemplates parses all .gotmpl files in fsys under "templates" into a
// shared template tree configured with missingkey=error and standard helpers
// ("include" and "indent").
func ParseTemplates(fsys fs.FS) (*Templates, error) {
	return ParseTemplatesDir(fsys, "templates")
}

// ParseTemplatesDir parses all .gotmpl files in fsys under root into a
// shared template tree configured with missingkey=error and standard helpers
// ("include" and "indent").
func ParseTemplatesDir(fsys fs.FS, root string) (*Templates, error) {
	t := &Templates{}
	funcMap := template.FuncMap{
		"include": func(name string, data any) (string, error) {
			out, err := t.Execute(name, data)
			if err != nil {
				return "", err
			}
			if out == "" {
				return "", nil
			}
			return "\n" + strings.TrimSuffix(out, "\n"), nil
		},
		"indent": indent,
	}
	tmpl := template.New("sidekick").Funcs(funcMap).Option("missingkey=error")
	t.tmpl = tmpl
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".gotmpl") {
			return nil
		}
		contents, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(path, ".gotmpl")
		_, err = tmpl.New(name).Parse(string(contents))
		return err
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Validate inspects the parsed AST of all templates in t and returns an error
// if any template uses banned constructs (variable mutation, break/continue,
// computational built-ins, or parameterized method calls).
func (t *Templates) Validate() error {
	for _, tmpl := range t.tmpl.Templates() {
		if tmpl.Tree == nil || tmpl.Root == nil {
			continue
		}
		if err := validateNode(tmpl.Root); err != nil {
			return fmt.Errorf("template %q: %w", tmpl.Name(), err)
		}
	}
	return nil
}

// Execute renders the named template (with or without ".gotmpl" suffix) and
// trims a single leading newline emitted after the template license comment.
func (t *Templates) Execute(name string, data any) (string, error) {
	name = strings.TrimSuffix(filepath.ToSlash(name), ".gotmpl")
	var buf bytes.Buffer
	if err := t.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return strings.TrimPrefix(buf.String(), "\n"), nil
}

// GenerateElement renders a single template file for element and writes it to outDir.
func (t *Templates) GenerateElement(outDir string, element any, gen GeneratedFile) error {
	s, err := t.Execute(gen.TemplatePath, element)
	if err != nil {
		return err
	}
	destination := filepath.Join(outDir, gen.OutputPath)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, []byte(s), 0o666)
}

// GenerateFromModel renders a slice of files using model as the root template context.
func (t *Templates) GenerateFromModel(outDir string, model *api.API, generatedFiles []GeneratedFile) error {
	for _, gen := range generatedFiles {
		if err := t.GenerateElement(outDir, model, gen); err != nil {
			return err
		}
	}
	return nil
}

// GenerateService renders a single file using the api.Service model.
func (t *Templates) GenerateService(outDir string, service *api.Service, gen GeneratedFile) error {
	return t.GenerateElement(outDir, service, gen)
}

// GenerateMethod renders a single file using the api.Method model.
func (t *Templates) GenerateMethod(outDir string, method *api.Method, gen GeneratedFile) error {
	return t.GenerateElement(outDir, method, gen)
}

// GenerateMessage renders a single file using the api.Message model.
func (t *Templates) GenerateMessage(outDir string, message *api.Message, gen GeneratedFile) error {
	return t.GenerateElement(outDir, message, gen)
}

// GenerateEnum renders a single file using the api.Enum model.
func (t *Templates) GenerateEnum(outDir string, enum *api.Enum, gen GeneratedFile) error {
	return t.GenerateElement(outDir, enum, gen)
}

func indent(spaces int, s string) string {
	if s == "" || spaces <= 0 {
		return s
	}
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = pad + line
		}
	}
	return strings.Join(lines, "\n")
}

func validateNode(node parse.Node) error {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return nil
		}
		for _, child := range n.Nodes {
			if err := validateNode(child); err != nil {
				return err
			}
		}
	case *parse.ActionNode:
		return validateNode(n.Pipe)
	case *parse.IfNode:
		return validateBranch(&n.BranchNode)
	case *parse.RangeNode:
		return validateBranch(&n.BranchNode)
	case *parse.WithNode:
		return validateBranch(&n.BranchNode)
	case *parse.TemplateNode:
		return validateNode(n.Pipe)
	case *parse.PipeNode:
		if n == nil {
			return nil
		}
		if n.IsAssign {
			return fmt.Errorf("%w: %s", ErrVariableMutation, n)
		}
		for _, cmd := range n.Cmds {
			if err := validateNode(cmd); err != nil {
				return err
			}
		}
	case *parse.CommandNode:
		if len(n.Args) > 1 {
			switch n.Args[0].(type) {
			case *parse.FieldNode, *parse.ChainNode, *parse.VariableNode:
				return fmt.Errorf("%w: %s", ErrParameterizedMethod, n)
			}
		}
		for _, arg := range n.Args {
			if err := validateNode(arg); err != nil {
				return err
			}
		}
	case *parse.ChainNode:
		return validateNode(n.Node)
	case *parse.IdentifierNode:
		if bannedIdentifiers[n.Ident] {
			return fmt.Errorf("%w: %q", ErrBannedBuiltin, n.Ident)
		}
	case *parse.BreakNode, *parse.ContinueNode:
		return fmt.Errorf("%w: %s", ErrLoopControl, n)
	}
	return nil
}

func validateBranch(b *parse.BranchNode) error {
	if err := validateNode(b.Pipe); err != nil {
		return err
	}
	if err := validateNode(b.List); err != nil {
		return err
	}
	return validateNode(b.ElseList)
}
