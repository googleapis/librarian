// Copyright 2025 Google LLC
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

package rust

import (
	"testing"

	cmdtest "github.com/googleapis/librarian/internal/command"
)

func TestGetPackageName(t *testing.T) {
	expectedPackageName := "google-cloud-accessapproval-v1"
	got, err := getPackageName("testdata/package")
	if err != nil {
		t.Fatalf("error getting package name %v", err)
	}
	if got != expectedPackageName {
		t.Errorf("want packageName %s, got %s", expectedPackageName, got)
	}
}

func TestPrepareCargoWorkspace(t *testing.T) {
	cmdtest.RequireCommand(t, "cargo")
	cmdtest.RequireCommand(t, "taplo")
	prepareCargoWorkspace(t.Context(), "testdata")
}

func TestFormatAndValidateCreatedLibrary(t *testing.T) {
	cmdtest.RequireCommand(t, "cargo")
	cmdtest.RequireCommand(t, "env")
	cmdtest.RequireCommand(t, "typos")
	cmdtest.RequireCommand(t, "git")
	formatAndValidateLibrary(t.Context(), "testdata/package")
}
