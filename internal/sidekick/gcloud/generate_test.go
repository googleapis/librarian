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
	stdflag "flag"
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

// update refreshes testdata goldens in place when set. Run as
// `go test ./internal/sidekick/gcloud -update -run TestGenerate`.
//
// The "flag" package is aliased because the surrounding gcloud package
// already exports an unrelated helper named flag().
var update = stdflag.Bool("update", false, "update golden files")

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
	if err := Generate([]*api.API{model}, outDir, ""); err != nil {
		t.Fatal(err)
	}
	mainFile := filepath.Join(outDir, "cmd", "gcloud", "main.go")
	if _, err := os.Stat(mainFile); err != nil {
		t.Fatalf("missing %s: %s", mainFile, err)
	}
}

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	makeModel := func(serviceConfig, source string) *api.API {
		t.Helper()
		cfg := &parser.ModelConfig{
			SpecificationFormat: config.SpecProtobuf,
			ServiceConfig:       serviceConfig,
			SpecificationSource: source,
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
		m, err := parser.CreateModel(cfg)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	parallelstoreModel := makeModel("google/cloud/parallelstore/v1/service.yaml", "google/cloud/parallelstore/v1")
	publiccaModel := makeModel("google/cloud/security/publicca/v1/publicca_v1.yaml", "google/cloud/security/publicca/v1")

	if err := Generate([]*api.API{parallelstoreModel, publiccaModel}, outDir, ""); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		filepath.Join("cmd", "gcloud", "main.go"),
		filepath.Join("internal", "generated", "parallelstore", "commands.go"),
		filepath.Join("internal", "generated", "publicca", "commands.go"),
	} {
		t.Run(rel, func(t *testing.T) {
			got, err := os.ReadFile(filepath.Join(outDir, rel))
			if err != nil {
				t.Fatal(err)
			}
			goldenPath := filepath.Join("testdata", rel+".golden")
			if *update {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath, got, 0o666); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s\n\nHint: run 'go test ./internal/sidekick/gcloud -update -run TestGenerate' to refresh goldens.", diff)
			}
		})
	}
}

func TestRenderSurface(t *testing.T) {
	for _, test := range []struct {
		name  string
		model SurfaceModel
		wants []string
	}{
		{
			name: "empty",
			model: SurfaceModel{
				PackageName: "parallelstore",
				Group: Group{
					Name:  "parallelstore",
					Usage: "manage Parallelstore API resources",
				},
			},
			wants: []string{
				"package parallelstore",
				`"parallelstore"`,
				`"manage Parallelstore API resources"`,
				"func Command() *cli.Command",
			},
		},
		{
			name: "subgroup with command",
			model: SurfaceModel{
				PackageName: "parallelstore",
				Group: Group{
					Name:  "parallelstore",
					Usage: "manage Parallelstore API resources",
					Subgroups: []Subgroup{{
						Name:  "instances",
						Usage: "Manage instances resources",
						Commands: []Command{{
							Name:  "list",
							Usage: "list instances",
						}},
					}},
				},
			},
			wants: []string{
				`Name:  "instances"`,
				`Name:  "list"`,
				`fmt.Println("Executing list...")`,
			},
		},
		{
			name: "command with project in path emits guard",
			model: SurfaceModel{
				PackageName: "parallelstore",
				Group: Group{
					Name: "parallelstore",
					Subgroups: []Subgroup{{
						Name: "instances",
						Commands: []Command{{
							Name:       "list",
							Args:       []string{"project", "location"},
							PathFormat: "projects/%s/locations/%s",
							PathLabel:  "parent",
						}},
					}},
				},
			},
			wants: []string{
				`if cmd.String("project") == ""`,
				`return fmt.Errorf("--project is required")`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := renderSurface(test.model)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.wants {
				if !strings.Contains(got, want) {
					t.Errorf("rendered output missing %q\n%s", want, got)
				}
			}
		})
	}
}

