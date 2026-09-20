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
	"errors"
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

func TestFindAuthScopes(t *testing.T) {
	for _, test := range []struct {
		name       string
		yamlBody   string
		wantScopes []string
	}{
		{
			name: "multiple scopes with comma and newline",
			yamlBody: `
authentication:
  rules:
    - selector: "*"
      oauth:
        canonical_scopes: |-
          https://www.googleapis.com/auth/cloud-platform,
          https://www.googleapis.com/auth/userinfo.email
`,
			wantScopes: []string{
				"https://www.googleapis.com/auth/cloud-platform",
				"https://www.googleapis.com/auth/userinfo.email",
			},
		},
		{
			name: "empty scopes falls back to default",
			yamlBody: `
authentication:
  rules:
    - selector: "*"
      oauth:
        canonical_scopes: ""
`,
			wantScopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
		},
		{
			name:       "missing config file falls back to default",
			yamlBody:   "",
			wantScopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			svc := api.NewTestService("ScopeService").
				WithPackage("google.cloud.scope.v1")
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			lib := &config.Library{
				Roots: []string{tempDir},
			}

			if test.yamlBody != "" {
				pkgDir := filepath.Join(tempDir, "google/cloud/scope/v1")
				if err := os.MkdirAll(pkgDir, 0o755); err != nil {
					t.Fatal(err)
				}
				configFile := filepath.Join(pkgDir, "scope_v1.yaml")
				if err := os.WriteFile(configFile, []byte(test.yamlBody), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			c := newTestCodec(t, model, lib)
			cfg, err := c.loadServiceConfig(svc)
			if err != nil {
				t.Fatal(err)
			}
			got := c.findAuthScopes(cfg)
			if diff := cmp.Diff(test.wantScopes, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsRestAsyncIOEnabled(t *testing.T) {
	for _, test := range []struct {
		name     string
		yamlBody string
		want     bool
	}{
		{
			name: "enabled in library_settings",
			yamlBody: `
publishing:
  library_settings:
    - version: google.cloud.async.v1
      python_settings:
        experimental_features:
          rest_async_io_enabled: true
`,
			want: true,
		},
		{
			name: "disabled in library_settings",
			yamlBody: `
publishing:
  library_settings:
    - version: google.cloud.async.v1
      python_settings:
        experimental_features:
          rest_async_io_enabled: false
`,
			want: false,
		},
		{
			name:     "missing config",
			yamlBody: "",
			want:     false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			svc := api.NewTestService("AsyncService").
				WithPackage("google.cloud.async.v1")
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			lib := &config.Library{
				Roots: []string{tempDir},
			}

			if test.yamlBody != "" {
				pkgDir := filepath.Join(tempDir, "google/cloud/async/v1")
				if err := os.MkdirAll(pkgDir, 0o755); err != nil {
					t.Fatal(err)
				}
				configFile := filepath.Join(pkgDir, "async_v1.yaml")
				if err := os.WriteFile(configFile, []byte(test.yamlBody), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			c := newTestCodec(t, model, lib)
			cfg, err := c.loadServiceConfig(svc)
			if err != nil {
				t.Fatal(err)
			}
			got := c.isRestAsyncIOEnabled(cfg, svc)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindConfigFileForService(t *testing.T) {
	tempDir := t.TempDir()
	apiDir := filepath.Join(tempDir, "google/cloud/myapi/v1")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(apiDir, "service_v1.yaml")
	if err := os.WriteFile(targetFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	otherDir := filepath.Join(tempDir, "google/cloud/other/v1")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	otherFile := filepath.Join(otherDir, "other_v1.yaml")
	if err := os.WriteFile(otherFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := api.NewTestService("MyService").
		WithPackage("google.cloud.myapi.v1")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	lib := &config.Library{
		Roots: []string{tempDir},
		APIs: []*config.API{
			{Path: "google/cloud/other/v1"},
			{Path: "google/cloud/myapi/v1"},
		},
	}

	c := newTestCodec(t, model, lib)
	got := c.findConfigFileForService(svc, "*_v1.yaml")
	if diff := cmp.Diff(targetFile, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateTransport_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		filename string
		content  string
		wantErr  error
	}{
		{
			name:     "malformed_service_config_yaml",
			filename: "myapi_v1.yaml",
			content:  "authentication:\n  rules: [invalid yaml",
			wantErr:  errLoadServiceConfig,
		},
		{
			name:     "malformed_grpc_service_config_json",
			filename: "myapi_grpc_service_config.json",
			content:  "{invalid_json",
			wantErr:  errLoadGRPCConfig,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			pkgDir := filepath.Join(tempDir, "google", "cloud", "myapi", "v1")
			if err := os.MkdirAll(pkgDir, 0o755); err != nil {
				t.Fatal(err)
			}
			filePath := filepath.Join(pkgDir, test.filename)
			if err := os.WriteFile(filePath, []byte(test.content), 0o644); err != nil {
				t.Fatal(err)
			}

			svc := api.NewTestService("MyService").
				WithPackage("google.cloud.myapi.v1")
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			lib := &config.Library{
				Roots: []string{tempDir},
				APIs: []*config.API{
					{Path: "google/cloud/myapi/v1"},
				},
			}

			c := newTestCodec(t, model, lib)
			_, err := c.annotateTransport(svc)
			if err == nil {
				t.Fatalf("annotateTransport(%s) error = nil, want %v", svc.Name, test.wantErr)
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("annotateTransport(%s) error = %v, want %v", svc.Name, err, test.wantErr)
			}
		})
	}
}

func TestGetGRPCStubType(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   string
	}{
		{
			name:   "unary_unary",
			method: api.NewTestMethod("Test"),
			want:   "unary_unary",
		},
		{
			name:   "unary_stream",
			method: api.NewTestMethod("Test").WithServerSideStreaming(),
			want:   "unary_stream",
		},
		{
			name:   "stream_unary",
			method: api.NewTestMethod("Test").WithClientSideStreaming(),
			want:   "stream_unary",
		},
		{
			name:   "stream_stream",
			method: api.NewTestMethod("Test").WithBidiStreaming(),
			want:   "stream_stream",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getGRPCStubType(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
