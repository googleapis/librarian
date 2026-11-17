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

func TestSampleInfo(t *testing.T) {
	rnField := NewFieldBuilder().WithName("rn").Build()
	msgField := NewFieldBuilder().WithName("msg").Build()
	idField := NewFieldBuilder().WithName("id").Build()
	maskField := NewFieldBuilder().WithName("mask").Build()
	gotSI := NewSampleInfoBuilder().
		WithResourceNameField(rnField).
		WithMessageField(msgField).
		WithResourceIDField(idField).
		WithUpdateMaskField(maskField).
		WithIsRequestResourceName(true).
		WithIsMessageResourceName(true).
		Build()
	gotSI.SetAnnotations("si-codec")
	wantSI := &SampleInfo{
		resourceNameField:     rnField,
		messageField:          msgField,
		resourceIDField:       idField,
		updateMaskField:       maskField,
		isRequestResourceName: true,
		isMessageResourceName: true,
		codec:                 "si-codec",
	}
	if diff := cmp.Diff(wantSI, gotSI, allowUnexported()); diff != "" {
		t.Errorf("SampleInfo mismatch (-want +got):\n%s", diff)
	}
	if gotSI.ResourceNameField() != wantSI.resourceNameField ||
		gotSI.MessageField() != wantSI.messageField ||
		gotSI.ResourceIDField() != wantSI.resourceIDField ||
		gotSI.UpdateMaskField() != wantSI.updateMaskField ||
		gotSI.IsRequestResourceName() != wantSI.isRequestResourceName ||
		gotSI.IsMessageResourceName() != wantSI.isMessageResourceName ||
		gotSI.Codec() != wantSI.codec {
		t.Errorf("unexpected SampleInfo getters: %+v", gotSI)
	}
}
