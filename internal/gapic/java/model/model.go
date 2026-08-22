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

// Package model defines data structures for GAPIC generator representation.
package model

import (
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/gapic/java/engine/ast"
)

// Transport defines the communication transport protocol for the GAPIC client.
type Transport string

const (
	// TransportGRPC indicates gRPC transport.
	TransportGRPC Transport = "grpc"
	// TransportREST indicates REST (HTTP/JSON) transport.
	TransportREST Transport = "rest"
	// TransportGRPCRest indicates hybrid gRPC + REST transport.
	TransportGRPCRest Transport = "grpc+rest"
)

// ParseTransport parses a string into a Transport value.
func ParseTransport(s string) Transport {
	switch strings.ToLower(s) {
	case "rest":
		return TransportREST
	case "grpc+rest", "rest+grpc":
		return TransportGRPCRest
	default:
		return TransportGRPC
	}
}

// GapicClassKind categorizes generated Java source files.
type GapicClassKind int

const (
	// KindMain indicates a main GAPIC client class.
	KindMain GapicClassKind = iota
	// KindStub indicates a client stub or transport channel class.
	KindStub
	// KindTest indicates a unit test class.
	KindTest
	// KindMock indicates a mock service class.
	KindMock
)

// GapicClass represents a generated Java class.
type GapicClass struct {
	Kind            GapicClassKind
	ClassDefinition *ast.ClassDefinition
}

// GapicPackageInfo represents package-info.java generation info.
type GapicPackageInfo struct {
	PackageName string
	Description string
	Annotations []*ast.AnnotationNode
}

// ReflectConfig represents GraalVM reflection configuration.
type ReflectConfig struct {
	Name      string            `json:"name"`
	Condition *ReflectCondition `json:"condition,omitempty"`
	Fields    []*ReflectField   `json:"fields,omitempty"`
	Methods   []*ReflectMethod  `json:"methods,omitempty"`
}

// ReflectCondition represents the reachability condition in reflection config.
type ReflectCondition struct {
	TypeReachable string `json:"typeReachable"`
}

// ReflectField represents a field entry in reflection config.
type ReflectField struct {
	Name string `json:"name"`
}

// ReflectMethod represents a method entry in reflection config.
type ReflectMethod struct {
	Name           string   `json:"name"`
	ParameterTypes []string `json:"parameterTypes"`
}

// StreamType defines RPC streaming style.
type StreamType int

const (
	// StreamNone indicates a unary (non-streaming) RPC.
	StreamNone StreamType = iota
	// StreamServer indicates a server-streaming RPC.
	StreamServer
	// StreamClient indicates a client-streaming RPC.
	StreamClient
	// StreamBidi indicates a bidirectional streaming RPC.
	StreamBidi
)

// FieldBehavior specifies field behavior annotations.
type FieldBehavior string

const (
	// FieldBehaviorRequired indicates a required field.
	FieldBehaviorRequired FieldBehavior = "REQUIRED"
	// FieldBehaviorOptional indicates an optional field.
	FieldBehaviorOptional FieldBehavior = "OPTIONAL"
	// FieldBehaviorOutputOnly indicates an output-only field.
	FieldBehaviorOutputOnly FieldBehavior = "OUTPUT_ONLY"
	// FieldBehaviorImmutable indicates an immutable field.
	FieldBehaviorImmutable FieldBehavior = "IMMUTABLE"
)

// ResourceReference points to a resource definition.
type ResourceReference struct {
	Type        string
	ChildType   string
	IsChildType bool
}

// Field represents a proto message field.
type Field struct {
	Name              string
	Type              *ast.TypeNode
	IsRepeated        bool
	IsMap             bool
	Behaviors         []FieldBehavior
	ResourceReference *ResourceReference
	Format            string
	Description       string
}

// IsRequired returns true if the field has the REQUIRED behavior.
func (f *Field) IsRequired() bool {
	return slices.Contains(f.Behaviors, FieldBehaviorRequired)
}

// ResourceName represents a Google API resource name definition.
type ResourceName struct {
	Type        string   // e.g. "pubsub.googleapis.com/Topic"
	Patterns    []string // e.g. ["projects/{project}/topics/{topic}"]
	PackageName string   // e.g. "com.google.pubsub.v1"
	ClassName   string   // e.g. "TopicName"
	IsWildcard  bool
}

