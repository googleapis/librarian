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
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestValidate(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "no libraries",
			cfg:  &config.Config{Language: config.LanguageGo},
		},
		{
			name: "library without copies",
			cfg: &config.Config{
				Language:  config.LanguageGo,
				Libraries: []*config.Library{{Name: "foo", APIs: []*config.API{{Path: "google/cloud/foo/v1"}}}},
			},
		},
		{
			name: "copy in library with explicit output",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{{
					Name:   "foo",
					Output: "foo",
					APIs: []*config.API{{
						Path: "google/cloud/foo/v1",
						Go: &config.GoAPI{
							InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
						},
					}},
				}},
			},
		},
		{
			name: "copy in library with default output",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Default:  &config.Default{Output: "."},
				Libraries: []*config.Library{{
					Name: "foo",
					APIs: []*config.API{{
						Path: "google/cloud/foo/v1",
						Go: &config.GoAPI{
							InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
						},
					}},
				}},
			},
		},
		{
			name: "copy in preview library",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{{
					Name: "foo",
					Preview: &config.Library{
						Name: "foo",
						APIs: []*config.API{{
							Path: "google/cloud/foo/v1",
							Go: &config.GoAPI{
								InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
							},
						}},
					},
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.cfg); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestValidate_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		cfg      *config.Config
		wantErrs []error
	}{
		{
			name: "null copy",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{{
					Name: "foo",
					APIs: []*config.API{{Path: "google/cloud/foo/v1", Go: &config.GoAPI{InternalCopies: []*config.GoInternalCopy{nil}}}},
				}},
			},
			wantErrs: []error{errInternalCopyNil},
		},
		{
			name: "copy outside the library",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{{
					Name: "foo",
					APIs: []*config.API{{
						Path: "google/cloud/foo/v1",
						Go: &config.GoAPI{
							InternalCopies: []*config.GoInternalCopy{{ImportPath: "other/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
						},
					}},
				}},
			},
			wantErrs: []error{errInternalCopyOutsideLibrary},
		},
		{
			name: "errors from several libraries are reported together",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{
					{
						Name: "foo",
						APIs: []*config.API{{
							Path: "google/cloud/foo/v1",
							Go: &config.GoAPI{
								InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/fastpb", ProtoPackage: "google.cloud.foo.v1"}},
							},
						}},
					},
					{
						Name: "bar",
						APIs: []*config.API{{
							Path: "google/cloud/bar/v1",
							Go: &config.GoAPI{
								InternalCopies: []*config.GoInternalCopy{{
									ImportPath:   "bar/internal/fastpb",
									ProtoPackage: "google.cloud.bar.v1.fastinternal",
									Plugins:      []*config.GoProtocPlugin{{Name: "go-vtproto", Options: []string{"module=cloud.google.com/go/bar"}}},
								}},
							},
						}},
					},
				},
			},
			wantErrs: []error{errInternalCopyProtoPackage, errProtocPluginOption},
		},
		{
			name: "copy in preview library outside the library",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{{
					Name: "foo",
					Preview: &config.Library{
						Name: "foo",
						APIs: []*config.API{{
							Path: "google/cloud/foo/v1",
							Go: &config.GoAPI{
								InternalCopies: []*config.GoInternalCopy{{ImportPath: "foo/internal/../../other/internal/fastpb", ProtoPackage: "google.cloud.foo.v1.fastinternal"}},
							},
						}},
					},
				}},
			},
			wantErrs: []error{errInternalCopyImportPath},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := Validate(test.cfg)
			for _, wantErr := range test.wantErrs {
				if !errors.Is(gotErr, wantErr) {
					t.Errorf("Validate error = %v, want it to wrap %v", gotErr, wantErr)
				}
			}
		})
	}
}
