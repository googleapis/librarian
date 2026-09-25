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
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
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
	message := api.NewTestMessage("Message").WithPackage("package")
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantEnum := api.NewTestEnum("networkTier").
		WithPackage("package").
		WithID(".package.Message.networkTier").
		WithDocumentation("The enumerated type for the [networkTier][package.Message.networkTier] field.").
		WithValues(
			api.NewTestEnumValue("FIXED_STANDARD", 0).
				WithID(".package.Message.networkTier.FIXED_STANDARD").
				WithDocumentation("Public internet quality with fixed bandwidth."),
			api.NewTestEnumValue("PREMIUM", 1).
				WithID(".package.Message.networkTier.PREMIUM").
				WithDocumentation("High quality, Google-grade network tier, support for all networking products."),
			api.NewTestEnumValue("STANDARD", 2).
				WithID(".package.Message.networkTier.STANDARD").
				WithDocumentation("Public internet quality, only limited support for other networking products."),
			api.NewTestEnumValue("STANDARD_OVERRIDES_FIXED_STANDARD", 3).
				WithID(".package.Message.networkTier.STANDARD_OVERRIDES_FIXED_STANDARD").
				WithDocumentation("(Output only) Temporary tier for FIXED_STANDARD when fixed standard tier is expired or not configured."),
		)
	gotEnum := model.Enum(wantEnum.ID)
	if gotEnum == nil {
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

	want := api.NewTestMessage("Message").
		WithPackage("package").
		WithFields(
			api.NewTestField("networkTier").
				WithDocumentation("The networking tier.").
				WithType(api.TypezEnum).
				WithTypezID(".package.Message.networkTier").
				WithOptional(),
		)
	want.Fields[0].Parent = nil
	apitest.CheckMessage(t, message, want)
	wantEnums := []*api.Enum{wantEnum}
	if diff := cmp.Diff(wantEnums, message.Enums, cmpopts.IgnoreFields(api.Enum{}, "Parent"), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
	message := api.NewTestMessage("Message").WithPackage("package")
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantEnum := api.NewTestEnum("networkTier").
		WithPackage("package").
		WithID(".package.Message.networkTier").
		WithDocumentation("The enumerated type for the [networkTier][package.Message.networkTier] field.").
		WithDeprecated(true).
		WithValues(
			api.NewTestEnumValue("FIXED_STANDARD", 0).
				WithID(".package.Message.networkTier.FIXED_STANDARD").
				WithDocumentation("Public internet quality with fixed bandwidth."),
			api.NewTestEnumValue("PREMIUM", 1).
				WithID(".package.Message.networkTier.PREMIUM").
				WithDocumentation("High quality, Google-grade network tier, support for all networking products."),
		)
	gotEnum := model.Enum(wantEnum.ID)
	if gotEnum == nil {
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

	want := api.NewTestMessage("Message").
		WithPackage("package").
		WithFields(
			api.NewTestField("networkTier").
				WithDocumentation("The networking tier.").
				WithDeprecated(true).
				WithType(api.TypezEnum).
				WithTypezID(".package.Message.networkTier").
				WithOptional(),
		)
	want.Fields[0].Parent = nil
	apitest.CheckMessage(t, message, want)
	wantEnums := []*api.Enum{wantEnum}
	if diff := cmp.Diff(wantEnums, message.Enums, cmpopts.IgnoreFields(api.Enum{}, "Parent"), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
	message := api.NewTestMessage("Message").WithPackage("package")
	if err := makeMessageFields(model, message, input); err != nil {
		t.Fatal(err)
	}

	wantEnum := api.NewTestEnum("networkTier").
		WithPackage("package").
		WithID(".package.Message.networkTier").
		WithDocumentation("The enumerated type for the [networkTier][package.Message.networkTier] field.").
		WithValues(
			api.NewTestEnumValue("FIXED_STANDARD", 0).
				WithID(".package.Message.networkTier.FIXED_STANDARD").
				WithDocumentation("Public internet quality with fixed bandwidth.").
				WithDeprecated(true),
			api.NewTestEnumValue("PREMIUM", 1).
				WithID(".package.Message.networkTier.PREMIUM").
				WithDocumentation("High quality, Google-grade network tier, support for all networking products."),
			api.NewTestEnumValue("STANDARD", 2).
				WithID(".package.Message.networkTier.STANDARD").
				WithDocumentation("Public internet quality, only limited support for other networking products.").
				WithDeprecated(true),
			api.NewTestEnumValue("STANDARD_OVERRIDES_FIXED_STANDARD", 3).
				WithID(".package.Message.networkTier.STANDARD_OVERRIDES_FIXED_STANDARD").
				WithDocumentation("(Output only) Temporary tier for FIXED_STANDARD when fixed standard tier is expired or not configured."),
		)
	gotEnum := model.Enum(wantEnum.ID)
	if gotEnum == nil {
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

	want := api.NewTestMessage("Message").
		WithPackage("package").
		WithFields(
			api.NewTestField("networkTier").
				WithDocumentation("The networking tier.").
				WithType(api.TypezEnum).
				WithTypezID(".package.Message.networkTier").
				WithOptional(),
		)
	want.Fields[0].Parent = nil
	apitest.CheckMessage(t, message, want)
	wantEnums := []*api.Enum{wantEnum}
	if diff := cmp.Diff(wantEnums, message.Enums, cmpopts.IgnoreFields(api.Enum{}, "Parent"), cmpopts.IgnoreFields(api.EnumValue{}, "Parent")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
	message := api.NewTestMessage("Message").WithPackage("package")
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
	message := api.NewTestMessage("Message").WithPackage("package")
	if err := makeMessageFields(model, message, input); err == nil {
		t.Errorf("expected error in enum with mismatched deprecated values count, got=%v", message)
	}
}
