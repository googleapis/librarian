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
	"strings"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/googleapis/librarian/internal/gapic/java/engine/ast"
	"github.com/googleapis/librarian/internal/gapic/java/model"
	"github.com/iancoleman/strcase"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Parse converts a CodeGeneratorRequest into a populated GapicContext.
func Parse(req *pluginpb.CodeGeneratorRequest) (*model.GapicContext, error) {
	pluginArgs := ParsePluginArguments(req)

	svcCfg, err := ParseServiceConfigJSON(pluginArgs.GrpcServiceConfigPath)
	if err != nil {
		return nil, err
	}
	batching, lroSettings, langSettings, err := ParseGapicYaml(pluginArgs.GapicYamlConfigPath)
	if err != nil {
		return nil, err
	}

	ctx := &model.GapicContext{
		Messages:               make(map[string]*model.Message),
		ResourceNames:          make(map[string]*model.ResourceName),
		HelperResourceNames:    make(map[string]*model.ResourceName),
		ServiceConfig:          svcCfg,
		BatchingSettings:       batching,
		LroRetrySettings:       lroSettings,
		LanguageSettings:       langSettings,
		Transport:              model.ParseTransport(pluginArgs.Transport),
		Repo:                   pluginArgs.Repo,
		Artifact:               pluginArgs.Artifact,
		HasMetadata:            pluginArgs.HasMetadata,
		HasNumericEnum:         pluginArgs.HasNumericEnum,
		HasGenerateVersionJava: pluginArgs.HasGenerateVersionJava,
	}

	filesToGenerate := make(map[string]bool)
	for _, f := range req.GetFileToGenerate() {
		filesToGenerate[f] = true
	}

	// First pass: collect resource definitions and messages
	for _, file := range req.GetProtoFile() {
		pkg := getJavaPackage(file)

		// File-level resource definitions
		if file.Options != nil && proto.HasExtension(file.Options, annotations.E_ResourceDefinition) {
			resDefs := proto.GetExtension(file.Options, annotations.E_ResourceDefinition).([]*annotations.ResourceDescriptor)
			for _, rd := range resDefs {
				if rd == nil {
					continue
				}
				resName := parseResourceDescriptor(rd, pkg)
				ctx.ResourceNames[resName.Type] = resName
				ctx.HelperResourceNames[resName.Type] = resName
			}
		}

		for _, msg := range file.GetMessageType() {
			parsedMsg := parseMessage(msg, file.GetPackage(), pkg)
			ctx.Messages[file.GetPackage()+"."+msg.GetName()] = parsedMsg
			ctx.Messages[msg.GetName()] = parsedMsg

			if parsedMsg.ResourceName != nil {
				ctx.ResourceNames[parsedMsg.ResourceName.Type] = parsedMsg.ResourceName
				ctx.HelperResourceNames[parsedMsg.ResourceName.Type] = parsedMsg.ResourceName
			}
		}
	}

	// Second pass: parse services in target files
	for _, file := range req.GetProtoFile() {
		if !filesToGenerate[file.GetName()] && len(filesToGenerate) > 0 {
			continue
		}

		pkg := getJavaPackage(file)
		if langSettings != nil && langSettings.PackageName != "" {
			pkg = langSettings.PackageName
		}

		for _, svc := range file.GetService() {
			parsedSvc := parseService(svc, file, pkg, ctx)
			ctx.Services = append(ctx.Services, parsedSvc)
		}
	}

	return ctx, nil
}

func getJavaPackage(file *descriptorpb.FileDescriptorProto) string {
	if file.GetOptions() != nil && file.GetOptions().GetJavaPackage() != "" {
		return file.GetOptions().GetJavaPackage()
	}
	protoPkg := file.GetPackage()
	if protoPkg == "" {
		return "com.google.cloud"
	}
	return "com." + protoPkg
}

func parseResourceDescriptor(rd *annotations.ResourceDescriptor, javaPkg string) *model.ResourceName {
	resType := rd.GetType()
	typeName := rd.GetSingular()
	if typeName == "" {
		parts := strings.Split(resType, "/")
		typeName = parts[len(parts)-1]
	}
	className := strcase.ToCamel(typeName) + "Name"
	return &model.ResourceName{
		Type:        resType,
		Patterns:    rd.GetPattern(),
		PackageName: javaPkg,
		ClassName:   className,
	}
}

