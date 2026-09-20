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

package python

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerate(t *testing.T) {
	for _, test := range []struct {
		name        string
		library     *config.Library
		wantVersion string
		wantTyped   string
	}{
		{
			name: "emits gapic_version.py and py.typed",
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				Version:       "2.1.0",
				CopyrightYear: "2026",
			},
			wantVersion: `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
__version__ = "2.1.0"  # {x-release-please-version}
`,
			wantTyped: `# Marker file for PEP 561.
# The google-cloud-secretmanager package uses inline types.
`,
		},
		{
			name:    "emits gapic_version.py and py.typed with nil library defaults",
			library: nil,
			wantVersion: fmt.Sprintf(`# -*- coding: utf-8 -*-
# Copyright %d Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
__version__ = "0.0.0"  # {x-release-please-version}
`, time.Now().Year()),
			wantTyped: `# Marker file for PEP 561.
# The google-cloud-secretmanager package uses inline types.
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()

			msg := api.NewTestMessage("Secret").WithFields(
				api.NewTestField("name").WithType(api.TypezString),
			)
			svc := api.NewTestService("SecretManagerService")
			model := api.NewTestAPI([]*api.Message{msg}, nil, []*api.Service{svc})
			model.Name = "google-cloud-secretmanager"

			if err := Generate(t.Context(), model, outdir, test.library); err != nil {
				t.Fatal(err)
			}

			versionFile := filepath.Join(outdir, "gapic_version.py")
			versionBytes, err := os.ReadFile(versionFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantVersion, string(versionBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			typedFile := filepath.Join(outdir, "py.typed")
			typedBytes, err := os.ReadFile(typedFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantTyped, string(typedBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerate_Error(t *testing.T) {
	outdir := t.TempDir()
	err := Generate(t.Context(), nil, outdir, nil)
	if !errors.Is(err, ErrNilModel) {
		t.Errorf("got %v, want %v", err, ErrNilModel)
	}
}
