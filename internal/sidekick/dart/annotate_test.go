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
	if diff := cmp.Diff("test.dart", codec.MainFileNameWithExtension); diff != "" {
		t.Errorf("mismatch in Codec.MainFileNameWithExtension (-want, +got)\n:%s", diff)
	}
}

func TestAnnotateModel_Options(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})

	var tests = []struct {
		options map[string]string
		verify  func(*testing.T, *annotateModel)
	}{
		{
			map[string]string{"library-path-override": "src/buffers.dart"},
			func(t *testing.T, am *annotateModel) {
				codec := model.Codec.(*modelAnnotations)
				if diff := cmp.Diff("src/buffers.dart", codec.MainFileNameWithExtension); diff != "" {
					t.Errorf("mismatch in Codec.MainFileNameWithExtension (-want, +got)\n:%s", diff)
				}
			},
		},
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
			[]string{"if (result.bool$ case final $1 when $1.isNotDefault) 'bool': '${$1}'"},
		}, {
			&api.Field{Name: "bytes", JSONName: "bytes", Typez: api.BYTES_TYPE},
			[]string{"if (result.bytes case final $1 when $1.isNotDefault) 'bytes': encodeBytes($1)!"},
		}, {
			&api.Field{Name: "int32", JSONName: "int32", Typez: api.INT32_TYPE},
			[]string{"if (result.int32 case final $1 when $1.isNotDefault) 'int32': '${$1}'"},
		}, {
			&api.Field{Name: "fixed32", JSONName: "fixed32", Typez: api.FIXED32_TYPE},
			[]string{"if (result.fixed32 case final $1 when $1.isNotDefault) 'fixed32': '${$1}'"},
		}, {
			&api.Field{Name: "sfixed32", JSONName: "sfixed32", Typez: api.SFIXED32_TYPE},
			[]string{"if (result.sfixed32 case final $1 when $1.isNotDefault) 'sfixed32': '${$1}'"},
		}, {
			&api.Field{Name: "int64", JSONName: "int64", Typez: api.INT64_TYPE},
			[]string{"if (result.int64 case final $1 when $1.isNotDefault) 'int64': '${$1}'"},
		}, {
			&api.Field{Name: "fixed64", JSONName: "fixed64", Typez: api.FIXED64_TYPE},
			[]string{"if (result.fixed64 case final $1 when $1.isNotDefault) 'fixed64': '${$1}'"},
		}, {
			&api.Field{Name: "sfixed64", JSONName: "sfixed64", Typez: api.SFIXED64_TYPE},
			[]string{"if (result.sfixed64 case final $1 when $1.isNotDefault) 'sfixed64': '${$1}'"},
		}, {
			&api.Field{Name: "double", JSONName: "double", Typez: api.DOUBLE_TYPE},
			[]string{"if (result.double$ case final $1 when $1.isNotDefault) 'double': '${$1}'"},
		}, {
			&api.Field{Name: "string", JSONName: "string", Typez: api.STRING_TYPE},
			[]string{"if (result.string case final $1 when $1.isNotDefault) 'string': $1"},
		},

		// optional primitives
		{
			&api.Field{Name: "bool_opt", JSONName: "bool", Typez: api.BOOL_TYPE, Optional: true},
			[]string{"if (result.boolOpt case final $1?) 'bool': '${$1}'"},
		}, {
			&api.Field{Name: "bytes_opt", JSONName: "bytes", Typez: api.BYTES_TYPE, Optional: true},
			[]string{"if (result.bytesOpt case final $1?) 'bytes': encodeBytes($1)!"},
		}, {
			&api.Field{Name: "int32_opt", JSONName: "int32", Typez: api.INT32_TYPE, Optional: true},
			[]string{"if (result.int32Opt case final $1?) 'int32': '${$1}'"},
		}, {
			&api.Field{Name: "fixed32_opt", JSONName: "fixed32", Typez: api.FIXED32_TYPE, Optional: true},
			[]string{"if (result.fixed32Opt case final $1?) 'fixed32': '${$1}'"},
		}, {
			&api.Field{Name: "sfixed32_opt", JSONName: "sfixed32", Typez: api.SFIXED32_TYPE, Optional: true},
			[]string{"if (result.sfixed32Opt case final $1?) 'sfixed32': '${$1}'"},
		}, {
			&api.Field{Name: "int64_opt", JSONName: "int64", Typez: api.INT64_TYPE, Optional: true},
			[]string{"if (result.int64Opt case final $1?) 'int64': '${$1}'"},
		}, {
			&api.Field{Name: "fixed64_opt", JSONName: "fixed64", Typez: api.FIXED64_TYPE, Optional: true},
			[]string{"if (result.fixed64Opt case final $1?) 'fixed64': '${$1}'"},
		}, {
			&api.Field{Name: "sfixed64_opt", JSONName: "sfixed64", Typez: api.SFIXED64_TYPE, Optional: true},
			[]string{"if (result.sfixed64Opt case final $1?) 'sfixed64': '${$1}'"},
		}, {
			&api.Field{Name: "double_opt", JSONName: "double", Typez: api.DOUBLE_TYPE, Optional: true},
			[]string{"if (result.doubleOpt case final $1?) 'double': '${$1}'"},
		}, {
			&api.Field{Name: "string_opt", JSONName: "string", Typez: api.STRING_TYPE, Optional: true},
			[]string{"if (result.stringOpt case final $1?) 'string': $1"},
		},

		// one ofs
		{
			&api.Field{Name: "bool", JSONName: "bool", Typez: api.BOOL_TYPE, IsOneOf: true},
			[]string{"if (result.bool$ case final $1?) 'bool': '${$1}'"},
		},

		// repeated primitives
		{
			&api.Field{Name: "boolList", JSONName: "boolList", Typez: api.BOOL_TYPE, Repeated: true},
			[]string{"if (result.boolList case final $1 when $1.isNotDefault) 'boolList': $1.map((e) => '$e')"},
		}, {
			&api.Field{Name: "bytesList", JSONName: "bytesList", Typez: api.BYTES_TYPE, Repeated: true},
			[]string{"if (result.bytesList case final $1 when $1.isNotDefault) 'bytesList': $1.map((e) => encodeBytes(e)!)"},
		}, {
			&api.Field{Name: "int32List", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true},
			[]string{"if (result.int32List case final $1 when $1.isNotDefault) 'int32List': $1.map((e) => '$e')"},
		}, {
			&api.Field{Name: "int64List", JSONName: "int64List", Typez: api.INT64_TYPE, Repeated: true},
			[]string{"if (result.int64List case final $1 when $1.isNotDefault) 'int64List': $1.map((e) => '$e')"},
		}, {
			&api.Field{Name: "doubleList", JSONName: "doubleList", Typez: api.DOUBLE_TYPE, Repeated: true},
			[]string{"if (result.doubleList case final $1 when $1.isNotDefault) 'doubleList': $1.map((e) => '$e')"},
		}, {
			&api.Field{Name: "stringList", JSONName: "stringList", Typez: api.STRING_TYPE, Repeated: true},
			[]string{"if (result.stringList case final $1 when $1.isNotDefault) 'stringList': $1"},
		},

		// repeated primitives w/ optional
		{
			&api.Field{Name: "int32List_opt", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true, Optional: true},
			[]string{"if (result.int32ListOpt case final $1 when $1.isNotDefault) 'int32List': $1.map((e) => '$e')"},
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
			[]string{"if (result.enumName case final $1 when $1.isNotDefault) 'jsonEnumName': $1.value"},
		},
		{
			&api.Field{
				Name:     "optionalEnum",
				JSONName: "optionalJsonEnum",
				Typez:    api.ENUM_TYPE,
				TypezID:  enum.ID,
				Optional: true},
			[]string{"if (result.optionalEnum case final $1?) 'optionalJsonEnum': $1.value"},
		},
		{
			&api.Field{
				Name:     "enumName",
				JSONName: "jsonEnumName",
				Typez:    api.ENUM_TYPE,
				TypezID:  foreignEnumState.ID,
				Optional: false},
			[]string{"if (result.enumName case final $1 when $1.isNotDefault) 'jsonEnumName': $1.value"},
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
		"if (result.message1!.name case final $1 when $1.isNotDefault) 'message1.name': $1",
		"if (result.message1!.state case final $1 when $1.isNotDefault) 'message1.state': $1.value",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", "", messageField2, model.State)
	want = []string{
		"if (result.message2!.data case final $1?) 'message2.data': encodeBytes($1)!",
		"if (result.message2!.dataCrc32C case final $1?) 'message2.dataCrc32c': '${$1}'",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	// nested messages
	got = annotate.buildQueryLines([]string{}, "result.", "", messageField3, model.State)
	want = []string{
		"if (result.message3!.secret!.name case final $1 when $1.isNotDefault) 'message3.secret.name': $1",
		"if (result.message3!.fieldMask case final $1?) 'message3.fieldMask': $1.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	// custom encoded messages
	got = annotate.buildQueryLines([]string{}, "result.", "", fieldMaskField, model.State)
	want = []string{
		"if (result.fieldMask case final $1?) 'fieldMask': $1.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", "", durationField, model.State)
	want = []string{
		"if (result.duration case final $1?) 'duration': $1.toJson()",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	got = annotate.buildQueryLines([]string{}, "result.", "", timestampField, model.State)
	want = []string{
		"if (result.time case final $1?) 'time': $1.toJson()",
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
	mapInt32ToBytes := &api.Message{
		Name:  "$Int32ToBytes",
		ID:    "..$Int32ToBytes",
		IsMap: true,
		Fields: []*api.Field{
			{
				Name:  "key",
				Typez: api.INT32_TYPE,
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
			"switch (json['bool']) { null => false, Object $1 => decodeBool($1)}",
		}, {
			&api.Field{Name: "bytes", JSONName: "bytes", Typez: api.BYTES_TYPE},
			"switch (json['bytes']) { null => Uint8List(0), Object $1 => decodeBytes($1)}",
		}, {
			&api.Field{Name: "double", JSONName: "double", Typez: api.DOUBLE_TYPE},
			"switch (json['double']) { null => 0, Object $1 => decodeDouble($1)}",
		}, {
			&api.Field{Name: "fixed32", JSONName: "fixed32", Typez: api.FIXED32_TYPE},
			"switch (json['fixed32']) { null => 0, Object $1 => decodeInt($1)}",
		}, {
			&api.Field{Name: "fixed64", JSONName: "fixed64", Typez: api.FIXED64_TYPE},
			"switch (json['fixed64']) { null => BigInt.zero, Object $1 => decodeUint64($1)}",
		}, {
			&api.Field{Name: "float", JSONName: "float", Typez: api.FLOAT_TYPE},
			"switch (json['float']) { null => 0, Object $1 => decodeDouble($1)}",
		}, {
			&api.Field{Name: "int32", JSONName: "int32", Typez: api.INT32_TYPE},
			"switch (json['int32']) { null => 0, Object $1 => decodeInt($1)}",
		}, {
			&api.Field{Name: "int64", JSONName: "int64", Typez: api.INT64_TYPE},
			"switch (json['int64']) { null => 0, Object $1 => decodeInt64($1)}",
		}, {
			&api.Field{Name: "sfixed32", JSONName: "sfixed32", Typez: api.SFIXED32_TYPE},
			"switch (json['sfixed32']) { null => 0, Object $1 => decodeInt($1)}",
		}, {
			&api.Field{Name: "sfixed64", JSONName: "sfixed64", Typez: api.SFIXED64_TYPE},
			"switch (json['sfixed64']) { null => 0, Object $1 => decodeInt64($1)}",
		}, {
			&api.Field{Name: "sint64", JSONName: "sint64", Typez: api.SINT64_TYPE},
			"switch (json['sint64']) { null => 0, Object $1 => decodeInt64($1)}",
		}, {
			&api.Field{Name: "string", JSONName: "string", Typez: api.STRING_TYPE},
			"switch (json['string']) { null => '', Object $1 => decodeString($1)}",
		}, {
			&api.Field{Name: "uint32", JSONName: "uint32", Typez: api.UINT32_TYPE},
			"switch (json['uint32']) { null => 0, Object $1 => decodeInt($1)}",
		}, {
			&api.Field{Name: "uint64", JSONName: "uint64", Typez: api.UINT64_TYPE},
			"switch (json['uint64']) { null => BigInt.zero, Object $1 => decodeUint64($1)}",
		},

		// optional primitives
		{
			&api.Field{Name: "bool_opt", JSONName: "bool", Typez: api.BOOL_TYPE, Optional: true},
			"switch (json['bool']) { null => null, Object $1 => decodeBool($1)}",
		}, {
			&api.Field{Name: "bytes_opt", JSONName: "bytes", Typez: api.BYTES_TYPE, Optional: true},
			"switch (json['bytes']) { null => null, Object $1 => decodeBytes($1)}",
		}, {
			&api.Field{Name: "double_opt", JSONName: "double", Typez: api.DOUBLE_TYPE, Optional: true},
			"switch (json['double']) { null => null, Object $1 => decodeDouble($1)}",
		}, {
			&api.Field{Name: "fixed64_opt", JSONName: "fixed64", Typez: api.FIXED64_TYPE, Optional: true},
			"switch (json['fixed64']) { null => null, Object $1 => decodeUint64($1)}",
		}, {
			&api.Field{Name: "float_opt", JSONName: "float", Typez: api.FLOAT_TYPE, Optional: true},
			"switch (json['float']) { null => null, Object $1 => decodeDouble($1)}",
		}, {
			&api.Field{Name: "int32_opt", JSONName: "int32", Typez: api.INT32_TYPE, Optional: true},
			"switch (json['int32']) { null => null, Object $1 => decodeInt($1)}",
		}, {
			&api.Field{Name: "int64_opt", JSONName: "int64", Typez: api.INT64_TYPE, Optional: true},
			"switch (json['int64']) { null => null, Object $1 => decodeInt64($1)}",
		}, {
			&api.Field{Name: "sfixed32_opt", JSONName: "sfixed32", Typez: api.SFIXED32_TYPE, Optional: true},
			"switch (json['sfixed32']) { null => null, Object $1 => decodeInt($1)}",
		}, {
			&api.Field{Name: "sfixed64_opt", JSONName: "sfixed64", Typez: api.SFIXED64_TYPE, Optional: true},
			"switch (json['sfixed64']) { null => null, Object $1 => decodeInt64($1)}",
		}, {
			&api.Field{Name: "sint64_opt", JSONName: "sint64", Typez: api.SINT64_TYPE, Optional: true},
			"switch (json['sint64']) { null => null, Object $1 => decodeInt64($1)}",
		}, {
			&api.Field{Name: "string_opt", JSONName: "string", Typez: api.STRING_TYPE, Optional: true},
			"switch (json['string']) { null => null, Object $1 => decodeString($1)}",
		}, {
			&api.Field{Name: "uint32_opt", JSONName: "uint32", Typez: api.UINT32_TYPE, Optional: true},
			"switch (json['uint32']) { null => null, Object $1 => decodeInt($1)}",
		}, {
			&api.Field{Name: "uint64_opt", JSONName: "uint64", Typez: api.UINT64_TYPE, Optional: true},
			"switch (json['uint64']) { null => null, Object $1 => decodeUint64($1)}",
		},

		// one ofs
		{
			&api.Field{Name: "bool", JSONName: "bool", Typez: api.BOOL_TYPE, IsOneOf: true},
			"switch (json['bool']) { null => null, Object $1 => decodeBool($1)}",
		},

		// repeated primitives
		{
			&api.Field{Name: "boolList", JSONName: "boolList", Typez: api.BOOL_TYPE, Repeated: true},
			"switch (json['boolList']) { null => [], List<Object?> $1 => [for (final i in $1) decodeBool(i)], _ => throw const FormatException('\"boolList\" is not a list') }",
		}, {
			&api.Field{Name: "bytesList", JSONName: "bytesList", Typez: api.BYTES_TYPE, Repeated: true},
			"switch (json['bytesList']) { null => [], List<Object?> $1 => [for (final i in $1) decodeBytes(i)], _ => throw const FormatException('\"bytesList\" is not a list') }",
		}, {
			&api.Field{Name: "doubleList", JSONName: "doubleList", Typez: api.DOUBLE_TYPE, Repeated: true},
			"switch (json['doubleList']) { null => [], List<Object?> $1 => [for (final i in $1) decodeDouble(i)], _ => throw const FormatException('\"doubleList\" is not a list') }",
		}, {
			&api.Field{Name: "fixed32List", JSONName: "fixed32List", Typez: api.FIXED32_TYPE, Repeated: true},
			"switch (json['fixed32List']) { null => [], List<Object?> $1 => [for (final i in $1) decodeInt(i)], _ => throw const FormatException('\"fixed32List\" is not a list') }",
		}, {
			&api.Field{Name: "int32List", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true},
			"switch (json['int32List']) { null => [], List<Object?> $1 => [for (final i in $1) decodeInt(i)], _ => throw const FormatException('\"int32List\" is not a list') }",
		}, {
			&api.Field{Name: "stringList", JSONName: "stringList", Typez: api.STRING_TYPE, Repeated: true},
			"switch (json['stringList']) { null => [], List<Object?> $1 => [for (final i in $1) decodeString(i)], _ => throw const FormatException('\"stringList\" is not a list') }",
		},

		// repeated primitives w/ optional
		{
			&api.Field{Name: "int32List_opt", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true, Optional: true},
			"switch (json['int32List']) { null => [], List<Object?> $1 => [for (final i in $1) decodeInt(i)], _ => throw const FormatException('\"int32List\" is not a list') }",
		},

		// enums
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.ENUM_TYPE, TypezID: enumState.ID},
			"switch (json['message']) { null => State.$default, Object $1 => State.fromJson($1)}",
		},
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.ENUM_TYPE, TypezID: foreignEnumState.ID},
			"switch (json['message']) { null => foo.ForeignEnum.$default, Object $1 => foo.ForeignEnum.fromJson($1)}",
		},

		// messages
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: secret.ID},
			"switch (json['message']) { null => null, Object $1 => Secret.fromJson($1)}",
		},
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: foreignMessage.ID},
			"switch (json['message']) { null => null, Object $1 => foo.Foo.fromJson($1)}",
		},
		{
			// Custom encoding.
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: ".google.protobuf.Duration"},
			"switch (json['message']) { null => null, Object $1 => Duration.fromJson($1)}",
		},

		// maps
		{
			// string -> bytes
			&api.Field{Name: "message", JSONName: "message", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapStringToBytes.ID},
			"switch (json['message']) { null => {}, Map<String, Object?> $1 => {for (final e in $1.entries) decodeString(e.key): decodeBytes(e.value)}, _ => throw const FormatException('\"message\" is not an object') }",
		},
		{
			// int32 -> bytes
			&api.Field{Name: "message", JSONName: "message", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapInt32ToBytes.ID},
			"switch (json['message']) { null => {}, Map<String, Object?> $1 => {for (final e in $1.entries) decodeIntKey(e.key): decodeBytes(e.value)}, _ => throw const FormatException('\"message\" is not an object') }",
		},
	} {
		t.Run(test.field.Name, func(t *testing.T) {
			message := &api.Message{
				Name:    "UpdateSecretRequest",
				ID:      "..UpdateRequest",
				Package: sample.Package,
				Fields:  []*api.Field{test.field},
			}
			model := api.NewTestAPI([]*api.Message{message,
				secret, foreignMessage, mapStringToBytes, mapInt32ToBytes},
				[]*api.Enum{enumState, foreignEnumState},
				[]*api.Service{})
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

	mapStringToString := &api.Message{
		Name:  "$StringToString",
		ID:    "..$StringToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.STRING_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapInt32ToString := &api.Message{
		Name:  "$Int32ToString",
		ID:    "..$Int32ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.INT32_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapBoolToString := &api.Message{
		Name:  "$BoolToString",
		ID:    "..$BoolToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.BOOL_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapStringToInt64 := &api.Message{
		Name:  "$StringToInt64",
		ID:    "..$StringToInt64",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.STRING_TYPE},
			{Name: "value", Typez: api.INT64_TYPE},
		},
	}
	mapInt64ToString := &api.Message{
		Name:  "$Int64ToString",
		ID:    "..$Int64ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.INT64_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapUint32ToString := &api.Message{
		Name:  "$Uint32ToString",
		ID:    "..$Uint32ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.UINT32_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapUint64ToString := &api.Message{
		Name:  "$Uint64ToString",
		ID:    "..$Uint64ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.UINT64_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapSint32ToString := &api.Message{
		Name:  "$Sint32ToString",
		ID:    "..$Sint32ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.SINT32_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapSint64ToString := &api.Message{
		Name:  "$Sint64ToString",
		ID:    "..$Sint64ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.SINT64_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapFixed32ToString := &api.Message{
		Name:  "$Fixed32ToString",
		ID:    "..$Fixed32ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.FIXED32_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapFixed64ToString := &api.Message{
		Name:  "$Fixed64ToString",
		ID:    "..$Fixed64ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.FIXED64_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapSfixed32ToString := &api.Message{
		Name:  "$Sfixed32ToString",
		ID:    "..$Sfixed32ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.SFIXED32_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
		},
	}
	mapSfixed64ToString := &api.Message{
		Name:  "$Sfixed64ToString",
		ID:    "..$Sfixed64ToString",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", Typez: api.SFIXED64_TYPE},
			{Name: "value", Typez: api.STRING_TYPE},
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
			&api.Field{Name: "bytes", JSONName: "bytes", Typez: api.BYTES_TYPE},
			"encodeBytes(bytes)",
		}, {
			&api.Field{Name: "double", JSONName: "double", Typez: api.DOUBLE_TYPE},
			"encodeDouble(double$)",
		}, {
			&api.Field{Name: "fixed32", JSONName: "fixed32", Typez: api.FIXED32_TYPE},
			"fixed32",
		}, {
			&api.Field{Name: "fixed64", JSONName: "fixed64", Typez: api.FIXED64_TYPE},
			"fixed64.toString()",
		}, {
			&api.Field{Name: "float", JSONName: "float", Typez: api.FLOAT_TYPE},
			"encodeDouble(float)",
		}, {
			&api.Field{Name: "int32", JSONName: "int32", Typez: api.INT32_TYPE},
			"int32",
		}, {
			&api.Field{Name: "int64", JSONName: "int64", Typez: api.INT64_TYPE},
			"int64.toString()",
		}, {
			&api.Field{Name: "sfixed32", JSONName: "sfixed32", Typez: api.SFIXED32_TYPE},
			"sfixed32",
		}, {
			&api.Field{Name: "sfixed64", JSONName: "sfixed64", Typez: api.SFIXED64_TYPE},
			"sfixed64.toString()",
		}, {
			&api.Field{Name: "sint32", JSONName: "sint32", Typez: api.SINT32_TYPE},
			"sint32",
		}, {
			&api.Field{Name: "sint64", JSONName: "sint64", Typez: api.SINT64_TYPE},
			"sint64.toString()",
		}, {
			&api.Field{Name: "string", JSONName: "string", Typez: api.STRING_TYPE},
			"string",
		}, {
			&api.Field{Name: "uint32", JSONName: "uint32", Typez: api.UINT32_TYPE},
			"uint32",
		}, {
			&api.Field{Name: "uint64", JSONName: "uint64", Typez: api.UINT64_TYPE},
			"uint64.toString()",
		},

		// enums
		{
			&api.Field{Name: "enum1", JSONName: "enum1", Typez: api.ENUM_TYPE, TypezID: enum.ID},
			"enum1.toJson()",
		},

		// messages
		{
			&api.Field{Name: "message", JSONName: "message", Typez: api.MESSAGE_TYPE, TypezID: secret.ID},
			"message.toJson()",
		},

		// repeated primitives
		{
			&api.Field{Name: "boolList", JSONName: "boolList", Typez: api.BOOL_TYPE, Repeated: true},
			"boolList",
		}, {
			&api.Field{Name: "bytesList", JSONName: "bytesList", Typez: api.BYTES_TYPE, Repeated: true},
			"[for (final i in bytesList) encodeBytes(i)]",
		}, {
			&api.Field{Name: "doubleList", JSONName: "doubleList", Typez: api.DOUBLE_TYPE, Repeated: true},
			"[for (final i in doubleList) encodeDouble(i)]",
		}, {
			&api.Field{Name: "fixed32List", JSONName: "fixed32List", Typez: api.FIXED32_TYPE, Repeated: true},
			"fixed32List",
		}, {
			&api.Field{Name: "fixed64List", JSONName: "fixed64List", Typez: api.FIXED64_TYPE, Repeated: true},
			"[for (final i in fixed64List) i.toString()]",
		}, {
			&api.Field{Name: "floatList", JSONName: "floatList", Typez: api.FLOAT_TYPE, Repeated: true},
			"[for (final i in floatList) encodeDouble(i)]",
		}, {
			&api.Field{Name: "int32List", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true},
			"int32List",
		}, {
			&api.Field{Name: "int64List", JSONName: "int64List", Typez: api.INT64_TYPE, Repeated: true},
			"[for (final i in int64List) i.toString()]",
		}, {
			&api.Field{Name: "sfixed32List", JSONName: "sfixed32List", Typez: api.SFIXED32_TYPE, Repeated: true},
			"sfixed32List",
		}, {
			&api.Field{Name: "sfixed64List", JSONName: "sfixed64List", Typez: api.SFIXED64_TYPE, Repeated: true},
			"[for (final i in sfixed64List) i.toString()]",
		}, {
			&api.Field{Name: "sint32List", JSONName: "sint32List", Typez: api.SINT32_TYPE, Repeated: true},
			"sint32List",
		}, {
			&api.Field{Name: "sint64List", JSONName: "sint64List", Typez: api.SINT64_TYPE, Repeated: true},
			"[for (final i in sint64List) i.toString()]",
		}, {
			&api.Field{Name: "stringList", JSONName: "stringList", Typez: api.STRING_TYPE, Repeated: true},
			"stringList",
		}, {
			&api.Field{Name: "uint32List", JSONName: "uint32List", Typez: api.UINT32_TYPE, Repeated: true},
			"uint32List",
		}, {
			&api.Field{Name: "uint64List", JSONName: "uint64List", Typez: api.UINT64_TYPE, Repeated: true},
			"[for (final i in uint64List) i.toString()]",
		},

		// repeated enums
		{
			&api.Field{Name: "enumList", JSONName: "enumList", Typez: api.ENUM_TYPE, TypezID: enum.ID, Repeated: true},
			"[for (final i in enumList) i.toJson()]",
		},

		// repeated messages
		{
			&api.Field{Name: "messageList", JSONName: "messageList", Typez: api.MESSAGE_TYPE, TypezID: secret.ID, Repeated: true},
			"[for (final i in messageList) i.toJson()]",
		},

		// maps
		{
			&api.Field{Name: "map_string_to_string", JSONName: "mapStringToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapStringToString.ID},
			"mapStringToString",
		},
		{
			&api.Field{Name: "map_int32_to_string", JSONName: "mapInt32ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapInt32ToString.ID},
			"{for (final e in mapInt32ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_bool_to_string", JSONName: "mapBoolToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapBoolToString.ID},
			"{for (final e in mapBoolToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_string_to_int64", JSONName: "mapStringToInt64", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapStringToInt64.ID},
			"{for (final e in mapStringToInt64.entries) e.key: e.value.toString()}",
		},
		{
			&api.Field{Name: "map_int64_to_string", JSONName: "mapInt64ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapInt64ToString.ID},
			"{for (final e in mapInt64ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_uint32_to_string", JSONName: "mapUint32ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapUint32ToString.ID},
			"{for (final e in mapUint32ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_uint64_to_string", JSONName: "mapUint64ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapUint64ToString.ID},
			"{for (final e in mapUint64ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_sint32_to_string", JSONName: "mapSint32ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapSint32ToString.ID},
			"{for (final e in mapSint32ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_sint64_to_string", JSONName: "mapSint64ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapSint64ToString.ID},
			"{for (final e in mapSint64ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_fixed32_to_string", JSONName: "mapFixed32ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapFixed32ToString.ID},
			"{for (final e in mapFixed32ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_fixed64_to_string", JSONName: "mapFixed64ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapFixed64ToString.ID},
			"{for (final e in mapFixed64ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_sfixed32_to_string", JSONName: "mapSfixed32ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapSfixed32ToString.ID},
			"{for (final e in mapSfixed32ToString.entries) e.key.toString(): e.value}",
		},
		{
			&api.Field{Name: "map_sfixed64_to_string", JSONName: "mapSfixed64ToString", Map: true, Typez: api.MESSAGE_TYPE, TypezID: mapSfixed64ToString.ID},
			"{for (final e in mapSfixed64ToString.entries) e.key.toString(): e.value}",
		},
	} {
		t.Run(test.field.Name, func(t *testing.T) {
			message := &api.Message{
				Name:    "UpdateSecretRequest",
				ID:      "..UpdateRequest",
				Package: sample.Package,
				Fields:  []*api.Field{test.field},
			}
			model := api.NewTestAPI([]*api.Message{
				message, secret, foreignMessage,
				mapStringToString, mapInt32ToString, mapBoolToString, mapStringToInt64,
				mapInt64ToString, mapUint32ToString, mapUint64ToString,
				mapSint32ToString, mapSint64ToString,
				mapFixed32ToString, mapFixed64ToString,
				mapSfixed32ToString, mapSfixed64ToString,
			}, []*api.Enum{enum, foreignEnumState}, []*api.Service{})
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{
				"prefix:google.cloud.foo": "foo",
			})

			got := createToJsonLine(test.field, model.State)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch in TestCreateToJsonLine (-want, +got)\n:%s", diff)
			}
		})
	}
}

