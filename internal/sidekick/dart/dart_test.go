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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestMessageNames(t *testing.T) {
	r := sample.Replication()
	a := sample.Automatic()
	f := api.NewTestMessage("Function").WithPackage("google.cloud.functions.v2")

	for _, test := range []struct {
		name    string
		message *api.Message
		want    string
	}{
		{name: "Replication", message: r, want: "Replication"},
		{name: "Automatic", message: a, want: "Replication_Automatic"},
		{name: "Function", message: f, want: "Function$"},
		{name: "SecretPayload", message: sample.SecretPayload(), want: "SecretPayload"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := messageName(test.message); got != test.want {
				t.Errorf("messageName(%v) = %q, want %q", test.message.Name, got, test.want)
			}
		})
	}
}

func TestEnumNames(t *testing.T) {
	nested := api.NewTestEnum("State").WithParent(api.NewTestMessage("SecretVersion"))
	nonNested := api.NewTestEnum("Code")

	for _, test := range []struct {
		name string
		enum *api.Enum
		want string
	}{
		{name: "non-nested", enum: nonNested, want: "Code"},
		{name: "nested", enum: nested, want: "SecretVersion_State"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := enumName(test.enum); got != test.want {
				t.Errorf("enumName(%v) = %q, want %q", test.enum.Name, got, test.want)
			}
		})
	}
}

func TestResolveMessageName(t *testing.T) {
	message := sample.CreateRequest()
	duration := api.NewTestMessage("Duration").WithPackage("google.protobuf")
	empty := api.NewTestMessage("Empty").WithPackage("google.protobuf")
	timestamp := api.NewTestMessage("Timestamp").WithPackage("google.protobuf")
	model := api.NewTestAPI([]*api.Message{
		message,
		duration,
		empty,
		timestamp,
	}, nil, nil)

	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	for _, test := range []struct {
		name    string
		message *api.Message
		want    string
	}{
		{name: "CreateSecretRequest", message: message, want: "CreateSecretRequest"},
		{name: "Empty", message: empty, want: "void"},
		{name: "Timestamp", message: timestamp, want: "Timestamp"},
		{name: "Duration", message: duration, want: "Duration"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := annotate.resolveMessageName(test.message, true)
			if got != test.want {
				t.Errorf("resolveMessageName(%s) = %s, want %s", test.message.Name, got, test.want)
			}
		})
	}
}

func TestResolveMessageName_ImportsMessages(t *testing.T) {
	anyMsg := api.NewTestMessage("Any").WithPackage("google.protobuf")
	statusMsg := api.NewTestMessage("Status").WithPackage("google.rpc")
	exprMsg := api.NewTestMessage("Expr").WithPackage("google.type")
	model := api.NewTestAPI([]*api.Message{
		anyMsg,
		statusMsg,
		exprMsg,
	}, nil, nil).WithPackageName("google.sample")

	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	annotate.packageMapping = map[string]string{
		"google.protobuf": "package:google_cloud_protobuf/protobuf.dart",
		"google.rpc":      "package:google_cloud_rpc/rpc.dart",
		"google.type":     "package:google_cloud_type/type.dart",
	}

	for _, test := range []struct {
		name    string
		message *api.Message
		want    string
	}{
		{name: "Any", message: anyMsg, want: "package:google_cloud_protobuf/protobuf.dart"},
		{name: "Status", message: statusMsg, want: "package:google_cloud_rpc/rpc.dart"},
		{name: "Expr", message: exprMsg, want: "package:google_cloud_type/type.dart"},
	} {
		t.Run(test.name, func(t *testing.T) {
			annotate.imports = map[string]bool{}
			annotate.resolveMessageName(test.message, true)
			if _, ok := annotate.imports[test.want]; !ok {
				t.Errorf("import not added, got: %v want: %s", annotate.imports, test.want)
			}
		})
	}
}

