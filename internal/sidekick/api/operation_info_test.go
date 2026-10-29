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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOperationInfo(t *testing.T) {
	got := NewOperationInfoBuilder().
		WithMetadataTypeID(".test.Metadata").
		WithResponseTypeID(".test.Response").
		Build()
	got.SetAnnotations("op-codec")

	want := &OperationInfo{
		metadataTypeID: ".test.Metadata",
		responseTypeID: ".test.Response",
		codec:          "op-codec",
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(OperationInfo{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if got.MetadataTypeID() != want.metadataTypeID {
		t.Errorf("MetadataTypeID() = %q, want %q", got.MetadataTypeID(), want.metadataTypeID)
	}
	if got.ResponseTypeID() != want.responseTypeID {
		t.Errorf("ResponseTypeID() = %q, want %q", got.ResponseTypeID(), want.responseTypeID)
	}
	if got.Method() != nil {
		t.Errorf("Method() = %v, want nil", got.Method())
	}
	if got.Codec() != "op-codec" {
		t.Errorf("Codec() = %v, want op-codec", got.Codec())
	}
}
