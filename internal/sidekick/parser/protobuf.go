// Copyright 2024 Google LLC
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
	"bytes"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser/httprule"
	"github.com/googleapis/librarian/internal/sidekick/parser/svcconfig"
	"github.com/googleapis/librarian/internal/sidekick/protobuf"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/tool/protoc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ParseProtobuf reads Protobuf specifications and converts them into
// the `api.API` model.
func ParseProtobuf(cfg *ModelConfig) (*api.API, error) {
	var request *pluginpb.CodeGeneratorRequest
	var err error

	if cfg.DescriptorFiles != "" {
		if cfg.DescriptorFilesToGenerate == "" {
			return nil, fmt.Errorf("descriptorFilesToGenerate must be specified when using descriptorFiles")
		}
		request, err = codeGeneratorRequestFromDescriptors(cfg.DescriptorFiles, cfg.DescriptorFilesToGenerate)
		if err != nil {
			return nil, err
		}
	} else {
		source := cfg.SpecificationSource
		request, err = codeGeneratorRequestFromSource(source, cfg.Source, cfg.Protoc)
		if err != nil {
			return nil, err
		}
	}

	serviceConfig, err := loadServiceConfig(cfg)
	if err != nil {
		return nil, err
	}
	return makeAPIForProtobuf(serviceConfig, request)
}

func codeGeneratorRequestFromDescriptors(descriptorFiles, generateFiles string) (*pluginpb.CodeGeneratorRequest, error) {
	allFiles, err := loadDescriptorSet(descriptorFiles)
	if err != nil {
		return nil, err
	}

	generateList := parseCommaSeparatedList(generateFiles)
	target, err := filterTargetDescriptors(allFiles, generateList)
	if err != nil {
		return nil, err
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate:        generateList,
		SourceFileDescriptors: target,
		ProtoFile:             allFiles,
		CompilerVersion:       newCompilerVersion(),
	}, nil
}

// Create a temporary files to store `protoc`'s output.
func codeGeneratorRequestFromSource(source string, sourceCfg *sources.SourceConfig, pc *config.Protoc) (*pluginpb.CodeGeneratorRequest, error) {
	files, err := protobuf.DetermineInputFiles(source, sourceCfg)
	if err != nil {
		return nil, err
	}

	contents, err := runProtoc(files, sourceCfg, pc)
	if err != nil {
		return nil, err
	}

	descriptors := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(contents, descriptors); err != nil {
		return nil, err
	}

	target := matchDescriptorsToFiles(descriptors.File, files)

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate:        files,
		SourceFileDescriptors: target,
		ProtoFile:             descriptors.File,
		CompilerVersion:       newCompilerVersion(),
	}, nil
}

func loadDescriptorSet(descriptorFiles string) ([]*descriptorpb.FileDescriptorProto, error) {
	var allFiles []*descriptorpb.FileDescriptorProto
	for _, f := range parseCommaSeparatedList(descriptorFiles) {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read descriptor file %q: %w", f, err)
		}
		set := &descriptorpb.FileDescriptorSet{}
		if err := proto.Unmarshal(data, set); err != nil {
			return nil, fmt.Errorf("failed to unmarshal descriptor file %q: %w", f, err)
		}
		allFiles = append(allFiles, set.File...)
	}
	return allFiles, nil
}

func parseCommaSeparatedList(s string) []string {
	var list []string
	for f := range strings.SplitSeq(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			list = append(list, f)
		}
	}
	return list
}

