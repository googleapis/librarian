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
	"strings"
	"text/template"
)

const (
	librarianDesc = `Librarian CLI runs local workflow that 
	adds, generates, updates and publishes client libraries.

Usage:

	librarian <command> [arguments]
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
{{.Description}}

The commands are:
{{range .Commands}}{{template "command" .}}{{end}}
*/
package main

{{define "command"}}

# {{.Name}}

{{.HelpText}}
{{if .Commands}}
{{range .Commands}}{{template "command" .}}{{end}}
{{end}}
{{end}}
`
)

// CommandDoc holds the documentation for a single CLI command.
type CommandDoc struct {
	Name     string
	HelpText string
	Commands []CommandDoc
}

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
		year = "2026"
	}

	tmpl := template.Must(template.New("doc").Parse(docTemplate))
	if err := tmpl.Execute(docFile, struct {
		Year        string
		Description string
		Commands    []CommandDoc
	}{
		Year:        year,
		Description: pkg,
		Commands:    commands,
	}); err != nil {
		return fmt.Errorf("could not execute template: %v", err)
	}
	return nil
}

func buildCommandDocs(parentCommand string) ([]CommandDoc, error) {
	var parentParts []string
	if parentCommand != "" {
		parentParts = strings.Fields(parentCommand)
	}

	// Get help text for parent to find subcommands.
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
		// Not an error, just means no subcommands.
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

		// Recurse.
		subCommands, err := buildCommandDocs(fullCommandName)
		if err != nil {
			return nil, err
		}

		commands = append(commands, CommandDoc{
			Name:     fullCommandName,
			HelpText: helpText,
			Commands: subCommands,
		})
	}

	return commands, nil
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
		// The help command also exits with status 1.
		if out.Len() == 0 {
			return "", fmt.Errorf("cmd.Run() for '%s --help' failed with %s\n%s", command, err, out.String())
		}
	}
	return out.String(), nil
}

func extractCommandNames(helpText []byte) ([]string, error) {
	ss := string(helpText)
	var start int
	// handle both legacy tool and urfave/cli/v3 style
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
			// Handle urfave/cli "help, h" style and filter out help command.
			name = strings.TrimSuffix(name, ",")
			if name == "help" || name == "h" {
				continue
			}
			commandNames = append(commandNames, name)
		}
	}
	return commandNames, nil
}
