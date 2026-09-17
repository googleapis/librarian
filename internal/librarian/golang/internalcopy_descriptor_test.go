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
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestRewriteDescriptorSet(t *testing.T) {
	cp := &config.GoInternalCopy{
		ImportPath:   "foo/internal/fastpb",
		ProtoPackage: "foo.v1.fastinternal",
	}
	fds := testDescriptorSet()
	file := filepath.Join(t.TempDir(), "api.pb")
	if err := writeDescriptorSet(file, fds); err != nil {
		t.Fatal(err)
	}
	fds, err := readDescriptorSet(file)
	if err != nil {
		t.Fatal(err)
	}
	got, err := rewriteDescriptorSet(fds, []string{"foo/v1/a.proto", "foo/v1/b.proto"}, cp)
	if err != nil {
		t.Fatal(err)
	}
	wantRenamed := []string{"foo/v1/fastinternal/a.proto", "foo/v1/fastinternal/b.proto"}
	if diff := cmp.Diff(wantRenamed, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	want := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:    new("google/protobuf/struct.proto"),
				Package: new("google.protobuf"),
				Options: &descriptorpb.FileOptions{
					GoPackage: new("google.golang.org/protobuf/types/known/structpb"),
				},
				MessageType: []*descriptorpb.DescriptorProto{{Name: new("Value")}},
			},
			{
				Name:        new("foo/v1beta/d.proto"),
				Package:     new("foo.v1beta"),
				MessageType: []*descriptorpb.DescriptorProto{{Name: new("D")}},
			},
			{
				Name:       new("foo/v1/fastinternal/a.proto"),
				Package:    new("foo.v1.fastinternal"),
				Dependency: []string{"google/protobuf/struct.proto", "foo/v1beta/d.proto"},
				Options: &descriptorpb.FileOptions{
					GoPackage:   new("cloud.google.com/go/foo/internal/fastpb;fastpb"),
					JavaPackage: new("com.foo.v1"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("A"),
						Field: []*descriptorpb.FieldDescriptorProto{
							testField("inner", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".foo.v1.fastinternal.A.Inner"),
							testField("kind", 2, descriptorpb.FieldDescriptorProto_TYPE_ENUM, ".foo.v1.fastinternal.Kind"),
							testField("value", 3, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".google.protobuf.Value"),
							testField("d", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".foo.v1beta.D"),
						},
						NestedType: []*descriptorpb.DescriptorProto{
							{
								Name: new("Inner"),
								Field: []*descriptorpb.FieldDescriptorProto{
									testField("mode", 1, descriptorpb.FieldDescriptorProto_TYPE_ENUM, ".foo.v1.fastinternal.A.Inner.Mode"),
								},
								EnumType: []*descriptorpb.EnumDescriptorProto{testEnum("Mode")},
							},
						},
					},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{testEnum("Kind")},
			},
			{
				Name:       new("foo/v1/fastinternal/b.proto"),
				Package:    new("foo.v1.fastinternal"),
				Dependency: []string{"foo/v1/fastinternal/a.proto"},
				Options: &descriptorpb.FileOptions{
					GoPackage: new("cloud.google.com/go/foo/internal/fastpb;fastpb"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("B"),
						Field: []*descriptorpb.FieldDescriptorProto{
							testField("a", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".foo.v1.fastinternal.A"),
						},
					},
				},
				Extension: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     new("ext"),
						Number:   new(int32(100)),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: new(".foo.v1.fastinternal.B"),
						Extendee: new(".foo.v1.fastinternal.A"),
					},
				},
			},
			{
				Name:       new("bar/v1/c.proto"),
				Package:    new("bar.v1"),
				Dependency: []string{"foo/v1/fastinternal/b.proto", "foo/v1/fastinternal/a.proto"},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("C"),
						Field: []*descriptorpb.FieldDescriptorProto{
							testField("b", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".foo.v1.fastinternal.B"),
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: new("BarService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:       new("Get"),
								InputType:  new(".foo.v1.fastinternal.A"),
								OutputType: new(".bar.v1.C"),
							},
						},
					},
				},
			},
		},
	}
	if diff := cmp.Diff(want, fds, protocmp.Transform()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRewriteDescriptorSet_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		apiFiles []string
		cp       *config.GoInternalCopy
		setup    func(fds *descriptorpb.FileDescriptorSet)
		wantErr  error
		wantMsg  []string
	}{
		{
			name:     "api file not in set",
			apiFiles: []string{"foo/v1/missing.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr:  errInternalCopyFileNotFound,
		},
		{
			name:    "no api files",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyFileNotFound,
		},
		{
			name:     "api files in different packages",
			apiFiles: []string{"foo/v1/a.proto", "bar/v1/c.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr:  errInternalCopyPackages,
		},
		{
			name:     "same proto package",
			apiFiles: []string{"foo/v1/a.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1"},
			wantErr:  errInternalCopyProtoPackage,
		},
		{
			name:     "proto package of a dependency",
			apiFiles: []string{"foo/v1/a.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.protobuf"},
			wantErr:  errInternalCopyProtoPackage,
			wantMsg:  []string{"google/protobuf/struct.proto"},
		},
		{
			name:     "empty proto package",
			apiFiles: []string{"foo/v1/a.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb"},
			wantErr:  errInternalCopyProtoPackage,
		},
		{
			name:     "api file without a proto package",
			apiFiles: []string{"foo/v1/a.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			setup: func(fds *descriptorpb.FileDescriptorSet) {
				for _, fd := range fds.File {
					if fd.GetName() == "foo/v1/a.proto" {
						fd.Package = nil
					}
				}
			},
			wantErr: errInternalCopyNoPackage,
			wantMsg: []string{"foo/v1/a.proto"},
		},
		{
			name:     "invalid proto package",
			apiFiles: []string{"foo/v1/a.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo/v1"},
			wantErr:  errInternalCopyProtoPackage,
		},
		{
			name:     "extension of a message outside the copy",
			apiFiles: []string{"foo/v1/a.proto", "foo/v1/b.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			setup: func(fds *descriptorpb.FileDescriptorSet) {
				b := fds.File[3]
				b.Extension = append(b.Extension, &descriptorpb.FieldDescriptorProto{
					Name:     new("note"),
					Number:   new(int32(50001)),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Extendee: new(".foo.v1beta.D"),
				})
			},
			wantErr: errInternalCopyExtension,
			wantMsg: []string{"foo/v1/b.proto", "extension foo.v1.note", "of foo.v1beta.D"},
		},
		{
			name:     "nested extension of a message outside the copy",
			apiFiles: []string{"foo/v1/a.proto", "foo/v1/b.proto"},
			cp:       &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			setup: func(fds *descriptorpb.FileDescriptorSet) {
				inner := fds.File[2].MessageType[0].NestedType[0]
				inner.Extension = append(inner.Extension, &descriptorpb.FieldDescriptorProto{
					Name:     new("note"),
					Number:   new(int32(50001)),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Extendee: new(".google.protobuf.Value"),
				})
			},
			wantErr: errInternalCopyExtension,
			wantMsg: []string{"foo/v1/a.proto", "extension foo.v1.A.Inner.note", "of google.protobuf.Value"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fds := testDescriptorSet()
			if test.setup != nil {
				test.setup(fds)
			}
			_, gotErr := rewriteDescriptorSet(fds, test.apiFiles, test.cp)
			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("rewriteDescriptorSet error = %v, wantErr %v", gotErr, test.wantErr)
			}
			for _, want := range test.wantMsg {
				if !strings.Contains(gotErr.Error(), want) {
					t.Errorf("error %q does not mention %q", gotErr, want)
				}
			}
		})
	}
}