func filterTargetDescriptors(allFiles []*descriptorpb.FileDescriptorProto, generateList []string) ([]*descriptorpb.FileDescriptorProto, error) {
	var target []*descriptorpb.FileDescriptorProto
	for _, name := range generateList {
		found := false
		for _, pb := range allFiles {
			// Protobuf descriptor names always use forward slashes "/" regardless of the operating system.
			if pb.GetName() == name || strings.HasSuffix(pb.GetName(), "/"+name) {
				target = append(target, pb)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("file to generate %q not found in descriptor files", name)
		}
	}
	return target, nil
}

func runProtoc(files []string, sourceCfg *sources.SourceConfig, pc *config.Protoc) ([]byte, error) {
	tempFile, err := os.CreateTemp("", "protoc-out-")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	args := []string{
		"--include_imports",
		"--include_source_info",
		"--retain_options",
		"--descriptor_set_out", tempFile.Name(),
	}
	if sourceCfg != nil {
		for _, root := range sourceCfg.ActiveRoots {
			if path := sourceCfg.Root(root); path != "" {
				args = append(args, "--proto_path")
				args = append(args, path)
			}
		}
	}

	args = append(args, files...)

	protocPath, err := protoc.BinaryPathOrSystem(pc)
	if err != nil {
		return nil, fmt.Errorf("failed to find protoc: %w", err)
	}

	var stderr, stdout bytes.Buffer
	cmd := exec.Command(protocPath, args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error calling protoc\ndetails:\n%s\nargs:\n%v\n: %w", stderr.String(), args, err)
	}

	return os.ReadFile(tempFile.Name())
}

func matchDescriptorsToFiles(descriptors []*descriptorpb.FileDescriptorProto, files []string) []*descriptorpb.FileDescriptorProto {
	var target []*descriptorpb.FileDescriptorProto
	for _, filename := range files {
		for _, pb := range descriptors {
			if strings.HasSuffix(filename, *pb.Name) {
				target = append(target, pb)
			}
		}
	}
	return target
}

func newCompilerVersion() *pluginpb.Version {
	var (
		i int32
		s = "test"
	)
	return &pluginpb.Version{
		Major:  &i,
		Minor:  &i,
		Patch:  &i,
		Suffix: &s,
	}
}

const (
	// From https://pkg.go.dev/google.golang.org/protobuf/types/descriptorpb#FileDescriptorProto

	fileDescriptorName             = 1
	fileDescriptorPackage          = 2
	fileDescriptorDependency       = 3
	fileDescriptorMessageType      = 4
	fileDescriptorEnumType         = 5
	fileDescriptorService          = 6
	fileDescriptorExtension        = 7
	fileDescriptorOptions          = 8
	fileDescriptorSourceCodeInfo   = 9
	fileDescriptorPublicDependency = 10
	fileDescriptorWeakDependency   = 11
	fileDescriptorSyntax           = 12
	fileDescriptorEdition          = 14

	// From https://pkg.go.dev/google.golang.org/protobuf/types/descriptorpb#ServiceDescriptorProto

	serviceDescriptorProtoMethod = 2
	serviceDescriptorProtoOption = 3

	// From https://pkg.go.dev/google.golang.org/protobuf/types/descriptorpb#DescriptorProto

	messageDescriptorField          = 2
	messageDescriptorNestedType     = 3
	messageDescriptorEnum           = 4
	messageDescriptorExtensionRange = 5
	messageDescriptorExtension      = 6
	messageDescriptorOptions        = 7
	messageDescriptorOneOf          = 8
	messageDescriptorReservedRange  = 9
	messageDescriptorReservedName   = 10

	// From https://pkg.go.dev/google.golang.org/protobuf/types/descriptorpb#EnumDescriptorProto

	enumDescriptorName          = 1
	enumDescriptorValue         = 2
	enumDescriptorOptions       = 3
	enumDescriptorReservedRange = 4
	enumDescriptorReservedName  = 5
	enumDescriptorVisibility    = 6
)

func makeAPIForProtobuf(serviceConfig *serviceconfig.Service, req *pluginpb.CodeGeneratorRequest) (*api.API, error) {
	var (
		mixinFileDesc       []*descriptorpb.FileDescriptorProto
		enabledMixinMethods mixinMethods = make(map[string]bool)
	)
	result := &api.API{}
	if serviceConfig != nil {
		result.Title = serviceConfig.Title
		if serviceConfig.Documentation != nil {
			result.Description = serviceConfig.Documentation.Summary
		}
		withLongrunning := requiresLongrunningMixin(req)
		enabledMixinMethods, mixinFileDesc = loadMixins(serviceConfig, withLongrunning)
		names := svcconfig.ExtractPackageName(serviceConfig)
		if names != nil {
			result.PackageName = names.PackageName
		}
	}

	// First we need to add all the message and enums types to the
	// `state.MessageByID` and `state.EnumByID` symbol tables. We may not need
	// to generate these elements, but we need them to be available to generate
	// any RPC that uses them.
	fileComments := map[*descriptorpb.FileDescriptorProto]map[string]string{}
	for _, f := range req.GetSourceFileDescriptors() {
		fileComments[f] = extractSourceComments(f)
	}

	for _, f := range append(req.GetProtoFile(), mixinFileDesc...) {
		fFQN := "." + f.GetPackage()
		comments := fileComments[f]
		for i, m := range f.MessageType {
			mFQN := fFQN + "." + m.GetName()
			if _, err := processMessage(result, m, mFQN, f.GetPackage(), isProto2(f), nil, comments, []int32{fileDescriptorMessageType, int32(i)}); err != nil {
				return nil, err
			}
		}

		for i, e := range f.EnumType {
			eFQN := fFQN + "." + e.GetName()
			_ = processEnum(result, e, eFQN, f.GetPackage(), nil, comments, []int32{fileDescriptorEnumType, int32(i)})
		}
		resources, err := processFileResourceDefinitions(f)
		if err != nil {
			return nil, err
		}
		result.ResourceDefinitions = append(result.ResourceDefinitions, resources...)
	}

	// Consolidate resources.
	// Message-level resources (in result.AllResources take precedence over
	// file-level resources (already in result.ResourceDefinitions).
	seen := map[string]int{}
	for i, r := range result.ResourceDefinitions {
		seen[r.Type] = i
	}
	for r := range result.AllResources() {
		if i, found := seen[r.Type]; found {
			result.ResourceDefinitions[i] = r
		} else {
			seen[r.Type] = len(result.ResourceDefinitions)
			result.ResourceDefinitions = append(result.ResourceDefinitions, r)
		}
	}

	// Sort to ensure deterministic output.
	slices.SortFunc(result.ResourceDefinitions, func(a, b *api.Resource) int {
		return strings.Compare(a.Type, b.Type)
	})

	// Collect all mixin methods enabled for services in this API.
	type mixinMethodSpec struct {
		proto      *descriptorpb.MethodDescriptorProto
		mixin      *api.Service
		serviceID  string
		apiVersion string
	}
	var mixinSpecs []mixinMethodSpec
	for _, f := range mixinFileDesc {
		fFQN := "." + f.GetPackage()
		for _, mixinProto := range f.Service {
			sFQN := fFQN + "." + mixinProto.GetName()
			mixin := processService(result, mixinProto, sFQN, f.GetPackage(), "", nil)
			apiVersion := parseAPIVersion(sFQN, mixinProto.GetOptions())
			for _, m := range mixinProto.Method {
				originalFQN := sFQN + "." + m.GetName()
				if !enabledMixinMethods[originalFQN] {
					continue
				}
				mixinSpecs = append(mixinSpecs, mixinMethodSpec{
					proto:      m,
					mixin:      mixin,
					serviceID:  sFQN,
					apiVersion: apiVersion,
				})
			}
		}
	}

	// Then we need to add the messages, enums and services to the list of
	// elements to be generated.
	for _, f := range req.GetSourceFileDescriptors() {
		var fileServices []*api.Service
		fFQN := "." + f.GetPackage()
		comments := fileComments[f]

		if value, err := protobufUpdateFileOption(result.CsharpNamespace, f.GetOptions().GetCsharpNamespace()); err == nil {
			result.CsharpNamespace = value
		} else {
			return nil, err
		}
		if value, err := protobufUpdateFileOption(result.PhpNamespace, f.GetOptions().GetPhpNamespace()); err == nil {
			result.PhpNamespace = value
		} else {
			return nil, err
		}
		// TODO(https://github.com/googleapis/librarian/issues/7091) - also use protobufUpdateFileOption() for Ruby
		if result.RubyPackage == "" {
			result.RubyPackage = f.GetOptions().GetRubyPackage()
		}

		// Messages
		for _, m := range f.MessageType {
			mFQN := fFQN + "." + m.GetName()
			if msg := result.Message(mFQN); msg != nil {
				result.Messages = append(result.Messages, msg)
			} else {
				slog.Warn("missing message in symbol table", "message", mFQN)
			}
		}

		// Enums
		for _, e := range f.EnumType {
			eFQN := fFQN + "." + e.GetName()
			if e := result.Enum(eFQN); e != nil {
				result.Enums = append(result.Enums, e)
			} else {
				slog.Warn("missing enum in symbol table", "message", eFQN)
			}
		}

		// Services
		for si, s := range f.Service {
			sFQN := fFQN + "." + s.GetName()
			sPath := []int32{fileDescriptorService, int32(si)}
			apiVersion := parseAPIVersion(sFQN, s.GetOptions())
			var methods []*api.Method
			for mi, m := range s.Method {
				mFQN := sFQN + "." + m.GetName()
				mDoc := lookupComment(comments, append(sPath, serviceDescriptorProtoMethod, int32(mi)))
				method, err := processMethod(result, m, mFQN, f.GetPackage(), sFQN, apiVersion, mDoc, serviceConfig)
				if err != nil {
					return nil, err
				}
				methods = append(methods, method)
			}
			for _, spec := range mixinSpecs {
				mFQN := sFQN + "." + spec.proto.GetName()
				if result.Method(mFQN) != nil {
					// The method already exists. This happens when services
					// require a mixin in the service config yaml *and* also
					// define the mixin method in the code.
					continue
				}
				originalFQN := spec.serviceID + "." + spec.proto.GetName()
				method, err := processMixinMethod(result, spec.proto, mFQN, originalFQN, f.GetPackage(), spec.serviceID, spec.apiVersion, spec.mixin, serviceConfig)
				if err != nil {
					return nil, err
				}
				methods = append(methods, method)
			}
			service := processService(result, s, sFQN, f.GetPackage(), lookupComment(comments, sPath), methods)
			fileServices = append(fileServices, service)
		}
		result.Services = append(result.Services, fileServices...)
	}

	if result.Name == "" && serviceConfig != nil {
		result.Name = strings.TrimSuffix(serviceConfig.Name, ".googleapis.com")
	}
	updatePackageName(result)
	return result, nil
}

// requiresLongrunningMixin finds out if any method returns a LRO. This is used
// to forcibly load the longrunning mixin. It needs to happen before the proto
// descriptors are converted to the `api.*`, as that conversion requires the
// mixin.
func requiresLongrunningMixin(req *pluginpb.CodeGeneratorRequest) bool {
	for _, f := range req.GetSourceFileDescriptors() {
		for _, s := range f.Service {
			for _, m := range s.Method {
				info := parseOperationInfo(f.GetPackage(), m)
				if info != nil && m.GetOutputType() == ".google.longrunning.Operation" {
					return true
				}
			}
		}
	}
	return false
}

var descriptorpbToTypez = map[descriptorpb.FieldDescriptorProto_Type]api.Typez{
	descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:   api.TypezDouble,
	descriptorpb.FieldDescriptorProto_TYPE_FLOAT:    api.TypezFloat,
	descriptorpb.FieldDescriptorProto_TYPE_INT64:    api.TypezInt64,
	descriptorpb.FieldDescriptorProto_TYPE_UINT64:   api.TypezUint64,
	descriptorpb.FieldDescriptorProto_TYPE_INT32:    api.TypezInt32,
	descriptorpb.FieldDescriptorProto_TYPE_FIXED64:  api.TypezFixed64,
	descriptorpb.FieldDescriptorProto_TYPE_FIXED32:  api.TypezFixed32,
	descriptorpb.FieldDescriptorProto_TYPE_BOOL:     api.TypezBool,
	descriptorpb.FieldDescriptorProto_TYPE_STRING:   api.TypezString,
	descriptorpb.FieldDescriptorProto_TYPE_BYTES:    api.TypezBytes,
	descriptorpb.FieldDescriptorProto_TYPE_UINT32:   api.TypezUint32,
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED32: api.TypezSfixed32,
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED64: api.TypezSfixed64,
	descriptorpb.FieldDescriptorProto_TYPE_SINT32:   api.TypezSint32,
	descriptorpb.FieldDescriptorProto_TYPE_SINT64:   api.TypezSint64,
	descriptorpb.FieldDescriptorProto_TYPE_GROUP:    api.TypezGroup,
	descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:  api.TypezMessage,
	descriptorpb.FieldDescriptorProto_TYPE_ENUM:     api.TypezEnum,
}

// protobufUpdateFileOption validates an update of `current` to `got`.
//
// This returns an error if `current` != "" and `current` != got.
func protobufUpdateFileOption(current, got string) (string, error) {
	if current == "" {
		return got, nil
	}
	if current == got {
		return got, nil
	}
	// The Well-Known Types have inconsistent annotations for C#. Just ignore
	// the inconsistency, it is not a real problem.
	if got == "" || strings.HasPrefix(got, "Google.Protobuf.") {
		return current, nil
	}
	return "", fmt.Errorf("mismatched file option value, want=%s got=%s", current, got)
}

func normalizeTypes(model *api.API, in *descriptorpb.FieldDescriptorProto, field *api.Field) {
	field.Typez = descriptorpbToTypez[in.GetType()]
	field.TypezID = in.GetTypeName()
	field.Repeated = in.Label != nil && *in.Label == descriptorpb.FieldDescriptorProto_LABEL_REPEATED

	switch in.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		field.TypezID = in.GetTypeName()
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		field.TypezID = in.GetTypeName()
		// Repeated fields are not optional, they can be empty, but always have
		// presence. Oneof fields are also not optional because presence is
		// managed by the oneof itself.
		field.Optional = !field.Repeated && !field.IsOneOf
		if model.Map(field.TypezID) != nil {
			// Map fields appear as repeated in Protobuf. This is confusing,
			// as they typically are represented by a single `map<k, v>`-like
			// datatype. Protobuf leaks the wire-representation of maps, i.e.,
			// repeated pairs.
			field.Typez = api.TypezMap
			field.Map = true
			field.Repeated = false
			field.Optional = false
		}
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		field.TypezID = in.GetTypeName()

	case
		descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		// These do not need normalization
		return

	default:
		slog.Warn("found undefined field", "field", in.GetName())
	}
}

