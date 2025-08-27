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
