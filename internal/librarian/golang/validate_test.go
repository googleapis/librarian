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
		copy *config.GoOpaqueCopy
	}{
		{name: "no opaque copy"},
		{
			name: "default proto package",
			copy: &config.GoOpaqueCopy{ImportPath: "spanner/internal/opaquepb"},
		},
		{
			name: "explicit proto package and extra proto",
			copy: &config.GoOpaqueCopy{
				ExtraProtos:  []string{"google/protobuf/struct.proto"},
				ImportPath:   "spanner/internal/opaquepb",
				ProtoPackage: "google.spanner.v1.internalopaque",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := goConfigWithOpaqueCopy(test.copy)
			if err := Validate(cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidate_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		copy    *config.GoOpaqueCopy
		wantErr error
	}{
		{
			name:    "missing import path",
			copy:    &config.GoOpaqueCopy{},
			wantErr: errOpaqueCopyImportPath,
		},
		{
			name:    "absolute import path",
			copy:    &config.GoOpaqueCopy{ImportPath: "/spanner/internal/opaquepb"},
			wantErr: errOpaqueCopyImportPath,
		},
		{
			name: "invalid proto package",
			copy: &config.GoOpaqueCopy{
				ImportPath:   "spanner/internal/opaquepb",
				ProtoPackage: "google.spanner.1internal",
			},
			wantErr: errOpaqueCopyProtoPackage,
		},
		{
			name: "invalid extra proto path",
			copy: &config.GoOpaqueCopy{
				ExtraProtos: []string{"../struct.proto"},
				ImportPath:  "spanner/internal/opaquepb",
			},
			wantErr: errOpaqueCopyExtraProto,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(goConfigWithOpaqueCopy(test.copy))
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func goConfigWithOpaqueCopy(copy *config.GoOpaqueCopy) *config.Config {
	return &config.Config{
		Language: config.LanguageGo,
		Libraries: []*config.Library{{
			Name: "spanner",
			APIs: []*config.API{{
				Path: "google/spanner/v1",
				Go:   &config.GoAPI{OpaqueCopy: copy},
			}},
		}},
	}
}
