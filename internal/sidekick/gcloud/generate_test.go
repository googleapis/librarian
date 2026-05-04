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

package gcloud

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFromProtobuf(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: filepath.Join(testdataDir, "googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"copyright-year": "2026",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(model, outDir); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(outDir, "README.md")
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("missing %s: %s", filename, err)
		}
		t.Fatal(err)
	}
}

func TestParallelstore(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		ServiceConfig:       "google/cloud/parallelstore/v1/service.yaml",
		SpecificationSource: "google/cloud/parallelstore/v1",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: filepath.Join(testdataDir, "googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"copyright-year": "2026",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(model, outDir); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(outDir, "main.go")
	gotMain, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatal(err)
	}
	wantMain, err := os.ReadFile(filepath.Join("testdata", "parallelstore", "main.go.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(wantMain), string(gotMain)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	readmeFile := filepath.Join(outDir, "README.md")
	gotReadme, err := os.ReadFile(readmeFile)
	if err != nil {
		t.Fatal(err)
	}
	wantReadme, err := os.ReadFile(filepath.Join("testdata", "parallelstore", "README.md.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(wantReadme), string(gotReadme)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRenderMain(t *testing.T) {
	for _, test := range []struct {
		name  string
		model CLIModel
		wants []string
	}{
		{
			name: "empty",
			model: CLIModel{
				Groups: []Group{{
					Name:  "gcloud",
					Usage: "Google Cloud CLI",
				}},
			},
			wants: []string{
				`Name:  "gcloud"`,
				`Usage: "Google Cloud CLI"`,
			},
		},
		{
			name: "subgroup with command",
			model: CLIModel{
				Groups: []Group{{
					Name:  "gcloud",
					Usage: "Google Cloud CLI",
					Subgroups: []Subgroup{{
						Name:  "compute",
						Usage: "Manage compute resources",
						Commands: []Command{{
							Name:  "list",
							Usage: "list compute",
						}},
					}},
				}},
			},
			wants: []string{
				`Name:  "compute"`,
				`Name:  "list"`,
				`fmt.Println("Executing list...")`,
			},
		},
		{
			name: "top-level command",
			model: CLIModel{
				Groups: []Group{{
					Name:  "gcloud",
					Usage: "Google Cloud CLI",
					Commands: []Command{{
						Name:  "version",
						Usage: "show version",
					}},
				}},
			},
			wants: []string{
				`Name:  "version"`,
				`fmt.Println("Executing version...")`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := renderMain(test.model)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(got, "package main") {
				t.Errorf("rendered output does not start with %q\n%s", "package main", got)
			}
			for _, want := range test.wants {
				if !strings.Contains(got, want) {
					t.Errorf("rendered output missing %q\n%s", want, got)
				}
			}
		})
	}
}

func TestWriteMain(t *testing.T) {
	const contents = "package main\n"
	for _, test := range []struct {
		name   string
		outdir func(t *testing.T) string
	}{
		{
			name: "existing dir",
			outdir: func(t *testing.T) string {
				return t.TempDir()
			},
		},
		{
			name: "nested dir creation",
			outdir: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nested", "deep")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := test.outdir(t)
			if err := writeMain(outdir, contents); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(outdir, "main.go"))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(contents, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderReadme(t *testing.T) {
	for _, test := range []struct {
		name  string
		model *api.API
		wants []string
	}{
		{
			name:  "title only",
			model: &api.API{Title: "Parallelstore API"},
			wants: []string{
				"# Google Cloud CLI (gcloud)",
				"Parallelstore API",
			},
		},
		{
			name:  "title and description",
			model: &api.API{Title: "Parallelstore API", Description: "Manages parallelstore instances."},
			wants: []string{
				"Parallelstore API",
				"Manages parallelstore instances.",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()
			if err := renderReadme(outdir, test.model); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(outdir, "README.md"))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.wants {
				if !strings.Contains(string(got), want) {
					t.Errorf("README missing %q\n%s", want, got)
				}
			}
		})
	}
}

func TestPathFlagsFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     []Flag
	}{
		{"nil", nil, nil},
		{
			"single",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				Segments,
			[]Flag{{Name: "project", Kind: "String", Required: true, Usage: "The project."}},
		},
		{
			"multi",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
				WithLiteral("instances").WithVariable(api.NewPathVariable("instance")).
				Segments,
			[]Flag{
				{Name: "project", Kind: "String", Required: true, Usage: "The project."},
				{Name: "location", Kind: "String", Required: true, Usage: "The location."},
				{Name: "instance", Kind: "String", Required: true, Usage: "The instance."},
			},
		},
		{
			"dedup",
			(&api.PathTemplate{}).
				WithLiteral("users").WithVariable(api.NewPathVariable("user")).
				WithLiteral("users").WithVariable(api.NewPathVariable("user")).
				Segments,
			[]Flag{{Name: "user", Kind: "String", Required: true, Usage: "The user."}},
		},
		{
			"trailing-literal",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				WithLiteral("config").
				Segments,
			[]Flag{{Name: "project", Kind: "String", Required: true, Usage: "The project."}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pathFlagsFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommandHasPath(t *testing.T) {
	for _, test := range []struct {
		name string
		cmd  Command
		want bool
	}{
		{"empty", Command{}, false},
		{"with-path", Command{PathFormat: "projects/%s"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.cmd.HasPath()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommandPathFormatArgs(t *testing.T) {
	for _, test := range []struct {
		name string
		cmd  Command
		want string
	}{
		{"empty", Command{}, ""},
		{"single", Command{Args: []string{"project"}}, `cmd.String("project")`},
		{
			"multi",
			Command{Args: []string{"project", "location", "instance"}},
			`cmd.String("project"), cmd.String("location"), cmd.String("instance")`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.cmd.PathFormatArgs()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPathFormatFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{"nil", nil, ""},
		{
			"no-variable",
			(&api.PathTemplate{}).WithLiteral("projects").Segments,
			"",
		},
		{
			"single",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				Segments,
			"projects/%s",
		},
		{
			"multi",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
				WithLiteral("instances").WithVariable(api.NewPathVariable("instance")).
				Segments,
			"projects/%s/locations/%s/instances/%s",
		},
		{
			"trailing-literal",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				WithLiteral("config").
				Segments,
			"projects/%s/config",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pathFormatFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPathArgsFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     []string
	}{
		{"nil", nil, nil},
		{
			"no-variable",
			(&api.PathTemplate{}).WithLiteral("projects").Segments,
			nil,
		},
		{
			"multi",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
				WithLiteral("instances").WithVariable(api.NewPathVariable("instance")).
				Segments,
			[]string{"project", "location", "instance"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pathArgsFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