func TestFieldType_EnumImports(t *testing.T) {
	dayOfWeek := api.NewTestEnum("DayOfWeek").WithPackage("google.type")
	model := api.NewTestAPI(nil, []*api.Enum{
		dayOfWeek,
	}, nil).WithPackageName("google.sample")

	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	annotate.packageMapping = map[string]string{
		"google.type": "package:google_cloud_type/type.dart",
	}

	field := api.NewTestField("testField").
		WithType(api.TypezEnum).
		WithTypezID(dayOfWeek.ID)
	annotate.imports = map[string]bool{}
	annotate.fieldType(field)
	want := "package:google_cloud_type/type.dart"
	if _, ok := annotate.imports[want]; !ok {
		t.Errorf("import not added, got: %v want: %s", annotate.imports, want)
	}
}

func TestResolveMessageNameImportPrefixes(t *testing.T) {
	timestamp := api.NewTestMessage("Timestamp").WithPackage("google.protobuf")
	duration := api.NewTestMessage("Duration").WithPackage("google.protobuf")
	status := api.NewTestMessage("Status").WithPackage("google.rpc")
	dayOfWeek := api.NewTestMessage("DayOfWeek").WithPackage("google.type")
	model := api.NewTestAPI([]*api.Message{
		timestamp,
		duration,
		status,
		dayOfWeek,
	}, nil, nil)

	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{
		"prefix:google.protobuf": "protobuf",
		"prefix:google.type":     "type",
	})

	for _, test := range []struct {
		name    string
		message *api.Message
		want    string
	}{
		{name: "Status", message: status, want: "Status"},
		{name: "Timestamp", message: timestamp, want: "protobuf.Timestamp"},
		{name: "Duration", message: duration, want: "protobuf.Duration"},
		{name: "DayOfWeek", message: dayOfWeek, want: "type.DayOfWeek"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := annotate.resolveMessageName(test.message, true)
			if got != test.want {
				t.Errorf("resolveMessageName(%s) = %s, want %s", test.message.Name, got, test.want)
			}
		})
	}
}

func TestFieldType(t *testing.T) {
	// Test simple fields.
	for _, test := range []struct {
		name  string
		typez api.Typez
		want  string
	}{
		{name: "bool", typez: api.TypezBool, want: "bool"},
		{name: "int32", typez: api.TypezInt32, want: "int"},
		{name: "uint32", typez: api.TypezUint32, want: "int"},
		{name: "fixed32", typez: api.TypezFixed32, want: "int"},
		{name: "sfixed32", typez: api.TypezSfixed32, want: "int"},
		{name: "int64", typez: api.TypezInt64, want: "int"},
		{name: "uint64", typez: api.TypezUint64, want: "BigInt"},
		{name: "fixed64", typez: api.TypezFixed64, want: "BigInt"},
		{name: "sfixed64", typez: api.TypezSfixed64, want: "int"},
		{name: "float", typez: api.TypezFloat, want: "double"},
		{name: "double", typez: api.TypezDouble, want: "double"},
		{name: "string", typez: api.TypezString, want: "String"},
		{name: "bytes", typez: api.TypezBytes, want: "Uint8List"},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := api.NewTestField("parent").WithType(test.typez)
			message := api.NewTestMessage("UpdateSecretRequest").
				WithPackage(sample.Package).
				WithID("..UpdateRequest").
				WithDocumentation("Request message for SecretManagerService.UpdateSecret").
				WithFields(field)
			model := api.NewTestAPI([]*api.Message{message}, nil, nil)
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{})

			got := annotate.fieldType(field)
			if got != test.want {
				t.Errorf("fieldType(%v) = %s, want %s", test.typez, got, test.want)
			}
		})
	}

	// Test message and enum fields.
	sampleMessage := sample.CreateRequest()
	sampleEnum := sample.EnumState()

	msgField := api.NewTestField("msgField").
		WithMessageType(sampleMessage)
	enumField := api.NewTestField("enumField").
		WithType(api.TypezEnum).
		WithTypezID(sampleEnum.ID)
	message := api.NewTestMessage("UpdateSecretRequest").
		WithPackage(sample.Package).
		WithID("..UpdateRequest").
		WithDocumentation("Request message for SecretManagerService.UpdateSecret").
		WithFields(msgField, enumField)
	model := api.NewTestAPI(
		[]*api.Message{message, sampleMessage},
		[]*api.Enum{sampleEnum},
		nil,
	)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	got := annotate.fieldType(msgField)
	want := "CreateSecretRequest"
	if got != want {
		t.Errorf("fieldType(%s) = %s, want %s", msgField.Name, got, want)
	}

	got = annotate.fieldType(enumField)
	want = "State"
	if got != want {
		t.Errorf("fieldType(%s) = %s, want %s", enumField.Name, got, want)
	}
}

