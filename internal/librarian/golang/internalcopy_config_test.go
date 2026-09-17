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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestValidateInternalCopyConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		cp   *config.GoInternalCopy
	}{
		{
			name: "minimal",
			cp:   &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
		},
		{
			name: "internal as last element",
			cp:   &config.GoInternalCopy{ImportPath: "foo/internal", ProtoPackage: "foo.v1.fastinternal"},
		},
		{
			name: "plugin with comma-packed options",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
				Plugins: []*config.GoProtocPlugin{{
					Name:    "go-vtproto",
					Options: []string{"features=marshal+unmarshal+size+pool,pool=cloud.google.com/go/foo/internal/fastpb.A"},
				}},
			},
		},
		{
			name: "plugin with options",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
				Plugins: []*config.GoProtocPlugin{{
					Name:    "go-vtproto",
					Options: []string{"features=marshal+unmarshal+size+pool", "pool=cloud.google.com/go/foo/internal/fastpb.A"},
				}, {
					Name: "go_json",
				}},
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
			cp:      &config.GoInternalCopy{ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "absolute import path",
			cp:      &config.GoInternalCopy{ImportPath: "/foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "backslash in import path",
			cp:      &config.GoInternalCopy{ImportPath: `foo\internal\fastpb`, ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "parent element",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/../publicpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "leading parent element",
			cp:      &config.GoInternalCopy{ImportPath: "../foo/internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "dot element",
			cp:      &config.GoInternalCopy{ImportPath: "foo/./internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "trailing slash",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb/", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "double slash",
			cp:      &config.GoInternalCopy{ImportPath: "foo//internal/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "import path without internal element",
			cp:      &config.GoInternalCopy{ImportPath: "foo/apiv1/fastpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "internal only as a name fragment",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internalpb", ProtoPackage: "foo.v1.fastinternal"},
			wantErr: errInternalCopyImportPath,
		},
		{
			name:    "empty proto package",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb"},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name:    "invalid proto package",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: "invalid/package"},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name:    "proto package with leading dot",
			cp:      &config.GoInternalCopy{ImportPath: "foo/internal/fastpb", ProtoPackage: ".foo.v1.fastinternal"},
			wantErr: errInternalCopyProtoPackage,
		},
		{
			name: "null plugin",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
				Plugins:      []*config.GoProtocPlugin{nil},
			},
			wantErr: errProtocPluginNil,
		},
		{
			name: "empty plugin name",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
				Plugins:      []*config.GoProtocPlugin{{}},
			},
			wantErr: errProtocPluginName,
		},
		{
			name: "plugin name with path",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
				Plugins:      []*config.GoProtocPlugin{{Name: "../go-vtproto"}},
			},
			wantErr: errProtocPluginName,
		},
		{
			name: "paths option",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
				Plugins:      []*config.GoProtocPlugin{{Name: "go-vtproto", Options: []string{"paths=source_relative"}}},
			},
			wantErr: errProtocPluginOption,
		},
		{
			name: "module option",
			cp: &config.GoInternalCopy{
				ImportPath:   "foo/internal/fastpb",
				ProtoPackage: "foo.v1.fastinternal",
				Plugins:      []*config.GoProtocPlugin{{Name: "go-vtproto", Options: []string{"features=marshal", "module=cloud.google.com/go/foo"}}},
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

func TestResolveProtocPlugins_Error(t *testing.T) {
	t.Setenv(cache.EnvLibrarianBin, t.TempDir())
	t.Setenv("PATH", t.TempDir())
	for _, test := range []struct {
		name    string
		plugins []*config.GoProtocPlugin
		wantErr error
	}{
		{
			name:    "null plugin",
			plugins: []*config.GoProtocPlugin{nil},
			wantErr: errProtocPluginNil,
		},
		{
			name:    "plugin not installed",
			plugins: []*config.GoProtocPlugin{{Name: "does-not-exist"}},
			wantErr: errProtocPluginNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, gotErr := resolveProtocPlugins(test.plugins)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("resolveProtocPlugins error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestProtocPluginPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping executable lookup test on Windows")
	}
	binDir := t.TempDir()
	pathDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, binDir)
	t.Setenv("PATH", pathDir)
	if err := os.MkdirAll(filepath.Join(binDir, toolsDir), 0o755); err != nil {
		t.Fatal(err)
	}
	toolPlugin := filepath.Join(binDir, toolsDir, "protoc-gen-tool")
	testhelper.WriteExecutable(t, toolPlugin, "#!/bin/sh\nexit 0\n")
	pathPlugin := filepath.Join(pathDir, "protoc-gen-path")
	testhelper.WriteExecutable(t, pathPlugin, "#!/bin/sh\nexit 0\n")
	for _, test := range []struct {
		name   string
		plugin string
		want   string
	}{
		{"go tool bin dir", "tool", toolPlugin},
		{"PATH fallback", "path", pathPlugin},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := protocPluginPath(test.plugin)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateInternalCopyConfig_LayoutOption(t *testing.T) {
	for _, layout := range []string{"paths=source_relative", "module=cloud.google.com/go", "Mfoo.proto=example.com/elsewhere"} {
		for _, option := range []string{layout, layout + ",features=marshal", "features=marshal," + layout, "features=marshal," + layout + ",allow-empty=true"} {
			t.Run(option, func(t *testing.T) {
				cp := &config.GoInternalCopy{
					ImportPath: "foo/internal/pb", ProtoPackage: "foo.private",
					Plugins: []*config.GoProtocPlugin{{Name: "go-vtproto", Options: []string{"pool=cloud.google.com/go/foo/internal/pb.Item", option}}},
				}
				if err := validateInternalCopyConfig(cp); !errors.Is(err, errProtocPluginOption) {
					t.Errorf("validateInternalCopyConfig error = %v, wantErr %v", err, errProtocPluginOption)
				}
			})
		}
	}
}