func TestAnnotateEnum(t *testing.T) {
	type wantedValueAnnotation struct {
		wantValueName string
	}

	enumValueSimple := &api.EnumValue{
		Name: "NAME",
		ID:   ".test.v1.SomeMessage.SomeEnum.NAME",
	}
	enumValueReservedName := &api.EnumValue{
		Name: "in",
		ID:   ".test.v1.SomeMessage.SomeEnum.in",
	}
	enumValueCompound := &api.EnumValue{
		Name: "ENUM_VALUE",
		ID:   ".test.v1.SomeMessage.SomeEnum.ENUM_VALUE",
	}
	enumValueNameDifferentCaseOnly := &api.EnumValue{
		Name: "name",
		ID:   ".test.v1.SomeMessage.SomeEnum.name",
	}
	someEnum := &api.Enum{
		Name:    "SomeEnum",
		ID:      ".test.v1.SomeMessage.SomeEnum",
		Values:  []*api.EnumValue{enumValueSimple, enumValueReservedName, enumValueCompound},
		Package: "test.v1",
	}
	noValuesEnum := &api.Enum{
		Name:    "NoValuesEnum",
		ID:      ".test.v1.NoValuesEnum",
		Values:  []*api.EnumValue{},
		Package: "test.v1",
	}
	someEnumNameDifferentCaseOnly := &api.Enum{
		Name:    "DifferentCaseOnlyEnum",
		ID:      ".test.v1.SomeMessage.SomeDifferentCaseOnlyEnum",
		Values:  []*api.EnumValue{enumValueSimple, enumValueNameDifferentCaseOnly},
		Package: "test.v1",
	}

	model := api.NewTestAPI(
		[]*api.Message{},
		[]*api.Enum{someEnum, noValuesEnum, someEnumNameDifferentCaseOnly},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)

	for _, test := range []struct {
		enum                 *api.Enum
		wantEnumName         string
		wantEnumDefaultValue string
		wantValueAnnotations []wantedValueAnnotation
	}{
		{enum: someEnum,
			wantEnumName:         "SomeEnum",
			wantEnumDefaultValue: "name",
			wantValueAnnotations: []wantedValueAnnotation{{"name"}, {"in$"}, {"enumValue"}},
		},
		{enum: noValuesEnum,
			wantEnumName:         "NoValuesEnum",
			wantEnumDefaultValue: "",
			wantValueAnnotations: []wantedValueAnnotation{},
		},
		{enum: someEnumNameDifferentCaseOnly,
			wantEnumName:         "DifferentCaseOnlyEnum",
			wantEnumDefaultValue: "NAME",
			wantValueAnnotations: []wantedValueAnnotation{{"NAME"}, {"name"}},
		},
	} {
		annotate.annotateEnum(test.enum)
		codec := test.enum.Codec.(*enumAnnotation)
		gotEnumName := codec.Name
		gotEnumDefaultValue := codec.DefaultValue

		if diff := cmp.Diff(test.wantEnumName, gotEnumName); diff != "" {
			t.Errorf("mismatch in TestAnnotateEnum(%q) (-want, +got)\n:%s", test.enum.Name, diff)
		}
		if diff := cmp.Diff(test.wantEnumDefaultValue, gotEnumDefaultValue); diff != "" {
			t.Errorf("mismatch in TestAnnotateEnum(%q) (-want, +got)\n:%s", test.enum.Name, diff)
		}

		for i, value := range test.enum.Values {
			wantValueAnnotation := test.wantValueAnnotations[i]
			gotValueAnnotation := value.Codec.(*enumValueAnnotation)
			if diff := cmp.Diff(wantValueAnnotation.wantValueName, gotValueAnnotation.Name); diff != "" {
				t.Errorf("mismatch in TestAnnotateEnum(%q) [value annotation %d] (-want, +got)\n:%s", test.enum.Name, i, diff)
			}
		}
	}
}
