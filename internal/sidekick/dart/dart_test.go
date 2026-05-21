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
	f := &api.Message{
		Name: "Function",
		ID:   ".google.cloud.functions.v2.Function",
	}
	model := api.NewTestAPI(
		[]*api.Message{r, a, f, sample.CustomerManagedEncryption()},
		[]*api.Enum{},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	for _, test := range []struct {
		message *api.Message
		want    string
	}{
		{message: r, want: "Replication"},
		{message: a, want: "Replication_Automatic"},
		{message: f, want: "Function$"},
		{message: sample.SecretPayload(), want: "SecretPayload"},
	} {
		t.Run(test.want, func(t *testing.T) {
			if got := messageName(test.message); got != test.want {
				t.Errorf("mismatched message name, got=%q, want=%q", got, test.want)
			}
		})
	}
}

func TestEnumNames(t *testing.T) {
	parent := &api.Message{
		Name:    "SecretVersion",
		ID:      sample.SecretVersion().ID,
		Package: "test",
		Fields: []*api.Field{
			{
				Name:     "automatic",
				Typez:    api.TypezMessage,
				TypezID:  sample.Automatic().ID,
				Optional: true,
				Repeated: false,
			},
		},
	}
	nested := &api.Enum{
		Name:    "State",
		ID:      ".test.SecretVersion.State",
		Parent:  parent,
		Package: "test",
	}
	non_nested := &api.Enum{
		Name:    "Code",
		ID:      ".test.Code",
		Package: "test",
	}

	model := api.NewTestAPI(
		[]*api.Message{parent, sample.Automatic(), sample.CustomerManagedEncryption()},
		[]*api.Enum{nested, non_nested},
		[]*api.Service{})
	model.PackageName = "test"
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	for _, test := range []struct {
		enum     *api.Enum
		wantEnum string
	}{
		{non_nested, "Code"},
		{nested, "SecretVersion_State"},
	} {
		if got := enumName(test.enum); got != test.wantEnum {
			t.Errorf("c.enumName(%q) = %q; want = %s", test.enum.Name, got, test.wantEnum)
		}
	}
}

func TestResolveMessageName(t *testing.T) {
	message := sample.CreateRequest()
	model := api.NewTestAPI([]*api.Message{
		message, {
			ID:   ".google.protobuf.Duration",
			Name: "Duration",
		}, {
			ID:   ".google.protobuf.Empty",
			Name: "Empty",
		}, {
			ID:   ".google.protobuf.Timestamp",
			Name: "Timestamp",
		},
	}, []*api.Enum{}, []*api.Service{})

	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	for _, test := range []struct {
		typeId string
		want   string
	}{
		{message.ID, "CreateSecretRequest"},
		{".google.protobuf.Empty", "void"},
		{".google.protobuf.Timestamp", "Timestamp"},
		{".google.protobuf.Duration", "Duration"},
	} {
		got := annotate.resolveMessageName(model.Message(test.typeId), true)
		if got != test.want {
			t.Errorf("unexpected type name, got: %s want: %s", got, test.want)
		}
	}
}

func TestResolveMessageName_ImportsMessages(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{
		{
			ID:      ".google.protobuf.Any",
			Package: "google.protobuf",
		}, {
			ID:      ".google.rpc.Status",
			Package: "google.rpc",
		}, {
			ID:      ".google.type.Expr",
			Package: "google.type",
		},
	}, []*api.Enum{}, []*api.Service{})

	// We use an explicit package name here; NewTestAPI will otherwise default to
	// 'google.type' and we won't be able to test that package name below.
	model.PackageName = "google.sample"

	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	annotate.packageMapping = map[string]string{
		"google.protobuf": "package:google_cloud_protobuf/protobuf.dart",
		"google.rpc":      "package:google_cloud_rpc/rpc.dart",
		"google.type":     "package:google_cloud_type/type.dart",
	}

	for _, test := range []struct {
		typeId string
		want   string
	}{
		{".google.protobuf.Any", "package:google_cloud_protobuf/protobuf.dart"},
		{".google.rpc.Status", "package:google_cloud_rpc/rpc.dart"},
		{".google.type.Expr", "package:google_cloud_type/type.dart"},
	} {
		annotate.imports = map[string]bool{}
		annotate.resolveMessageName(model.Message(test.typeId), true)
		if _, ok := annotate.imports[test.want]; !ok {
			t.Errorf("import not added, got: %v want: %s", annotate.imports, test.want)
		}
	}
}