func processService(model *api.API, s *descriptorpb.ServiceDescriptorProto, sFQN, packagez, doc string, methods []*api.Method) *api.Service {
	service := &api.Service{
		Name:          s.GetName(),
		ID:            sFQN,
		Package:       packagez,
		Documentation: doc,
		DefaultHost:   parseDefaultHost(s.GetOptions()),
		Deprecated:    s.GetOptions().GetDeprecated(),
		Methods:       methods,
	}
	model.AddService(service)
	return service
}

func processMethod(model *api.API, m *descriptorpb.MethodDescriptorProto, mFQN, packagez, serviceID, apiVersion, doc string, serviceConfig *serviceconfig.Service) (*api.Method, error) {
	pathInfo, err := parsePathInfo(m, model)
	if err != nil {
		return nil, fmt.Errorf("unsupported http method for %q: %w", mFQN, err)
	}
	return buildProtobufMethod(model, m, mFQN, packagez, serviceID, apiVersion, doc, pathInfo, serviceConfig)
}

func processMixinMethod(model *api.API, m *descriptorpb.MethodDescriptorProto, mFQN, originalFQN, packagez, serviceID, apiVersion string, mixin *api.Service, serviceConfig *serviceconfig.Service) (*api.Method, error) {
	defaultPathInfo, err := parsePathInfo(m, model)
	if err != nil {
		return nil, fmt.Errorf("unsupported http method for %q: %w", mFQN, err)
	}
	pathInfo, doc, err := serviceConfigMethodOverrides(mFQN, originalFQN, m.GetInputType(), defaultPathInfo, serviceConfig, model, mixin)
	if err != nil {
		return nil, err
	}
	return buildProtobufMethod(model, m, mFQN, packagez, serviceID, apiVersion, doc, pathInfo, serviceConfig)
}