func TestFieldType_Maps(t *testing.T) {
	mapMsg := api.NewTestMapMessage("$map<string, string>", api.TypezString, api.TypezInt32)
	field := api.NewTestField("map").
		WithMessageType(mapMsg)
	model := api.NewTestAPI([]*api.Message{mapMsg}, nil, nil)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	got := annotate.fieldType(field)
	want := "Map<String, int>"
	if got != want {
		t.Errorf("fieldType(%s) = %s, want %s", field.Name, got, want)
	}
}

func TestFieldType_Bytes(t *testing.T) {
	field := api.NewTestField("test").WithType(api.TypezBytes)
	message := api.NewTestMessage("$test").
		WithID("$test").
		WithIsMap().
		WithFields(field)
	model := api.NewTestAPI([]*api.Message{message}, nil, nil)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	got := annotate.fieldType(field)
	want := "Uint8List"
	if got != want {
		t.Errorf("fieldType(%s) = %s, want %s", field.Name, got, want)
	}
}

func TestFieldType_Repeated(t *testing.T) {
	// Test repeated simple fields.
	for _, test := range []struct {
		name  string
		typez api.Typez
		want  string
	}{
		{name: "bool", typez: api.TypezBool, want: "List<bool>"},
		{name: "int32", typez: api.TypezInt32, want: "List<int>"},
		{name: "uint32", typez: api.TypezUint32, want: "List<int>"},
		{name: "fixed32", typez: api.TypezFixed32, want: "List<int>"},
		{name: "sfixed32", typez: api.TypezSfixed32, want: "List<int>"},
		{name: "int64", typez: api.TypezInt64, want: "List<int>"},
		{name: "uint64", typez: api.TypezUint64, want: "List<BigInt>"},
		{name: "fixed64", typez: api.TypezFixed64, want: "List<BigInt>"},
		{name: "sfixed64", typez: api.TypezSfixed64, want: "List<int>"},
		{name: "float", typez: api.TypezFloat, want: "List<double>"},
		{name: "double", typez: api.TypezDouble, want: "List<double>"},
		{name: "string", typez: api.TypezString, want: "List<String>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := api.NewTestField("parent").
				WithType(test.typez).
				WithRepeated()
			message := api.NewTestMessage("UpdateSecretRequest").
				WithPackage(sample.Package).
				WithID("..UpdateRequest").
				WithDocumentation("Request message for SecretManagerService.UpdateSecret").
				WithFields(field)
			model := api.NewTestAPI([]*api.Message{message}, nil, nil)
			annotate := newAnnotateModel(model)
			annotate.annotateModel(map[string]string{})

			got := annotate.fieldType(field)
			if got != test.want {
				t.Errorf("fieldType(%v) = %s, want %s", test.typez, got, test.want)
			}
		})
	}

	// Test repeated message and enum fields.
	sampleMessage := sample.CreateRequest()
	sampleEnum := sample.EnumState()

	repeatedMsgField := api.NewTestField("msgField").
		WithMessageType(sampleMessage).
		WithRepeated()
	repeatedEnumField := api.NewTestField("enumField").
		WithType(api.TypezEnum).
		WithTypezID(sampleEnum.ID).
		WithRepeated()
	message := api.NewTestMessage("UpdateSecretRequest").
		WithPackage(sample.Package).
		WithID("..UpdateRequest").
		WithDocumentation("Request message for SecretManagerService.UpdateSecret").
		WithFields(repeatedMsgField, repeatedEnumField)
	model := api.NewTestAPI(
		[]*api.Message{message, sampleMessage},
		[]*api.Enum{sampleEnum},
		nil,
	)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	got := annotate.fieldType(repeatedMsgField)
	want := "List<CreateSecretRequest>"
	if got != want {
		t.Errorf("fieldType(%s) = %s, want %s", repeatedMsgField.Name, got, want)
	}

	got = annotate.fieldType(repeatedEnumField)
	want = "List<State>"
	if got != want {
		t.Errorf("fieldType(%s) = %s, want %s", repeatedEnumField.Name, got, want)
	}
}

