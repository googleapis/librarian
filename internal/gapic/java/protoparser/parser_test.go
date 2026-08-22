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

package protoparser

import (
	"testing"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestParserWithCodeGeneratorRequest(t *testing.T) {
	httpRule := &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Post{
			Post: "/v1/echo:echo",
		},
		Body: "*",
	}

	methodOpts := &descriptorpb.MethodOptions{}
	proto.SetExtension(methodOpts, annotations.E_Http, httpRule)
	proto.SetExtension(methodOpts, annotations.E_MethodSignature, []string{"content"})

	svcOpts := &descriptorpb.ServiceOptions{}
	proto.SetExtension(svcOpts, annotations.E_DefaultHost, "echo.googleapis.com")

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"google/example/v1/echo.proto"},
		Parameter:      new("transport=grpc+rest,metadata,generate-version-java"),
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    new("google/example/v1/echo.proto"),
				Package: new("google.example.v1"),
				Options: &descriptorpb.FileOptions{
					JavaPackage: new("com.google.example.v1"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("EchoRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:  new("content"),
								Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							},
						},
					},
					{
						Name: new("EchoResponse"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:  new("content"),
								Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							},
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name:    new("Echo"),
						Options: svcOpts,
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:       new("Echo"),
								InputType:  new(".google.example.v1.EchoRequest"),
								OutputType: new(".google.example.v1.EchoResponse"),
								Options:    methodOpts,
							},
						},
					},
				},
			},
		},
	}

	ctx, err := Parse(req)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(ctx.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(ctx.Services))
	}

	svc := ctx.Services[0]
	if svc.Name != "Echo" {
		t.Errorf("expected service name Echo, got %s", svc.Name)
	}
	if svc.HostName != "echo.googleapis.com" {
		t.Errorf("expected host echo.googleapis.com, got %s", svc.HostName)
	}
	if len(svc.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(svc.Methods))
	}

	m := svc.Methods[0]
	if m.Name != "Echo" {
		t.Errorf("expected method Echo, got %s", m.Name)
	}
	if m.HttpBindings == nil || m.HttpBindings.HttpMethod != "POST" {
		t.Errorf("expected POST http binding, got %+v", m.HttpBindings)
	}
	if len(m.MethodSignatures) != 1 || m.MethodSignatures[0][0] != "content" {
		t.Errorf("expected method signature [content], got %+v", m.MethodSignatures)
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestParserWithLROStreamingAndPagination(t *testing.T) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"google/example/v1/library.proto"},
		Parameter:      new("transport=grpc+rest,metadata"),
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    new("google/example/v1/library.proto"),
				Package: new("google.example.v1"),
				Options: &descriptorpb.FileOptions{
					JavaPackage: new("com.google.example.v1"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("ListShelvesRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:  new("page_size"),
								Type:  descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
								Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							},
							{
								Name:  new("page_token"),
								Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							},
						},
					},
					{
						Name: new("ListShelvesResponse"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:  new("next_page_token"),
								Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							},
							{
								Name:  new("shelves"),
								Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
							},
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: new("Library"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:            new("StreamShelves"),
								InputType:       new(".google.example.v1.ListShelvesRequest"),
								OutputType:      new(".google.example.v1.ListShelvesResponse"),
								ClientStreaming: new(true),
								ServerStreaming: new(true),
							},
							{
								Name:       new("ListShelves"),
								InputType:  new(".google.example.v1.ListShelvesRequest"),
								OutputType: new(".google.example.v1.ListShelvesResponse"),
							},
						},
					},
				},
			},
		},
	}

	ctx, err := Parse(req)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(ctx.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(ctx.Services))
	}

	svc := ctx.Services[0]
	if !svc.HasStreaming {
		t.Errorf("expected service to have streaming methods")
	}

	for _, m := range svc.Methods {
		if m.Name == "StreamShelves" {
			if m.StreamType != 3 /* StreamBidi */ {
				t.Errorf("expected bidi streaming, got %d", m.StreamType)
			}
		}
		if m.Name == "ListShelves" {
			if !m.IsPaged {
				t.Errorf("expected ListShelves to be recognized as paged")
			}
			if m.PageSizeField != "page_size" || m.PageTokenField != "page_token" || m.NextPageTokenField != "next_page_token" || m.ResourceListField != "shelves" {
				t.Errorf("unexpected pagination fields: %+v", m)
			}
		}
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestParserWithImportedTypes(t *testing.T) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"google/example/v1/event.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    new("google/example/v1/event.proto"),
				Package: new("google.example.v1"),
				Options: &descriptorpb.FileOptions{
					JavaPackage: new("com.google.example.v1"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("Event"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     new("timestamp"),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
								TypeName: new(".google.protobuf.Timestamp"),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							},
						},
					},
				},
			},
		},
	}

	ctx, err := Parse(req)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	msg := ctx.FindMessage("google.example.v1.Event")
	if msg == nil {
		t.Fatalf("message Event not found")
	}
	f := msg.Fields["timestamp"]
	if f == nil {
		t.Fatalf("field timestamp not found")
	}
	if f.Type.Package != "com.google.protobuf" || f.Type.Name != "Timestamp" {
		t.Errorf("expected package com.google.protobuf and type Timestamp, got package=%s name=%s", f.Type.Package, f.Type.Name)
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestParserInvalidConfigPaths(t *testing.T) {
	req := &pluginpb.CodeGeneratorRequest{
		Parameter: new("grpc-service-config=/nonexistent/path.json"),
	}
	if _, err := Parse(req); err == nil {
		t.Errorf("expected error for non-existent grpc-service-config")
	}

	reqYaml := &pluginpb.CodeGeneratorRequest{
		Parameter: new("gapic-config=/nonexistent/path.yaml"),
	}
	if _, err := Parse(reqYaml); err == nil {
		t.Errorf("expected error for non-existent gapic-config")
	}
}