func buildProtobufMethod(model *api.API, m *descriptorpb.MethodDescriptorProto, mFQN, packagez, serviceID, apiVersion, doc string, pathInfo *api.PathInfo, serviceConfig *serviceconfig.Service) (*api.Method, error) {
	routing, err := parseRoutingAnnotations(mFQN, m)
	if err != nil {
		return nil, fmt.Errorf("cannot parse routing annotations for %q: %w", mFQN, err)
	}
	inputTypeID := m.GetInputType()
	outputTypeID := m.GetOutputType()
	signatures, err := protobufMethodSignatures(m)
	if err != nil {
		return nil, err
	}
	method := &api.Method{
		ID:                  mFQN,
		PathInfo:            pathInfo,
		Name:                m.GetName(),
		Documentation:       doc,
		Deprecated:          m.GetOptions().GetDeprecated(),
		InputTypeID:         inputTypeID,
		OutputTypeID:        outputTypeID,
		ClientSideStreaming: m.GetClientStreaming(),
		ServerSideStreaming: m.GetServerStreaming(),
		OperationInfo:       parseOperationInfo(packagez, m),
		Routing:             routing,
		ReturnsEmpty:        outputTypeID == ".google.protobuf.Empty",
		SourceServiceID:     serviceID,
		APIVersion:          apiVersion,
		Signatures:          signatures,
		AutoPopulated:       findAutoPopulatedFields(serviceConfig, mFQN, model.Message(inputTypeID)),
	}
	model.AddMethod(method)
	return method, nil
}