func TestFieldType_EnumImports(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{
		{
			ID:      ".google.type.DayOfWeek",
			Package: "google.type",
		},
	}, []*api.Service{})

	// We use an explicit package name here; NewTestAPI will otherwise default to
	// 'google.type' and we won't be able to test that package name below.
	model.PackageName = "google.sample"

	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	annotate.packageMapping = map[string]string{
		"google.type": "package:google_cloud_type/type.dart",
	}

	field := &api.Field{
		Name:    "testField",
		Typez:   api.TypezEnum,
		TypezID: ".google.type.DayOfWeek",
	}
	annotate.imports = map[string]bool{}
	annotate.fieldType(field)
	want := "package:google_cloud_type/type.dart"
	if _, ok := annotate.imports[want]; !ok {
		t.Errorf("import not added, got: %v want: %s", annotate.imports, want)
	}
}

func TestResolveMessageNameImportPrefixes(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{
		{
			ID:      ".google.protobuf.Timestamp",
			Name:    "Timestamp",
			Package: "google.protobuf",
		}, {
			ID:      ".google.protobuf.Duration",
			Name:    "Duration",
			Package: "google.protobuf",
		}, {
			ID:      ".google.rpc.Status",
			Name:    "Status",
			Package: "google.rpc",
		}, {
			ID:      ".google.type.DayOfWeek",
			Name:    "DayOfWeek",
			Package: "google.type",
		},
	}, []*api.Enum{}, []*api.Service{})

	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{
		"prefix:google.protobuf": "protobuf",
		"prefix:google.type":     "type",
	})

	for _, test := range []struct {
		typeId string
		want   string
	}{
		{".google.rpc.Status", "Status"},
		{".google.protobuf.Timestamp", "protobuf.Timestamp"},
		{".google.protobuf.Duration", "protobuf.Duration"},
		{".google.type.DayOfWeek", "type.DayOfWeek"},
	} {
		t.Run(test.want, func(t *testing.T) {
			got := annotate.resolveMessageName(model.Message(test.typeId), true)
			if got != test.want {
				t.Errorf("unexpected type name, got: %s want: %s", got, test.want)
			}
		})
	}
}

func TestFieldType(t *testing.T) {
	// Test simple fields.
	for _, test := range []struct {
		typez api.Typez
		want  string
	}{
		{api.TypezBool, "bool"},
		{api.TypezInt32, "int"},
		{api.TypezUint32, "int"},
		{api.TypezFixed32, "int"},
		{api.TypezSfixed32, "int"},
		{api.TypezInt64, "int"},
		{api.TypezUint64, "BigInt"},
		{api.TypezFixed64, "BigInt"},
		{api.TypezSfixed64, "int"},
		{api.TypezFloat, "double"},
		{api.TypezDouble, "double"},
		{api.TypezString, "String"},
		{api.TypezBytes, "Uint8List"},
	} {
		field := &api.Field{
			Name:     "parent",
			JSONName: "parent",
			Typez:    test.typez,
		}
		message := &api.Message{
			Name:          "UpdateSecretRequest",
			ID:            "..UpdateRequest",
			Documentation: "Request message for SecretManagerService.UpdateSecret",
			Package:       sample.Package,
			Fields:        []*api.Field{field},
		}
		model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
		annotate := newAnnotateModel(model)
		annotate.annotateModel(map[string]string{})

		got := annotate.fieldType(field)
		if got != test.want {
			t.Errorf("unexpected type name, got: %s want: %s", got, test.want)
		}
	}

	// Test message and enum fields.
	sampleMessage := sample.CreateRequest()
	sampleEnum := sample.EnumState()

	field1 := &api.Field{
		Name:     "parent",
		JSONName: "parent",
		Typez:    api.TypezMessage,
		TypezID:  sampleMessage.ID,
	}
	field2 := &api.Field{
		Name:     "parent",
		JSONName: "parent",
		Typez:    api.TypezEnum,
		TypezID:  sampleEnum.ID,
	}
	message := &api.Message{
		Name:          "UpdateSecretRequest",
		ID:            "..UpdateRequest",
		Documentation: "Request message for SecretManagerService.UpdateSecret",
		Package:       sample.Package,
		Fields:        []*api.Field{field1, field2},
	}
	model := api.NewTestAPI(
		[]*api.Message{message, sampleMessage},
		[]*api.Enum{sampleEnum},
		[]*api.Service{},
	)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	got := annotate.fieldType(field1)
	want := "CreateSecretRequest"
	if got != want {
		t.Errorf("unexpected type name, got: %s want: %s", got, want)
	}

	got = annotate.fieldType(field2)
	want = "State"
	if got != want {
		t.Errorf("unexpected type name, got: %s want: %s", got, want)
	}
}

