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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/iancoleman/strcase"
)

type groupTracks struct {
	ga    *CommandGroup
	beta  *CommandGroup
	alpha *CommandGroup
}

func writeCommandGroupTree(outputDir string, baseModule string, tree *CommandGroupsByTrack) error {
	bundle := groupTracks{ga: tree.GA, beta: tree.BETA, alpha: tree.ALPHA}
	return writeGroup(outputDir, baseModule, bundle)
}

func writeGroup(outputDir string, baseModule string, b groupTracks) error {
	name := groupName(b)
	if name == "" {
		return nil
	}

	moduleName := strcase.ToSnake(name)
	groupDir := filepath.Join(outputDir, moduleName)
	if err := os.MkdirAll(groupDir, 0755); err != nil {
		return err
	}

	if err := writeCommandGroupFile(groupDir, baseModule, name, b); err != nil {
		return err
	}

	if err := writeGroupCommands(groupDir, b); err != nil {
		return err
	}

	return writeGroupSubgroups(groupDir, baseModule, b)
}

func groupName(b groupTracks) string {
	if b.ga != nil {
		return b.ga.Name
	}
	if b.beta != nil {
		return b.beta.Name
	}
	if b.alpha != nil {
		return b.alpha.Name
	}
	return ""
}

func writeGroupCommands(groupDir string, b groupTracks) error {
	cmdNames := make(map[string]bool)
	tracks := []*CommandGroup{b.ga, b.beta, b.alpha}
	for _, g := range tracks {
		if g != nil {
			for n := range g.Commands {
				cmdNames[n] = true
			}
		}
	}

	for verb := range cmdNames {
		var gaCmd, betaCmd, alphaCmd *Command
		if b.ga != nil {
			gaCmd = b.ga.Commands[verb]
		}
		if b.beta != nil {
			betaCmd = b.beta.Commands[verb]
		}
		if b.alpha != nil {
			alphaCmd = b.alpha.Commands[verb]
		}

		if err := writeCommandFiles(groupDir, verb, gaCmd, betaCmd, alphaCmd); err != nil {
			return err
		}
	}
	return nil
}

func writeGroupSubgroups(groupDir string, baseModule string, b groupTracks) error {
	subNames := make(map[string]bool)
	tracks := []*CommandGroup{b.ga, b.beta, b.alpha}
	for _, g := range tracks {
		if g != nil {
			for n := range g.Groups {
				subNames[n] = true
			}
		}
	}

	for sub := range subNames {
		subBundle := groupTracks{}
		if b.ga != nil {
			subBundle.ga = b.ga.Groups[sub]
		}
		if b.beta != nil {
			subBundle.beta = b.beta.Groups[sub]
		}
		if b.alpha != nil {
			subBundle.alpha = b.alpha.Groups[sub]
		}

		if err := writeGroup(groupDir, baseModule, subBundle); err != nil {
			return err
		}
	}
	return nil
}

func writeCommandGroupFile(dir string, baseModule string, name string, b groupTracks) error {
	initPath := filepath.Join(dir, "__init__.py")
	extPath := filepath.Join(dir, "_init_extensions.py")

	var path []string
	var primaryHelpText string
	var gaView, betaView, alphaView *trackView

	if b.ga != nil {
		gaView = &trackView{Name: "GA", HelpText: b.ga.HelpText}
		primaryHelpText = b.ga.HelpText
		path = b.ga.Path
	}
	if b.beta != nil {
		betaView = &trackView{Name: "BETA", HelpText: b.beta.HelpText}
		if primaryHelpText == "" {
			primaryHelpText = b.beta.HelpText
		}
		if len(path) == 0 {
			path = b.beta.Path
		}
	}
	if b.alpha != nil {
		alphaView = &trackView{Name: "ALPHA", HelpText: b.alpha.HelpText}
		if primaryHelpText == "" {
			primaryHelpText = b.alpha.HelpText
		}
		if len(path) == 0 {
			path = b.alpha.Path
		}
	}

	modulePathParts := make([]string, 0, 2+len(path))
	if baseModule != "" {
		modulePathParts = append(modulePathParts, baseModule)
		modulePathParts = append(modulePathParts, "surface")
	}
	for _, p := range path {
		modulePathParts = append(modulePathParts, strcase.ToSnake(p))
	}
	fullModulePath := strings.Join(modulePathParts, ".")

	isRoot := len(path) <= 1

	view := commandGroupView{
		Name:       name,
		ModulePath: fullModulePath,
		HelpText:   primaryHelpText,
		Year:       time.Now().Year(),
		IsRoot:     isRoot,
		GA:         gaView,
		BETA:       betaView,
		ALPHA:      alphaView,
	}

	var buf bytes.Buffer
	if err := commandGroupTemplate.Execute(&buf, view); err != nil {
		return err
	}

	if err := os.WriteFile(initPath, []byte(strings.TrimSpace(buf.String())+"\n"), 0644); err != nil {
		return err
	}

	buf.Reset()
	if err := initExtensionsTemplate.Execute(&buf, view); err != nil {
		return err
	}

	return os.WriteFile(extPath, []byte(strings.TrimSpace(buf.String())+"\n"), 0644)
}

type trackView struct {
	Name     string
	HelpText string
}

type commandGroupView struct {
	Name       string
	ModulePath string
	HelpText   string
	Year       int
	IsRoot     bool
	GA         *trackView
	BETA       *trackView
	ALPHA      *trackView
}

var commandGroupTemplate = template.Must(template.New("__init__.py").Funcs(template.FuncMap{
	"toCamel": strcase.ToCamel,
}).Parse(autogenTemplate + `"""{{.HelpText}}"""

from googlecloudsdk.calliope import base
from {{.ModulePath}} import _init_extensions as extensions


{{if .GA}}@base.ReleaseTracks(base.ReleaseTrack.GA)
@base.Autogenerated
class {{$.Name | toCamel}}Ga(extensions.{{$.Name | toCamel}}Ga):
  """{{.GA.HelpText}}"""
{{end}}{{if .BETA}}

@base.ReleaseTracks(base.ReleaseTrack.BETA)
@base.Autogenerated
class {{$.Name | toCamel}}Beta(extensions.{{$.Name | toCamel}}Beta):
  """{{.BETA.HelpText}}"""
{{end}}{{if .ALPHA}}

@base.ReleaseTracks(base.ReleaseTrack.ALPHA)
@base.Autogenerated
class {{$.Name | toCamel}}Alpha(extensions.{{$.Name | toCamel}}Alpha):
  """{{.ALPHA.HelpText}}"""
{{end}}
`))

var initExtensionsTemplate = template.Must(template.New("_init_extensions.py").Funcs(template.FuncMap{
	"toCamel": strcase.ToCamel,
}).Parse(licenseTemplate + `"""File to add optional custom code to extend __init__.py."""` + "\n" + `from googlecloudsdk.calliope import base


class {{$.Name | toCamel}}Alpha(base.Group):
  """Optional no-auto-generated code for ALPHA."""
{{if .IsRoot}}  category = base.UNCATEGORIZED_CATEGORY
{{end}}

class {{$.Name | toCamel}}Beta(base.Group):
  """Optional no-auto-generated code for BETA."""
{{if .IsRoot}}  category = base.UNCATEGORIZED_CATEGORY
{{end}}

class {{$.Name | toCamel}}Ga(base.Group):
  """Optional no-auto-generated code for GA."""
{{if .IsRoot}}  category = base.UNCATEGORIZED_CATEGORY
{{end}}`))