func isProto2(f *descriptorpb.FileDescriptorProto) bool {
	// protoc leaves the syntax field empty for proto2 files.
	return f.GetSyntax() == "" || f.GetSyntax() == "proto2"
}

func processMessage(model *api.API, m *descriptorpb.DescriptorProto, mFQN, packagez string, proto2 bool, parent *api.Message, comments map[string]string, pathPrefix []int32) (*api.Message, error) {
	message := &api.Message{
		Name:          m.GetName(),
		ID:            mFQN,
		Parent:        parent,
		Package:       packagez,
		Documentation: lookupComment(comments, pathPrefix),
		Deprecated:    m.GetOptions().GetDeprecated(),
	}
	model.AddMessage(message)

	if opts := m.GetOptions(); opts != nil {
		if err := processResourceAnnotation(opts, message, model); err != nil {
			return nil, err
		}
	}
	if len(m.GetNestedType()) > 0 {
		for i, nm := range m.GetNestedType() {
			nmFQN := mFQN + "." + nm.GetName()
			if nm.GetOptions().GetMapEntry() {
				if err := processMapEntry(model, nm, nmFQN); err != nil {
					return nil, err
				}
				continue
			}
			nmsg, err := processMessage(model, nm, nmFQN, packagez, proto2, message, comments, append(pathPrefix, messageDescriptorNestedType, int32(i)))
			if err != nil {
				return nil, err
			}
			message.Messages = append(message.Messages, nmsg)
		}
	}
	for i, e := range m.GetEnumType() {
		eFQN := mFQN + "." + e.GetName()
		e := processEnum(model, e, eFQN, packagez, message, comments, append(pathPrefix, messageDescriptorEnum, int32(i)))
		message.Enums = append(message.Enums, e)
	}
	for i, oneof := range m.OneofDecl {
		oneOfs := &api.OneOf{
			Name:          oneof.GetName(),
			ID:            mFQN + "." + oneof.GetName(),
			Documentation: lookupComment(comments, append(pathPrefix, messageDescriptorOneOf, int32(i))),
		}
		message.OneOfs = append(message.OneOfs, oneOfs)
	}
	for i, mf := range m.Field {
		field, err := processField(model, mf, mFQN, proto2, lookupComment(comments, append(pathPrefix, messageDescriptorField, int32(i))))
		if err != nil {
			return nil, err
		}
		message.Fields = append(message.Fields, field)
		if field.IsOneOf {
			message.OneOfs[*mf.OneofIndex].Fields = append(message.OneOfs[*mf.OneofIndex].Fields, field)
		}
	}

	// Remove proto3 optionals from one-of
	var oneOfIdx int
	for _, oneof := range message.OneOfs {
		if len(oneof.Fields) > 0 {
			message.OneOfs[oneOfIdx] = oneof
			oneOfIdx++
		}
	}
	if oneOfIdx == 0 {
		message.OneOfs = nil
	} else {
		message.OneOfs = message.OneOfs[:oneOfIdx]
	}

	return message, nil
}