func TestFieldType_Maps(t *testing.T) {
	map1 := &api.Message{
		Name:  "$map<string, string>",
		ID:    "$map<string, string>",
		IsMap: true,
		Fields: []*api.Field{
			{
				Name:  "key",
				Typez: api.TypezString,
			},
			{
				Name:  "value",
				Typez: api.TypezInt32,
			},
		},
	}
	field := &api.Field{
		Name:     "map",
		JSONName: "map",
		Typez:    api.TypezMessage,
		TypezID:  map1.ID,
	}
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.AddMessage(map1)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	got := annotate.fieldType(field)
	want := "Map<String, int>"
	if got != want {
		t.Errorf("unexpected type name, got: %s want: %s", got, want)
	}
}

func TestFieldType_Bytes(t *testing.T) {
	field := &api.Field{
		Name:     "test",
		JSONName: "test",
		Typez:    api.TypezBytes,
	}
	message := &api.Message{
		Name:   "$test",
		ID:     "$test",
		IsMap:  true,
		Fields: []*api.Field{field},
	}
	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})
	annotate.imports = map[string]bool{}

	{
		got := annotate.fieldType(field)
		want := "Uint8List"
		if got != want {
			t.Errorf("unexpected type name, got: %s want: %s", got, want)
		}
	}
}

func TestFieldType_Repeated(t *testing.T) {
	// Test repeated simple fields.
	for _, test := range []struct {
		typez api.Typez
		want  string
	}{
		{api.TypezBool, "List<bool>"},
		{api.TypezInt32, "List<int>"},
		{api.TypezUint32, "List<int>"},
		{api.TypezFixed32, "List<int>"},
		{api.TypezSfixed32, "List<int>"},
		{api.TypezInt64, "List<int>"},
		{api.TypezUint64, "List<BigInt>"},
		{api.TypezFixed64, "List<BigInt>"},
		{api.TypezSfixed64, "List<int>"},
		{api.TypezFloat, "List<double>"},
		{api.TypezDouble, "List<double>"},
		{api.TypezString, "List<String>"},
	} {
		field := &api.Field{
			Name:     "parent",
			JSONName: "parent",
			Typez:    test.typez,
			Repeated: true,
		}
		message := &api.Message{
			Name:          "UpdateSecretRequest",
			ID:            "..UpdateRequest",
			Documentation: "Request message for SecretManagerService.UpdateSecret",
			Package:       sample.Package,
			Fields:        []*api.Field{field},
		}
		model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
		annotate := newAnnotateModel(model)
		annotate.annotateModel(map[string]string{})

		got := annotate.fieldType(field)
		if got != test.want {
			t.Errorf("unexpected type name, got: %s want: %s", got, test.want)
		}
	}

	// Test repeated message and enum fields.
	sampleMessage := sample.CreateRequest()
	sampleEnum := sample.EnumState()

	field1 := &api.Field{
		Name:     "parent",
		JSONName: "parent",
		Typez:    api.TypezMessage,
		TypezID:  sampleMessage.ID,
		Repeated: true,
	}
	field2 := &api.Field{
		Name:     "parent",
		JSONName: "parent",
		Typez:    api.TypezEnum,
		TypezID:  sampleEnum.ID,
		Repeated: true,
	}
	message := &api.Message{
		Name:          "UpdateSecretRequest",
		ID:            "..UpdateRequest",
		Documentation: "Request message for SecretManagerService.UpdateSecret",
		Package:       sample.Package,
		Fields:        []*api.Field{field1, field2},
	}
	model := api.NewTestAPI(
		[]*api.Message{message, sampleMessage},
		[]*api.Enum{sampleEnum},
		[]*api.Service{},
	)
	annotate := newAnnotateModel(model)
	annotate.annotateModel(map[string]string{})

	got := annotate.fieldType(field1)
	want := "List<CreateSecretRequest>"
	if got != want {
		t.Errorf("unexpected type name, got: %s want: %s", got, want)
	}

	got = annotate.fieldType(field2)
	want = "List<State>"
	if got != want {
		t.Errorf("unexpected type name, got: %s want: %s", got, want)
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
	model := api.NewTestAPI(nil, nil, nil)
	got := formatDocComments(input, model)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in FormatDocComments (-want, +got)\n:%s", diff)
	}
}

