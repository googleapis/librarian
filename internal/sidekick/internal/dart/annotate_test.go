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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/sample"
)

var (
	requiredConfig = map[string]string{
		"api-keys-environment-variables": "GOOGLE_API_KEY,GEMINI_API_KEY",
		"issue-tracker-url":              "http://www.example.com/issues",
		"package:googleapis_auth":        "^2.0.0",
		"package:google_cloud_gax":       "^1.2.3",
		"package:http":                   "^4.5.6",
		"package:google_cloud_protobuf":  "^7.8.9",
	}
)

func TestAnnotateModel(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "test"

	options := maps.Clone(requiredConfig)
	maps.Copy(options, map[string]string{"package:google_cloud_gax": "^1.2.3"})

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
				if diff := cmp.Diff([]string{"test", "mockito", "lints"}, codec.DevDependencies); diff != "" {
					t.Errorf("mismatch in Codec.PackageName (-want, +got)\n:%s", diff)
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
			map[string]string{"google_cloud_gax": "^1.2.3", "package:http": "1.2.0"},
			func(t *testing.T, am *annotateModel) {
				if diff := cmp.Diff(map[string]string{
					"google_cloud_gax":      "^1.2.3",
					"google_cloud_protobuf": "^7.8.9",
					"googleapis_auth":       "^2.0.0",
					"http":                  "1.2.0"},
					am.dependencyConstraints); diff != "" {
					t.Errorf("mismatch in annotateModel.dependencyConstraints (-want, +got)\n:%s", diff)
				}
			},
		},
	}

	for _, tt := range tests {
		annotate := newAnnotateModel(model)
		options := maps.Clone(requiredConfig)
		maps.Copy(options, tt.options)
		err := annotate.annotateModel(maps.Clone(options))
		if err != nil {
			t.Fatal(err)
		}
		tt.verify(t, annotate)
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

	for _, tt := range tests {
		annotate := newAnnotateModel(model)
		options := maps.Clone(requiredConfig)
		delete(options, tt)

		err := annotate.annotateModel(options)
		if err == nil {
			t.Fatalf("expected error when missing %q", tt)
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

func TestCalculateDependencies(t *testing.T) {
	for _, test := range []struct {
		testName string
		pkgName  string
		imports  []string
		want     []packageDependency
	}{
		{testName: "empty", pkgName: "google_cloud_bar", imports: []string{}, want: []packageDependency{}},
		{testName: "dart import", pkgName: "google_cloud_bar", imports: []string{typedDataImport}, want: []packageDependency{}},
		{testName: "package import", pkgName: "google_cloud_bar", imports: []string{httpImport}, want: []packageDependency{{Name: "http", Constraint: "^1.3.0"}}},
		{testName: "dart and package imports", pkgName: "google_cloud_bar", imports: []string{typedDataImport, httpImport}, want: []packageDependency{{Name: "http", Constraint: "^1.3.0"}}},
		{testName: "package imports", pkgName: "google_cloud_bar", imports: []string{
			httpImport,
			"package:google_cloud_foo/foo.dart",
		}, want: []packageDependency{{Name: "google_cloud_foo", Constraint: "^1.2.3"}, {Name: "http", Constraint: "^1.3.0"}}},
		{testName: "same package", pkgName: "google_cloud_bar", imports: []string{typedDataImport, httpImport, "google_cloud_bar"}, want: []packageDependency{{Name: "http", Constraint: "^1.3.0"}}},
	} {
		t.Run(test.testName, func(t *testing.T) {
			deps := map[string]bool{}
			for _, imp := range test.imports {
				deps[imp] = true
			}
			got, err := calculateDependencies(deps, map[string]string{"google_cloud_foo": "^1.2.3", "http": "^1.3.0"}, test.pkgName)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch in %q in calculateDependencies (-want, +got)\n:%s", test.testName, diff)
			}
		})
	}
}

func TestCalculateImports(t *testing.T) {
	for _, test := range []struct {
		name    string
		imports []string
		want    []string
	}{
		{name: "dart import", imports: []string{typedDataImport}, want: []string{
			"import 'dart:typed_data';",
		}},
		{name: "package import", imports: []string{httpImport}, want: []string{
			"import 'package:http/http.dart' as http;",
		}},
		{name: "dart and package imports", imports: []string{typedDataImport, httpImport}, want: []string{
			"import 'dart:typed_data';",
			"",
			"import 'package:http/http.dart' as http;",
		}},
		{name: "package imports", imports: []string{
			httpImport,
			"package:google_cloud_foo/foo.dart",
		}, want: []string{
			"import 'package:google_cloud_foo/foo.dart';",
			"import 'package:http/http.dart' as http;",
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			deps := map[string]bool{}
			for _, imp := range test.imports {
				deps[imp] = true
			}
			got := calculateImports(deps)

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

func TestCalculateRequiredFields(t *testing.T) {
	service := &api.Service{
		Name:          sample.ServiceName,
		Documentation: sample.APIDescription,
		DefaultHost:   sample.DefaultHost,
		Methods:       []*api.Method{sample.MethodListSecretVersions()},
		Package:       sample.Package,
	}
	model := api.NewTestAPI(
		[]*api.Message{sample.ListSecretVersionsRequest(), sample.ListSecretVersionsResponse(),
			sample.Secret(), sample.SecretVersion(), sample.Replication()},
		[]*api.Enum{sample.EnumState()},
		[]*api.Service{service},
	)
	api.Validate(model)

	// Test that field annotations correctly calculate their required state; this
	// uses the method's PathInfo.
	requiredFields := calculateRequiredFields(model)

	got := map[string]string{}
	for key, value := range requiredFields {
		got[key] = value.Name
	}

	want := map[string]string{
		"..Secret.parent": "parent",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestCalculateRequiredFields (-want, +got)\n:%s", diff)
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
			[]string{"if (result.bool != null) 'bool': '${result.bool}'"},
		}, {
			&api.Field{Name: "int32", JSONName: "int32", Typez: api.INT32_TYPE},
			[]string{"if (result.int32 != null) 'int32': '${result.int32}'"},
		}, {
			&api.Field{Name: "fixed32", JSONName: "fixed32", Typez: api.FIXED32_TYPE},
			[]string{"if (result.fixed32 != null) 'fixed32': '${result.fixed32}'"},
		}, {
			&api.Field{Name: "sfixed32", JSONName: "sfixed32", Typez: api.SFIXED32_TYPE},
			[]string{"if (result.sfixed32 != null) 'sfixed32': '${result.sfixed32}'"},
		}, {
			&api.Field{Name: "int64", JSONName: "int64", Typez: api.INT64_TYPE},
			[]string{"if (result.int64 != null) 'int64': '${result.int64}'"},
		}, {
			&api.Field{Name: "fixed64", JSONName: "fixed64", Typez: api.FIXED64_TYPE},
			[]string{"if (result.fixed64 != null) 'fixed64': '${result.fixed64}'"},
		}, {
			&api.Field{Name: "sfixed64", JSONName: "sfixed64", Typez: api.SFIXED64_TYPE},
			[]string{"if (result.sfixed64 != null) 'sfixed64': '${result.sfixed64}'"},
		}, {
			&api.Field{Name: "double", JSONName: "double", Typez: api.DOUBLE_TYPE},
			[]string{"if (result.double != null) 'double': '${result.double}'"},
		}, {
			&api.Field{Name: "string", JSONName: "string", Typez: api.STRING_TYPE},
			[]string{"if (result.string != null) 'string': result.string!"},
		},

		// repeated primitives
		{
			&api.Field{Name: "boolList", JSONName: "boolList", Typez: api.BOOL_TYPE, Repeated: true},
			[]string{"if (result.boolList != null) 'boolList': result.boolList!.map((e) => '$e')"},
		}, {
			&api.Field{Name: "int32List", JSONName: "int32List", Typez: api.INT32_TYPE, Repeated: true},
			[]string{"if (result.int32List != null) 'int32List': result.int32List!.map((e) => '$e')"},
		}, {
			&api.Field{Name: "int64List", JSONName: "int64List", Typez: api.INT64_TYPE, Repeated: true},
			[]string{"if (result.int64List != null) 'int64List': result.int64List!.map((e) => '$e')"},
		}, {
			&api.Field{Name: "doubleList", JSONName: "doubleList", Typez: api.DOUBLE_TYPE, Repeated: true},
			[]string{"if (result.doubleList != null) 'doubleList': result.doubleList!.map((e) => '$e')"},
		}, {
			&api.Field{Name: "stringList", JSONName: "stringList", Typez: api.STRING_TYPE, Repeated: true},
			[]string{"if (result.stringList != null) 'stringList': result.stringList!"},
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

			got := buildQueryLines([]string{}, "result.", "", test.field, model.State)
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
	model := api.NewTestAPI(
		[]*api.Message{r, a, sample.CustomerManagedEncryption()},
		[]*api.Enum{enum},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	enumField := &api.Field{
		Name:     "enumName",
		JSONName: "enumName",
		Typez:    api.ENUM_TYPE,
		TypezID:  enum.ID,
	}

	got := buildQueryLines([]string{}, "result.", "", enumField, model.State)
	want := []string{
		"if (result.enumName != null) 'enumName': result.enumName!.value",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
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

	// messages
	got := buildQueryLines([]string{}, "result.", "", messageField1, model.State)
	want := []string{
		"if (result.message1?.name != null) 'message1.name': result.message1?.name!",
		"if (result.message1?.state != null) 'message1.state': result.message1?.state!.value",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	got = buildQueryLines([]string{}, "result.", "", messageField2, model.State)
	want = []string{
		"if (result.message2?.data != null) 'message2.data': encodeBytes(result.message2?.data)!",
		"if (result.message2?.dataCrc32C != null) 'message2.dataCrc32c': '${result.message2?.dataCrc32C}'",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}

	// nested messages
	got = buildQueryLines([]string{}, "result.", "", messageField3, model.State)
	want = []string{
		"if (result.message3?.secret?.name != null) 'message3.secret.name': result.message3?.secret?.name!",
		"if (result.message3?.fieldMask?.paths != null) 'message3.fieldMask.paths': result.message3?.fieldMask?.paths!",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestBuildQueryLines (-want, +got)\n:%s", diff)
	}
}