func parseMessage(msg *descriptorpb.DescriptorProto, protoPkg, javaPkg string) *model.Message {
	m := &model.Message{
		Name:          msg.GetName(),
		PackageName:   javaPkg,
		FullProtoName: protoPkg + "." + msg.GetName(),
		Fields:        make(map[string]*model.Field),
	}

	if msg.Options != nil && proto.HasExtension(msg.Options, annotations.E_Resource) {
		rd := proto.GetExtension(msg.Options, annotations.E_Resource).(*annotations.ResourceDescriptor)
		if rd != nil {
			m.ResourceName = parseResourceDescriptor(rd, javaPkg)
		}
	}

	for _, f := range msg.GetField() {
		field := &model.Field{
			Name:       f.GetName(),
			Type:       protoTypeToTypeNode(f, protoPkg, javaPkg),
			IsRepeated: f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED,
		}
		if f.Options != nil {
			if proto.HasExtension(f.Options, annotations.E_FieldBehavior) {
				behaviors := proto.GetExtension(f.Options, annotations.E_FieldBehavior).([]annotations.FieldBehavior)
				for _, b := range behaviors {
					field.Behaviors = append(field.Behaviors, model.FieldBehavior(b.String()))
				}
			}
			if proto.HasExtension(f.Options, annotations.E_ResourceReference) {
				rr := proto.GetExtension(f.Options, annotations.E_ResourceReference).(*annotations.ResourceReference)
				if rr != nil {
					field.ResourceReference = &model.ResourceReference{
						Type:        rr.GetType(),
						ChildType:   rr.GetChildType(),
						IsChildType: rr.GetChildType() != "",
					}
				}
			}
		}
		m.Fields[field.Name] = field
		m.FieldList = append(m.FieldList, field)
	}

	return m
}

func protoTypeToTypeNode(f *descriptorpb.FieldDescriptorProto, protoPkg, javaPkg string) *ast.TypeNode {
	switch f.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return ast.TypeDouble
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		return ast.TypeFloat
	case descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_SINT64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		return ast.TypeLong
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		return ast.TypeLong
	case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_SINT32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		return ast.TypeInt
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		return ast.TypeInt
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return ast.TypeBoolean
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return ast.TypeString
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return ast.ObjectType("ByteString", "com.google.protobuf")
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		rawType := f.GetTypeName()
		trimmed := strings.TrimPrefix(rawType, ".")
		parts := strings.Split(trimmed, ".")
		typeName := parts[len(parts)-1]
		pkg := javaPkg
		if len(parts) > 1 {
			typeProtoPkg := strings.Join(parts[:len(parts)-1], ".")
			if typeProtoPkg != protoPkg {
				pkg = "com." + typeProtoPkg
			}
		}
		return ast.ObjectType(typeName, pkg)
	default:
		return ast.TypeObject
	}
}

func parseService(svc *descriptorpb.ServiceDescriptorProto, file *descriptorpb.FileDescriptorProto, javaPkg string, ctx *model.GapicContext) *model.Service {
	s := &model.Service{
		Name:                svc.GetName(),
		PackageName:         javaPkg,
		OriginalJavaPackage: getJavaPackage(file),
		DefaultPort:         "443",
	}

	if svc.Options != nil {
		if proto.HasExtension(svc.Options, annotations.E_DefaultHost) {
			s.HostName = proto.GetExtension(svc.Options, annotations.E_DefaultHost).(string)
		}
	}

	for _, m := range svc.GetMethod() {
		method := parseMethod(m, javaPkg, ctx)
		s.Methods = append(s.Methods, method)
		if method.Lro != nil {
			s.HasLRO = true
		}
		if method.StreamType != model.StreamNone {
			s.HasStreaming = true
		}
	}

	return s
}

