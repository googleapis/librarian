// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build docgen

package doc

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

const docTemplate = `// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run -tags docgen doc_generate.go

/*
Librarian manages Google API client libraries by automating onboarding,
regeneration, and release. It runs language-agnostic workflows while
delegating language-specific tasks—such as code generation, building, and
testing—to Docker images.

Usage:

	librarian <command> [arguments]

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

// CommandDoc holds the documentation for a single CLI command.
type CommandDoc struct {
	Name     string
	HelpText string
	Commands []CommandDoc
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := processFile(); err != nil {
		return err
	}
	cmd := exec.Command("goimports", "-w", "doc.go")
	if err := cmd.Run(); err != nil {
		log.Fatalf("goimports: %v", err)
	}
	return nil
}

func processFile() error {
	commands, err := buildCommandDocs("")
	if err != nil {
		return err
	}

	docFile, err := os.Create("doc.go")
	if err != nil {
		return fmt.Errorf("could not create doc.go: %v", err)
	}
	defer docFile.Close()

	tmpl := template.Must(template.New("doc").Parse(docTemplate))
	if err := tmpl.Execute(docFile, struct{ Commands []CommandDoc }{Commands: commands}); err != nil {
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
	args := []string{"run", "../../cmd/librarian/"}
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
	const (
		commandsHeader = "Commands:\n\n"
	)
	ss := string(helpText)
	start := strings.Index(ss, commandsHeader)
	if start == -1 {
		return nil, errors.New("could not find commands header")
	}
	start += len(commandsHeader)

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
			commandNames = append(commandNames, fields[0])
		}
	}
	return commandNames, nil
}
