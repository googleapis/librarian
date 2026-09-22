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

package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateTransport_Basic(t *testing.T) {
	reqMsg := api.NewTestMessage("CreateFooRequest")
	respMsg := api.NewTestMessage("Foo")
	meth := api.NewTestMethod("CreateFoo").
		WithInput(reqMsg).
		WithOutput(respMsg)
	meth.Documentation = "Creates a new Foo."
	svc := api.NewTestService("FooService").
		WithPackage("google.cloud.foo.v1").
		WithMethods(meth)
	svc.Documentation = "Foo service documentation."
	svc.DefaultHost = "foo.googleapis.com"
	reqMsg.WithSourceLocation("foo_service.proto", 1)
	respMsg.WithSourceLocation("foo_service.proto", 10)
	svc.WithSourceLocation("foo_service.proto", 20)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc})
	model.PackageName = "google.cloud.foo.v1"

	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	sAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok || sAnn == nil {
		t.Fatalf("svc.Codec = %T, want *serviceAnnotations", svc.Codec)
	}

	got := sAnn.Transport
	want := &transportAnnotations{
		Name:               "FooService",
		TransportClassName: "FooServiceTransport",
		ServiceFQN:         "google.cloud.foo.v1.FooService",
		DefaultHost:        "foo.googleapis.com",
		VersionPackage:     "google.cloud.foo_v1",
		Scopes:             []string{"https://www.googleapis.com/auth/cloud-platform"},
		DocLines:           []string{"Foo service documentation."},
		HasDocLines:        true,
		Imports: []*transportImport{
			{
				From:   "google.cloud.foo_v1.types",
				Import: "foo_service",
			},
		},
		WrappedMethods: []*wrappedMethodAnnotations{
			{
				Name:           "create_foo",
				DefaultTimeout: "None",
			},
		},
		ServiceMethods: []*transportMethodAnnotations{
			{
				Name:                 "create_foo",
				InputTypeIdent:       "foo_service.CreateFooRequest",
				OutputTypeIdent:      "foo_service.Foo",
				InputTypeShortIdent:  "~.CreateFooRequest",
				OutputTypeShortIdent: "~.Foo",
				RPCPath:              "/google.cloud.foo.v1.FooService/CreateFoo",
				GRPCStubType:         "unary_unary",
				RequestSerializer:    "foo_service.CreateFooRequest.serialize",
				ResponseDeserializer: "foo_service.Foo.deserialize",
				DocSummaryLead:       "create foo",
				DocLines:             []string{"Creates a new Foo."},
				HasDocLines:          true,
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateTransport_MixinsAndLRO(t *testing.T) {
	reqMsg := api.NewTestMessage("DoSomethingRequest")
	nativeMeth := api.NewTestMethod("DoSomething").
		WithInput(reqMsg).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".google.cloud.bar.v1.BarResponse",
			MetadataTypeID: ".google.cloud.bar.v1.BarMetadata",
		})
	nativeMeth.OutputTypeID = ".google.longrunning.Operation"

	emptyMeth := api.NewTestMethod("DeleteSomething").
		WithInput(reqMsg)
	emptyMeth.OutputTypeID = api.WktEmptyID
	emptyMeth.ReturnsEmpty = true

	getOpMeth := api.NewTestMethod("GetOperation")
	getOpMeth.SourceServiceID = ".google.longrunning.Operations"
	getOpMeth.InputTypeID = ".google.longrunning.GetOperationRequest"
	getOpMeth.OutputTypeID = ".google.longrunning.Operation"

	getLocMeth := api.NewTestMethod("GetLocation")
	getLocMeth.SourceServiceID = ".google.cloud.location.Locations"
	getLocMeth.InputTypeID = ".google.cloud.location.GetLocationRequest"
	getLocMeth.OutputTypeID = ".google.cloud.location.Location"

	svc := api.NewTestService("BarService").
		WithPackage("google.cloud.bar.v1").
		WithMethods(nativeMeth, emptyMeth, getLocMeth, getOpMeth)
	svc.DefaultHost = "bar.googleapis.com"
	reqMsg.WithSourceLocation("bar.proto", 1)
	svc.WithSourceLocation("bar.proto", 10)
	model := api.NewTestAPI([]*api.Message{reqMsg}, nil, []*api.Service{svc})
	model.PackageName = "google.cloud.bar.v1"

	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	sAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok || sAnn == nil {
		t.Fatalf("svc.Codec = %T, want *serviceAnnotations", svc.Codec)
	}

	got := sAnn.Transport
	want := &transportAnnotations{
		Name:               "BarService",
		TransportClassName: "BarServiceTransport",
		ServiceFQN:         "google.cloud.bar.v1.BarService",
		DefaultHost:        "bar.googleapis.com",
		VersionPackage:     "google.cloud.bar_v1",
		Scopes:             []string{"https://www.googleapis.com/auth/cloud-platform"},
		HasLRO:             true,
		HasLocationMixin:   true,
		HasOperationsMixin: true,
		HasGetOperation:    true,
		HasGetLocation:     true,
		Imports: []*transportImport{
			{
				From:   "google.cloud.bar_v1.types",
				Import: "bar",
			},
			{
				From:   "google.cloud.location",
				Import: "locations_pb2",
				Ignore: true,
			},
			{
				From:   "google.longrunning",
				Import: "operations_pb2",
				Ignore: true,
			},
			{
				From:   "google.protobuf.empty_pb2",
				As:     "empty_pb2",
				Ignore: true,
			},
		},
		WrappedMethods: []*wrappedMethodAnnotations{
			{
				Name:           "do_something",
				DefaultTimeout: "None",
			},
			{
				Name:           "delete_something",
				DefaultTimeout: "None",
			},
			{
				Name:           "get_location",
				DefaultTimeout: "None",
			},
			{
				Name:           "get_operation",
				DefaultTimeout: "None",
			},
		},
		ServiceMethods: []*transportMethodAnnotations{
			{
				Name:                 "do_something",
				InputTypeIdent:       "bar.DoSomethingRequest",
				OutputTypeIdent:      "operations_pb2.Operation",
				InputTypeShortIdent:  "~.DoSomethingRequest",
				OutputTypeShortIdent: "~.Operation",
				RPCPath:              "/google.cloud.bar.v1.BarService/DoSomething",
				GRPCStubType:         "unary_unary",
				RequestSerializer:    "bar.DoSomethingRequest.serialize",
				ResponseDeserializer: "operations_pb2.Operation.FromString",
				DocSummaryLead:       "do something",
			},
			{
				Name:                 "delete_something",
				InputTypeIdent:       "bar.DoSomethingRequest",
				OutputTypeIdent:      "empty_pb2.Empty",
				InputTypeShortIdent:  "~.DoSomethingRequest",
				OutputTypeShortIdent: "~.Empty",
				RPCPath:              "/google.cloud.bar.v1.BarService/DeleteSomething",
				GRPCStubType:         "unary_unary",
				RequestSerializer:    "bar.DoSomethingRequest.serialize",
				ResponseDeserializer: "empty_pb2.Empty.FromString",
				DocSummaryLead:       "delete something",
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParseFloatDuration(t *testing.T) {
	for _, test := range []struct {
		name     string
		duration string
		wantSec  float64
	}{
		{
			name:     "integer seconds with s suffix",
			duration: "60s",
			wantSec:  60.0,
		},
		{
			name:     "large integer seconds with s suffix",
			duration: "600s",
			wantSec:  600.0,
		},
		{
			name:     "fractional seconds with s suffix",
			duration: "0.100s",
			wantSec:  0.1,
		},
		{
			name:     "without s suffix",
			duration: "1.3",
			wantSec:  1.3,
		},
		{
			name:     "with whitespace",
			duration: " 15s \n",
			wantSec:  15.0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotSec, err := parseFloatDuration(test.duration)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantSec, gotSec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseFloatDuration_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		duration string
	}{
		{
			name:     "invalid characters",
			duration: "invalid",
		},
		{
			name:     "multiple s suffixes",
			duration: "60ss",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseFloatDuration(test.duration); err == nil {
				t.Errorf("parseFloatDuration(%q) = nil, want error", test.duration)
			}
		})
	}
}

func TestFormatFloat(t *testing.T) {
	for _, test := range []struct {
		name    string
		val     float64
		wantFmt string
	}{
		{
			name:    "integer 60 formats with decimal",
			val:     60.0,
			wantFmt: "60.0",
		},
		{
			name:    "integer 0 formats with decimal",
			val:     0.0,
			wantFmt: "0.0",
		},
		{
			name:    "fractional 0.1 formats without trailing zero",
			val:     0.1,
			wantFmt: "0.1",
		},
		{
			name:    "fractional 1.3",
			val:     1.3,
			wantFmt: "1.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatFloat(test.val)
			if diff := cmp.Diff(test.wantFmt, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildWrappedMethod_WithGRPCServiceConfig(t *testing.T) {
	jsonConfig := `{
		"methodConfig": [
			{
				"name": [
					{"service": "google.cloud.example.v1.ExampleService", "method": "DoTask"}
				],
				"timeout": "120s",
				"retryPolicy": {
					"maxAttempts": 5,
					"initialBackoff": "0.2s",
					"maxBackoff": "30s",
					"backoffMultiplier": 1.5,
					"retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
				}
			}
		]
	}`

	tempDir := t.TempDir()
	pkgDir := filepath.Join(tempDir, "google/cloud/example/v1")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(pkgDir, "example_grpc_service_config.json")
	if err := os.WriteFile(cfgFile, []byte(jsonConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := api.NewTestService("ExampleService").
		WithPackage("google.cloud.example.v1")
	svc.ID = ".google.cloud.example.v1.ExampleService"
	m := api.NewTestMethod("DoTask")

	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	lib := &config.Library{
		Roots: []string{tempDir},
	}
	c := newTestCodec(t, model, lib)

	cfg, err := c.loadGRPCServiceConfig(svc)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("loadGRPCServiceConfig returned nil")
	}

	got := c.buildWrappedMethod(m, svc, cfg)
	want := &wrappedMethodAnnotations{
		Name:              "do_task",
		HasRetry:          true,
		InitialBackoff:    "0.2",
		MaxBackoff:        "30.0",
		BackoffMultiplier: "1.5",
		RetryExceptions:   []string{"DeadlineExceeded", "ServiceUnavailable"},
		Deadline:          "120.0",
		DefaultTimeout:    "120.0",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateTransport_IAM(t *testing.T) {
	reqMsg := api.NewTestMessage("SetIamPolicyRequest").WithPackage("google.iam.v1")
	respMsg := api.NewTestMessage("Policy").WithPackage("google.iam.v1")

	meth := api.NewTestMethod("SetIamPolicy").
		WithInput(reqMsg).
		WithOutput(respMsg)
	svc := api.NewTestService("SecretManagerService").
		WithPackage("google.cloud.secretmanager.v1").
		WithMethods(meth)
	svc.DefaultHost = "secretmanager.googleapis.com"
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc}).
		WithPackageName("google.cloud.secretmanager.v1")

	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	sAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok || sAnn == nil {
		t.Fatalf("svc.Codec = %T, want *serviceAnnotations", svc.Codec)
	}

	got := sAnn.Transport
	if len(got.ServiceMethods) != 1 {
		t.Fatalf("got %d service methods, want 1", len(got.ServiceMethods))
	}
	sm := got.ServiceMethods[0]
	want := &transportMethodAnnotations{
		Name:                 "set_iam_policy",
		RequestSerializer:    "iam_policy_pb2.SetIamPolicyRequest.SerializeToString",
		ResponseDeserializer: "policy_pb2.Policy.FromString",
		InputTypeIdent:       "iam_policy_pb2.SetIamPolicyRequest",
		OutputTypeIdent:      "policy_pb2.Policy",
		GRPCStubType:         "unary_unary",
	}
	gotMeth := &transportMethodAnnotations{
		Name:                 sm.Name,
		RequestSerializer:    sm.RequestSerializer,
		ResponseDeserializer: sm.ResponseDeserializer,
		InputTypeIdent:       sm.InputTypeIdent,
		OutputTypeIdent:      sm.OutputTypeIdent,
		GRPCStubType:         sm.GRPCStubType,
	}
	if diff := cmp.Diff(want, gotMeth); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateTransport_IAMPolicyMixin(t *testing.T) {
	meth := api.NewTestMethod("SetIamPolicy")
	meth.SourceServiceID = ".google.iam.v1.IAMPolicy"
	svc := api.NewTestService("ExampleService").
		WithPackage("google.example.v1").
		WithMethods(meth)
	svc.DefaultHost = "example.googleapis.com"
	model := api.NewTestAPI(nil, nil, []*api.Service{svc}).
		WithPackageName("google.example.v1")
	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}
	sAnn := svc.Codec.(*serviceAnnotations)
	if !sAnn.Transport.HasIAMPolicyMixin || !sAnn.Transport.HasSetIamPolicy {
		t.Errorf("got HasIAMPolicyMixin=%v, HasSetIamPolicy=%v, want true, true",
			sAnn.Transport.HasIAMPolicyMixin, sAnn.Transport.HasSetIamPolicy)
	}
}
