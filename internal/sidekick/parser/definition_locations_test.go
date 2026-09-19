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

package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestExtractDefinitionLocations_FromProto(t *testing.T) {
	requireProtoc(t)

	t.Run("comments.proto symbols", func(t *testing.T) {
		req := newTestCodeGeneratorRequest(t, "comments.proto")
		model, err := makeAPIForProtobuf(nil, req)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range []struct {
			symbol string
			want   api.SourceLocation
		}{
			{symbol: ".test.Request", want: api.SourceLocation{Filename: "comments.proto", Line: 31}},
			{symbol: ".test.Request.parent", want: api.SourceLocation{Filename: "comments.proto", Line: 35}},
			{symbol: ".test.Response", want: api.SourceLocation{Filename: "comments.proto", Line: 39}},
			{symbol: ".test.Response.name", want: api.SourceLocation{Filename: "comments.proto", Line: 41}},
			{symbol: ".test.Response.Status", want: api.SourceLocation{Filename: "comments.proto", Line: 47}},
			{symbol: ".test.Response.Status.NOT_READY", want: api.SourceLocation{Filename: "comments.proto", Line: 52}},
			{symbol: ".test.Response.Status.READY", want: api.SourceLocation{Filename: "comments.proto", Line: 54}},
			{symbol: ".test.Response.Nested", want: api.SourceLocation{Filename: "comments.proto", Line: 61}},
			{symbol: ".test.Response.Nested.path", want: api.SourceLocation{Filename: "comments.proto", Line: 68}},
			{symbol: ".test.Service", want: api.SourceLocation{Filename: "comments.proto", Line: 75}},
			{symbol: ".test.Service.Create", want: api.SourceLocation{Filename: "comments.proto", Line: 83}},
		} {
			got, ok := model.DefinitionLocation(test.symbol)
			if !ok {
				t.Errorf("missing definition location for %q", test.symbol)
				continue
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("location mismatch for %q (-want +got):\n%s", test.symbol, diff)
			}
		}
	})

	t.Run("enum.proto symbols", func(t *testing.T) {
		req := newTestCodeGeneratorRequest(t, "enum.proto")
		model, err := makeAPIForProtobuf(nil, req)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range []struct {
			symbol string
			want   api.SourceLocation
		}{
			{symbol: ".test.Code", want: api.SourceLocation{Filename: "enum.proto", Line: 19}},
			{symbol: ".test.Code.OK", want: api.SourceLocation{Filename: "enum.proto", Line: 21}},
			{symbol: ".test.Code.UNKNOWN", want: api.SourceLocation{Filename: "enum.proto", Line: 24}},
		} {
			got, ok := model.DefinitionLocation(test.symbol)
			if !ok {
				t.Errorf("missing definition location for %q", test.symbol)
				continue
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("location mismatch for %q (-want +got):\n%s", test.symbol, diff)
			}
		}
	})

	t.Run("test_service.proto symbols", func(t *testing.T) {
		req := newTestCodeGeneratorRequest(t, "test_service.proto")
		model, err := makeAPIForProtobuf(nil, req)
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range []struct {
			symbol string
			want   api.SourceLocation
		}{
			{symbol: ".test.TestService", want: api.SourceLocation{Filename: "test_service.proto", Line: 25}},
			{symbol: ".test.TestService.GetFoo", want: api.SourceLocation{Filename: "test_service.proto", Line: 31}},
			{symbol: ".test.TestService.CreateFoo", want: api.SourceLocation{Filename: "test_service.proto", Line: 39}},
			{symbol: ".test.TestService.DeleteFoo", want: api.SourceLocation{Filename: "test_service.proto", Line: 48}},
			{symbol: ".test.Foo", want: api.SourceLocation{Filename: "test_service.proto", Line: 69}},
			{symbol: ".test.Foo.name", want: api.SourceLocation{Filename: "test_service.proto", Line: 77}},
			{symbol: ".test.Foo.content", want: api.SourceLocation{Filename: "test_service.proto", Line: 80}},
		} {
			got, ok := model.DefinitionLocation(test.symbol)
			if !ok {
				t.Errorf("missing definition location for %q", test.symbol)
				continue
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("location mismatch for %q (-want +got):\n%s", test.symbol, diff)
			}
		}
	})
}

