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

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

func TestFetchGoogleapisWithCommit(t *testing.T) {
	const (
		wantCommit = "abcd123"
		wantSHA    = "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8" // sha256 of "password"
	)
	// Mock GitHub server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/commits/") {
			w.Write([]byte(wantCommit))
			return
		}
		if strings.Contains(r.URL.Path, ".tar.gz") {
			w.Write([]byte("password"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	endpoints := &fetch.Endpoints{
		API:      ts.URL,
		Download: ts.URL,
	}
	// Mock cache
	tmp := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", tmp)
	// Pre-populate cache to avoid RepoDir downloading (which ignores our mock download URL)
	cachePath := filepath.Join(tmp, fmt.Sprintf("%s@%s", googleapisRepo, wantCommit))
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cachePath, "dummy"), []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := fetchGoogleapisWithCommit(t.Context(), endpoints, "master")
	if err != nil {
		t.Fatal(err)
	}

	want := &config.Source{
		Commit: wantCommit,
		SHA256: wantSHA,
		Dir:    cachePath,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