func TestRenderMain(t *testing.T) {
	for _, test := range []struct {
		name  string
		model CLIModel
		wants []string
	}{
		{
			name: "no surfaces",
			model: CLIModel{
				ModulePath: "cloud.google.com/go/gcloud",
			},
			wants: []string{
				"package main",
				`"gcloud"`,
				`"Google Cloud CLI"`,
				`&cli.StringFlag{Name: "project", Usage: "The Google Cloud project ID."}`,
			},
		},
		{
			name: "two surfaces",
			model: CLIModel{
				ModulePath: "cloud.google.com/go/gcloud",
				Surfaces: []SurfaceRef{
					{PackageName: "parallelstore"},
					{PackageName: "publicca"},
				},
			},
			wants: []string{
				`"cloud.google.com/go/gcloud/internal/generated/parallelstore"`,
				`"cloud.google.com/go/gcloud/internal/generated/publicca"`,
				"parallelstore.Command()",
				"publicca.Command()",
				`&cli.StringFlag{Name: "project", Usage: "The Google Cloud project ID."}`,
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

func TestWriteSurface(t *testing.T) {
	outdir := t.TempDir()
	model := SurfaceModel{
		PackageName: "parallelstore",
		Group: Group{
			Name:  "parallelstore",
			Usage: "manage Parallelstore API resources",
		},
	}
	if err := writeSurface(outdir, model); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(outdir, "internal", "generated", "parallelstore", "commands.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "package parallelstore") {
		t.Errorf("missing package declaration in:\n%s", got)
	}
}

func TestWriteMain(t *testing.T) {
	outdir := t.TempDir()
	if err := writeMain(outdir, CLIModel{ModulePath: "cloud.google.com/go/gcloud"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(outdir, "cmd", "gcloud", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "package main") {
		t.Errorf("missing package main in:\n%s", got)
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
			"project-only",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				Segments,
			nil,
		},
		{
			"multi",
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
				WithLiteral("instances").WithVariable(api.NewPathVariable("instance")).
				Segments,
			[]Flag{
				{Name: "location", Kind: "String", Required: true, Usage: "The location."},
				{Name: "instance", Kind: "String", Required: true, Usage: "The instance."},
			},
		},
		{
			"no-project",
			(&api.PathTemplate{}).
				WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
				WithLiteral("instances").WithVariable(api.NewPathVariable("instance")).
				Segments,
			[]Flag{
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
			nil,
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

func TestCommandRequiresProject(t *testing.T) {
	for _, test := range []struct {
		name string
		cmd  Command
		want bool
	}{
		{"empty", Command{}, false},
		{"no-project", Command{Args: []string{"location", "instance"}}, false},
		{"project-only", Command{Args: []string{"project"}}, true},
		{"project-and-others", Command{Args: []string{"project", "location"}}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.cmd.RequiresProject()
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

func TestGoClientPackage(t *testing.T) {
	for _, test := range []struct {
		name     string
		protoPkg string
		override string
		want     *goClientInfo
	}{
		{
			name:     "parallelstore",
			protoPkg: "google.cloud.parallelstore.v1",
			want: &goClientInfo{
				Alias:      "parallelstore",
				ClientPath: "cloud.google.com/go/parallelstore/apiv1",
				PbPath:     "cloud.google.com/go/parallelstore/apiv1/parallelstorepb",
			},
		},
		{
			name:     "secretmanager",
			protoPkg: "google.cloud.secretmanager.v1",
			want: &goClientInfo{
				Alias:      "secretmanager",
				ClientPath: "cloud.google.com/go/secretmanager/apiv1",
				PbPath:     "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb",
			},
		},
		{name: "empty", protoPkg: ""},
		{name: "three-segments", protoPkg: "google.cloud.parallelstore"},
		{name: "beta-version", protoPkg: "google.cloud.parallelstore.v1beta1"},
		{name: "not-google-cloud", protoPkg: "google.api.X.v1"},
		{
			name:     "override-major-version",
			protoPkg: "google.cloud.recaptchaenterprise.v1",
			override: "cloud.google.com/go/recaptchaenterprise/v2/apiv1",
			want: &goClientInfo{
				Alias:      "recaptchaenterprise",
				ClientPath: "cloud.google.com/go/recaptchaenterprise/v2/apiv1",
				PbPath:     "cloud.google.com/go/recaptchaenterprise/v2/apiv1/recaptchaenterprisepb",
			},
		},
		{
			name:     "override-renamed-module",
			protoPkg: "google.cloud.tasks.v2",
			override: "cloud.google.com/go/cloudtasks/apiv2",
			want: &goClientInfo{
				Alias:      "cloudtasks",
				ClientPath: "cloud.google.com/go/cloudtasks/apiv2",
				PbPath:     "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb",
			},
		},
		{
			name:     "override-translate-apiv3",
			protoPkg: "google.cloud.translation.v3",
			override: "cloud.google.com/go/translate/apiv3",
			want: &goClientInfo{
				Alias:      "translate",
				ClientPath: "cloud.google.com/go/translate/apiv3",
				PbPath:     "cloud.google.com/go/translate/apiv3/translatepb",
			},
		},
		{
			name:     "override-ignores-proto-pkg",
			protoPkg: "ignored.totally",
			override: "cloud.google.com/go/translate/apiv3",
			want: &goClientInfo{
				Alias:      "translate",
				ClientPath: "cloud.google.com/go/translate/apiv3",
				PbPath:     "cloud.google.com/go/translate/apiv3/translatepb",
			},
		},
		{
			name:     "override-malformed-no-api",
			override: "cloud.google.com/go/parallelstore",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := goClientPackage(test.protoPkg, test.override)
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