func processField(model *api.API, mf *descriptorpb.FieldDescriptorProto, mFQN string, proto2 bool, doc string) (*api.Field, error) {
	isProtoOptional := mf.Proto3Optional != nil && *mf.Proto3Optional
	field := &api.Field{
		Name:          mf.GetName(),
		ID:            mFQN + "." + mf.GetName(),
		JSONName:      mf.GetJsonName(),
		Documentation: doc,
		Deprecated:    mf.GetOptions().GetDeprecated(),
		Optional:      isProtoOptional,
		IsOneOf:       mf.OneofIndex != nil && !isProtoOptional,
		AutoPopulated: protobufIsAutoPopulated(mf),
		Behavior:      protobufFieldBehavior(mf),
	}
	if err := processResourceReference(mf, field); err != nil {
		return nil, err
	}
	normalizeTypes(model, mf, field)
	field.Proto2Optional = proto2 && !field.IsOneOf &&
		mf.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL &&
		field.Typez != api.TypezMessage && field.Typez != api.TypezGroup
	return field, nil
}

func processMapEntry(model *api.API, nm *descriptorpb.DescriptorProto, nmFQN string) error {
	var keyField, valueField *api.Field
	for _, mf := range nm.Field {
		field, err := processField(model, mf, nmFQN, false, "")
		if err != nil {
			return err
		}
		switch field.Name {
		case "key":
			keyField = field
		case "value":
			valueField = field
		}
	}
	model.AddMap(&api.Map{
		ID:    nmFQN,
		Key:   keyField,
		Value: valueField,
	})
	return nil
}

