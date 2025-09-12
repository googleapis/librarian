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

package main

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
Package librarian contains the business logic for the Librarian CLI.
Implementation details for interacting with other systems (Git, GitHub,
Docker etc.) are abstracted into other packages.

Usage:

	librarian <command> [arguments]

The commands are:
{{range .Commands}}

# {{.Name}}

{{.HelpText}}
{{end}}
*/
package librarian
`

// CommandDoc holds the documentation for a single CLI command.
type CommandDoc struct {
	Name     string
	HelpText string
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
	// Get the help text.
	cmd := exec.Command("go", "run", "../../cmd/librarian/")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		// The command exits with status 1 if no subcommand is given, which is
		// the case when we are generating the help text. We can ignore the
		// error if there is output.
		if out.Len() == 0 {
			return fmt.Errorf("cmd.Run() failed with %s\n%s", err, out.String())
		}
	}
	helpText := out.Bytes()

	commandNames, err := extractCommandNames(helpText)
	if err != nil {
		return err
	}

	var commands []CommandDoc
	for _, name := range commandNames {
		help, err := getCommandHelp(name)
		if err != nil {
			return fmt.Errorf("getting help for command %s: %w", name, err)
		}
		commands = append(commands, CommandDoc{Name: name, HelpText: help})
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

func getCommandHelp(command string) (string, error) {
	cmd := exec.Command("go", "run", "../../cmd/librarian/", command, "--help")
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
