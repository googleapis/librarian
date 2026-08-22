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

package java

import (
	"strings"
	"testing"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestGenerateGapicEndToEnd(t *testing.T) {
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

	resp, err := GenerateGapic(req)
	if err != nil {
		t.Fatalf("GenerateGapic failed: %v", err)
	}

	if resp == nil || len(resp.GetFile()) == 0 {
		t.Fatalf("expected non-empty response files")
	}

	fileMap := make(map[string]string)
	for _, f := range resp.GetFile() {
		fileMap[f.GetName()] = f.GetContent()
	}

	expectedFiles := []string{
		"com/google/example/v1/EchoClient.java",
		"com/google/example/v1/EchoSettings.java",
		"com/google/example/v1/stub/EchoStub.java",
		"com/google/example/v1/stub/EchoStubSettings.java",
		"com/google/example/v1/stub/GrpcEchoStub.java",
		"com/google/example/v1/stub/GrpcEchoCallableFactory.java",
		"com/google/example/v1/stub/HttpJsonEchoStub.java",
		"com/google/example/v1/stub/HttpJsonEchoCallableFactory.java",
		"com/google/example/v1/package-info.java",
		"com/google/example/v1/Version.java",
		"gapic_metadata.json",
	}

	for _, exp := range expectedFiles {
		content, ok := fileMap[exp]
		if !ok {
			t.Errorf("missing expected file: %s", exp)
			continue
		}
		if len(content) == 0 {
			t.Errorf("file %s is empty", exp)
		}
	}

	// Verify content of EchoClient.java
	clientCode := fileMap["com/google/example/v1/EchoClient.java"]
	if !strings.Contains(clientCode, "public class EchoClient implements BackgroundResource {") {
		t.Errorf("client code missing class definition, got:\n%s", clientCode)
	}
	if !strings.Contains(clientCode, "public final EchoResponse echo(EchoRequest request)") {
		t.Errorf("client code missing echo(EchoRequest) method, got:\n%s", clientCode)
	}
	if !strings.Contains(clientCode, "public final EchoResponse echo(String content)") {
		t.Errorf("client code missing echo(String content) signature overload, got:\n%s", clientCode)
	}
}
