// Copyright 2026 Google LLC
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
package upgrade

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestGetLatestLibrarianVersion(t *testing.T) {
	ctx := context.Background()
	t.Run("Verify Version Command Run and Output", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Version}}", "github.com/googleapis/librarian@main")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to run manual go list command: %v", err)
		}
		expectedVersion := strings.TrimSpace(string(out))
		actualVersion, err := GetLatestLibrarianVersion(ctx)
		if err != nil {
			t.Fatalf("getLatestLibrarianVersion returned an error: %v", err)
		}
		if actualVersion != expectedVersion {
			t.Errorf("Version mismatch!\nExpected: %q\nActual:   %q", expectedVersion, actualVersion)
		}
	})

	t.Run("Handle Command Error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Immediately cancel the context to force an error.
		_, err := GetLatestLibrarianVersion(ctx)
		if err == nil {
			t.Fatal("Expected an error for a canceled context, but got nil")
		}
		t.Logf("Successfully captured error as expected: %v", err)
	})
}
