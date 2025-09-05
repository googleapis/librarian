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

//go:build ignore

package main

import (
	"bytes"
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

//go:generate go run doc_generate.go

/*
Package librarian contains the business logic for the Librarian CLI.
Implementation details for interacting with other systems (Git, GitHub,
Docker etc.) are abstracted into other packages.

Usage:

	librarian <command> [arguments]

The commands are:
{{.Commands}}
*/
package librarian
`

func main() {
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
			log.Fatalf("cmd.Run() failed with %s\n%s", err, out.String())
		}
	}
	helpText := out.Bytes()

	commands := extractCommands(helpText)

	docFile, err := os.Create("doc.go")
	if err != nil {
		log.Fatalf("could not create doc.go: %v", err)
	}
	defer docFile.Close()

	tmpl := template.Must(template.New("doc").Parse(docTemplate))
	if err := tmpl.Execute(docFile, struct{ Commands string }{Commands: string(commands)}); err != nil {
		log.Fatalf("could not execute template: %v", err)
	}

	cmd = exec.Command("goimports", "-w", "doc.go")
	if err := cmd.Run(); err != nil {
		log.Fatalf("goimports: %v", err)
	}
}

func extractCommands(helpText []byte) []byte {
	const (
		commandsHeader = "Commands:\n\n"
	)
	ss := string(helpText)
	start := strings.Index(ss, commandsHeader)
	if start == -1 {
		return helpText
	}
	start += len(commandsHeader)
	return []byte(ss[start:])
}
