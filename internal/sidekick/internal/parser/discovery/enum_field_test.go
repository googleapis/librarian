// Copyright 2025 Google LLC
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

package discovery

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/api/apitest"
)

func TestMakeEnumFields(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "networkTier",
				Schema: &schema{
					Description: "The networking tier.",
					Enums: []string{
						"FIXED_STANDARD",
						"PREMIUM",
						"STANDARD",
						"STANDARD_OVERRIDES_FIXED_STANDARD",
					},
					EnumDescriptions: []string{
						"Public internet quality with fixed bandwidth.",
						"High quality, Google-grade network tier, support for all networking products.",
						"Public internet quality, only limited support for other networking products.",
						"(Output only) Temporary tier for FIXED_STANDARD when fixed standard tier is expired or not configured.",
					},
					Type: "string",
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantEnum := &api.Enum{
		Name:          "networkTier",
		ID:            ".package.Message.networkTier",
		Documentation: "The enumerated type for the [networkTier][package.Message.networkTier] field.",
		Values: []*api.EnumValue{
			{
				Name:          "FIXED_STANDARD",
				ID:            ".package.Message.networkTier.FIXED_STANDARD",
				Number:        0,
				Documentation: "Public internet quality with fixed bandwidth.",
			},
			{
				Name:          "PREMIUM",
				ID:            ".package.Message.networkTier.PREMIUM",
				Number:        1,
				Documentation: "High quality, Google-grade network tier, support for all networking products.",
			},
			{
				Name:          "STANDARD",
				ID:            ".package.Message.networkTier.STANDARD",
				Number:        2,
				Documentation: "Public internet quality, only limited support for other networking products.",
			},
			{
				Name:          "STANDARD_OVERRIDES_FIXED_STANDARD",
				ID:            ".package.Message.networkTier.STANDARD_OVERRIDES_FIXED_STANDARD",
				Number:        3,
				Documentation: "(Output only) Temporary tier for FIXED_STANDARD when fixed standard tier is expired or not configured.",
			},
		},
	}
	wantEnum.UniqueNumberValues = wantEnum.Values
	gotEnum, ok := model.State.EnumByID[wantEnum.ID]
	if !ok {
		t.Fatalf("missing enum %s", wantEnum.ID)
	}
	apitest.CheckEnum(t, *gotEnum, *wantEnum)
	if gotEnum.Parent == nil {
		t.Errorf("expected non-nil parent in enum: %v", gotEnum)
	}
	for _, value := range gotEnum.Values {
		if value.Parent != gotEnum {
			t.Errorf("mismatched parent in enumValue: %v", value)
		}
	}

	want := &api.Message{
		ID: ".package.Message",
		Fields: []*api.Field{
			{
				Name:          "networkTier",
				JSONName:      "networkTier",
				ID:            ".package.Message.networkTier",
				Documentation: "The networking tier.",
				Typez:         api.ENUM_TYPE,
				TypezID:       ".package.Message.networkTier",
				Optional:      true,
			},
		},
	}
	apitest.CheckMessage(t, message, want)
	wantEnums := []*api.Enum{wantEnum}
	if diff := cmp.Diff(wantEnums, message.Enums, cmpopts.IgnoreFields(api.Enum{}, "Parent"), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestMakeEnumFieldsDeprecated(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "networkTier",
				Schema: &schema{
					Description: "The networking tier.",
					Deprecated:  true,
					Enums: []string{
						"FIXED_STANDARD",
						"PREMIUM",
					},
					EnumDescriptions: []string{
						"Public internet quality with fixed bandwidth.",
						"High quality, Google-grade network tier, support for all networking products.",
					},
					Type: "string",
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantEnum := &api.Enum{
		Name:          "networkTier",
		ID:            ".package.Message.networkTier",
		Documentation: "The enumerated type for the [networkTier][package.Message.networkTier] field.",
		Deprecated:    true,
		Values: []*api.EnumValue{
			{
				Name:          "FIXED_STANDARD",
				ID:            ".package.Message.networkTier.FIXED_STANDARD",
				Number:        0,
				Documentation: "Public internet quality with fixed bandwidth.",
			},
			{
				Name:          "PREMIUM",
				ID:            ".package.Message.networkTier.PREMIUM",
				Number:        1,
				Documentation: "High quality, Google-grade network tier, support for all networking products.",
			},
		},
	}
	wantEnum.UniqueNumberValues = wantEnum.Values
	gotEnum, ok := model.State.EnumByID[wantEnum.ID]
	if !ok {
		t.Fatalf("missing enum %s", wantEnum.ID)
	}
	apitest.CheckEnum(t, *gotEnum, *wantEnum)
	if gotEnum.Parent == nil {
		t.Errorf("expected non-nil parent in enum: %v", gotEnum)
	}
	for _, value := range gotEnum.Values {
		if value.Parent != gotEnum {
			t.Errorf("mismatched parent in enumValue: %v", value)
		}
	}

	want := &api.Message{
		ID: ".package.Message",
		Fields: []*api.Field{
			{
				Name:          "networkTier",
				JSONName:      "networkTier",
				ID:            ".package.Message.networkTier",
				Documentation: "The networking tier.",
				Deprecated:    true,
				Typez:         api.ENUM_TYPE,
				TypezID:       ".package.Message.networkTier",
				Optional:      true,
			},
		},
	}
	apitest.CheckMessage(t, message, want)
	wantEnums := []*api.Enum{wantEnum}
	if diff := cmp.Diff(wantEnums, message.Enums, cmpopts.IgnoreFields(api.Enum{}, "Parent"), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestMakeEnumFieldsWithDeprecatedValues(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "networkTier",
				Schema: &schema{
					Description: "The networking tier.",
					Enums: []string{
						"FIXED_STANDARD",
						"PREMIUM",
						"STANDARD",
						"STANDARD_OVERRIDES_FIXED_STANDARD",
					},
					EnumDescriptions: []string{
						"Public internet quality with fixed bandwidth.",
						"High quality, Google-grade network tier, support for all networking products.",
						"Public internet quality, only limited support for other networking products.",
						"(Output only) Temporary tier for FIXED_STANDARD when fixed standard tier is expired or not configured.",
					},
					EnumDeprecated: []bool{
						true,
						false,
						true,
						false,
					},
					Type: "string",
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantEnum := &api.Enum{
		Name:          "networkTier",
		ID:            ".package.Message.networkTier",
		Documentation: "The enumerated type for the [networkTier][package.Message.networkTier] field.",
		Values: []*api.EnumValue{
			{
				Name:          "FIXED_STANDARD",
				ID:            ".package.Message.networkTier.FIXED_STANDARD",
				Number:        0,
				Documentation: "Public internet quality with fixed bandwidth.",
				Deprecated:    true,
			},
			{
				Name:          "PREMIUM",
				ID:            ".package.Message.networkTier.PREMIUM",
				Number:        1,
				Documentation: "High quality, Google-grade network tier, support for all networking products.",
			},
			{
				Name:          "STANDARD",
				ID:            ".package.Message.networkTier.STANDARD",
				Number:        2,
				Documentation: "Public internet quality, only limited support for other networking products.",
				Deprecated:    true,
			},
			{
				Name:          "STANDARD_OVERRIDES_FIXED_STANDARD",
				ID:            ".package.Message.networkTier.STANDARD_OVERRIDES_FIXED_STANDARD",
				Number:        3,
				Documentation: "(Output only) Temporary tier for FIXED_STANDARD when fixed standard tier is expired or not configured.",
			},
		},
	}
	wantEnum.UniqueNumberValues = wantEnum.Values
	gotEnum, ok := model.State.EnumByID[wantEnum.ID]
	if !ok {
		t.Fatalf("missing enum %s", wantEnum.ID)
	}
	apitest.CheckEnum(t, *gotEnum, *wantEnum)
	if gotEnum.Parent == nil {
		t.Errorf("expected non-nil parent in enum: %v", gotEnum)
	}
	for _, value := range gotEnum.Values {
		if value.Parent != gotEnum {
			t.Errorf("mismatched parent in enumValue: %v", value)
		}
	}

	want := &api.Message{
		ID: ".package.Message",
		Fields: []*api.Field{
			{
				Name:          "networkTier",
				JSONName:      "networkTier",
				ID:            ".package.Message.networkTier",
				Documentation: "The networking tier.",
				Typez:         api.ENUM_TYPE,
				TypezID:       ".package.Message.networkTier",
				Optional:      true,
			},
		},
	}
	apitest.CheckMessage(t, message, want)
	wantEnums := []*api.Enum{wantEnum}
	if diff := cmp.Diff(wantEnums, message.Enums, cmpopts.IgnoreFields(api.Enum{}, "Parent"), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestMakeEnumFieldsDescriptionError(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "networkTier",
				Schema: &schema{
					Description: "The networking tier.",
					Enums: []string{
						"FIXED_STANDARD",
						"PREMIUM",
					},
					EnumDescriptions: []string{
						"Public internet quality with fixed bandwidth.",
						"High quality, Google-grade network tier, support for all networking products.",
						"Public internet quality, only limited support for other networking products.",
						"(Output only) Temporary tier for FIXED_STANDARD when fixed standard tier is expired or not configured.",
					},
					Type: "string",
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err == nil {
		t.Errorf("expected error in enum with mismatched description count, got=%v", message)
	}
}

func TestMakeEnumFieldsDeprecatedError(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "package"
	input := &schema{
		Properties: []*property{
			{
				Name: "networkTier",
				Schema: &schema{
					Description: "The networking tier.",
					Enums: []string{
						"FIXED_STANDARD",
						"PREMIUM",
					},
					EnumDescriptions: []string{
						"Public internet quality with fixed bandwidth.",
						"High quality, Google-grade network tier, support for all networking products.",
					},
					EnumDeprecated: []bool{
						false,
						true,
						true,
					},
					Type: "string",
				},
			},
		},
	}
	message := &api.Message{ID: ".package.Message"}
	if err := makeMessageFields(model, message, input); err == nil {
		t.Errorf("expected error in enum with mismatched deprecated values count, got=%v", message)
	}
}
