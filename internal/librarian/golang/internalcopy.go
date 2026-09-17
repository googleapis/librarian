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

package golang

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	errInternalCopyProtoPackage = errors.New("internal copy proto_package must be a valid proto package that is not used by the API or another copy")
	errInternalCopyPackages     = errors.New("internal copy requires the API proto files to share one proto package")
	errInternalCopyNoPackage    = errors.New("internal copy requires the API proto files to declare a proto package")
	errInternalCopyFileNotFound = errors.New("internal copy proto file not found in descriptor set")
	errInternalCopyExtension    = errors.New("internal copy cannot declare an extension of a message outside the copy")
)

// apiFileDescriptors returns the descriptors of the API files in the order of
// apiFiles and checks that they declare a single proto package.
func apiFileDescriptors(fds *descriptorpb.FileDescriptorSet, apiFiles []string) ([]*descriptorpb.FileDescriptorProto, error) {
	byName := make(map[string]*descriptorpb.FileDescriptorProto, len(fds.GetFile()))
	for _, fd := range fds.GetFile() {
		byName[fd.GetName()] = fd
	}
	var apiFds []*descriptorpb.FileDescriptorProto
	for _, name := range apiFiles {
		fd, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", errInternalCopyFileNotFound, name)
		}
		if len(apiFds) > 0 && fd.GetPackage() != apiFds[0].GetPackage() {
			return nil, fmt.Errorf("%w: %q and %q", errInternalCopyPackages, apiFds[0].GetPackage(), fd.GetPackage())
		}
		apiFds = append(apiFds, fd)
	}
	if len(apiFds) == 0 {
		return nil, errInternalCopyFileNotFound
	}
	return apiFds, nil
}

// rewriteDescriptorSet rewrites the given API files in place for the internal
// copy: it renames the proto package, moves each file under the directory
// matching the new package, points go_package at the copy's import path and
// drops service definitions. Every reference to a renamed type across the set
// is updated. It returns the new file names in the order of apiFiles.
//
// Only go_package is rewritten; every other option, annotation and comment
// keeps its original value, so resource names, HTTP bindings and similar
// metadata in the copy still describe the public API.
func rewriteDescriptorSet(fds *descriptorpb.FileDescriptorSet, apiFiles []string, cp *config.GoInternalCopy) ([]string, error) {
	apiFds, err := apiFileDescriptors(fds, apiFiles)
	if err != nil {
		return nil, err
	}
	if !protoreflect.FullName(cp.ProtoPackage).IsValid() {
		return nil, fmt.Errorf("%w: %q", errInternalCopyProtoPackage, cp.ProtoPackage)
	}
	// The copy's files are registered by path and its types by full name, both
	// derived from the proto package, so the package must be new to the set.
	for _, fd := range fds.GetFile() {
		if fd.GetPackage() == cp.ProtoPackage {
			return nil, fmt.Errorf("%w: %q is the proto package of %s", errInternalCopyProtoPackage, cp.ProtoPackage, fd.GetName())
		}
	}
	oldPackage := apiFds[0].GetPackage()
	// Full names are built from the package, so a file without one would
	// yield names such as "..Foo" that match nothing in the descriptor set.
	if oldPackage == "" {
		return nil, fmt.Errorf("%w: %s", errInternalCopyNoPackage, apiFds[0].GetName())
	}

	// Type references are fully qualified names such as ".google.spanner.v1.Type".
	// Collect the exact names defined by the API files rather than rewriting by
	// prefix so that a sibling package sharing the prefix is left untouched.
	renamedTypes := make(map[string]string)
	for _, fd := range apiFds {
		collectRenamedTypes(renamedTypes, "."+oldPackage, "."+cp.ProtoPackage, fd.GetMessageType(), fd.GetEnumType())
	}
	// An extension is registered on its extendee by field number, and renaming
	// the extension does not change that key: a copy extending a message that
	// is not itself copied would register a second extension with the same
	// number and panic at init as soon as both packages are linked.
	for _, fd := range apiFds {
		if err := checkExtendees(renamedTypes, fd.GetName(), "."+oldPackage, fd.GetExtension(), fd.GetMessageType()); err != nil {
			return nil, err
		}
	}
	renamedFiles := make(map[string]string, len(apiFds))
	newDir := strings.ReplaceAll(cp.ProtoPackage, ".", "/")
	goPackage := modulePathPrefix + cp.ImportPath + ";" + path.Base(cp.ImportPath)
	renamed := make([]string, 0, len(apiFds))
	for _, fd := range apiFds {
		newName := path.Join(newDir, path.Base(fd.GetName()))
		renamedFiles[fd.GetName()] = newName
		renamed = append(renamed, newName)
		fd.Name = new(newName)
		fd.Package = new(cp.ProtoPackage)
		if fd.Options == nil {
			fd.Options = &descriptorpb.FileOptions{}
		}
		fd.Options.GoPackage = new(goPackage)
		fd.Service = nil
	}
	for _, fd := range fds.GetFile() {
		for i, dep := range fd.GetDependency() {
			if newName, ok := renamedFiles[dep]; ok {
				fd.Dependency[i] = newName
			}
		}
		rewriteMessageTypes(renamedTypes, fd.GetMessageType())
		rewriteFieldTypes(renamedTypes, fd.GetExtension())
		for _, service := range fd.GetService() {
			for _, method := range service.GetMethod() {
				method.InputType = renameType(renamedTypes, method.InputType)
				method.OutputType = renameType(renamedTypes, method.OutputType)
			}
		}
	}
	return renamed, nil
}

