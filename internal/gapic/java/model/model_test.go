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

package model

import (
	"testing"

	"github.com/googleapis/librarian/internal/gapic/java/engine/ast"
)

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestTransportParsing(t *testing.T) {
	if ParseTransport("rest") != TransportREST {
		t.Errorf("expected REST transport")
	}
	if ParseTransport("grpc+rest") != TransportGRPCRest {
		t.Errorf("expected GRPCRest transport")
	}
	if ParseTransport("grpc") != TransportGRPC {
		t.Errorf("expected GRPC transport")
	}
	if ParseTransport("unknown") != TransportGRPC {
		t.Errorf("expected fallback to GRPC")
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestFieldRequired(t *testing.T) {
	fReq := &Field{
		Name:      "parent",
		Type:      ast.TypeString,
		Behaviors: []FieldBehavior{FieldBehaviorRequired},
	}
	if !fReq.IsRequired() {
		t.Errorf("expected field to be required")
	}

	fOpt := &Field{
		Name:      "page_size",
		Type:      ast.TypeInt,
		Behaviors: []FieldBehavior{FieldBehaviorOptional},
	}
	if fOpt.IsRequired() {
		t.Errorf("expected field not to be required")
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestGapicContextLookup(t *testing.T) {
	ctx := &GapicContext{
		Messages: map[string]*Message{
			"google.example.v1.EchoRequest": {
				Name:          "EchoRequest",
				PackageName:   "com.google.example.v1",
				FullProtoName: "google.example.v1.EchoRequest",
				Fields: map[string]*Field{
					"name": {
						Name:              "name",
						Type:              ast.TypeString,
						Behaviors:         []FieldBehavior{FieldBehaviorRequired},
						ResourceReference: &ResourceReference{Type: "example.googleapis.com/Echo"},
					},
				},
			},
		},
		ResourceNames: map[string]*ResourceName{
			"example.googleapis.com/Echo": {
				Type:        "example.googleapis.com/Echo",
				Patterns:    []string{"projects/{project}/echoes/{echo}"},
				PackageName: "com.google.example.v1",
				ClassName:   "EchoName",
			},
		},
		ServiceConfig: &GapicServiceConfig{
			RetryCodes: map[string][]string{
				"idempotent": {"DEADLINE_EXCEEDED", "UNAVAILABLE"},
			},
			RetryParams: map[string]*RetryParam{
				"default": {
					InitialRetryDelayMillis: 100,
					RetryDelayMultiplier:    1.3,
					MaxRetryDelayMillis:     60000,
					InitialRpcTimeoutMillis: 20000,
					RpcTimeoutMultiplier:    1.0,
					MaxRpcTimeoutMillis:     20000,
					TotalTimeoutMillis:      600000,
				},
			},
			MethodConfigs: map[string]*MethodConfig{
				"Echo": {
					RetryPolicyName: "default",
					TimeoutMillis:   60000,
				},
			},
		},
	}

	if msg := ctx.FindMessage("google.example.v1.EchoRequest"); msg == nil || msg.Name != "EchoRequest" {
		t.Errorf("failed to find message: %+v", msg)
	}
	if res := ctx.FindResourceName("example.googleapis.com/Echo"); res == nil || res.ClassName != "EchoName" {
		t.Errorf("failed to find resource: %+v", res)
	}
	if ctx.ServiceConfig == nil || len(ctx.ServiceConfig.RetryCodes["idempotent"]) != 2 {
		t.Errorf("unexpected service config: %+v", ctx.ServiceConfig)
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestMethodAndServiceModels(t *testing.T) {
	method := &Method{
		Name:       "Echo",
		InputType:  ast.ObjectType("EchoRequest", "com.google.example.v1"),
		OutputType: ast.ObjectType("EchoResponse", "com.google.example.v1"),
		StreamType: StreamServer,
		HttpBindings: &HttpBindings{
			HttpMethod:  "POST",
			PathPattern: "/v1/echo:echo",
			Body:        "*",
		},
		RoutingHeaderRule: &RoutingHeaderRule{
			Table: map[string]string{"name": "projects/{project}/echoes/{echo}"},
		},
		MethodSignatures: [][]string{{"name", "content"}},
		Lro: &LongrunningOperation{
			ResponseType: ast.ObjectType("EchoResponse", "com.google.example.v1"),
			MetadataType: ast.ObjectType("EchoMetadata", "com.google.example.v1"),
		},
		IsPaged:            true,
		PageSizeField:      "page_size",
		PageTokenField:     "page_token",
		NextPageTokenField: "next_page_token",
		ResourceListField:  "responses",
	}

	svc := &Service{
		Name:                "Echo",
		PackageName:         "com.google.example.v1",
		OriginalJavaPackage: "com.google.example.v1",
		HostName:            "echo.googleapis.com",
		DefaultPort:         "443",
		ClientDocumentation: "Documentation for Echo service.",
		Methods:             []*Method{method},
		HasLRO:              true,
		HasStreaming:        true,
	}

	if len(svc.Methods) != 1 || !svc.HasLRO || !svc.HasStreaming {
		t.Errorf("unexpected service configuration: %+v", svc)
	}
	if !method.IsPaged || method.StreamType != StreamServer {
		t.Errorf("unexpected method configuration: %+v", method)
	}
}