func TestExtractDefinitionLocations_ExtensionsAndEdgeCases(t *testing.T) {
	t.Run("file and message extensions", func(t *testing.T) {
		fileDesc := &descriptorpb.FileDescriptorProto{
			Name:    new("ext_test.proto"),
			Package: new("test.ext"),
			Extension: []*descriptorpb.FieldDescriptorProto{
				{Name: new("file_level_ext")},
			},
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: new("Container"),
					Extension: []*descriptorpb.FieldDescriptorProto{
						{Name: new("msg_level_ext")},
					},
				},
			},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{
				Location: []*descriptorpb.SourceCodeInfo_Location{
					{
						Path: []int32{fileDescriptorExtension, 0},
						Span: []int32{9, 0, 10},
					},
					{
						Path: []int32{fileDescriptorMessageType, 0, messageDescriptorExtension, 0},
						Span: []int32{19, 2, 20},
					},
				},
			},
		}

		model := api.NewTestAPI(nil, nil, nil)
		extractDefinitionLocations(model, fileDesc)

		for _, test := range []struct {
			name   string
			symbol string
			want   api.SourceLocation
		}{
			{
				name:   "file-level extension",
				symbol: ".test.ext.file_level_ext",
				want:   api.SourceLocation{Filename: "ext_test.proto", Line: 10},
			},
			{
				name:   "message-level extension",
				symbol: ".test.ext.Container.msg_level_ext",
				want:   api.SourceLocation{Filename: "ext_test.proto", Line: 20},
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				got, ok := model.DefinitionLocation(test.symbol)
				if !ok {
					t.Fatalf("missing definition location for %s", test.symbol)
				}
				if diff := cmp.Diff(test.want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("nil and empty edge cases", func(t *testing.T) {
		model := api.NewTestAPI(nil, nil, nil)
		extractDefinitionLocations(model, nil)
		extractDefinitionLocations(model, &descriptorpb.FileDescriptorProto{})
		extractDefinitionLocations(model, &descriptorpb.FileDescriptorProto{
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{
				Location: []*descriptorpb.SourceCodeInfo_Location{
					{Path: []int32{}},
					{Path: []int32{1}},
					{Path: []int32{fileDescriptorMessageType, 999}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorEnumType, 999}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorService, 999}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorExtension, 999}, Span: []int32{1, 0, 2}},
				},
			},
		})
		if len(model.DefinitionLocations) != 0 {
			t.Errorf("expected 0 locations recorded for empty/invalid inputs, got %d", len(model.DefinitionLocations))
		}
	})

	t.Run("source_file_descriptors takes precedence over stripped proto_file", func(t *testing.T) {
		stripped := &descriptorpb.FileDescriptorProto{
			Name:    new("stripped.proto"),
			Package: new("test.stripped"),
			MessageType: []*descriptorpb.DescriptorProto{
				{Name: new("StrippedMessage")},
			},
		}
		withSourceInfo := &descriptorpb.FileDescriptorProto{
			Name:    new("stripped.proto"),
			Package: new("test.stripped"),
			MessageType: []*descriptorpb.DescriptorProto{
				{Name: new("StrippedMessage")},
			},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{
				Location: []*descriptorpb.SourceCodeInfo_Location{
					{
						Path: []int32{fileDescriptorMessageType, 0},
						Span: []int32{41, 0, 42},
					},
				},
			},
		}
		req := &pluginpb.CodeGeneratorRequest{
			ProtoFile:             []*descriptorpb.FileDescriptorProto{stripped},
			SourceFileDescriptors: []*descriptorpb.FileDescriptorProto{withSourceInfo},
		}
		model, err := makeAPIForProtobuf(nil, req)
		if err != nil {
			t.Fatal(err)
		}
		gotLoc, ok := model.DefinitionLocation(".test.stripped.StrippedMessage")
		if !ok {
			t.Fatal("expected to find .test.stripped.StrippedMessage location from SourceFileDescriptors")
		}
		wantLoc := api.SourceLocation{Filename: "stripped.proto", Line: 42}
		if diff := cmp.Diff(wantLoc, gotLoc); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("negative indices do not panic", func(t *testing.T) {
		model := api.NewTestAPI(nil, nil, nil)
		fileDesc := &descriptorpb.FileDescriptorProto{
			Name:    new("negative.proto"),
			Package: new("test.negative"),
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: new("NegativeMsg"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{Name: new("field")},
					},
					NestedType: []*descriptorpb.DescriptorProto{
						{Name: new("NestedMsg")},
					},
					EnumType: []*descriptorpb.EnumDescriptorProto{
						{Name: new("NestedEnum")},
					},
					Extension: []*descriptorpb.FieldDescriptorProto{
						{Name: new("nested_ext")},
					},
				},
			},
			EnumType: []*descriptorpb.EnumDescriptorProto{
				{
					Name: new("NegativeEnum"),
					Value: []*descriptorpb.EnumValueDescriptorProto{
						{Name: new("V0")},
					},
				},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{
				{
					Name: new("NegativeSvc"),
					Method: []*descriptorpb.MethodDescriptorProto{
						{Name: new("M0")},
					},
				},
			},
			Extension: []*descriptorpb.FieldDescriptorProto{
				{Name: new("file_ext")},
			},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{
				Location: []*descriptorpb.SourceCodeInfo_Location{
					{Path: []int32{fileDescriptorMessageType, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorMessageType, 0, messageDescriptorField, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorMessageType, 0, messageDescriptorNestedType, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorMessageType, 0, messageDescriptorEnum, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorMessageType, 0, messageDescriptorExtension, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorEnumType, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorEnumType, 0, enumDescriptorValue, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorService, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorService, 0, serviceDescriptorProtoMethod, -1}, Span: []int32{1, 0, 2}},
					{Path: []int32{fileDescriptorExtension, -1}, Span: []int32{1, 0, 2}},
				},
			},
		}
		extractDefinitionLocations(model, fileDesc)
		if len(model.DefinitionLocations) != 0 {
			t.Errorf("expected 0 locations recorded for negative indices, got %d", len(model.DefinitionLocations))
		}
	})

	t.Run("deeply nested message extensions", func(t *testing.T) {
		fileDesc := &descriptorpb.FileDescriptorProto{
			Name:    new("deep_ext.proto"),
			Package: new("test.deep"),
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: new("Outer"),
					NestedType: []*descriptorpb.DescriptorProto{
						{
							Name: new("Middle"),
							NestedType: []*descriptorpb.DescriptorProto{
								{
									Name: new("Inner"),
									Extension: []*descriptorpb.FieldDescriptorProto{
										{Name: new("deep_ext")},
									},
								},
							},
						},
					},
				},
			},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{
				Location: []*descriptorpb.SourceCodeInfo_Location{
					{
						// Outer (0) -> Middle (0) -> Inner (0) -> deep_ext (0)
						Path: []int32{fileDescriptorMessageType, 0, messageDescriptorNestedType, 0, messageDescriptorNestedType, 0, messageDescriptorExtension, 0},
						Span: []int32{99, 4, 100},
					},
				},
			},
		}
		model := api.NewTestAPI(nil, nil, nil)
		extractDefinitionLocations(model, fileDesc)
		wantLoc := api.SourceLocation{Filename: "deep_ext.proto", Line: 100}
		gotLoc, ok := model.DefinitionLocation(".test.deep.Outer.Middle.Inner.deep_ext")
		if !ok {
			t.Fatal("missing definition location for deeply nested extension")
		}
		if diff := cmp.Diff(wantLoc, gotLoc); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}
