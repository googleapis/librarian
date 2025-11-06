// Copyright 2025 Google LLC
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

package doc

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestGoGenerateLibrarianDoc(t *testing.T) {
	for _, test := range []struct {
		name    string
		docFile string
	}{
		{
			name:    "automation_doc",
			docFile: "cmd/automation/doc.go",
		},
		{
			name:    "librarian_doc",
			docFile: "cmd/librarian/doc.go",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command("go", "generate", "./...")
			var stderr, stdout bytes.Buffer
			cmd.Stderr = &stderr
			cmd.Stdout = &stdout
			t.Log(stderr.String())
			t.Log(stdout.String())
			if err := cmd.Run(); err != nil {
				t.Fatalf("%v: %v", cmd, err)
			}
			cmd = exec.Command("git", "diff", "--exit-code", "--", test.docFile)
			if err := cmd.Run(); err != nil {
				t.Errorf("go generate produced a diff, please run `go generate ./...` and commit the changes")
				cmd = exec.Command("git", "diff", "--", test.docFile)
				out, _ := cmd.CombinedOutput()
				t.Logf("diff:\n%s", out)
			}
		})
	}
}
