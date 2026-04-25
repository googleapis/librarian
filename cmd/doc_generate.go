// Copyright 2024 Google LLC
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
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

const (
	librarianDesc = `Librarian manages Google Cloud client libraries. It runs a local workflow
that onboards new APIs, generates client code, bumps versions, publishes
releases, and tags release commits. Language-specific work, such as code
generation, building, and testing, is delegated to per-language tooling.

All behavior is driven by librarian.yaml at the root of the repository,
whose schema is documented at
https://github.com/googleapis/librarian/blob/main/doc/config-schema.md.

Usage:

	librarian <command> [arguments]

Global flags:

	--verbose, -v    enable verbose logging
`
	librarianopsDesc = `Librarianops orchestrates librarian operations across multiple repositories.

Usage:

	librarianops <command> [arguments]
`
	legacyLibrarianDesc = `Librarian manages Google API client libraries by automating onboarding,
regeneration, and release. It runs language-agnostic workflows while
delegating language-specific tasks—such as code generation, building, and
testing—to Docker images.

Usage:

	librarian <command> [arguments]
`
	automationDesc = `Automation provides logic to trigger Cloud Build jobs that run Librarian commands for
any repository listed in internal/legacylibrarian/legacyautomation/prod/repositories.yaml.

Usage:

	automation <command> [arguments]
`

	docTemplate = `// Copyright {{.Year}} Google LLC
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

//go:generate go run -tags docgen ../doc_generate.go -cmd .

/*
{{.Description}}{{range .Commands}}{{template "command" .}}{{end}}*/
package main

{{define "command"}}
# {{.Tagline}}

Usage:

	{{.Usage}}
{{if .Description}}
{{.Description}}
{{end}}{{if .Flags}}
Flags:

{{.Flags}}
{{end}}{{if .AfterFlags}}
{{.AfterFlags}}
{{end}}{{if .Commands}}{{range .Commands}}{{template "command" .}}{{end}}{{end}}{{end}}
`
)

// CommandDoc holds the documentation for a single CLI command, parsed from
// the urfave/cli "--help" output.
type CommandDoc struct {
	Name        string
	Summary     string
	Tagline     string
	Usage       string
	Description string
	Flags       string
	AfterFlags  string
	Commands    []CommandDoc
}

// afterFlagsMarker, when it appears on its own line inside a command's
// Description, splits the description: text above the marker stays in the
// Description (rendered before Flags), text below moves to AfterFlags
// (rendered after Flags). The marker itself is stripped from godoc output.
const afterFlagsMarker = "[after-flags]"

var (
	descriptions = map[string]string{
		"legacyautomation": automationDesc,
		"legacylibrarian":  legacyLibrarianDesc,
		"librarian":        librarianDesc,
		"librarianops":     librarianopsDesc,
	}

	years = map[string]string{
		"legacyautomation": "2025",
		"legacylibrarian":  "2025",
		"librarian":        "2026",
		"librarianops":     "2026",
	}

	cmdPath = flag.String("cmd", "", "Path to the command to generate docs for (e.g., ../../cmd/librarian)")

	sectionRE = regexp.MustCompile(`(?m)^([A-Z][A-Z ]*):\s*$`)
)

func main() {
	flag.Parse()
	if *cmdPath == "" {
		log.Fatal("must specify -cmd flag")
	}
	if err := run(cmdPath); err != nil {
		log.Fatal(err)
	}
}

func run(cmdPath *string) error {
	if err := processFile(cmdPath); err != nil {
		return err
	}
	cmd := exec.Command("goimports", "-w", "doc.go")
	if err := cmd.Run(); err != nil {
		log.Fatalf("goimports: %v", err)
	}
	return nil
}

func processFile(cmdPath *string) error {
	commands, err := buildCommandDocs("")
	if err != nil {
		return err
	}

	docFile, err := os.Create("doc.go")
	if err != nil {
		return fmt.Errorf("could not create doc.go: %v", err)
	}
	defer docFile.Close()

	pkgPath, err := filepath.Abs(*cmdPath)
	if err != nil {
		return fmt.Errorf("could not find path: %v", err)
	}

	pkgName := filepath.Base(pkgPath)
	pkg, ok := descriptions[pkgName]
	if !ok {
		return fmt.Errorf("cannot find description for command: %s", pkgPath)
	}
	year, ok := years[pkgName]
	if !ok {
		return fmt.Errorf("cannot find year for command: %s", pkgPath)
	}

	tmpl := template.Must(template.New("doc").Parse(docTemplate))
	if err := tmpl.Execute(docFile, struct {
		Year        string
		Description string
		Commands    []CommandDoc
	}{
		Year:        year,
		Description: sanitize(pkg),
		Commands:    commands,
	}); err != nil {
		return fmt.Errorf("could not execute template: %v", err)
	}
	return nil
}

func sanitize(s string) string {
	return strings.ReplaceAll(s, "*/", "* /")
}