func processResourceAnnotation(opts *descriptorpb.MessageOptions, message *api.Message, model *api.API) error {
	if !proto.HasExtension(opts, eResource) {
		return nil
	}
	ext := proto.GetExtension(opts, eResource)
	res, ok := ext.(*resourceDescriptor)
	if !ok {
		return fmt.Errorf("in message %q: unexpected type for eResource extension: %T", message.ID, ext)
	}

	patterns, err := parseResourcePatterns(res.GetPattern())
	if err != nil {
		return fmt.Errorf("in message %q: %w", message.ID, err)
	}

	resource := &api.Resource{
		Type:     res.GetType(),
		Patterns: patterns,
		Plural:   res.GetPlural(),
		Singular: res.GetSingular(),
		Self:     message,
	}
	message.Resource = resource
	model.AddResource(resource)
	return nil
}

// processFileResourceDefinitions extracts resource definitions from file-level options.
// This must be called for all files (including dependencies) to ensure that
// resources referenced but not defined in the source files are available.
func processFileResourceDefinitions(f *descriptorpb.FileDescriptorProto) ([]*api.Resource, error) {
	if f.Options == nil || !proto.HasExtension(f.Options, eResourceDefinition) {
		return nil, nil
	}

	ext := proto.GetExtension(f.Options, eResourceDefinition)
	res, ok := ext.([]*resourceDescriptor)
	if !ok {
		return nil, fmt.Errorf("unexpected type for eResourceDefinition extension: %T", ext)
	}

	var resources []*api.Resource
	for _, r := range res {
		patterns, err := parseResourcePatterns(r.GetPattern())
		if err != nil {
			return nil, fmt.Errorf("in file %q: %w", f.GetName(), err)
		}

		resources = append(resources, &api.Resource{
			Type:     r.GetType(),
			Patterns: patterns,
			Plural:   r.GetPlural(),
			Singular: r.GetSingular(),
		})
	}
	return resources, nil
}