func TestFormatDocCommentsEmpty(t *testing.T) {
	input := ``

	want := []string{}
	model := api.NewTestAPI(nil, nil, nil)
	got := formatDocComments(input, model)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in FormatDocComments (-want, +got)\n:%s", diff)
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
	model := api.NewTestAPI(nil, nil, nil)
	got := formatDocComments(input, model)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in FormatDocComments (-want, +got)\n:%s", diff)
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
	model := api.NewTestAPI(nil, nil, nil)
	got := formatDocComments(input, model)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in FormatDocComments (-want, +got)\n:%s", diff)
	}
}

func TestFormatDocCommentsRewriteReferences(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{
		{Name: "Secret", ID: "Secret"},
	}, nil, nil)

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
			output:   "/// foo Code ref bar",
		},
		{
			testName: "no match - missing brackets",
			input:    "foo [google.rpc.Code] bar",
			output:   "/// foo `[google.rpc.Code]` bar",
		},
		{
			testName: "array access with dot",
			input:    "foo logging.producer_destinations[0] bar",
			output:   "/// foo `logging.producer_destinations[0]` bar",
		},
		{
			testName: "simple array access",
			input:    "foo grounding_chunk[1] bar",
			output:   "/// foo `grounding_chunk[1]` bar",
		},
		{
			testName: "double backticks cleanup",
			input:    "foo ``Distribution`` bar",
			output:   "/// foo `Distribution` bar",
		},
		{
			testName: "complex path with array access",
			input:    "foo emailAddresses[3].type[2] bar",
			output:   "/// foo `emailAddresses[3].type[2]` bar",
		},
		{
			testName: "complex path ending with field",
			input:    "foo emailAddresses[1].email bar",
			output:   "/// foo `emailAddresses[1].email` bar",
		},
		{
			testName: "quoted URL with brackets",
			input:    `foo "https://[service.name]/[google.protobuf.Api.name]" bar`,
			output:   "/// foo `https://[service.name]/[google.protobuf.Api.name]` bar",
		},
		{
			testName: "non-quoted URL with brackets",
			input:    "foo https://[Service_name]/[API_name] bar",
			output:   "/// foo `https://[Service_name]/[API_name]` bar",
		},
		{
			testName: "URL at start of line with brackets",
			input:    "https://[Service_name]/[API_name]",
			output:   "/// `https://[Service_name]/[API_name]`",
		},
		{
			testName: "URL with reference targets",
			input:    "foo https://[Service_name][google.api.Service.name]/[API_name][google.protobuf.Api.name] bar",
			output:   "/// foo `https://[Service_name]/[API_name]` bar",
		},
		{
			testName: "URL with reference targets at start of line",
			input:    "https://[Service_name][google.api.Service.name]/[API_name][google.protobuf.Api.name]",
			output:   "/// `https://[Service_name]/[API_name]`",
		},
		{
			testName: "Markdown reference syntax example",
			input:    "You can reference [Java][Tutorial.Java] page using Markdown reference link syntax:\n`[Java][Tutorial.Java]`.",
			output:   "/// You can reference `Java` page using Markdown reference link syntax:\n/// `[Java][Tutorial.Java]`.",
		},
		{
			testName: "Code block indentation (3 spaces)",
			input:    "There are bounds:\n\n   Upper bound: bounds[i]\n   Lower bound: bounds[i - 1]\n\nEnd.",
			output:   "/// There are bounds:\n///\n/// ```text\n/// Upper bound: bounds[i]\n/// Lower bound: bounds[i - 1]\n/// ```\n///\n/// End.",
		},
		{
			testName: "Code block indentation mixed spaces",
			input:    "Required. The resource name:\n\n    \"projects/[PROJECT_ID]...\"\n\nFor example:\n\n   \"projects/my-project...\"",
			output:   "/// Required. The resource name:\n///\n/// ```text\n/// \"projects/[PROJECT_ID]...\"\n/// ```\n///\n/// For example:\n///\n/// ```text\n/// \"projects/my-project...\"\n/// ```",
		},
		{
			testName: "List items with 2 spaces should NOT be code block",
			input:    "Example values:\n\n  - `000000000000004a`\n  - `7a2190356c3fc94b`\n\nEnd.",
			output:   "/// Example values:\n///\n///   - `000000000000004a`\n///   - `7a2190356c3fc94b`\n///\n/// End.",
		},
		{
			testName: "Code block relative indentation",
			input:    "There is a block:\n\n    Line 1\n      Line 2\n    Line 3\n\nEnd.",
			output:   "/// There is a block:\n///\n/// ```text\n/// Line 1\n///   Line 2\n/// Line 3\n/// ```\n///\n/// End.",
		},
		{
			testName: "Existing code fence without name",
			input:    "See follows:\n\n```\nparent_contexts:\n\"projects/...\"\n```\nEnd.",
			output:   "/// See follows:\n///\n/// ```text\n/// parent_contexts:\n/// \"projects/...\"\n/// ```\n/// End.",
		},
		{
			testName: "Existing code fence with name",
			input:    "See follows:\n\n```json\n{\n  \"a\": 1\n}\n```\nEnd.",
			output:   "/// See follows:\n///\n/// ```json\n/// {\n///   \"a\": 1\n/// }\n/// ```\n/// End.",
		},
		{
			testName: "Existing code fence indented",
			input:    "See follows:\n\n   ```\n   parent_contexts:\n   \"projects/...\"\n   ```\nEnd.",
			output:   "/// See follows:\n///\n/// ```text\n/// parent_contexts:\n/// \"projects/...\"\n/// ```\n/// End.",
		},
		{
			testName: "double bracket with valid symbol",
			input:    "foo [Label][Secret] bar",
			output:   "/// foo [Label][Secret] bar",
		},
		{
			testName: "double bracket with invalid symbol (phrase)",
			input:    "foo [Some Label][NonExistent] bar",
			output:   "/// foo Some Label bar",
		},
		{
			testName: "double bracket with invalid symbol (symbol)",
			input:    "foo [SomeLabel][NonExistent] bar",
			output:   "/// foo `SomeLabel` bar",
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			gotLines := formatDocComments(test.input, model)
			got := strings.Join(gotLines, "\n")
			if diff := cmp.Diff(test.output, got); diff != "" {
				t.Errorf("mismatch in FormatDocComments (-want, +got)\n:%s", diff)
			}
		})
	}
}