func parseMethod(m *descriptorpb.MethodDescriptorProto, javaPkg string, ctx *model.GapicContext) *model.Method {
	inputTypeName := typeShortName(m.GetInputType())
	outputTypeName := typeShortName(m.GetOutputType())

	method := &model.Method{
		Name:       m.GetName(),
		InputType:  ast.ObjectType(inputTypeName, javaPkg),
		OutputType: ast.ObjectType(outputTypeName, javaPkg),
	}

	// Stream type
	if m.GetClientStreaming() && m.GetServerStreaming() {
		method.StreamType = model.StreamBidi
	} else if m.GetServerStreaming() {
		method.StreamType = model.StreamServer
	} else if m.GetClientStreaming() {
		method.StreamType = model.StreamClient
	} else {
		method.StreamType = model.StreamNone
	}

	if m.Options != nil {
		// HTTP Rule
		if proto.HasExtension(m.Options, annotations.E_Http) {
			httpRule := proto.GetExtension(m.Options, annotations.E_Http).(*annotations.HttpRule)
			if httpRule != nil {
				method.HttpBindings = parseHttpRule(httpRule)
			}
		}

		// Method Signatures
		if proto.HasExtension(m.Options, annotations.E_MethodSignature) {
			sigs := proto.GetExtension(m.Options, annotations.E_MethodSignature).([]string)
			for _, sig := range sigs {
				fields := strings.Split(sig, ",")
				for i := range fields {
					fields[i] = strings.TrimSpace(fields[i])
				}
				method.MethodSignatures = append(method.MethodSignatures, fields)
			}
		}

		// Longrunning Operation
		if proto.HasExtension(m.Options, longrunningpb.E_OperationInfo) {
			opInfo := proto.GetExtension(m.Options, longrunningpb.E_OperationInfo).(*longrunningpb.OperationInfo)
			if opInfo != nil {
				respType := typeShortName(opInfo.GetResponseType())
				metaType := typeShortName(opInfo.GetMetadataType())
				method.Lro = &model.LongrunningOperation{
					ResponseType: ast.ObjectType(respType, javaPkg),
					MetadataType: ast.ObjectType(metaType, javaPkg),
				}
			}
		}

		// Routing
		if proto.HasExtension(m.Options, annotations.E_Routing) {
			routing := proto.GetExtension(m.Options, annotations.E_Routing).(*annotations.RoutingRule)
			if routing != nil && len(routing.GetRoutingParameters()) > 0 {
				tbl := make(map[string]string)
				for _, param := range routing.GetRoutingParameters() {
					field := param.GetField()
					pattern := param.GetPathTemplate()
					if pattern == "" {
						pattern = "{**}"
					}
					tbl[field] = pattern
				}
				method.RoutingHeaderRule = &model.RoutingHeaderRule{Table: tbl}
			}
		}
	}

	// Detect Paging
	checkPaging(method, inputTypeName, outputTypeName, ctx)

	return method
}

func parseHttpRule(rule *annotations.HttpRule) *model.HttpBindings {
	bindings := &model.HttpBindings{
		Body: rule.GetBody(),
	}
	switch p := rule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		bindings.HttpMethod = "GET"
		bindings.PathPattern = p.Get
	case *annotations.HttpRule_Post:
		bindings.HttpMethod = "POST"
		bindings.PathPattern = p.Post
	case *annotations.HttpRule_Put:
		bindings.HttpMethod = "PUT"
		bindings.PathPattern = p.Put
	case *annotations.HttpRule_Delete:
		bindings.HttpMethod = "DELETE"
		bindings.PathPattern = p.Delete
	case *annotations.HttpRule_Patch:
		bindings.HttpMethod = "PATCH"
		bindings.PathPattern = p.Patch
	}

	for _, add := range rule.GetAdditionalBindings() {
		if add != nil {
			bindings.AdditionalBindings = append(bindings.AdditionalBindings, parseHttpRule(add))
		}
	}
	return bindings
}

func checkPaging(method *model.Method, inputMsgName, outputMsgName string, ctx *model.GapicContext) {
	inMsg := ctx.FindMessage(inputMsgName)
	outMsg := ctx.FindMessage(outputMsgName)
	if inMsg == nil || outMsg == nil {
		return
	}

	var hasPageSize, hasPageToken, hasNextPageToken bool
	var pageSizeField, pageTokenField, nextPageTokenField, resourceListField string

	for _, f := range inMsg.FieldList {
		name := f.Name
		if (name == "page_size" || name == "max_results") && (f.Type == ast.TypeInt || f.Type == ast.TypeLong) {
			hasPageSize = true
			pageSizeField = name
		}
		if name == "page_token" && f.Type == ast.TypeString {
			hasPageToken = true
			pageTokenField = name
		}
	}

	for _, f := range outMsg.FieldList {
		name := f.Name
		if name == "next_page_token" && f.Type == ast.TypeString {
			hasNextPageToken = true
			nextPageTokenField = name
		}
		if f.IsRepeated && resourceListField == "" {
			resourceListField = name
		}
	}

	if hasPageSize && hasPageToken && hasNextPageToken && resourceListField != "" {
		method.IsPaged = true
		method.PageSizeField = pageSizeField
		method.PageTokenField = pageTokenField
		method.NextPageTokenField = nextPageTokenField
		method.ResourceListField = resourceListField
	}
}

func typeShortName(full string) string {
	parts := strings.Split(strings.TrimPrefix(full, "."), ".")
	return parts[len(parts)-1]
}
