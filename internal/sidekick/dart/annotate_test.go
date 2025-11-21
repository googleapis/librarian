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
	"maps"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/sample"
)

var (
	requiredConfig = map[string]string{
		"api-keys-environment-variables": "GOOGLE_API_KEY,GEMINI_API_KEY",
		"issue-tracker-url":              "http://www.example.com/issues",
		"package:google_cloud_rpc":       "^1.2.3",
		"package:http":                   "^4.5.6",
		"package:google_cloud_protobuf":  "^7.8.9",
	}
)

func TestAnnotateModel(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "test"

	options := maps.Clone(requiredConfig)
	maps.Copy(options, map[string]string{"package:google_cloud_rpc": "^1.2.3"})

	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(options)
	if err != nil {
		t.Fatal(err)
	}

	codec := model.Codec.(*modelAnnotations)

	if diff := cmp.Diff("google_cloud_test", codec.PackageName); diff != "" {
		t.Errorf("mismatch in Codec.PackageName (-want, +got)\n:%s", diff)
	}
	if diff := cmp.Diff("test", codec.MainFileName); diff != "" {
		t.Errorf("mismatch in Codec.MainFileName (-want, +got)\n:%s", diff)
	}
}

func TestAnnotateModel_Options(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})

	var tests = []struct {
		options map[string]string
		verify  func(*testing.T, *annotateModel)
	}{
		{
			map[string]string{"package-name-override": "google-cloud-type"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("google-cloud-type", codec.PackageName); diff != "" {
					t.Errorf("mismatch in Codec.PackageName (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{"dev-dependencies": "test,mockito"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff([]string{"mockito", "test"}, codec.DevDependencies); diff != "" {
					t.Errorf("mismatch in Codec.PackageName (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{
				"dependencies":             "google_cloud_foo, google_cloud_bar",
				"package:google_cloud_bar": "^1.2.3",
				"package:google_cloud_foo": "^4.5.6"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if !slices.Contains(codec.PackageDependencies, packageDependency{Name: "google_cloud_foo", Constraint: "^4.5.6"}) {
					t.Errorf("missing 'google_cloud_foo' in Codec.PackageDependencies, got %v", codec.PackageDependencies)
				}
				if !slices.Contains(codec.PackageDependencies, packageDependency{Name: "google_cloud_bar", Constraint: "^1.2.3"}) {
					t.Errorf("missing 'google_cloud_bar' in Codec.PackageDependencies, got %v", codec.PackageDependencies)
				}
			},
		},
		{
			map[string]string{"extra-exports": "export 'package:google_cloud_gax/gax.dart' show Any; export 'package:google_cloud_gax/gax.dart' show Status;"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff([]string{
					"export 'package:google_cloud_gax/gax.dart' show Any",
					"export 'package:google_cloud_gax/gax.dart' show Status"}, codec.Exports); diff != "" {
					t.Errorf("mismatch in Codec.Exports (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{"version": "1.2.3"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("1.2.3", codec.PackageVersion); diff != "" {
					t.Errorf("mismatch in Codec.PackageVersion (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{"part-file": "src/test.p.dart"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("src/test.p.dart", codec.PartFileReference); diff != "" {
					t.Errorf("mismatch in Codec.PartFileReference (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{"readme-after-title-text": "> [!TIP] Still beta!"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("> [!TIP] Still beta!", codec.ReadMeAfterTitleText); diff != "" {
					t.Errorf("mismatch in Codec.ReadMeAfterTitleText (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{"readme-quickstart-text": "## Getting Started\n..."},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("## Getting Started\n...", codec.ReadMeQuickstartText); diff != "" {
					t.Errorf("mismatch in Codec.ReadMeQuickstartText (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{"repository-url": "http://example.com/repo"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("http://example.com/repo", codec.RepositoryURL); diff != "" {
					t.Errorf("mismatch in Codec.RepositoryURL (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{"issue-tracker-url": "http://example.com/issues"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("http://example.com/issues", codec.IssueTrackerURL); diff != "" {
					t.Errorf("mismatch in Codec.IssueTrackerURL (-want, +got)\n:%s", diff)
				}
			},
		},
		{
			map[string]string{"google_cloud_rpc": "^1.2.3", "package:http": "1.2.0"},
			func(t *testing.T, am *annotateModel) {
				if diff := cmp.Diff(map[string]string{
					"google_cloud_rpc":      "^1.2.3",
					"google_cloud_protobuf": "^7.8.9",
					"http":                  "1.2.0"},
					am.dependencyConstraints); diff != "" {
					t.Errorf("mismatch in annotateModel.dependencyConstraints (-want, +got)\n:%s", diff)
				}
			},
		},
	}

	for _, test := range tests {
		annotate := newAnnotateModel(model)
		options := maps.Clone(requiredConfig)
		maps.Copy(options, test.options)
		err := annotate.annotateModel(maps.Clone(options))
		if err != nil {
			t.Fatal(err)
		}
		test.verify(t, annotate)
	}
}

func TestAnnotateModel_Options_MissingRequired(t *testing.T) {
	method := sample.MethodListSecretVersions()
	service := &api.Service{
		Name:          sample.ServiceName,
		Documentation: sample.APIDescription,
		DefaultHost:   sample.DefaultHost,
		Methods:       []*api.Method{method},
		Package:       sample.Package,
	}
	model := api.NewTestAPI(
		[]*api.Message{sample.ListSecretVersionsRequest(), sample.ListSecretVersionsResponse(),
			sample.Secret(), sample.SecretVersion(), sample.Replication(), sample.Automatic(),
			sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{service},
	)

	var tests = []string{
		"api-keys-environment-variables",
		"issue-tracker-url",
	}

	for _, test := range tests {
		annotate := newAnnotateModel(model)
		options := maps.Clone(requiredConfig)
		delete(options, test)

		err := annotate.annotateModel(options)
		if err == nil {
			t.Fatalf("expected error when missing %q", test)
		}
	}
}

func TestAnnotateMethod(t *testing.T) {
	method := sample.MethodListSecretVersions()
	service := &api.Service{
		Name:          sample.ServiceName,
		Documentation: sample.APIDescription,
		DefaultHost:   sample.DefaultHost,
		Methods:       []*api.Method{method},
		Package:       sample.Package,
	}
	model := api.NewTestAPI(
		[]*api.Message{sample.ListSecretVersionsRequest(), sample.ListSecretVersionsResponse(),
			sample.Secret(), sample.SecretVersion(), sample.Replication(), sample.Automatic(),
			sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{service},
	)
	api.Validate(model)
	annotate := newAnnotateModel(model)
	err := annotate.annotateModel(requiredConfig)
	if err != nil {
		t.Fatal(err)
	}

	annotate.annotateMethod(method)
	codec := method.Codec.(*methodAnnotation)

	got := codec.Name
	want := "listSecretVersions"
	if got != want {
		t.Errorf("mismatched name, got=%q, want=%q", got, want)
	}

	got = codec.RequestType
	want = "ListSecretVersionRequest"
	if got != want {
		t.Errorf("mismatched type, got=%q, want=%q", got, want)
	}

	got = codec.ResponseType
	want = "ListSecretVersionsResponse"
	if got != want {
		t.Errorf("mismatched type, got=%q, want=%q", got, want)
	}
}

func TestCalculatePubPackages(t *testing.T) {
	for _, test := range []struct {
		imports map[string]bool
		want    map[string]bool
	}{
		{imports: map[string]bool{"dart:typed_data": true},
			want: map[string]bool{}},
		{imports: map[string]bool{"dart:typed_data as typed_data": true},
			want: map[string]bool{}},
		{imports: map[string]bool{"package:http/http.dart": true},
			want: map[string]bool{"http": true}},
		{imports: map[string]bool{"package:http/http.dart as http": true},
			want: map[string]bool{"http": true}},
		{imports: map[string]bool{"package:google_cloud_protobuf/src/encoding.dart": true},
			want: map[string]bool{"google_cloud_protobuf": true}},
		{imports: map[string]bool{"package:google_cloud_protobuf/src/encoding.dart as encoding": true},
			want: map[string]bool{"google_cloud_protobuf": true}},
		{imports: map[string]bool{"package:http/http.dart": true, "package:http/http.dart as http": true},
			want: map[string]bool{"http": true}},
		{imports: map[string]bool{
			"package:google_cloud_protobuf/src/encoding.dart": true,
			"package:http/http.dart":                          true,
			"dart:typed_data":                                 true},
			want: map[string]bool{"google_cloud_protobuf": true, "http": true}},
	} { // package:http/http.dart as http
		got := calculatePubPackages(test.imports)

		if !maps.Equal(got, test.want) {
			t.Errorf("calculatePubPackages(%v) = %v, want %v", test.imports, got, test.want)
		}
	}
}

func TestCalculateDependencies(t *testing.T) {
	for _, test := range []struct {
		testName    string
		packages    map[string]bool
		constraints map[string]string
		packageName string
		want        []packageDependency
		wantErr     bool
	}{
		{
			testName:    "empty",
			packages:    map[string]bool{},
			constraints: map[string]string{},
			packageName: "google_cloud_bar",
			want:        []packageDependency{},
		},
		{
			testName:    "self dependency",
			packages:    map[string]bool{"google_cloud_bar": true},
			constraints: map[string]string{},
			packageName: "google_cloud_bar",
			want:        []packageDependency{},
		},
		{
			testName:    "separate dependency",
			packages:    map[string]bool{"google_cloud_foo": true},
			constraints: map[string]string{"google_cloud_foo": "^1.2.3"},
			packageName: "google_cloud_bar",
			want:        []packageDependency{{Name: "google_cloud_foo", Constraint: "^1.2.3"}},
		},
		{
			testName:    "missing constraint",
			packages:    map[string]bool{"google_cloud_foo": true},
			constraints: map[string]string{},
			packageName: "google_cloud_bar",
			wantErr:     true,
		},
		{
			testName:    "multiple dependencies",
			packages:    map[string]bool{"google_cloud_bar": true, "google_cloud_baz": true, "google_cloud_foo": true},
			constraints: map[string]string{"google_cloud_baz": "^1.2.3", "google_cloud_foo": "^4.5.6"},
			packageName: "google_cloud_bar",
			want: []packageDependency{
				{Name: "google_cloud_baz", Constraint: "^1.2.3"},
				{Name: "google_cloud_foo", Constraint: "^4.5.6"}},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			got, err := calculateDependencies(test.packages, test.constraints, test.packageName)
			if (err != nil) != test.wantErr {
				t.Errorf("calculateDependencies(%v, %v, %v) error = %v, want error presence = %t",
					test.packages, test.constraints, test.packageName, err, test.wantErr)
			}

			if err != nil {
				return
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("calculateDependencies(%v, %v, %v) = %v, want %v",
					test.packages, test.constraints, test.packageName, got, test.want)
			}
		})
	}
}

func TestCalculateImports(t *testing.T) {
	for _, test := range []struct {
		name        string
		imports     []string
		packageName string
		fileName    string
		want        []string
	}{
		{
			name:        "dart: import",
			imports:     []string{"dart:typed_data"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want:        []string{"import 'dart:typed_data';"},
		},
		{
			name:        "dart: import with prefix",
			imports:     []string{"dart:typed_data as td"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want:        []string{"import 'dart:typed_data' as td;"},
		},
		{
			name:        "package: import",
			imports:     []string{"package:http/http.dart"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want:        []string{"import 'package:http/http.dart';"},
		},
		{
			name:        "package: import with prefix",
			imports:     []string{"package:http/http.dart as http"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want:        []string{"import 'package:http/http.dart' as http;"},
		},
		{
			name:        "dart: and package: imports",
			imports:     []string{"dart:typed_data", "package:http/http.dart"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want: []string{
				"import 'dart:typed_data';",
				"",
				"import 'package:http/http.dart';",
			},
		},
		{
			name:        "same file import",
			imports:     []string{"package:google_cloud_bar/bar.dart"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want:        nil,
		},
		{
			name:        "same file import with prefix",
			imports:     []string{"package:google_cloud_bar/bar.dart as bar"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want:        nil,
		},
		{
			name:        "same package import",
			imports:     []string{"package:google_cloud_bar/baz.dart"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want:        []string{"import 'baz.dart';"},
		},
		{
			name:        "same package import with prefix",
			imports:     []string{"package:google_cloud_bar/baz.dart as baz"},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want:        []string{"import 'baz.dart' as baz;"},
		},
		{
			name: "many imports", imports: []string{
				"package:google_cloud_foo/foo.dart",
				"package:google_cloud_bar/bar.dart as bar",
				"package:google_cloud_bar/src/foo.dart as foo",
				"package:google_cloud_bar/baz.dart",
				"dart:core",
				"dart:io as io",
			},
			packageName: "google_cloud_bar",
			fileName:    "bar.dart",
			want: []string{
				"import 'dart:core';",
				"import 'dart:io' as io;",
				"",
				"import 'package:google_cloud_foo/foo.dart';",
				"",
				"import 'baz.dart';",
				"import 'src/foo.dart' as foo;",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			deps := map[string]bool{}
			for _, imp := range test.imports {
				deps[imp] = true
			}
			got := calculateImports(deps, test.packageName, test.fileName)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch in calculateImports (-want, +got)\n:%s", diff)
			}
		})
	}
}

func TestAnnotateMessageToString(t *testing.T) {
	model := api.NewTestAPI(
		[]*api.Message{sample.Secret(), sample.SecretVersion(), sample.Replication(),
			sample.Automatic(), sample.CustomerManagedEncryption()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{},
	)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	for _, test := range []struct {
		message  *api.Message
		expected int
	}{
		// Expect the number of fields less the number of message fields.
		{message: sample.Secret(), expected: 1},
		{message: sample.SecretVersion(), expected: 2},
		{message: sample.Replication(), expected: 0},
		{message: sample.Automatic(), expected: 0},
	} {
		t.Run(test.message.Name, func(t *testing.T) {
			annotate.annotateMessage(test.message)

			codec := test.message.Codec.(*messageAnnotation)
			actual := codec.ToStringLines

			if len(actual) != test.expected {
				t.Errorf("Expected list of length %d, got %d", test.expected, len(actual))
			}
		})
	}
}

func TestBuildQueryLines(t *testing.T) {
	for _, test := range []struct {
		field *api.Field
		want  []string
	}{
		// primitives
		{
			&api.Field{Name: "bool", JSONName: "bool", Typez: api.BOOL_TYPE},
			[]string{"if (result.bool$.isNotDefault) 'bool': '${result.bool$}'"},
		}, {
			&api.Field{Name: "int32", JSONName: "int32", Typez: api.INT32_TYPE},
			[]string{"if (result.int32.isNotDefault) 'int32': '${result.int32}'"},
		}, {
			&api.Field{Name: "fixed32", JSONName: "fixed32", Typez: api.FIXED32_TYPE},
			[]string{"if (result.fixed32.isNotDefault) 'fixed32': '${result.fixed32}'"},
		}, {
			&api.Field{Name: "sfixed32", JSONName: "sfixed32", Typez: api.SFIXED32_TYPE},
			[]string{"if (result.sfixed32.isNotDefault) 'sfixed32': '${result.sfixed32}'"},
		}, {
			&api.Field{Name: "int64", JSONName: "int64", Typez: api.INT64_TYPE},
			[]string{"if (result.int64.isNotDefault) 'int64': '${result.int64}'"},
		}, {
			&api.Field{Name: "fixed64", JSONName: "fixed64", Typez: api.FIXED64_TYPE},
			[]string{"if (result.fixed64.isNotDefault) 'fixed64': '${result.fixed64}'"},
		}, {
			&api.Field{Name: "sfixed64", JSONName: "sfixed64", Typez: api.SFIXED64_TYPE},
			[]string{"if (result.sfixed64.isNotDefault) 'sfixed64': '${result.sfixed64}'"},
		}, {
			&api.Field{Name: "double", JSONName: "double", Typez: api.DOUBLE_TYPE},
			[]string{"if (result.double$.isNotDefault) 'double': '${result.double$}'"},
		}, {
			&api.Field{Name: "string", JSONName: "string", Typez: api.STRING_TYPE},
			[]string{"if (result.string.isNotDefault) 'string': result.string"},
		},

		// optional primitives
		{
			&api.Field{Name: "bool_opt", JSONName: "bool", Typez: api.BOOL_TYPE, Optional: true},
			[]string{"if (result.boolOpt != null) 'bool': '${result.boolOpt}'"},
		}, {
			&api.Field{Name: "int32_opt", JSONName: "int32", Typez: api.INT32_TYPE, Optional: true},
			[]string{"if (result.int32Opt != null) 'int32': '${result.int32Opt}'"},
		}, {
			&api.Field{Name: "fixed32_opt", JSONName: "fixed32", Typez: api.FIXED32_TYPE, Optional: true},
			[]string{"if (result.fixed32Opt != null) 'fixed32': '${result.fixed32Opt}'"},
		}, {
			&api.Field{Name: "sfixed32_opt", JSONName: "sfixed32", Typez: api.SFIXED32_TYPE, Optional: true},
			[]string{"if (result.sfixed32Opt != null) 'sfixed32': '${result.sfixed32Opt}'"},
		}, {
			&api.Field{Name: "int64_opt", JSONName: "int64", Typez: api.INT64_TYPE, Optional: true},
			[]string{"if (result.int64Opt != null) 'int64': '${result.int64Opt}'"},
		}, {
			&api.Field{Name: "fixed64_opt", JSONName: "fixed64", Typez: api.FIXED64_TYPE, Optional: true},
			[]string{"if (result.fixed64Opt != null) 'fixed64': '${result.fixed64Opt}'"},
		}, {
			&api.Field{Name: "sfixed64_opt", JSONName: "sfixed64", Typez: api.SFIXED64_TYPE, Optional: true},
			[]string{"if (result.sfixed64Opt != null) 'sfixed64': '${result.sfixed64Opt}'"},
		}, {
			&api.Field{Name: "double_opt", JSONName: "double", Typez: api.DOUBLE_TYPE, Optional: true},
			[]string{"if (result.doubleOpt != null) 'double': '${result.doubleOpt}'"},
		}, {
			&api.Field{Name: "string_opt", JSONName: "string", Typez: api.STRING_TYPE, Optional: true},
			[]string{"if (result.stringOpt != null) 'string': result.stringOpt!"},
		},

		// one ofs
		{
			&api.Field{Name: "bool", JSONName: "bool", Typez: api.BOOL_TYPE, IsOneOf: true},
			[]string{"if (result.bool$ != null) 'bool': '${result.bool$}'"},
		},

		// repeated primitives
		{
			&api.Field{Name: "boolList", JSONName: "boolList", Typez: api.BOOL_TYPE, Repeated: true},
			[]string{"if (result.boolList.isNotDefault) 'boolList': result.boolList!.map((e) => '$e')"},
		}, {
			&api.Field{Name: "int32List", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true},
			[]string{"if (result.int32List.isNotDefault) 'int32List': result.int32List!.map((e) => '$e')"},
		}, {
			&api.Field{Name: "int64List", JSONName: "int64List", Typez: api.INT64_TYPE, Repeated: true},
			[]string{"if (result.int64List.isNotDefault) 'int64List': result.int64List!.map((e) => '$e')"},
		}, {
			&api.Field{Name: "doubleList", JSONName: "doubleList", Typez: api.DOUBLE_TYPE, Repeated: true},
			[]string{"if (result.doubleList.isNotDefault) 'doubleList': result.doubleList!.map((e) => '$e')"},
		}, {
			&api.Field{Name: "stringList", JSONName: "stringList", Typez: api.STRING_TYPE, Repeated: true},
			[]string{"if (result.stringList.isNotDefault) 'stringList': result.stringList"},
		},

		// repeated primitives w/ optional
		{
			&api.Field{Name: "int32List_opt", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true, Optional: true},
			[]string{"if (result.int32ListOpt.isNotDefault) 'int32List': result.int32ListOpt!.map((e) => '$e')"},
		},

		// bytes, repeated bytes
		{
			&api.Field{Name: "bytes", JSONName: "bytes", Typez: api.BYTES_TYPE},
			[]string{"if (result.bytes != null) 'bytes': encodeBytes(result.bytes)!"},
		}, {
			&api.Field{Name: "bytesList", JSONName: "bytesList", Typez: api.BYTES_TYPE, Repeated: true},
			[]string{"if (result.bytesList != null) 'bytesList': result.bytesList!.map((e) => encodeBytes(e)!)"},
		},
	} {
		t.Run(test.field.Name, func(t *testing.T) {
			message := &api.Message{
				Name:    "UpdateSecretRequest",
				ID:      "..UpdateRequest",
				Package: sample.Package,
				Fields:  []*api.Field{test.field},
			}
			model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{})

			got := annotate.buildQueryLines([]string{}, "result.", "", test.field, model.State)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
			}
		})
	}
}

func TestBuildQueryLinesEnums(t *testing.T) {
	r := sample.Replication()
	a := sample.Automatic()
	enum := sample.EnumState()
	foreignEnumState := &api.Enum{
		Name:    "ForeignEnum",
		Package: "google.cloud.foo",
		ID:      "google.cloud.foo.ForeignEnum",
		Values: []*api.EnumValue{
			{
				Name:   "Enabled",
				Number: 1,
			},
		},
	}

	model := api.NewTestAPI(
		[]*api.Message{r, a, sample.CustomerManagedEncryption()},
		[]*api.Enum{enum, foreignEnumState},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{
		"prefix:google.cloud.foo": "foo",
	})
	for _, test := range []struct {
		enumField *api.Field
		want      []string
	}{
		{
			&api.Field{
				Name:     "enumName",
				JSONName: "jsonEnumName",
				Typez:    api.ENUM_TYPE,
				TypezID:  enum.ID},
			[]string{"if (result.enumName.isNotDefault) 'jsonEnumName': result.enumName.value"},
		},
		{
			&api.Field{
				Name:     "optionalEnum",
				JSONName: "optionalJsonEnum",
				Typez:    api.ENUM_TYPE,
				TypezID:  enum.ID,
				Optional: true},
			[]string{"if (result.optionalEnum != null) 'optionalJsonEnum': result.optionalEnum!.value"},
		},
		{
			&api.Field{
				Name:     "enumName",
				JSONName: "jsonEnumName",
				Typez:    api.ENUM_TYPE,
				TypezID:  foreignEnumState.ID,
				Optional: false},
			[]string{"if (result.enumName.isNotDefault) 'jsonEnumName': result.enumName.value"},
		},
	} {
		t.Run(test.enumField.Name, func(t *testing.T) {
			got := annotate.buildQueryLines([]string{}, "result.", "", test.enumField, model.State)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch in TestBuildQueryLinesEnums (-want, +got)\n:%s", diff)
			}
		})
	}
}

func TestBuildQueryLinesMessages(t *testing.T) {
	r := sample.Replication()
	a := sample.Automatic()
	secretVersion := sample.SecretVersion()
	updateRequest := sample.UpdateRequest()
	payload := sample.SecretPayload()
	model := api.NewTestAPI(
		[]*api.Message{r, a, sample.CustomerManagedEncryption(), secretVersion,
			updateRequest, sample.Secret(), payload},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	messageField1 := &api.Field{
		Name:     "message1",
		JSONName: "message1",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  secretVersion.ID,
	}
	messageField2 := &api.Field{
		Name:     "message2",
		JSONName: "message2",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  payload.ID,
	}
	messageField3 := &api.Field{
		Name:     "message3",
		JSONName: "message3",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  updateRequest.ID,
	}
	fieldMaskField := &api.Field{
		Name:     "field_mask",
		JSONName: "fieldMask",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".google.protobuf.FieldMask",
	}

	durationField := &api.Field{
		Name:     "duration",
		JSONName: "duration",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".google.protobuf.Duration",
	}

	timestampField := &api.Field{
		Name:     "time",
		JSONName: "time",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".google.protobuf.Timestamp",
	}

	// messages
	got := annotate.buildQueryLines([]string{}, "result.", "", messageField1, model.State)
	want := []string{
		"if (result.message1!.name.isNotDefault) 'message1.name': result.message1!.name",
		"if (result.message1!.state.isNotDefault) 'message1.state': result.message1!.state.value",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", "", messageField2, model.State)
	want = []string{
		"if (result.message2!.data != null) 'message2.data': encodeBytes(result.message2!.data)!",
		"if (result.message2!.dataCrc32C != null) 'message2.dataCrc32c': '${result.message2!.dataCrc32C}'",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	// nested messages
	got = annotate.buildQueryLines([]string{}, "result.", "", messageField3, model.State)
	want = []string{
		"if (result.message3!.secret!.name.isNotDefault) 'message3.secret.name': result.message3!.secret!.name",
		"if (result.message3!.fieldMask != null) 'message3.fieldMask': result.message3!.fieldMask!.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	// custom encoded messages
	got = annotate.buildQueryLines([]string{}, "result.", "", fieldMaskField, model.State)
	want = []string{
		"if (result.fieldMask != null) 'fieldMask': result.fieldMask!.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", "", durationField, model.State)
	want = []string{
		"if (result.duration != null) 'duration': result.duration!.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", "", timestampField, model.State)
	want = []string{
		"if (result.time != null) 'time': result.time!.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}
}

func TestCreateFromJsonLine(t *testing.T) {
	secret := sample.Secret()
	enumState := sample.EnumState()

	foreignMessage := &api.Message{
		Name:    "Foo",
		Package: "google.cloud.foo",
		ID:      "google.cloud.foo.Foo",
		Enums:   []*api.Enum{},
		Fields:  []*api.Field{},
	}
	foreignEnumState := &api.Enum{
		Name:    "ForeignEnum",
		Package: "google.cloud.foo",
		ID:      "google.cloud.foo.ForeignEnum",
		Values: []*api.EnumValue{
			{
				Name:   "Enabled",
				Number: 1,
			},
		},
	}
	mapStringToBytes := &api.Message{
		Name:  "$StringToBytes",
		ID:    "..$StringToBytes",
		IsMap: true,
		Fields: []*api.Field{
			{
				Name:  "key",
				Typez: api.STRING_TYPE,
			},
			{
				Name:  "value",
				Typez: api.BYTES_TYPE,
			},
		},
	}

	for _, test := range []struct {
		field *api.Field
		want  string
	}{
		// primitives
		{
			&api.Field{Name: "bool", JSONName: "bool", Typez: api.BOOL_TYPE},
			"json['bool'] ?? false",
		}, {
			&api.Field{Name: "int32", JSONName: "int32", Typez: api.INT32_TYPE},
			"json['int32'] ?? 0",
		}, {
			&api.Field{Name: "fixed32", JSONName: "fixed32", Typez: api.FIXED32_TYPE},
			"json['fixed32'] ?? 0",
		}, {
			&api.Field{Name: "string", JSONName: "string", Typez: api.STRING_TYPE},
			"json['string'] ?? ''",
		},

		// optional primitives
		{
			&api.Field{Name: "bool_opt", JSONName: "bool", Typez: api.BOOL_TYPE, Optional: true},
			"json['bool']",
		}, {
			&api.Field{Name: "int32_opt", JSONName: "int32", Typez: api.INT32_TYPE, Optional: true},
			"json['int32']",
		}, {
			&api.Field{Name: "string_opt", JSONName: "string", Typez: api.STRING_TYPE, Optional: true},
			"json['string']",
		},

		// one ofs
		{
			&api.Field{Name: "bool", JSONName: "bool", Typez: api.BOOL_TYPE, IsOneOf: true},
			"json['bool']",
		},

		// repeated primitives
		{
			&api.Field{Name: "boolList", JSONName: "boolList", Typez: api.BOOL_TYPE, Repeated: true},
			"decodeList(json['boolList']) ?? []",
		}, {
			&api.Field{Name: "int32List", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true},
			"decodeList(json['int32List']) ?? []",
		}, {
			&api.Field{Name: "stringList", JSONName: "stringList", Typez: api.STRING_TYPE, Repeated: true},
			"decodeList(json['stringList']) ?? []",
		},

		// repeated primitives w/ optional
		{
			&api.Field{Name: "int32List_opt", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true, Optional: true},
			"decodeList(json['int32List']) ?? []",
		},

		// bytes, repeated bytes
		{
			&api.Field{Name: "bytes", JSONName: "bytes", Typez: api.BYTES_TYPE},
			"decodeBytes(json['bytes'])",
		}, {
			&api.Field{Name: "bytesList", JSONName: "bytesList", Typez: api.BYTES_TYPE, Repeated: true},
			"decodeListBytes(json['bytesList'])",
		},

		// enums
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.ENUM_TYPE, TypezID: enumState.ID},
			"decodeEnum(json['message'], State.fromJson) ?? State.$default",
		},
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.ENUM_TYPE, TypezID: foreignEnumState.ID},
			"decodeEnum(json['message'], foo.ForeignEnum.fromJson) ?? foo.ForeignEnum.$default",
		},

		// messages
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: secret.ID},
			"decode(json['message'], Secret.fromJson)",
		},
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: foreignMessage.ID},
			"decode(json['message'], foo.Foo.fromJson)",
		},
		{
			// Custom encoding.
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: ".google.protobuf.Duration"},
			"decodeCustom(json['message'], Duration.fromJson)",
		},
		{
			// Map of bytes.
			&api.Field{Name: "message", JSONName: "message", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapStringToBytes.ID},
			"decodeMapBytes(json['message']) ?? {}",
		},
	} {
		t.Run(test.field.Name, func(t *testing.T) {
			message := &api.Message{
				Name:    "UpdateSecretRequest",
				ID:      "..UpdateRequest",
				Package: sample.Package,
				Fields:  []*api.Field{test.field},
			}
			model := api.NewTestAPI([]*api.Message{message, secret, foreignMessage, mapStringToBytes}, []*api.Enum{enumState, foreignEnumState}, []*api.Service{})
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{
				"prefix:google.cloud.foo": "foo",
			})
			codec := test.field.Codec.(*fieldAnnotation)

			got := annotate.createFromJsonLine(test.field, model.State, codec.Required)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
			}
		})
	}
}

func TestCreateToJsonLine(t *testing.T) {
	secret := sample.Secret()
	enum := sample.EnumState()

	foreignMessage := &api.Message{
		Name:    "Foo",
		Package: "google.cloud.foo",
		ID:      "google.cloud.foo.Foo",
		Enums:   []*api.Enum{},
		Fields:  []*api.Field{},
	}
	foreignEnumState := &api.Enum{
		Name:    "ForeignEnum",
		Package: "google.cloud.foo",
		ID:      "google.cloud.foo.ForeignEnum",
		Values: []*api.EnumValue{
			{
				Name:   "Enabled",
				Number: 1,
			},
		},
	}

	for _, test := range []struct {
		field *api.Field
		want  string
	}{
		// primitives
		{
			&api.Field{Name: "bool", JSONName: "bool", Typez: api.BOOL_TYPE},
			"bool$",
		}, {
			&api.Field{Name: "int32", JSONName: "int32", Typez: api.INT32_TYPE},
			"int32",
		}, {
			&api.Field{Name: "fixed32", JSONName: "fixed32", Typez: api.FIXED32_TYPE},
			"fixed32",
		}, {
			&api.Field{Name: "string", JSONName: "string", Typez: api.STRING_TYPE},
			"string",
		},

		// optional primitives
		{
			&api.Field{Name: "bool_opt", JSONName: "bool", Typez: api.BOOL_TYPE, Optional: true},
			"boolOpt",
		}, {
			&api.Field{Name: "int32_opt", JSONName: "int32", Typez: api.INT32_TYPE, Optional: true},
			"int32Opt",
		}, {
			&api.Field{Name: "string_opt", JSONName: "string", Typez: api.STRING_TYPE, Optional: true},
			"stringOpt",
		},

		// repeated primitives
		{
			&api.Field{Name: "boolList", JSONName: "boolList", Typez: api.BOOL_TYPE, Repeated: true},
			"boolList",
		}, {
			&api.Field{Name: "int32List", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true},
			"int32List",
		}, {
			&api.Field{Name: "stringList", JSONName: "stringList", Typez: api.STRING_TYPE, Repeated: true},
			"stringList",
		},

		// repeated enums
		{
			&api.Field{Name: "enumList", JSONName: "enumList", Typez: api.ENUM_TYPE, TypezID: enum.ID, Repeated: true},
			"encodeList(enumList)",
		},

		// repeated primitives w/ optional
		{
			&api.Field{Name: "int32List_opt", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true, Optional: true},
			"int32ListOpt",
		},

		// bytes, repeated bytes
		{
			&api.Field{Name: "bytes", JSONName: "bytes", Typez: api.BYTES_TYPE},
			"encodeBytes(bytes)",
		}, {
			&api.Field{Name: "bytesList", JSONName: "bytesList", Typez: api.BYTES_TYPE, Repeated: true},
			"encodeListBytes(bytesList)",
		},

		// enums
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.ENUM_TYPE, TypezID: enum.ID},
			"message.toJson()",
		},
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.ENUM_TYPE, TypezID: foreignEnumState.ID},
			"message.toJson()",
		},

		// messages
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: secret.ID},
			"message!.toJson()",
		},
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: foreignMessage.ID},
			"message!.toJson()",
		},
	} {
		t.Run(test.field.Name, func(t *testing.T) {
			message := &api.Message{
				Name:    "UpdateSecretRequest",
				ID:      "..UpdateRequest",
				Package: sample.Package,
				Fields:  []*api.Field{test.field},
			}
			model := api.NewTestAPI([]*api.Message{message, secret, foreignMessage}, []*api.Enum{enum, foreignEnumState}, []*api.Service{})
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{
				"prefix:google.cloud.foo": "foo",
			})
			codec := test.field.Codec.(*fieldAnnotation)

			got := createToJsonLine(test.field, model.State, codec.Required)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
			}
		})
	}
}