// TODO(https://github.com/googleapis/librarian/issues/3036): This function needs
// to be made more robust. For methods that operate on
//
// collections (e.g., a `List` method), the `(google.api.resource_reference)`
// annotation often uses the `type` field to refer to the *parent* resource
// (e.g., `cloudresourcemanager.googleapis.com/Project`). The actual resource
// that the method returns is specified in the `child_type` field.
//
// The current implementation correctly captures both fields, but the logic in
// the gcloud generator that *uses* this information does not yet handle this
// distinction correctly. Future work should involve creating a more robust
// model that correctly determines the primary resource for a method, using
// `child_type` when it is present for collection-based methods.
func processResourceReference(f *descriptorpb.FieldDescriptorProto, field *api.Field) error {
	if f.Options == nil {
		return nil
	}
	if !proto.HasExtension(f.Options, eResourceReference) {
		return nil
	}
	ext := proto.GetExtension(f.Options, eResourceReference)
	ref, ok := ext.(*resourceReference)
	if !ok {
		return fmt.Errorf("in field %q: unexpected type for eResourceReference extension: %T", field.ID, ext)
	}
	field.ResourceReference = api.NewResourceReferenceBuilder().
		WithType(ref.Type).
		WithChildType(ref.ChildType).
		Build()
	return nil
}

func processEnum(model *api.API, e *descriptorpb.EnumDescriptorProto, eFQN, packagez string, parent *api.Message, comments map[string]string, pathPrefix []int32) *api.Enum {
	enum := &api.Enum{
		Name:          e.GetName(),
		ID:            eFQN,
		Parent:        parent,
		Package:       packagez,
		Documentation: lookupComment(comments, pathPrefix),
		Deprecated:    e.GetOptions().GetDeprecated(),
	}
	model.AddEnum(enum)
	for i, ev := range e.Value {
		enumValue := &api.EnumValue{
			Name:          ev.GetName(),
			Number:        ev.GetNumber(),
			Parent:        enum,
			Documentation: lookupComment(comments, append(pathPrefix, enumDescriptorValue, int32(i))),
			Deprecated:    ev.GetOptions().GetDeprecated(),
		}
		enum.Values = append(enum.Values, enumValue)
	}
	numbers := map[int32]*api.EnumValue{}
	for _, v := range enum.Values {
		matchesStyle := func(v *api.EnumValue) bool {
			return strings.ToUpper(v.Name) == v.Name
		}
		if ev, ok := numbers[v.Number]; ok {
			if len(ev.Name) > len(v.Name) || (matchesStyle(v) && !matchesStyle(ev)) {
				numbers[v.Number] = v
			}
		} else {
			numbers[v.Number] = v
		}
	}
	unique := slices.Collect(maps.Values(numbers))
	slices.SortFunc(unique, func(a, b *api.EnumValue) int { return int(a.Number - b.Number) })
	enum.UniqueNumberValues = unique
	return enum
}

func extractSourceComments(f *descriptorpb.FileDescriptorProto) map[string]string {
	comments := map[string]string{}
	for _, loc := range f.GetSourceCodeInfo().GetLocation() {
		p := loc.GetPath()
		if loc.GetLeadingComments() == "" || len(p) == 0 {
			continue
		}
		comments[commentPathKey(p)] = trimLeadingSpacesInDocumentation(loc.GetLeadingComments())
	}
	return comments
}

func lookupComment(comments map[string]string, path []int32) string {
	if len(comments) == 0 {
		return ""
	}
	return comments[commentPathKey(path)]
}

func commentPathKey(path []int32) string {
	var b strings.Builder
	for i, v := range path {
		if i > 0 {
			b.WriteByte('.')
		}
		fmt.Fprintf(&b, "%d", v)
	}
	return b.String()
}

func parseResourcePatterns(patterns []string) ([]api.ResourcePattern, error) {
	var parsedPatterns []api.ResourcePattern
	for _, p := range patterns {
		tmpl, err := httprule.ParseResourcePattern(p)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource pattern %q: %w", p, err)
		}
		parsedPatterns = append(parsedPatterns, tmpl.Segments())
	}
	return parsedPatterns, nil
}

// trimLeadingSpacesInDocumentation removes the leading spaces from each line in the documentation.
// Protobuf removes the `//` leading characters, but leaves the leading
// whitespace. It is easier to reason about the comments in the rest of the
// generator if they are better normalized.
func trimLeadingSpacesInDocumentation(doc string) string {
	lines := strings.Split(doc, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, " ")
	}
	return strings.TrimSuffix(strings.Join(lines, "\n"), "\n")
}
