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

package serviceconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReadCacheHitReturnsSamePointer verifies a second Read of an unchanged
// file is served from the cache (same pointer, no re-parse).
func TestReadCacheHitReturnsSamePointer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "svc.yaml")
	writeConfig(t, path, "first")

	a, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Error("expected cached Read to return the same *Service pointer")
	}
}

// TestReadCacheInvalidatesOnChange verifies that editing the file (new size and
// modtime) invalidates the cache and re-parses.
func TestReadCacheInvalidatesOnChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "svc.yaml")
	writeConfig(t, path, "first")
	first, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := first.GetName(); got != "first.googleapis.com" {
		t.Fatalf("name = %q, want first.googleapis.com", got)
	}

	// Rewrite with different content and a strictly newer modtime.
	writeConfig(t, path, "second-longer-name")
	if err := os.Chtimes(path, time.Now().Add(time.Second), time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	second, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := second.GetName(); got != "second-longer-name.googleapis.com" {
		t.Errorf("after edit, name = %q, want second-longer-name.googleapis.com (stale cache?)", got)
	}
}

func writeConfig(t *testing.T, path, name string) {
	t.Helper()
	data := []byte("type: google.api.Service\nname: " + name + ".googleapis.com\ntitle: Test API\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
