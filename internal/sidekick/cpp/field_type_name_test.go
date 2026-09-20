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

package cpp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestCppParamName(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "keyword delete",
			input: "delete",
			want:  "delete_",
		},
		{
			name:  "keyword export",
			input: "export",
			want:  "export_",
		},
		{
			name:  "keyword class",
			input: "class",
			want:  "class_",
		},
		{
			name:  "keyword template",
			input: "template",
			want:  "template_",
		},
		{
			name:  "keyword namespace",
			input: "namespace",
			want:  "namespace_",
		},
		{
			name:  "keyword default",
			input: "default",
			want:  "default_",
		},
		{
			name:  "keyword friend",
			input: "friend",
			want:  "friend_",
		},
		{
			name:  "keyword operator",
			input: "operator",
			want:  "operator_",
		},
		{
			name:  "regular parameter name",
			input: "regular_name",
			want:  "regular_name",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := CppParamName(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtoNameToCppName(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "leading dot",
			input: ".google.cloud.secretmanager.v1.Secret",
			want:  "google::cloud::secretmanager::v1::Secret",
		},
		{
			name:  "no leading dot",
			input: "google.protobuf.Empty",
			want:  "google::protobuf::Empty",
		},
		{
			name:  "simple identifier",
			input: "Simple",
			want:  "Simple",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ProtoNameToCppName(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCppTypeToString(t *testing.T) {
	mapEntryMsg := api.NewTestMessage("MapEntry").WithFields(
		api.NewTestField("key").WithType(api.TypezString),
		api.NewTestField("value").WithType(api.TypezInt32),
	)

	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name:  "nil field",
			field: nil,
			want:  "",
		},
		{
			name:  "int32",
			field: api.NewTestField("foo").WithType(api.TypezInt32),
			want:  "std::int32_t",
		},
		{
			name:  "int64",
			field: api.NewTestField("foo").WithType(api.TypezInt64),
			want:  "std::int64_t",
		},
		{
			name:  "uint32",
			field: api.NewTestField("foo").WithType(api.TypezUint32),
			want:  "std::uint32_t",
		},
		{
			name:  "uint64",
			field: api.NewTestField("foo").WithType(api.TypezUint64),
			want:  "std::uint64_t",
		},
		{
			name:  "double",
			field: api.NewTestField("foo").WithType(api.TypezDouble),
			want:  "double",
		},
		{
			name:  "float",
			field: api.NewTestField("foo").WithType(api.TypezFloat),
			want:  "float",
		},
		{
			name:  "bool",
			field: api.NewTestField("foo").WithType(api.TypezBool),
			want:  "bool",
		},
		{
			name:  "string",
			field: api.NewTestField("foo").WithType(api.TypezString),
			want:  "std::string",
		},
		{
			name:  "bytes",
			field: api.NewTestField("foo").WithType(api.TypezBytes),
			want:  "std::string",
		},
		{
			name:  "message",
			field: api.NewTestField("foo").WithMessageType(api.NewTestMessage("FooBar").WithPackage("test.pkg")),
			want:  "test::pkg::FooBar",
		},
		{
			name:  "repeated string",
			field: api.NewTestField("foo").WithType(api.TypezString).WithRepeated(),
			want:  "std::vector<std::string>",
		},
		{
			name:  "repeated message",
			field: api.NewTestField("foo").WithMessageType(api.NewTestMessage("Item").WithPackage("test")).WithRepeated(),
			want:  "std::vector<test::Item>",
		},
		{
			name:  "map",
			field: api.NewTestField("attributes").WithMap().WithMessageType(mapEntryMsg),
			want:  "std::map<std::string, std::int32_t>",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := CppTypeToString(test.field)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCppParamTypeToString(t *testing.T) {
	for _, test := range []struct {
		name  string
		field *api.Field
		want  string
	}{
		{
			name:  "nil field",
			field: nil,
			want:  "",
		},
		{
			name:  "int32 value",
			field: api.NewTestField("foo").WithType(api.TypezInt32),
			want:  "std::int32_t",
		},
		{
			name:  "bool value",
			field: api.NewTestField("foo").WithType(api.TypezBool),
			want:  "bool",
		},
		{
			name:  "string const reference",
			field: api.NewTestField("foo").WithType(api.TypezString),
			want:  "std::string const&",
		},
		{
			name:  "bytes const reference",
			field: api.NewTestField("foo").WithType(api.TypezBytes),
			want:  "std::string const&",
		},
		{
			name:  "message const reference",
			field: api.NewTestField("foo").WithMessageType(api.NewTestMessage("Item").WithPackage("test")),
			want:  "test::Item const&",
		},
		{
			name:  "repeated const reference",
			field: api.NewTestField("foo").WithType(api.TypezInt32).WithRepeated(),
			want:  "std::vector<std::int32_t> const&",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := CppParamTypeToString(test.field)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
