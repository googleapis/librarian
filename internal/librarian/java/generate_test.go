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

package java

import (
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateLibraries(t *testing.T) {
	libraries := []*config.Library{
		{
			Name: "test-lib",
			APIs: []*config.API{
				{Path: "google/cloud/test/v1"},
			},
		},
	}
	googleapisDir := "/tmp/googleapis"

	if err := GenerateLibraries(t.Context(), libraries, googleapisDir); err != nil {
		t.Errorf("GenerateLibraries() error = %v, want nil", err)
	}
}

func TestGenerateLibraries_Error(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries []*config.Library
	}{
		{
			name: "no apis",
			libraries: []*config.Library{
				{
					Name: "test-lib",
					APIs: nil,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := GenerateLibraries(t.Context(), test.libraries, "/tmp"); err == nil {
				t.Error("GenerateLibraries() error = nil, want error")
			}
		})
	}
}

func TestFormat(t *testing.T) {
	library := &config.Library{Name: "test-lib"}

	if err := Format(t.Context(), library); err != nil {
		t.Errorf("Format() error = %v, want nil", err)
	}
}

func TestClean(t *testing.T) {
	library := &config.Library{Name: "test-lib"}

	if err := Clean(library); err != nil {
		t.Errorf("Clean() error = %v, want nil", err)
	}
}
