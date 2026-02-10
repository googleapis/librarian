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
	"github.com/googleapis/librarian/internal/config"
)

const (
	secretManagerYamlRelative = "google/cloud/secretmanager/v1/secretmanager_v1.yaml"
	discoSourceFileRelative   = "discovery/compute.v1.json"
)

var (
	mainTestdataDir, _        = filepath.Abs("../../testdata")
	discoSourceFile           = path.Join(mainTestdataDir, discoSourceFileRelative)
	secretManagerYamlFullPath = path.Join(mainTestdataDir, "googleapis", secretManagerYamlRelative)
	openAPIFile               = path.Join(mainTestdataDir, "secretmanager_openapi_v1.json")
	protobufFile              = path.Join("testdata", "scalar.proto")
)

func TestCreateModelDisco(t *testing.T) {
	cfg := ModelConfig{
		SpecificationFormat: config.SpecDiscovery,
		ServiceConfig:       secretManagerYamlFullPath,
		SpecificationSource: discoSourceFile,
	}
	got, err := CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	wantName := "secretmanager"
	wantTitle := "Secret Manager API"
	wantDescription := "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security."
	wantPackageName := "google.cloud.secretmanager.v1"
	if got.Name != wantName {
		t.Errorf("want = %q; got = %q", wantName, got.Name)
	}
	if got.Title != wantTitle {
		t.Errorf("want = %q; got = %q", wantTitle, got.Title)
	}
	if diff := cmp.Diff(wantDescription, got.Description); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if got.PackageName != wantPackageName {
		t.Errorf("want = %q; got = %q", wantPackageName, got.PackageName)
	}
	// This is strange, but we want to verify the package name override from
	// the service config YAML applies to the message IDs too.
	wantMessage := ".google.cloud.secretmanager.v1.ZoneSetPolicyRequest"
	if _, ok := got.State.MessageByID[wantMessage]; !ok {
		t.Errorf("missing message %s in MessageByID index", wantMessage)
	}
}

func TestCreateModelOpenAPI(t *testing.T) {
	cfg := ModelConfig{
		SpecificationFormat: config.SpecOpenAPI,
		ServiceConfig:       secretManagerYamlFullPath,
		SpecificationSource: openAPIFile,
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
	cfg := ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       secretManagerYamlRelative,
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: map[string]string{
			"googleapis-root": path.Join(mainTestdataDir, "googleapis"),
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
	cfg := ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       secretManagerYamlRelative,
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: map[string]string{
			"googleapis-root":      path.Join(mainTestdataDir, "googleapis"),
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
	cfg := ModelConfig{
		SpecificationFormat: "none",
		ServiceConfig:       secretManagerYamlRelative,
		SpecificationSource: "none",
		Source: map[string]string{
			"googleapis-root":      path.Join(mainTestdataDir, "googleapis"),
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

func TestCreateModelUnknown(t *testing.T) {
	cfg := ModelConfig{
		SpecificationFormat: "--unknown--",
		ServiceConfig:       secretManagerYamlRelative,
		SpecificationSource: "none",
		Source: map[string]string{
			"googleapis-root":      path.Join(mainTestdataDir, "googleapis"),
			"name-override":        "Name Override",
			"title-override":       "Title Override",
			"description-override": "Description Override",
		},
	}
	if got, err := CreateModel(cfg); err == nil {
		t.Errorf("expected error with unknown specification format, got=%v", got)
	}
}

func TestCreateModelBadParse(t *testing.T) {
	cfg := ModelConfig{
		SpecificationFormat: config.SpecOpenAPI,
		ServiceConfig:       secretManagerYamlRelative,
		// Note the mismatch between the format and the file contents.
		SpecificationSource: discoSourceFile,
		Source: map[string]string{
			"googleapis-root":      path.Join(mainTestdataDir, "googleapis"),
			"name-override":        "Name Override",
			"title-override":       "Title Override",
			"description-override": "Description Override",
		},
	}
	if got, err := CreateModel(cfg); err == nil {
		t.Errorf("expected error with bad specification, got=%v", got)
	}
}
