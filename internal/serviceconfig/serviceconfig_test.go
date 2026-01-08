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

package serviceconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

const googleapisDir = "../testdata/googleapis"

func TestRead(t *testing.T) {
	got, err := Read(filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want := &Service{
		Name:  "secretmanager.googleapis.com",
		Title: "Secret Manager API",
		Documentation: &Documentation{
			Summary: "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
		},
	}
	opts := cmp.Options{
		protocmp.Transform(),
		protocmp.IgnoreFields(&Service{}, "apis", "authentication", "config_version", "http", "publishing"),
		protocmp.IgnoreFields(&Documentation{}, "overview", "rules"),
	}
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// TestNoGenprotoServiceConfigImports verifies that the genproto serviceconfig
// dependency is isolated to this package.
func TestNoGenprotoServiceConfigImports(t *testing.T) {
	const genprotoImport = "google.golang.org/genproto/googleapis/api/serviceconfig"
	root := filepath.Join("..", "..")

	var violations []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil ||
			!strings.HasSuffix(path, ".go") ||
			strings.Contains(path, "/vendor/") ||
			strings.Contains(path, "/testdata/") ||
			strings.Contains(path, "internal/serviceconfig/") {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.Contains(string(content), genprotoImport) {
			relPath, _ := filepath.Rel(root, path)
			violations = append(violations, relPath)
		}
		return nil
	})
	if len(violations) > 0 {
		t.Errorf("Found %d file(s) importing %q outside of internal/serviceconfig:\n  %s",
			len(violations), genprotoImport, strings.Join(violations, "\n  "))
	}
}

func TestFind(t *testing.T) {
	for _, test := range []struct {
		name    string
		channel string
		want    string
		wantErr bool
	}{
		{
			name:    "found",
			channel: "google/cloud/secretmanager/v1",
			want:    "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		},
		{
			name:    "not found",
			channel: "google/cloud/compute/v1",
			want:    "",
		},
		{
			name:    "directory does not exist",
			channel: "google/cloud/nonexistent/v1",
			wantErr: true,
		},
		{
			name:    "override",
			channel: "google/cloud/aiplatform/schema/predict/instance",
			want:    "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Find(googleapisDir, test.channel)
			if err != nil {
				if !test.wantErr {
					t.Fatal(err)
				}
				return
			}
			if got != test.want {
				t.Errorf("Find() = %q, want %q", got, test.want)
			}
		})
	}
}
