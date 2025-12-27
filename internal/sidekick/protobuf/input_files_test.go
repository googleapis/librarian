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

package protobuf

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	testdataDir, _ = filepath.Abs("../../testdata")
)

const (
	sourceDir = "googleapis/google/cloud/secretmanager/v1"
)

func TestBasic(t *testing.T) {
	source := sourceDir
	options := map[string]string{
		"googleapis-root": testdataDir,
	}
	got, err := DetermineInputFiles(source, options)
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		got[i] = filepath.ToSlash(got[i])
	}
	want := []string{
		filepath.ToSlash(path.Join(testdataDir, source, "resources.proto")),
		filepath.ToSlash(path.Join(testdataDir, source, "service.proto")),
	}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched merged config (-want, +got):\n%s", diff)
	}
}

func TestTooManyOptions(t *testing.T) {
	source := sourceDir
	options := map[string]string{
		"googleapis-root": testdataDir,
		"exclude-list":    "d,e,f",
		"include-list":    "a,b,c",
	}
	_, err := DetermineInputFiles(source, options)
	if err == nil {
		t.Errorf("expected an error when setting both exclude-list and include-list")
	}
}

func TestIncludeList(t *testing.T) {
	source := sourceDir
	options := map[string]string{
		"googleapis-root": testdataDir,
		"include-list":    "resources.proto",
	}
	got, err := DetermineInputFiles(source, options)
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		got[i] = filepath.ToSlash(got[i])
	}
	want := []string{
		filepath.ToSlash(path.Join(testdataDir, source, "resources.proto")),
	}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched merged config (-want, +got):\n%s", diff)
	}
}

func TestExcludeList(t *testing.T) {
	source := sourceDir
	options := map[string]string{
		"googleapis-root": testdataDir,
		"exclude-list":    "resources.proto",
	}
	got, err := DetermineInputFiles(source, options)
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		got[i] = filepath.ToSlash(got[i])
	}
	want := []string{
		filepath.ToSlash(path.Join(testdataDir, source, "service.proto")),
	}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched merged config (-want, +got):\n%s", diff)
	}
}
