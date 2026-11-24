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

package dart

import (
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	testdataDir, _ = filepath.Abs("../testdata")
)

func TestFromProtobuf(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: path.Join(testdataDir, "../../testdata/googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"api-keys-environment-variables": "GOOGLE_API_KEY,GEMINI_API_KEY",
			"issue-tracker-url":              "http://www.example.com/issues",
			"copyright-year":                 "2025",
			"not-for-publication":            "true",
			"version":                        "0.1.0",
			"sdk-version":                    "^3.9.0",
			"skip-format":                    "true",
			"package:google_cloud_rpc":       "^1.2.3",
			"package:http":                   "^4.5.6",
			"package:google_cloud_location":  "^7.8.9",
			"package:google_cloud_protobuf":  "^0.1.2",
			"proto:google.protobuf":          "package:google_cloud_protobuf/protobuf.dart",
			"proto:google.cloud.location":    "package:google_cloud_location/location.dart",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg.Codec); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		".gitattributes",
		"pubspec.yaml",
		"lib/secretmanager.dart",
		"lib/src/api.g.dart",
		"lib/src/version.dart",
		"lib/testing.dart",
		"LICENSE",
		"README.md",
		"skills/google_cloud_secretmanager_v1-tests/SKILL.md",
		"skills/google_cloud_secretmanager_v1-setup/SKILL.md",
	} {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if err != nil {
			t.Errorf("missing or cannot stat %s: %s", filename, err)
			continue
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}

	gitattributesContent, err := os.ReadFile(filepath.Join(outDir, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	wantGitattributes := `.gitattributes linguist-generated=true
LICENSE linguist-generated=true
README.md linguist-generated=true
pubspec.yaml linguist-generated=true
lib/secretmanager.dart linguist-generated=true
lib/src/api.g.dart linguist-generated=true
lib/src/version.dart linguist-generated=true
lib/testing.dart linguist-generated=true
skills/** linguist-generated=true
`
	if diff := cmp.Diff(wantGitattributes, string(gitattributesContent)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFromProtobuf_NoServices(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       "google/type/type.yaml",
		SpecificationSource: "google/type",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: path.Join(testdataDir, "../../testdata/googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"issue-tracker-url":             "http://www.example.com/issues",
			"copyright-year":                "2025",
			"not-for-publication":           "true",
			"version":                       "0.1.0",
			"sdk-version":                   "^3.9.0",
			"skip-format":                   "true",
			"package:google_cloud_protobuf": "^0.1.2",
			"proto:google.protobuf":         "package:google_cloud_protobuf/protobuf.dart",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg.Codec); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		".gitattributes",
		"pubspec.yaml",
		"lib/type.dart",
		"lib/src/api.g.dart",
		"lib/src/version.dart",
		"LICENSE",
		"README.md",
	} {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if err != nil {
			t.Errorf("missing or cannot stat %s: %s", filename, err)
			continue
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}

	gitattributesContent, err := os.ReadFile(filepath.Join(outDir, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	wantGitattributes := `.gitattributes linguist-generated=true
LICENSE linguist-generated=true
README.md linguist-generated=true
pubspec.yaml linguist-generated=true
lib/type.dart linguist-generated=true
lib/src/api.g.dart linguist-generated=true
lib/src/version.dart linguist-generated=true
`
	if diff := cmp.Diff(wantGitattributes, string(gitattributesContent)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerate_SDKVersion(t *testing.T) {
	type pubspec struct {
		Environment struct {
			SDK string `yaml:"sdk"`
		} `yaml:"environment"`
	}
	for _, test := range []struct {
		name       string
		sdkVersion string
		wantLine   string
	}{
		{
			name:       "caret",
			sdkVersion: "^3.10.0",
			wantLine:   "  sdk: ^3.10.0\n",
		},
		{
			name:       "range",
			sdkVersion: ">=3.9.0 <5.0.0",
			wantLine:   "  sdk: '>=3.9.0 <5.0.0'\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			options := maps.Clone(requiredConfig)
			options["skip-format"] = "true"
			options["sdk-version"] = test.sdkVersion
			if err := Generate(t.Context(), api.NewTestAPI(nil, nil, nil), outDir, options); err != nil {
				t.Fatal(err)
			}
			pubspecPath := filepath.Join(outDir, "pubspec.yaml")
			raw, err := os.ReadFile(pubspecPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), test.wantLine) {
				t.Errorf("pubspec.yaml missing line %q:\n%s", test.wantLine, raw)
			}
			got, err := yaml.Read[pubspec](pubspecPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.sdkVersion, got.Environment.SDK); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGeneratedFiles(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	annotate := newAnnotateModel(model)

	options := maps.Clone(requiredConfig)
	maps.Copy(options, map[string]string{"package:google_cloud_rpc": "^1.2.3", "package:http": "^4.5.6"})

	annotate.annotateModel(options)
	files := generatedFiles(model)
	if len(files) == 0 {
		t.Errorf("expected a non-empty list of template files from generatedFiles()")
	}

	// Validate that main.dart was replaced with {servicename}.dart.
	for _, fileInfo := range files {
		if filepath.Base(fileInfo.OutputPath) == "main.dart" {
			t.Errorf("expected the main.dart template to be generated as {servicename}.dart")
		}
	}
}

func TestTemplatesAvailable(t *testing.T) {
	var count = 0
	fs.WalkDir(dartTemplates, "templates", func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) != ".gotmpl" {
			return nil
		}
		if strings.Count(d.Name(), ".") == 1 {
			// skip partials
			return nil
		}
		count++
		return nil
	})

	if count == 0 {
		t.Errorf("no dart templates found")
	}
}

func TestValidateTemplates(t *testing.T) {
	if err := parsedTemplates.Validate(); err != nil {
		t.Fatal(err)
	}
}

func requireProtoc(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
}
