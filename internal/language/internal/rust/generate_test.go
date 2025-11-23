// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rust

import (
	"path/filepath"
	"testing"

	cmdtest "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerate(t *testing.T) {
	cmdtest.RequireCommand(t, "protoc")
	testdataDir, err := filepath.Abs("../../../sidekick/testdata")
	if err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	googleapisDir := filepath.Join(testdataDir, "googleapis")
	library := &config.Library{
		Name:          "secretmanager",
		Output:        outDir,
		ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		Channel:       "google/cloud/secretmanager/v1",
		Version:       "0.1.0",
		ReleaseLevel:  "preview",
		CopyrightYear: "2025",
	}
	if err := Generate(t.Context(), library, googleapisDir); err != nil {
		t.Fatal(err)
	}
}