// checkExtendees rejects extensions, including ones declared inside messages,
// whose extendee is not among the renamed types of the copy.
func checkExtendees(renamed map[string]string, file, scope string, extensions []*descriptorpb.FieldDescriptorProto, messages []*descriptorpb.DescriptorProto) error {
	for _, ext := range extensions {
		if _, ok := renamed[ext.GetExtendee()]; !ok {
			return fmt.Errorf("%w: %s declares extension %s of %s, which would register the same extension number on the public message; internal_copies is not supported for this API", errInternalCopyExtension, file, strings.TrimPrefix(scope+"."+ext.GetName(), "."), strings.TrimPrefix(ext.GetExtendee(), "."))
		}
	}
	for _, message := range messages {
		if err := checkExtendees(renamed, file, scope+"."+message.GetName(), message.GetExtension(), message.GetNestedType()); err != nil {
			return err
		}
	}
	return nil
}

// collectRenamedTypes records the old and new fully qualified name of every
// message and enum, including nested ones, under the given package prefixes.
func collectRenamedTypes(renamed map[string]string, oldPrefix, newPrefix string, messages []*descriptorpb.DescriptorProto, enums []*descriptorpb.EnumDescriptorProto) {
	for _, enum := range enums {
		renamed[oldPrefix+"."+enum.GetName()] = newPrefix + "." + enum.GetName()
	}
	for _, message := range messages {
		oldName := oldPrefix + "." + message.GetName()
		newName := newPrefix + "." + message.GetName()
		renamed[oldName] = newName
		collectRenamedTypes(renamed, oldName, newName, message.GetNestedType(), message.GetEnumType())
	}
}

func rewriteMessageTypes(renamed map[string]string, messages []*descriptorpb.DescriptorProto) {
	for _, message := range messages {
		rewriteFieldTypes(renamed, message.GetField())
		rewriteFieldTypes(renamed, message.GetExtension())
		rewriteMessageTypes(renamed, message.GetNestedType())
	}
}

func rewriteFieldTypes(renamed map[string]string, fields []*descriptorpb.FieldDescriptorProto) {
	for _, field := range fields {
		field.TypeName = renameType(renamed, field.TypeName)
		field.Extendee = renameType(renamed, field.Extendee)
	}
}

func renameType(renamed map[string]string, name *string) *string {
	if name == nil {
		return nil
	}
	if newName, ok := renamed[*name]; ok {
		return new(newName)
	}
	return name
}

func readDescriptorSet(path string) (*descriptorpb.FileDescriptorSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fds); err != nil {
		return nil, fmt.Errorf("failed to parse descriptor set %s: %w", path, err)
	}
	return fds, nil
}

func writeDescriptorSet(path string, fds *descriptorpb.FileDescriptorSet) error {
	data, err := proto.Marshal(fds)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