// Message represents a protobuf message.
type Message struct {
	Name          string
	PackageName   string
	FullProtoName string
	Fields        map[string]*Field
	FieldList     []*Field
	ResourceName  *ResourceName
	Description   string
}

// HttpBindings represents google.api.http REST mappings.
type HttpBindings struct {
	HttpMethod         string // GET, POST, PUT, DELETE, PATCH
	PathPattern        string // /v1/{name=projects/*/topics/*}
	Body               string // * or field name
	AdditionalBindings []*HttpBindings
	PathVariables      []string
	QueryParams        []string
}

// RoutingHeaderRule represents google.api.routing annotations.
type RoutingHeaderRule struct {
	Table map[string]string // param -> regex pattern
}

// LongrunningOperation represents LRO info for an RPC.
type LongrunningOperation struct {
	ResponseType *ast.TypeNode
	MetadataType *ast.TypeNode
}

// Method represents an RPC method in a service.
type Method struct {
	Name                string
	InputType           *ast.TypeNode
	OutputType          *ast.TypeNode
	StreamType          StreamType
	HttpBindings        *HttpBindings
	RoutingHeaderRule   *RoutingHeaderRule
	MethodSignatures    [][]string
	Lro                 *LongrunningOperation
	IsPaged             bool
	PageSizeField       string
	PageTokenField      string
	NextPageTokenField  string
	ResourceListField   string
	IsDeprecated        bool
	Description         string
	AutoPopulatedFields []string
}

// Service represents a gRPC/GAPIC service.
type Service struct {
	Name                string
	PackageName         string
	OriginalJavaPackage string
	HostName            string
	DefaultPort         string
	ClientDocumentation string
	Methods             []*Method
	HasLRO              bool
	HasStreaming        bool
	IsDeprecated        bool
}

// GapicServiceConfig contains parsed retry/timeout configuration from grpc_service_config.json.
type GapicServiceConfig struct {
	RetryCodes    map[string][]string // name -> list of status codes
	RetryParams   map[string]*RetryParam
	MethodConfigs map[string]*MethodConfig // method -> config
}

// RetryParam defines retry parameters for an RPC method.
type RetryParam struct {
	InitialRetryDelayMillis int64
	RetryDelayMultiplier    float64
	MaxRetryDelayMillis     int64
	InitialRpcTimeoutMillis int64
	RpcTimeoutMultiplier    float64
	MaxRpcTimeoutMillis     int64
	TotalTimeoutMillis      int64
}

// MethodConfig associates an RPC method with its retry policy and timeout.
type MethodConfig struct {
	RetryPolicyName string
	TimeoutMillis   int64
}

// GapicBatchingSettings contains batching settings from gapic.yaml.
type GapicBatchingSettings struct {
	MethodName            string
	ElementCountThreshold int
	RequestByteThreshold  int64
	DelayThresholdMillis  int64
}

// GapicLroRetrySettings contains LRO retry settings from gapic.yaml.
type GapicLroRetrySettings struct {
	MethodName             string
	InitialPollDelayMillis int64
	PollDelayMultiplier    float64
	MaxPollDelayMillis     int64
	TotalPollTimeoutMillis int64
}

// GapicLanguageSettings contains language specific overrides from gapic.yaml.
type GapicLanguageSettings struct {
	PackageName    string
	InterfaceNames map[string]string
}

// GapicContext holds all parsed and derived data for GAPIC client generation.
type GapicContext struct {
	Services               []*Service
	Messages               map[string]*Message
	ResourceNames          map[string]*ResourceName
	HelperResourceNames    map[string]*ResourceName
	ServiceConfig          *GapicServiceConfig
	BatchingSettings       []*GapicBatchingSettings
	LroRetrySettings       []*GapicLroRetrySettings
	LanguageSettings       *GapicLanguageSettings
	Transport              Transport
	Repo                   string
	Artifact               string
	HasMetadata            bool
	HasNumericEnum         bool
	HasGenerateVersionJava bool
}

// FindMessage returns the Message definition matching the full proto name.
func (c *GapicContext) FindMessage(fullName string) *Message {
	if c.Messages == nil {
		return nil
	}
	return c.Messages[fullName]
}

// FindResourceName returns the ResourceName definition matching the given resource type.
func (c *GapicContext) FindResourceName(resType string) *ResourceName {
	if c.ResourceNames == nil {
		return nil
	}
	return c.ResourceNames[resType]
}