// testDescriptorSet builds a descriptor set with an API package foo.v1 spread
// over two files, a sibling package sharing its name prefix, a well-known type
// dependency and a dependent package that references the API types.
func testDescriptorSet() *descriptorpb.FileDescriptorSet {
	return &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:    new("google/protobuf/struct.proto"),
				Package: new("google.protobuf"),
				Options: &descriptorpb.FileOptions{
					GoPackage: new("google.golang.org/protobuf/types/known/structpb"),
				},
				MessageType: []*descriptorpb.DescriptorProto{{Name: new("Value")}},
			},
			{
				Name:        new("foo/v1beta/d.proto"),
				Package:     new("foo.v1beta"),
				MessageType: []*descriptorpb.DescriptorProto{{Name: new("D")}},
			},
			{
				Name:       new("foo/v1/a.proto"),
				Package:    new("foo.v1"),
				Dependency: []string{"google/protobuf/struct.proto", "foo/v1beta/d.proto"},
				Options: &descriptorpb.FileOptions{
					GoPackage:   new("cloud.google.com/go/foo/apiv1/foopb;foopb"),
					JavaPackage: new("com.foo.v1"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("A"),
						Field: []*descriptorpb.FieldDescriptorProto{
							testField("inner", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".foo.v1.A.Inner"),
							testField("kind", 2, descriptorpb.FieldDescriptorProto_TYPE_ENUM, ".foo.v1.Kind"),
							testField("value", 3, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".google.protobuf.Value"),
							testField("d", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".foo.v1beta.D"),
						},
						NestedType: []*descriptorpb.DescriptorProto{
							{
								Name: new("Inner"),
								Field: []*descriptorpb.FieldDescriptorProto{
									testField("mode", 1, descriptorpb.FieldDescriptorProto_TYPE_ENUM, ".foo.v1.A.Inner.Mode"),
								},
								EnumType: []*descriptorpb.EnumDescriptorProto{testEnum("Mode")},
							},
						},
					},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{testEnum("Kind")},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: new("FooService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:       new("Get"),
								InputType:  new(".foo.v1.A"),
								OutputType: new(".foo.v1.A"),
							},
						},
					},
				},
			},
			{
				Name:       new("foo/v1/b.proto"),
				Package:    new("foo.v1"),
				Dependency: []string{"foo/v1/a.proto"},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("B"),
						Field: []*descriptorpb.FieldDescriptorProto{
							testField("a", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".foo.v1.A"),
						},
					},
				},
				Extension: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     new("ext"),
						Number:   new(int32(100)),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: new(".foo.v1.B"),
						Extendee: new(".foo.v1.A"),
					},
				},
			},
			{
				Name:       new("bar/v1/c.proto"),
				Package:    new("bar.v1"),
				Dependency: []string{"foo/v1/b.proto", "foo/v1/a.proto"},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("C"),
						Field: []*descriptorpb.FieldDescriptorProto{
							testField("b", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".foo.v1.B"),
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: new("BarService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:       new("Get"),
								InputType:  new(".foo.v1.A"),
								OutputType: new(".bar.v1.C"),
							},
						},
					},
				},
			},
		},
	}
}

func testField(name string, number int32, typ descriptorpb.FieldDescriptorProto_Type, typeName string) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     new(name),
		Number:   new(number),
		Type:     typ.Enum(),
		TypeName: new(typeName),
	}
}

func testEnum(name string) *descriptorpb.EnumDescriptorProto {
	return &descriptorpb.EnumDescriptorProto{
		Name: new(name),
		Value: []*descriptorpb.EnumValueDescriptorProto{
			{Name: new(strings.ToUpper(name) + "_UNSPECIFIED"), Number: new(int32(0))},
		},
	}
}
