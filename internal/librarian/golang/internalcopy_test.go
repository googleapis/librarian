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
	"io/fs"
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

func TestDescriptorSetFile(t *testing.T) {
	want := testDescriptorSet()
	path := filepath.Join(t.TempDir(), "api.pb")
	if err := writeDescriptorSet(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := readDescriptorSet(path)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDescriptorSetFile_Error(t *testing.T) {
	if _, err := readDescriptorSet(filepath.Join(t.TempDir(), "missing.pb")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("readDescriptorSet error = %v, wantErr %v", err, fs.ErrNotExist)
	}
}

func TestRewriteDescriptorSet_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		apiFiles []string
		cp       *config.GoInternalCopy
		setup    func(fds *descriptorpb.FileDescriptorSet)
		wantErr  error
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

func TestInternalCopyDir(t *testing.T) {
	for _, test := range []struct {
		name       string
		library    *config.Library
		outDir     string
		importPath string
		want       string
	}{
		{
			name:       "library",
			library:    &config.Library{Name: "foo"},
			outDir:     "repo/foo",
			importPath: "foo/internal/fastpb",
			want:       "repo/foo/internal/fastpb",
		},
		{
			name:       "nested library",
			library:    &config.Library{Name: "foo/bar"},
			outDir:     "repo/foo/bar",
			importPath: "foo/bar/internal/fastpb",
			want:       "repo/foo/bar/internal/fastpb",
		},
		{
			name:       "preview library",
			library:    &config.Library{Name: "foo", Output: "preview/internal/foo"},
			outDir:     "preview/internal/foo",
			importPath: "foo/internal/fastpb",
			want:       "preview/internal/foo/internal/fastpb",
		},
		{
			name:       "module path version",
			library:    &config.Library{Name: "pubsub", Go: &config.GoModule{ModulePathVersion: "v2"}},
			outDir:     "repo/pubsub",
			importPath: "pubsub/v2/internal/fastpb",
			want:       "repo/pubsub/internal/fastpb",
		},
		{
			name:       "absolute output",
			library:    &config.Library{Name: "foo"},
			outDir:     "/repo/foo",
			importPath: "foo/internal/fastpb",
			want:       "/repo/foo/internal/fastpb",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := internalCopyDir(test.library, filepath.FromSlash(test.outDir), &config.GoInternalCopy{ImportPath: test.importPath})
			if diff := cmp.Diff(filepath.FromSlash(test.want), got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateInternalCopies(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		outDir  string
	}{
		{
			name: "no copies",
			library: &config.Library{Name: "foo", APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go:   &config.GoAPI{ImportPath: "foo/apiv1"},
			}}},
			outDir: "repo/foo",
		},
		{
			name: "copy inside library",
			library: &config.Library{Name: "foo", APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go: &config.GoAPI{
					ImportPath:     "foo/apiv1",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
				},
			}}},
			outDir: "repo/foo",
		},
		{
			name: "kept file outside the copy",
			library: &config.Library{Name: "foo", Keep: []string{"internal/helpers/handwritten.go"}, APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go: &config.GoAPI{
					ImportPath:     "foo/apiv1",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
				},
			}}},
			outDir: "repo/foo",
		},
		{
			name: "two copies",
			library: &config.Library{Name: "foo", APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go: &config.GoAPI{
					ImportPath: "foo/apiv1",
					InternalCopies: []*config.GoInternalCopy{
						{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
						{ImportPath: "foo/internal/otherpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.otherinternal"},
					},
				},
			}}},
			outDir: "repo/foo",
		},
		{
			name: "preview library",
			library: &config.Library{Name: "foo", APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go: &config.GoAPI{
					ImportPath:     "foo/apiv1",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
				},
			}}},
			outDir: "preview/internal/foo",
		},
		{
			name: "module path version",
			library: &config.Library{
				Name: "pubsub",
				Go:   &config.GoModule{ModulePathVersion: "v2"},
				APIs: []*config.API{{
					Path: "google/pubsub/v1",
					Go: &config.GoAPI{
						ImportPath:     "pubsub/v2/apiv1",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "pubsub/v2/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.pubsub.v1.fastinternal"}},
					},
				}},
			},
			outDir: "repo/pubsub",
		},
		{
			name: "root module",
			library: &config.Library{Name: rootModule, APIs: []*config.API{{
				Path: "google/longrunning",
				Go: &config.GoAPI{
					ImportPath:     "longrunning/autogen",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "longrunning/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.longrunning.fastinternal"}},
				},
			}}},
			outDir: "repo",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.library.Output = filepath.FromSlash(test.outDir)
			if err := validateInternalCopies(test.library); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestValidateInternalCopies_Error(t *testing.T) {
	newLibrary := func(copies ...*config.GoInternalCopy) *config.Library {
		return &config.Library{Name: "foo", Output: filepath.FromSlash("repo/foo"), APIs: []*config.API{{
			Path: "google/cloud/foo/v1",
			Go:   &config.GoAPI{ImportPath: "foo/apiv1", InternalCopies: copies},
		}}}
	}
	withKeep := func(library *config.Library, keep ...string) *config.Library {
		library.Keep = keep
		return library
	}
	for _, test := range []struct {
		name    string
		library *config.Library
		wantErr error
	}{
		{
			name:    "null copy",
			library: newLibrary(nil),
			wantErr: errInternalCopyNil,
		},
		{
			name:    "parent element",
			library: newLibrary(&config.GoInternalCopy{ImportPath: "foo/internal/../../other", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}),
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "sibling library",
			library: newLibrary(&config.GoInternalCopy{ImportPath: "other/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}),
			wantErr: errInternalCopyOutsideLibrary,
		},
		{
			name: "parent of the library",
			library: &config.Library{Name: "foo/bar", Output: filepath.FromSlash("repo/foo/bar"), APIs: []*config.API{{
				Path: "google/cloud/foo/bar/v1",
				Go: &config.GoAPI{
					ImportPath:     "foo/bar/apiv1",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.bar.v1.fastinternal"}},
				},
			}}},
			wantErr: errInternalCopyOutsideLibrary,
		},
		{
			name: "library directory itself",
			library: &config.Library{Name: "internal", Output: filepath.FromSlash("repo/internal"), APIs: []*config.API{{
				Path: "google/cloud/internal/v1",
				Go: &config.GoAPI{
					ImportPath:     "internal/apiv1",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "internal", Plugin: "go-vtproto", ProtoPackage: "google.cloud.internal.v1.fastinternal"}},
				},
			}}},
			wantErr: errInternalCopyOutsideLibrary,
		},
		{
			name:    "kept file under the copy",
			library: withKeep(newLibrary(&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}), "internal/fastpb/handwritten.go"),
			wantErr: errInternalCopyKeep,
		},
		{
			name:    "proto package of the api",
			library: newLibrary(&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1"}),
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "configured proto package of the api",
			library: &config.Library{Name: "foo", Output: filepath.FromSlash("repo/foo"), APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go: &config.GoAPI{
					ImportPath:     "foo/apiv1",
					ProtoPackage:   "foo.v1",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1"}},
				},
			}}},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "proto package of another copy",
			library: newLibrary(
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
				&config.GoInternalCopy{ImportPath: "foo/internal/otherpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
			),
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "proto package of a copy of another api",
			library: &config.Library{Name: "foo", Output: filepath.FromSlash("repo/foo"), APIs: []*config.API{
				{
					Path: "google/cloud/foo/v1",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv1",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.fastinternal"}},
					},
				},
				{
					Path: "google/cloud/foo/v2",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv2",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastv2pb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.fastinternal"}},
					},
				},
			}},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "duplicate import path",
			library: newLibrary(
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.otherinternal"},
			),
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "import paths differing only by case",
			library: newLibrary(
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
				&config.GoInternalCopy{ImportPath: "foo/internal/FastPB", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.otherinternal"},
			),
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "nested copy directories",
			library: newLibrary(
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb/nested", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.otherinternal"},
			),
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "copy of another api in the same directory",
			library: &config.Library{Name: "foo", Output: filepath.FromSlash("repo/foo"), APIs: []*config.API{
				{
					Path: "google/cloud/foo/v1",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv1",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
					},
				},
				{
					Path: "google/cloud/foo/v2",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv2",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v2.fastinternal"}},
					},
				},
			}},
			wantErr: errInternalCopyOverlap,
		},
		{
			name:    "copy inside the client directory",
			library: newLibrary(&config.GoInternalCopy{ImportPath: "foo/apiv1/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}),
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "client directory of another api inside the copy",
			library: &config.Library{Name: "foo", Output: filepath.FromSlash("repo/foo"), APIs: []*config.API{
				{
					Path: "google/cloud/foo/v1",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv1",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal", Plugin: "go-vtproto", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
					},
				},
				{
					Path: "google/cloud/foo/v1beta",
					Go:   &config.GoAPI{ImportPath: "foo/internal/apiv1beta"},
				},
			}},
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "malformed plugin",
			library: newLibrary(&config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "google.cloud.foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"paths=source_relative"},
			}),
			wantErr: errProtocPluginOption,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := validateInternalCopies(test.library)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("validateInternalCopies error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestValidateInternalCopyConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		cp   *config.GoInternalCopy
	}{
		{
			name: "minimal",
			cp:   &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
		},
		{
			name: "internal as last element",
			cp:   &config.GoInternalCopy{ImportPath: "foo/internal", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
		},
		{
			name: "plugin with underscore",
			cp:   &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go_json", ProtoPackage: "foo.v1.fastinternal"},
		},
		{
			name: "plugin with comma-packed options",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"features=marshal+unmarshal+size+pool,pool=cloud.google.com/go/foo/internal/fastpb.A"},
			},
		},
		{
			name: "plugin with options",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"features=marshal+unmarshal+size+pool", "pool=cloud.google.com/go/foo/internal/fastpb.A"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateInternalCopyConfig(test.cp); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestValidateInternalCopyConfig_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		cp      *config.GoInternalCopy
		wantErr error
	}{
		{
			name:    "null copy",
			wantErr: errInternalCopyNil,
		},
		{
			name:    "empty import path",
			cp:      &config.GoInternalCopy{Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "absolute import path",
			cp:      &config.GoInternalCopy{ImportPath: "/foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "backslash in import path",
			cp:      &config.GoInternalCopy{ImportPath: `foo\internal\fastpb`, Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "parent element",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/../publicpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "leading parent element",
			cp:      &config.GoInternalCopy{ImportPath: "../foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "dot element",
			cp:      &config.GoInternalCopy{ImportPath: "foo/./internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "trailing slash",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb/", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "double slash",
			cp:      &config.GoInternalCopy{ImportPath: "foo//internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "import path without internal element",
			cp:      &config.GoInternalCopy{ImportPath: "foo/apiv1/fastpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "internal only as a name fragment",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internalpb", Plugin: "go-vtproto", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "empty proto package",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb"},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name:    "invalid proto package",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: "invalid/package"},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name:    "proto package with leading dot",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", Plugin: "go-vtproto", ProtoPackage: ".foo.v1.fastinternal"},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "empty plugin name",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
			},
			wantErr: errProtocPluginName,
		},
		{
			name: "plugin name with path",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
				Plugin:       "../go-vtproto",
			},
			wantErr: errProtocPluginName,
		},
		{
			name: "paths option",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"paths=source_relative"},
			},
			wantErr: errProtocPluginOption,
		},
		{
			name: "paths option leading a parameter list",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"paths=source_relative,features=marshal"},
			},
			wantErr: errProtocPluginOption,
		},
		{
			name: "paths option inside a parameter list",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"features=marshal,paths=source_relative,allow-empty=true"},
			},
			wantErr: errProtocPluginOption,
		},
		{
			name: "module option",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"features=marshal", "module=cloud.google.com/go/foo"},
			},
			wantErr: errProtocPluginOption,
		},
		{
			name: "module option trailing a parameter list",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"features=marshal,module=cloud.google.com/go"},
			},
			wantErr: errProtocPluginOption,
		},
		{
			name: "import mapping option",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"Mfoo.proto=example.com/elsewhere"},
			},
			wantErr: errProtocPluginOption,
		},
		{
			name: "import mapping option inside a parameter list",
			cp: &config.GoInternalCopy{
				ImportPath:    "foo/internal/fastpb",
				ProtoPackage:  "foo.v1.fastinternal",
				Plugin:        "go-vtproto",
				PluginOptions: []string{"pool=cloud.google.com/go/foo/internal/fastpb.Item", "features=marshal,Mfoo.proto=example.com/elsewhere,allow-empty=true"},
			},
			wantErr: errProtocPluginOption,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := validateInternalCopyConfig(test.cp)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("validateInternalCopyConfig error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}
