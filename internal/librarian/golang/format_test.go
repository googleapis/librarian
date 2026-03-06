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

package golang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "goimports")
	outDir := t.TempDir()
	goFile := filepath.Join(outDir, "test.go")
	unformatted := `package main

import (
"fmt"
"os"
)

func main() {
fmt.Println("Hello World")
}
`
	if err := os.WriteFile(goFile, []byte(unformatted), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Output: outDir,
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}

	gotBytes, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	want := `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello World")
}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
