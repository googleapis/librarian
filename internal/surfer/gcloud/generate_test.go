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

package gcloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateService(t *testing.T) {
	for _, test := range []struct {
		name    string
		service *api.Service
		model   *api.API
		wantErr bool
	}{
		{
			name: "Valid Service",
			service: &api.Service{
				Name:        "parallelstore.googleapis.com",
				DefaultHost: "parallelstore.googleapis.com",
				Methods: []*api.Method{
					{
						Name: "CreateInstance",
						InputType: &api.Message{
							Fields: []*api.Field{},
						},
						// Annotations needed for resource resolution would be complex to mock here completely
						// without a full parser run or extensive manual setup.
						// So we test the basic flow: it should create the service directory.
					},
				},
			},
			model: &api.API{
				ResourceDefinitions: []*api.Resource{},
			},
			wantErr: false,
		},
		{
			name: "Empty DefaultHost",
			service: &api.Service{
				Name:        "parallelstore.googleapis.com",
				DefaultHost: "",
			},
			model:   &api.API{},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			err := generateService(test.service, &Config{}, test.model, tmpDir)
			if (err != nil) != test.wantErr {
				t.Errorf("generateService() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestGenerateResourceCommands verifies that command files are generated.
func TestGenerateResourceCommands(t *testing.T) {
	// This tests the file writing logic specifically.
	tmpDir := t.TempDir()

	err := generateResourceCommands("instances", []*api.Method{
		{
			Name:      "CreateInstance",
			Service:   &api.Service{Package: "google.cloud.parallelstore.v1"},
			InputType: &api.Message{},
		},
	}, tmpDir, &Config{}, &api.API{}, &api.Service{DefaultHost: "parallelstore.googleapis.com"})

	if err != nil {
		t.Fatalf("generateResourceCommands() error = %v", err)
	}

	// Check if main command file exists
	mainFile := filepath.Join(tmpDir, "instances", "create.yaml")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", mainFile)
	}

	// Check content of main file
	content, _ := os.ReadFile(mainFile)
	wantContent := "_PARTIALS_: true\n"
	if diff := cmp.Diff(wantContent, string(content)); diff != "" {
		t.Errorf("main file content mismatch (-want +got):\n%s", diff)
	}
}
