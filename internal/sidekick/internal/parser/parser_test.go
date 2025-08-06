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

package parser

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
)

var (
	testdataDir, _ = filepath.Abs("../../testdata")
)

func TestCreateModelOpenAPI(t *testing.T) {
	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "openapi",
			ServiceConfig:       path.Join(testdataDir, "googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
			SpecificationSource: path.Join(testdataDir, "openapi/secretmanager_openapi_v1.json"),
		},
	}
	model, err := CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := model.State.ServiceByID[".google.cloud.secretmanager.v1.SecretManagerService"]
	if !ok {
		t.Errorf("missing service (.google.cloud.secretmanager.v1.SecretManagerService) in ServiceByID index")
		return
	}
}

func TestCreateModelProtobuf(t *testing.T) {
	requireProtoc(t)
	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "protobuf",
			ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			SpecificationSource: "google/cloud/secretmanager/v1",
		},
		Source: map[string]string{
			"googleapis-root": path.Join(testdataDir, "googleapis"),
		},
	}
	model, err := CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := model.State.ServiceByID[".google.cloud.secretmanager.v1.SecretManagerService"]
	if !ok {
		t.Errorf("missing service (.google.cloud.secretmanager.v1.SecretManagerService) in ServiceByID index")
		return
	}
}

func TestCreateModelOverrides(t *testing.T) {
	requireProtoc(t)
	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "protobuf",
			ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			SpecificationSource: "google/cloud/secretmanager/v1",
		},
		Source: map[string]string{
			"googleapis-root":      path.Join(testdataDir, "googleapis"),
			"name-override":        "Name Override",
			"title-override":       "Title Override",
			"description-override": "Description Override",
		},
	}
	model, err := CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	type TestCase struct {
		got  string
		want string
	}
	testCases := []TestCase{
		{model.Name, "Name Override"},
		{model.Title, "Title Override"},
		{model.Description, "Description Override"},
	}
	for _, c := range testCases {
		if c.got != c.want {
			t.Errorf("mimatched override got=%q, want=%q", c.got, c.want)
		}
	}
}

func TestCreateModelNone(t *testing.T) {
	requireProtoc(t)
	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "none",
			ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			SpecificationSource: "none",
		},
		Source: map[string]string{
			"googleapis-root":      path.Join(testdataDir, "googleapis"),
			"name-override":        "Name Override",
			"title-override":       "Title Override",
			"description-override": "Description Override",
		},
	}
	model, err := CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if model != nil {
		t.Errorf("expected `nil` model with source format == none")
	}
}

func checkMessage(t *testing.T, got *api.Message, want *api.Message) {
	t.Helper()
	// Checking Parent, Messages, Fields, and OneOfs requires special handling.
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Message{}, "Fields", "OneOfs", "Parent", "Messages")); diff != "" {
		t.Errorf("message attributes mismatch (-want +got):\n%s", diff)
	}
	less := func(a, b *api.Field) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want.Fields, got.Fields, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("field mismatch (-want, +got):\n%s", diff)
	}
	// Ignore parent because types are cyclic
	if diff := cmp.Diff(want.OneOfs, got.OneOfs, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("oneofs mismatch (-want, +got):\n%s", diff)
	}
}

func checkEnum(t *testing.T, got api.Enum, want api.Enum) {
	t.Helper()
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Enum{}, "Values", "UniqueNumberValues", "Parent")); diff != "" {
		t.Errorf("mismatched service attributes (-want, +got):\n%s", diff)
	}
	less := func(a, b *api.EnumValue) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want.Values, got.Values, cmpopts.SortSlices(less), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("method mismatch (-want, +got):\n%s", diff)
	}
}

func checkService(t *testing.T, got *api.Service, want *api.Service) {
	t.Helper()
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Service{}, "Methods")); diff != "" {
		t.Errorf("mismatched service attributes (-want, +got):\n%s", diff)
	}
	less := func(a, b *api.Method) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want.Methods, got.Methods, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("method mismatch (-want, +got):\n%s", diff)
	}
}

func checkMethod(t *testing.T, service *api.Service, name string, want *api.Method) {
	t.Helper()
	findMethod := func(name string) (*api.Method, bool) {
		for _, method := range service.Methods {
			if method.Name == name {
				return method, true
			}
		}
		return nil, false
	}
	got, ok := findMethod(name)
	if !ok {
		t.Errorf("missing method %s", name)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatched data for method %s (-want, +got):\n%s", name, diff)
	}
}
