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
	f := NewFieldBuilder().WithName("f").Build()
	gotSI := NewSampleInfoBuilder().
		WithResourceNameField(f).
		WithMessageField(f).
		WithResourceIDField(f).
		WithUpdateMaskField(f).
		WithIsRequestResourceName(true).
		WithIsMessageResourceName(true).
		Build()
	gotSI.SetAnnotations("si-codec")
	wantSI := &SampleInfo{
		resourceNameField:     f,
		messageField:          f,
		resourceIDField:       f,
		updateMaskField:       f,
		isRequestResourceName: true,
		isMessageResourceName: true,
		codec:                 "si-codec",
	}
	if diff := cmp.Diff(wantSI, gotSI, allowUnexported()); diff != "" {
		t.Errorf("SampleInfo mismatch (-want +got):\n%s", diff)
	}
	if gotSI.ResourceNameField() != f || gotSI.MessageField() != f || gotSI.ResourceIDField() != f || gotSI.UpdateMaskField() != f || !gotSI.IsRequestResourceName() || !gotSI.IsMessageResourceName() || gotSI.Codec() != "si-codec" {
		t.Errorf("unexpected SampleInfo getters: %+v", gotSI)
	}
}

func TestSampleInfoSetAnnotationsPanicsWhenAlreadySet(t *testing.T) {
	si := NewSampleInfoBuilder().Build()
	si.SetAnnotations("first")
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when calling SetAnnotations a second time, got none")
		}
	}()
	si.SetAnnotations("second")
}