func TestFormatDocComments(t *testing.T) {
	input := `Some comments describing the thing.

We want to respect whitespace at the beginning, because it important in Markdown:
- A thing
  - A nested thing
- The next thing
`

	want := []string{
		"/// Some comments describing the thing.",
		"///",
		"/// We want to respect whitespace at the beginning, because it important in Markdown:",
		"/// - A thing",
		"///   - A nested thing",
		"/// - The next thing",
	}
	got := formatDocComments(input, nil)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatDocCommentsEmpty(t *testing.T) {
	input := ``

	want := []string{}
	got := formatDocComments(input, nil)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatDocCommentsTrimTrailingSpaces(t *testing.T) {
	input := `The next line contains spaces.

This line has trailing spaces.  `

	want := []string{
		"/// The next line contains spaces.",
		"///",
		"/// This line has trailing spaces.",
	}
	got := formatDocComments(input, nil)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatDocCommentsTrimTrailingEmptyLines(t *testing.T) {
	input := `Lorem ipsum dolor sit amet, consectetur adipiscing elit,
sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.

`

	want := []string{
		"/// Lorem ipsum dolor sit amet, consectetur adipiscing elit,",
		"/// sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
		"/// Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.",
	}
	got := formatDocComments(input, nil)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatDocCommentsRewriteReferences(t *testing.T) {
	for _, test := range []struct {
		testName string
		input    string
		output   string
	}{
		{
			testName: "regular api ref",
			input:    "foo [Code][google.rpc.Code] bar",
			output:   "/// foo `Code` bar",
		},
		{
			testName: "implicit api ref",
			input:    "foo [google.rpc.Code][] bar",
			output:   "/// foo `google.rpc.Code` bar",
		},
		{
			testName: "two on a line",
			input:    "foo [Code][google.rpc.Code] and [AnalyzeSentiment][] bar",
			output:   "/// foo `Code` and `AnalyzeSentiment` bar",
		},
		{
			testName: "multi-line",
			input: `For calls to [AnalyzeSentiment][] or if
[AnnotateTextRequest.Features.extract_document_sentiment][google.cloud.language.v2.AnnotateTextRequest.Features.extract_document_sentiment]
is set to true, this field will contain the sentiment for the sentence.`,
			output: "/// For calls to `AnalyzeSentiment` or if\n" +
				"/// `AnnotateTextRequest.Features.extract_document_sentiment`\n" +
				"/// is set to true, this field will contain the sentiment for the sentence.",
		},
		{
			testName: "no match - spaces",
			input:    "foo [Code ref][google.rpc.Code] bar",
			output:   "/// foo [Code ref][google.rpc.Code] bar",
		},
		{
			testName: "no match - missing brackets",
			input:    "foo [google.rpc.Code] bar",
			output:   "/// foo [google.rpc.Code] bar",
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			gotLines := formatDocComments(test.input, nil)
			got := strings.Join(gotLines, "\n")
			if diff := cmp.Diff(test.output, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHttpPathFmt(t *testing.T) {
	for _, test := range []struct {
		method *api.Method
		want   string
	}{
		{method: sample.MethodCreate(), want: "/v1/projects/${request.project}/secrets"},
		{method: sample.MethodUpdate(), want: "/v1/${request.secret!.name}"},
		{method: sample.MethodAddSecretVersion(), want: "/v1/projects/${request.project}/secrets/${request.secret}:addVersion"},
		{method: sample.MethodListSecretVersions(), want: "/v1/projects/${request.parent}/secrets/${request.secret}:listSecretVersions"},
	} {
		t.Run(test.method.Name, func(t *testing.T) {
			if got := httpPathFmt(test.method.PathInfo); got != test.want {
				t.Errorf("unexpected httpPathFmt, got=%q, want=%q", got, test.want)
			}
		})
	}
}
