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

package dart

import (
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestInstall(t *testing.T) {
	for _, test := range []struct {
		name  string
		tools *config.Tools
	}{
		{
			name:  "nil tools",
			tools: nil,
		},
		{
			name:  "empty tools",
			tools: &config.Tools{},
		},
		{
			name: "with protoc",
			tools: &config.Tools{
				Protoc: &config.Protoc{Version: "29.3"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Install(t.Context(), test.tools); err != nil {
				t.Fatalf("Install() unexpected error = %v", err)
			}
		})
	}
}
