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

func TestMethod(t *testing.T) {
	f := NewFieldBuilder().WithName("name").Build()
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

	inMsg := NewMessageBuilder().WithName("Req").Build()
	outMsg := NewMessageBuilder().WithName("Resp").Build()
	pi := NewPathInfoBuilder().Build()
	r1 := NewRoutingInfoBuilder().WithName("r1").Build()
	op := NewOperationInfoBuilder().Build()
	lro := NewDiscoveryLroBuilder().Build()
	sig := NewMethodSignatureBuilder().WithNames("name").Build()
	svc := NewTestService("Svc")
	srcSvc := NewTestService("SrcSvc")
	model := NewTestAPI(nil, nil, nil)

	got := NewMethodBuilder().
		WithName("GetBook").
		WithID(".test.Svc.GetBook").
		WithDocumentation("GetBook docs.").
		WithDeprecated(true).
		WithInputTypeID(".test.Req").
		WithOutputTypeID(".test.Resp").
		WithInputType(inMsg).
		WithOutputType(outMsg).
		WithReturnsEmpty(true).
		WithClientSideStreaming(true).
		WithServerSideStreaming(true).
		WithIsLroPoller(true).
		WithPathInfo(pi).
		WithRouting([]*RoutingInfo{r1}).
		WithAutoPopulated([]*Field{f}).
		WithAPIVersion("v1").
		WithOperationInfo(op).
		WithDiscoveryLro(lro).
		WithPagination(f).
		WithService(svc).
		WithSourceService(srcSvc).
		WithSourceServiceID(".test.SrcSvc").
		WithModel(model).
		WithSignatures([]*MethodSignature{sig}).
		WithSampleInfo(gotSI).
		WithIsAIPStandard(true).
		Build()

	got.SetAnnotations("method-codec")

	want := &Method{
		name:                "GetBook",
		id:                  ".test.Svc.GetBook",
		documentation:       "GetBook docs.",
		deprecated:          true,
		inputTypeID:         ".test.Req",
		outputTypeID:        ".test.Resp",
		inputType:           inMsg,
		outputType:          outMsg,
		returnsEmpty:        true,
		clientSideStreaming: true,
		serverSideStreaming: true,
		isLroPoller:         true,
		pathInfo:            pi,
		routing:             []*RoutingInfo{r1},
		autoPopulated:       []*Field{f},
		apiVersion:          "v1",
		operationInfo:       op,
		discoveryLro:        lro,
		pagination:          f,
		service:             svc,
		sourceService:       srcSvc,
		sourceServiceID:     ".test.SrcSvc",
		model:               model,
		signatures:          []*MethodSignature{sig},
		sampleInfo:          gotSI,
		isAIPStandard:       true,
		codec:               "method-codec",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("Method mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name || got.ID() != want.id || got.Documentation() != want.documentation || !got.Deprecated() || got.InputTypeID() != want.inputTypeID || got.OutputTypeID() != want.outputTypeID || got.APIVersion() != want.apiVersion || got.SourceServiceID() != want.sourceServiceID {
		t.Errorf("unexpected Method identity: %+v", got)
	}
	if got.InputType() != inMsg || got.OutputType() != outMsg || !got.ReturnsEmpty() || !got.ClientSideStreaming() || !got.ServerSideStreaming() || !got.IsLroPoller() || got.IsSimple() {
		t.Errorf("unexpected Method streaming/types: %+v", got)
	}
	if got.PathInfo() != pi || got.OperationInfo() != op || got.DiscoveryLro() != lro || got.Pagination() != f || got.Service() != svc || got.SourceService() != srcSvc || got.Model() != model || got.SampleInfo() != gotSI || !got.IsAIPStandard() || got.Codec() != "method-codec" {
		t.Errorf("unexpected Method pointers: %+v", got)
	}
	if len(got.Routing()) != 1 || got.Routing()[0] != r1 || len(got.AutoPopulated()) != 1 || got.AutoPopulated()[0] != f || len(got.Signatures()) != 1 || got.Signatures()[0] != sig {
		t.Errorf("Method slice inputs were not cloned")
	}

	got.Routing()[0] = nil
	got.AutoPopulated()[0] = nil
	got.Signatures()[0] = nil
	if got.Routing()[0] != r1 || got.AutoPopulated()[0] != f || got.Signatures()[0] != sig {
		t.Errorf("Method slice getters should return defensive copies")
	}
}
