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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"google.golang.org/genproto/googleapis/api/annotations"
)

func TestPopulateMixinDefinitionLocations(t *testing.T) {
	// Verify nil model does not crash
	populateMixinDefinitionLocations(nil)

	model := api.NewTestAPI(nil, nil, nil)
	populateMixinDefinitionLocations(model)

	for _, test := range []struct {
		symbol string
		want   api.SourceLocation
	}{
		// Locations
		{symbol: ".google.cloud.location.ListLocationsRequest", want: api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 58}},
		{symbol: "google.cloud.location.ListLocationsRequest", want: api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 58}},
		{symbol: ".google.cloud.location.ListLocationsResponse", want: api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 73}},
		{symbol: "google.cloud.location.ListLocationsResponse", want: api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 73}},
		{symbol: ".google.cloud.location.GetLocationRequest", want: api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 82}},
		{symbol: "google.cloud.location.GetLocationRequest", want: api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 82}},
		{symbol: ".google.cloud.location.Location", want: api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 88}},
		{symbol: "google.cloud.location.Location", want: api.SourceLocation{Filename: "google/cloud/location/locations.proto", Line: 88}},

		// IAM Policy
		{symbol: ".google.iam.v1.SetIamPolicyRequest", want: api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 100}},
		{symbol: "google.iam.v1.SetIamPolicyRequest", want: api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 100}},
		{symbol: ".google.iam.v1.GetIamPolicyRequest", want: api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 123}},
		{symbol: "google.iam.v1.GetIamPolicyRequest", want: api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 123}},
		{symbol: ".google.iam.v1.TestIamPermissionsRequest", want: api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 137}},
		{symbol: "google.iam.v1.TestIamPermissionsRequest", want: api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 137}},
		{symbol: ".google.iam.v1.TestIamPermissionsResponse", want: api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 153}},
		{symbol: "google.iam.v1.TestIamPermissionsResponse", want: api.SourceLocation{Filename: "google/iam/v1/iam_policy.proto", Line: 153}},

		// Policy
		{symbol: ".google.iam.v1.Policy", want: api.SourceLocation{Filename: "google/iam/v1/policy.proto", Line: 102}},
		{symbol: "google.iam.v1.Policy", want: api.SourceLocation{Filename: "google/iam/v1/policy.proto", Line: 102}},

		// Longrunning Operations
		{symbol: ".google.longrunning.Operation", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 121}},
		{symbol: "google.longrunning.Operation", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 121}},
		{symbol: ".google.longrunning.GetOperationRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 160}},
		{symbol: "google.longrunning.GetOperationRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 160}},
		{symbol: ".google.longrunning.ListOperationsRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 167}},
		{symbol: "google.longrunning.ListOperationsRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 167}},
		{symbol: ".google.longrunning.ListOperationsResponse", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 196}},
		{symbol: "google.longrunning.ListOperationsResponse", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 196}},
		{symbol: ".google.longrunning.CancelOperationRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 212}},
		{symbol: "google.longrunning.CancelOperationRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 212}},
		{symbol: ".google.longrunning.DeleteOperationRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 219}},
		{symbol: "google.longrunning.DeleteOperationRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 219}},
		{symbol: ".google.longrunning.WaitOperationRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 226}},
		{symbol: "google.longrunning.WaitOperationRequest", want: api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 226}},
	} {
		t.Run(test.symbol, func(t *testing.T) {
			got, ok := model.DefinitionLocation(test.symbol)
			if !ok {
				t.Errorf("missing definition location for %s", test.symbol)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}

	if got, want := len(model.DefinitionLocations), 32; got != want {
		t.Errorf("len(model.DefinitionLocations) = %d, want %d", got, want)
	}
}

func TestPopulateMixinDefinitionLocations_DoesNotOverwriteExisting(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil)

	// Pre-populate dynamic SourceCodeInfo locations for mixin symbols.
	existingOpLoc := api.SourceLocation{Filename: "custom/dynamic/operations.proto", Line: 42}
	existingLocLoc := api.SourceLocation{Filename: "custom/dynamic/locations.proto", Line: 77}

	model.AddDefinitionLocation(".google.longrunning.Operation", existingOpLoc)
	model.AddDefinitionLocation("google.cloud.location.Location", existingLocLoc)

	populateMixinDefinitionLocations(model)

	for _, test := range []struct {
		name   string
		symbol string
		want   api.SourceLocation
	}{
		{
			name:   "pre-existing operation with leading dot is preserved",
			symbol: ".google.longrunning.Operation",
			want:   existingOpLoc,
		},
		{
			name:   "pre-existing operation queried without leading dot is preserved",
			symbol: "google.longrunning.Operation",
			want:   existingOpLoc,
		},
		{
			name:   "pre-existing location without leading dot is preserved",
			symbol: "google.cloud.location.Location",
			want:   existingLocLoc,
		},
		{
			name:   "pre-existing location queried with leading dot is preserved",
			symbol: ".google.cloud.location.Location",
			want:   existingLocLoc,
		},
		{
			name:   "un-populated mixin symbol receives static fallback",
			symbol: ".google.longrunning.GetOperationRequest",
			want:   api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 160},
		},
		{
			name:   "un-populated mixin symbol without leading dot receives static fallback",
			symbol: "google.longrunning.GetOperationRequest",
			want:   api.SourceLocation{Filename: "google/longrunning/operations.proto", Line: 160},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := model.DefinitionLocation(test.symbol)
			if !ok {
				t.Fatalf("missing location for %q", test.symbol)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtobuf_ForceLongrunning(t *testing.T) {
	sc := sample.ServiceConfig()
	sc.Http = &annotations.Http{
		Rules: []*httpRule{
			{
				Selector: "google.longrunning.Operations.CancelOperation",
				Pattern: &httpRulePost{
					Post: "/v2/{name=operations/**}:cancel",
				},
			},
			{
				Selector: "google.longrunning.Operations.GetOperation",
				Pattern: &httpRuleGet{
					Get: "/v2/{name=operations/**}:cancel",
				},
			},
		},
	}

	wantMethods := mixinMethods{
		".google.longrunning.Operations.GetOperation":    true,
		".google.longrunning.Operations.CancelOperation": true,
	}
	gotMethods, gotDescriptors := loadMixins(sc, true)
	if diff := cmp.Diff(wantMethods, gotMethods); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	names := map[string]bool{}
	for _, d := range gotDescriptors {
		names[d.GetName()] = true
	}
	if _, ok := names["google/cloud/location/locations.proto"]; !ok {
		t.Errorf("Missing longrunning descriptor in %v", gotDescriptors)
	}
}

func TestProtobuf_ForceLongrunningNoRules(t *testing.T) {
	sc := sample.ServiceConfig()
	sc.Http = &annotations.Http{}

	wantMethods := mixinMethods{
		".google.longrunning.Operations.GetOperation": true,
	}
	gotMethods, gotDescriptors := loadMixins(sc, true)
	if diff := cmp.Diff(wantMethods, gotMethods); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	names := map[string]bool{}
	for _, d := range gotDescriptors {
		names[d.GetName()] = true
	}
	if _, ok := names["google/cloud/location/locations.proto"]; !ok {
		t.Errorf("Missing longrunning descriptor in %v", gotDescriptors)
	}
}