func TestFormatDocCommentsNestedMessageReferences(t *testing.T) {
	parent := &api.Message{Name: "NasJobSpec", ID: ".test.NasJobSpec"}
	msg := &api.Message{Name: "MultiTrialAlgorithmSpec", ID: ".test.NasJobSpec.MultiTrialAlgorithmSpec"}
	nested := &api.Message{Name: "TrainTrialSpec", ID: ".test.NasJobSpec.MultiTrialAlgorithmSpec.TrainTrialSpec"}

	nested.Fields = []*api.Field{
		{Name: "frequency", JSONName: "frequency"},
		{Name: "max_parallel_trial_count", JSONName: "maxParallelTrialCount"},
	}

	model := api.NewTestAPI([]*api.Message{parent, msg, nested}, nil, nil)

	input := "Spec for train trials. Top N [TrainTrialSpec.maxParallelTrialCount]\nsearch trials will be trained for every M\n[TrainTrialSpec.frequency] trials searched."

	want := []string{
		"/// Spec for train trials. Top N [NasJobSpec_MultiTrialAlgorithmSpec_TrainTrialSpec.maxParallelTrialCount]",
		"/// search trials will be trained for every M",
		"/// [NasJobSpec_MultiTrialAlgorithmSpec_TrainTrialSpec.frequency] trials searched.",
	}

	got := formatDocComments(input, model)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestFormatDocCommentsNestedMessageReferences (-want, +got)\n:%s", diff)
	}
}

func TestFormatDocCommentsDuplicateMessageNames(t *testing.T) {
	// Message 1: named "Model" but without the field.
	msg1 := &api.Message{Name: "Model", ID: "msg1"}

	// Message 2: named "Model" with the field!
	msg2 := &api.Message{Name: "Model", ID: "msg2"}
	msg2.Fields = []*api.Field{
		{Name: "supported_export_formats", JSONName: "supportedExportFormats"},
	}

	model := api.NewTestAPI([]*api.Message{msg1, msg2}, nil, nil)

	input := "Check [Model.supported_export_formats] object."

	want := []string{
		"/// Check [Model.supportedExportFormats] object.",
	}

	got := formatDocComments(input, model)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in TestFormatDocCommentsDuplicateMessageNames (-want, +got)\n:%s", diff)
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