func buildCommandDocs(parentCommand string) ([]CommandDoc, error) {
	var parentParts []string
	if parentCommand != "" {
		parentParts = strings.Fields(parentCommand)
	}

	args := []string{"run", "main.go"}
	args = append(args, parentParts...)
	cmd := exec.Command("go", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	// Ignore error, help text is printed on error when no subcommand is provided.
	_ = cmd.Run()

	commandNames, err := extractCommandNames(out.Bytes())
	if err != nil {
		return nil, nil
	}

	var commands []CommandDoc
	for _, name := range commandNames {
		fullCommandName := name
		if parentCommand != "" {
			fullCommandName = parentCommand + " " + name
		}

		helpText, err := getCommandHelpText(fullCommandName)
		if err != nil {
			return nil, fmt.Errorf("getting help text for command %s: %w", fullCommandName, err)
		}

		subCommands, err := buildCommandDocs(fullCommandName)
		if err != nil {
			return nil, err
		}

		doc := parseHelp(helpText)
		doc.Name = sanitize(fullCommandName)
		doc.Commands = subCommands
		commands = append(commands, doc)
	}

	return commands, nil
}

// parseHelp parses urfave/cli "--help" output into a CommandDoc, populating
// Tagline, Usage, Description, and Flags. The expected input format is:
//
//	NAME:
//	   <name> - <tagline>
//
//	USAGE:
//	   <usage>
//
//	DESCRIPTION:
//	   <description>
//
//	OPTIONS:
//	   <flag>  <flag help>
//
//	GLOBAL OPTIONS:
//	   <flag>  <flag help>
//
// GLOBAL OPTIONS are dropped (they are documented once at the package level).
// The "--help, -h" flag is filtered out of OPTIONS.
func parseHelp(help string) CommandDoc {
	sections := splitSections(help)

	var doc CommandDoc
	if name, ok := sections["NAME"]; ok {
		name = strings.TrimSpace(name)
		if i := strings.Index(name, " - "); i >= 0 {
			doc.Summary = strings.TrimSpace(name[i+3:])
			doc.Tagline = sentenceCase(doc.Summary)
		}
	}
	if u, ok := sections["USAGE"]; ok {
		doc.Usage = strings.TrimSpace(u)
	}
	if d, ok := sections["DESCRIPTION"]; ok {
		desc, after := splitAfterFlags(strings.TrimRight(dedent(d), "\n"))
		doc.Description = sanitize(desc)
		doc.AfterFlags = sanitize(after)
	}
	if o, ok := sections["OPTIONS"]; ok {
		doc.Flags = filterFlags(o)
	}
	return doc
}

// splitSections splits urfave/cli help text into sections keyed by their
// uppercase header (NAME, USAGE, DESCRIPTION, OPTIONS, GLOBAL OPTIONS).
func splitSections(help string) map[string]string {
	matches := sectionRE.FindAllStringIndex(help, -1)
	sections := make(map[string]string, len(matches))
	for i, m := range matches {
		header := strings.TrimSpace(help[m[0]:m[1]])
		header = strings.TrimSuffix(header, ":")
		start := m[1]
		end := len(help)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		sections[header] = help[start:end]
	}
	return sections
}

// splitAfterFlags splits a dedented description on a line that is exactly
// afterFlagsMarker. Text before the marker is returned as the description;
// text after the marker is returned as the after-flags content. If the
// marker is absent, the whole input is returned as the description.
func splitAfterFlags(desc string) (before, after string) {
	lines := strings.Split(desc, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == afterFlagsMarker {
			before = strings.TrimRight(strings.Join(lines[:i], "\n"), "\n")
			after = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
			return
		}
	}
	return desc, ""
}

// dedent strips the smallest common leading-whitespace prefix from every
// non-blank line of s and trims surrounding blank lines.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	min := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		i := 0
		for i < len(l) && (l[i] == ' ' || l[i] == '\t') {
			i++
		}
		if min == -1 || i < min {
			min = i
		}
	}
	if min <= 0 {
		return strings.Join(lines, "\n")
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if len(l) >= min {
			out[i] = l[min:]
		} else {
			out[i] = l
		}
	}
	return strings.Join(out, "\n")
}

// filterFlags removes the help flag from an OPTIONS block, dedents, and
// re-indents each remaining line with a single tab so godoc renders the
// block as preformatted text.
func filterFlags(opts string) string {
	body := dedent(opts)
	if body == "" {
		return ""
	}
	var out []string
	for _, l := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "--help") {
			continue
		}
		out = append(out, "\t"+l)
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n")
}

// sentenceCase capitalizes the first letter of s.
func sentenceCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func getCommandHelpText(command string) (string, error) {
	parts := strings.Fields(command)
	args := []string{"run", *cmdPath}
	args = append(args, parts...)
	args = append(args, "--help")
	cmd := exec.Command("go", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		if out.Len() == 0 {
			return "", fmt.Errorf("cmd.Run() for '%s --help' failed with %s\n%s", command, err, out.String())
		}
	}
	return out.String(), nil
}

func extractCommandNames(helpText []byte) ([]string, error) {
	ss := string(helpText)
	var start int
	headers := []string{"Commands:\n\n", "COMMANDS:\n"}
	for _, header := range headers {
		start = strings.Index(ss, header)
		if start != -1 {
			start += len(header)
			break
		}
	}
	if start == -1 {
		return nil, errors.New("could not find commands header")
	}

	commandsBlock := ss[start:]
	if end := strings.Index(commandsBlock, "\n\n"); end != -1 {
		commandsBlock = commandsBlock[:end]
	}

	var commandNames []string
	lines := strings.Split(strings.TrimSpace(commandsBlock), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			name := fields[0]
			name = strings.TrimSuffix(name, ",")
			if name == "help" || name == "h" {
				continue
			}
			commandNames = append(commandNames, name)
		}
	}
	return commandNames, nil
}
