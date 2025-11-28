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

package language

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestReleaseAll(t *testing.T) {
	cfg := &config.Config{
		Language: "testhelper",
		Libraries: []*config.Library{
			{Name: "lib1", Version: "0.1.0"},
			{Name: "lib2", Version: "0.2.0"},
		},
	}
	cfg, err := ReleaseAll(cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"lib1": TestReleaseVersion,
		"lib2": TestReleaseVersion,
	}
	if diff := cmp.Diff(want, libraryVersions(cfg)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReleaseLibrary(t *testing.T) {
	cfg := &config.Config{
		Language: "testhelper",
		Libraries: []*config.Library{
			{Name: "lib1", Version: "0.1.0"},
			{Name: "lib2", Version: "0.2.0"},
		},
	}
	cfg, err := ReleaseLibrary(cfg, "lib1")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"lib1": TestReleaseVersion,
		"lib2": "0.2.0",
	}
	if diff := cmp.Diff(want, libraryVersions(cfg)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func libraryVersions(cfg *config.Config) map[string]string {
	m := make(map[string]string)
	for _, lib := range cfg.Libraries {
		m[lib.Name] = lib.Version
	}
	return m
}
