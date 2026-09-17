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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
)

func TestValidateInternalCopies(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		outDir  string
	}{
		{
			name:    "no go api",
			library: &config.Library{Name: "foo", APIs: []*config.API{{Path: "google/cloud/foo/v1"}}},
			outDir:  "repo/foo",
		},
		{
			name: "copy inside library",
			library: &config.Library{Name: "foo", APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go: &config.GoAPI{
					ImportPath:     "foo/apiv1",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
				},
			}}},
			outDir: "repo/foo",
		},
		{
			name: "default import path",
			library: &config.Library{Name: "foo", APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go: &config.GoAPI{
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
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
						{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
						{ImportPath: "foo/internal/otherpb", ProtoPackage: "google.cloud.foo.v1.otherinternal"},
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
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
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
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "pubsub/v2/internal/fastpb", ProtoPackage: "google.pubsub.v1.fastinternal"}},
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
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "longrunning/internal/fastpb", ProtoPackage: "google.longrunning.fastinternal"}},
				},
			}}},
			outDir: "repo",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateInternalCopies(test.library, filepath.FromSlash(test.outDir)); err != nil {
				t.Error(err)
			}
			if err := validateInternalCopyPaths(test.library, filepath.FromSlash(test.outDir)); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestValidateInternalCopies_Error(t *testing.T) {
	newLibrary := func(copies ...*config.GoInternalCopy) *config.Library {
		return &config.Library{Name: "foo", APIs: []*config.API{{
			Path: "google/cloud/foo/v1",
			Go:   &config.GoAPI{ImportPath: "foo/apiv1", InternalCopies: copies},
		}}}
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
			library: newLibrary(&config.GoInternalCopy{ImportPath: "foo/internal/../../other", ProtoPackage: "google.cloud.foo.v1.fastinternal"}),
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "sibling library",
			library: newLibrary(&config.GoInternalCopy{ImportPath: "other/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}),
			wantErr: errInternalCopyOutsideLibrary,
		},
		{
			name:    "proto package of the api",
			library: newLibrary(&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1"}),
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "configured proto package of the api",
			library: &config.Library{Name: "foo", APIs: []*config.API{{
				Path: "google/cloud/foo/v1",
				Go: &config.GoAPI{
					ImportPath:     "foo/apiv1",
					ProtoPackage:   "foo.v1",
					InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1"}},
				},
			}}},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "proto package of another copy",
			library: newLibrary(
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
				&config.GoInternalCopy{ImportPath: "foo/internal/otherpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
			),
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "proto package of a copy of another api",
			library: &config.Library{Name: "foo", APIs: []*config.API{
				{
					Path: "google/cloud/foo/v1",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv1",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.fastinternal"}},
					},
				},
				{
					Path: "google/cloud/foo/v2",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv2",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastv2pb", ProtoPackage: "google.cloud.foo.fastinternal"}},
					},
				},
			}},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "duplicate import path",
			library: newLibrary(
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.otherinternal"},
			),
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "import paths differing only by case",
			library: newLibrary(
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
				&config.GoInternalCopy{ImportPath: "foo/internal/FastPB", ProtoPackage: "google.cloud.foo.v1.otherinternal"},
			),
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "nested copy directories",
			library: newLibrary(
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"},
				&config.GoInternalCopy{ImportPath: "foo/internal/fastpb/nested", ProtoPackage: "google.cloud.foo.v1.otherinternal"},
			),
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "copy of another api in the same directory",
			library: &config.Library{Name: "foo", APIs: []*config.API{
				{
					Path: "google/cloud/foo/v1",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv1",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
					},
				},
				{
					Path: "google/cloud/foo/v2",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv2",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v2.fastinternal"}},
					},
				},
			}},
			wantErr: errInternalCopyOverlap,
		},
		{
			name:    "copy inside the client directory",
			library: newLibrary(&config.GoInternalCopy{ImportPath: "foo/apiv1/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}),
			wantErr: errInternalCopyOverlap,
		},
		{
			name: "client directory of another api inside the copy",
			library: &config.Library{Name: "foo", APIs: []*config.API{
				{
					Path: "google/cloud/foo/v1",
					Go: &config.GoAPI{
						ImportPath:     "foo/apiv1",
						InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
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
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "google.cloud.foo.v1.fastinternal",
				Plugins:      []*config.GoProtocPlugin{{Name: "go-vtproto", Options: []string{"paths=source_relative"}}},
			}),
			wantErr: errProtocPluginOption,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := validateInternalCopies(test.library, filepath.FromSlash("repo/foo"))
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("validateInternalCopies error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
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
			got, err := internalCopyDir(test.library, filepath.FromSlash(test.outDir), &config.GoInternalCopy{ImportPath: test.importPath})
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(filepath.FromSlash(test.want), got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternalCopyDir_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		library    *config.Library
		outDir     string
		importPath string
	}{
		{
			name:       "sibling library",
			library:    &config.Library{Name: "foo"},
			outDir:     "repo/foo",
			importPath: "other/internal/fastpb",
		},
		{
			name:       "parent of the library",
			library:    &config.Library{Name: "foo/bar"},
			outDir:     "repo/foo/bar",
			importPath: "foo/internal/fastpb",
		},
		{
			name:       "library directory itself",
			library:    &config.Library{Name: "internal"},
			outDir:     "repo/internal",
			importPath: "internal",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, gotErr := internalCopyDir(test.library, filepath.FromSlash(test.outDir), &config.GoInternalCopy{ImportPath: test.importPath})
			if !errors.Is(gotErr, errInternalCopyOutsideLibrary) {
				t.Errorf("internalCopyDir error = %v, wantErr %v", gotErr, errInternalCopyOutsideLibrary)
			}
		})
	}
}

func TestInternalCopySymlink_Error(t *testing.T) {
	for _, test := range []struct {
		name   string
		link   string
		target string
	}{
		{name: "parent outside library", link: "internal/link", target: "../../other"},
		{name: "copy directory outside library", link: "internal/link/pb", target: "../../../other/pb"},
		{name: "parent inside library", link: "internal/link", target: "../private"},
		{name: "dangling parent", link: "internal/link", target: "../../missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, action := range []string{"Clean", "Generate"} {
				t.Run(action, func(t *testing.T) {
					root := t.TempDir()
					outDir := filepath.Join(root, "foo")
					files := []string{"other/pb/valuable.pb.go", "foo/README.md", "foo/apiv1/public.pb.go", "foo/private/pb/valuable.pb.go"}
					createFiles(t, root, files)
					link := filepath.Join(outDir, test.link)
					if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(test.target, link); err != nil {
						t.Fatal(err)
					}
					library := &config.Library{
						Name: "foo", Output: outDir,
						APIs: []*config.API{{
							Path: "foo/v1",
							Go: &config.GoAPI{
								ImportPath:     "foo/apiv1",
								InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/link/pb", ProtoPackage: "foo.private"}},
							},
						}},
					}
					// Tidy validates configuration without consulting the filesystem.
					if err := Validate(&config.Config{Libraries: []*config.Library{library}}); err != nil {
						t.Fatal(err)
					}
					var err error
					switch action {
					case "Clean":
						err = Clean(library)
					case "Generate":
						err = Generate(t.Context(), nil, library, &sources.Sources{Googleapis: googleapisDir})
					}
					if err == nil || !strings.Contains(err.Error(), link) {
						t.Errorf("%s error = %v, want rejection naming symlink %s", action, err, link)
					}
					for _, file := range files {
						got, err := os.ReadFile(filepath.Join(root, file))
						if err != nil {
							t.Error(err)
							continue
						}
						if diff := cmp.Diff("test", string(got)); diff != "" {
							t.Errorf("mismatch (-want +got):\n%s", diff)
						}
					}
					if _, err := os.Stat(filepath.Join(root, "missing")); !errors.Is(err, fs.ErrNotExist) {
						t.Errorf("missing directory: Stat error = %v, want not exist", err)
					}
				})
			}
		})
	}
}

func TestInternalCopyResolvedOverlap_Error(t *testing.T) {
	root := t.TempDir()
	library := &config.Library{
		Name: "foo", Output: filepath.Join(root, "foo"),
		APIs: []*config.API{{
			Path: "foo/v1",
			Go: &config.GoAPI{
				ImportPath:     "foo/apiv1",
				InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/pb", ProtoPackage: "foo.private"}},
			},
		}},
	}
	createFiles(t, library.Output, []string{"README.md", "internal/pb/valuable.pb.go"})
	if err := os.Symlink("internal/pb", filepath.Join(library.Output, "apiv1")); err != nil {
		t.Fatal(err)
	}
	if err := Validate(&config.Config{Libraries: []*config.Library{library}}); err != nil {
		t.Fatal(err)
	}
	if err := Clean(library); !errors.Is(err, errInternalCopyOverlap) {
		t.Errorf("Clean error = %v, wantErr %v", err, errInternalCopyOverlap)
	}
	for _, file := range []string{"README.md", "internal/pb/valuable.pb.go"} {
		if _, err := os.Stat(filepath.Join(library.Output, file)); err != nil {
			t.Error(err)
		}
	}
}

func TestInternalCopyLibrarySymlink(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	createFiles(t, realDir, []string{"internal/pb/stale.pb.go", "internal/pb/doc.go"})
	outDir := filepath.Join(root, "foo")
	if err := os.Symlink(realDir, outDir); err != nil {
		t.Fatal(err)
	}
	library := &config.Library{
		Name: "foo", Output: outDir,
		APIs: []*config.API{{
			Path: "foo/v1",
			Go: &config.GoAPI{
				ImportPath:     "foo/apiv1",
				InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/pb", ProtoPackage: "foo.private"}},
			},
		}},
	}
	if err := Clean(library); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(realDir, "internal/pb/stale.pb.go")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stale file: Stat error = %v, want not exist", err)
	}
	got, err := os.ReadFile(filepath.Join(realDir, "internal/pb/doc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff("test", string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveExistingPath(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	createFiles(t, root, []string{"real/existing/file"})
	if err := os.Symlink("real", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path string
		want string
	}{
		{path: "real/existing", want: "real/existing"},
		{path: "link/existing", want: "real/existing"},
		{path: "link/new/deep", want: "real/new/deep"},
		{path: "new/library/internal/pb", want: "new/library/internal/pb"},
	} {
		t.Run(test.path, func(t *testing.T) {
			got, err := resolveExistingPath(filepath.Join(root, test.path))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(filepath.Join(root, test.want), got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveExistingPath_Error(t *testing.T) {
	root := t.TempDir()
	createFiles(t, root, []string{"file"})
	if err := os.Symlink("missing", filepath.Join(root, "dangling")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"file/child", "dangling/child"} {
		t.Run(path, func(t *testing.T) {
			_, err := resolveExistingPath(filepath.Join(root, path))
			if _, ok := errors.AsType[*os.PathError](err); !ok {
				t.Errorf("resolveExistingPath error = %v, want *os.PathError", err)
			}
		})
	}
}
