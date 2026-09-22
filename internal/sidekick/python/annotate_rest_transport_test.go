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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateRestTransport_Basic(t *testing.T) {
	reqMsg := api.NewTestMessage("CreateBookRequest").
		WithPackage("google.example.library.v1").
		WithSourceLocation("google/example/library/v1/library.proto", 10)
	respMsg := api.NewTestMessage("Book").
		WithPackage("google.example.library.v1").
		WithSourceLocation("google/example/library/v1/library.proto", 20)

	pt := (&api.PathTemplate{}).WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithMatch()).WithLiteral("books")
	method := makeTestRestMethod("CreateBook", "POST", "*", pt, reqMsg).WithOutput(respMsg)

	svc := api.NewTestService("LibraryService").
		WithPackage("google.example.library.v1").
		WithSourceLocation("google/example/library/v1/library.proto", 30).
		WithMethods(method)
	svc.DefaultHost = "library.googleapis.com"

	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc}).
		WithPackageName("google.example.library.v1")

	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	svcAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok {
		t.Fatalf("svc.Codec is %T, want *serviceAnnotations", svc.Codec)
	}

	restAnn := svcAnn.RestTransport
	if restAnn == nil {
		t.Fatal("svcAnn.RestTransport is nil, want non-nil")
	}

	if restAnn.Name != "LibraryService" {
		t.Errorf("restAnn.Name = %q, want %q", restAnn.Name, "LibraryService")
	}
	if restAnn.BaseTransportClassName != "LibraryServiceTransport" {
		t.Errorf("restAnn.BaseTransportClassName = %q, want %q", restAnn.BaseTransportClassName, "LibraryServiceTransport")
	}
	if restAnn.TransportClassName != "LibraryServiceRestTransport" {
		t.Errorf("restAnn.TransportClassName = %q, want %q", restAnn.TransportClassName, "LibraryServiceRestTransport")
	}
	if restAnn.DefaultHost != "library.googleapis.com" {
		t.Errorf("restAnn.DefaultHost = %q, want %q", restAnn.DefaultHost, "library.googleapis.com")
	}
	if len(restAnn.BaseMethods) != 1 {
		t.Fatalf("len(restAnn.BaseMethods) = %d, want 1", len(restAnn.BaseMethods))
	}
	if restAnn.BaseMethods[0].BaseClassName != "_BaseCreateBook" {
		t.Errorf("BaseMethods[0].BaseClassName = %q, want %q", restAnn.BaseMethods[0].BaseClassName, "_BaseCreateBook")
	}
	if len(restAnn.PrimaryMethods) != 1 {
		t.Fatalf("len(restAnn.PrimaryMethods) = %d, want 1", len(restAnn.PrimaryMethods))
	}
	pm := restAnn.PrimaryMethods[0]
	if pm.Name != "create_book" || pm.MethodPascalName != "CreateBook" {
		t.Errorf("PrimaryMethod names mismatch: got (%q, %q)", pm.Name, pm.MethodPascalName)
	}
	if !pm.HasBody {
		t.Errorf("PrimaryMethod.HasBody = false, want true")
	}
	if diff := cmp.Diff("Test", restAnn.PackageName); diff != "" {
		t.Errorf("PackageName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, restAnn.RestAsyncIOEnabled); diff != "" {
		t.Errorf("RestAsyncIOEnabled mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("False", restAnn.RestNumericEnumsBool); diff != "" {
		t.Errorf("RestNumericEnumsBool mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateRestTransport_RestNumericEnums(t *testing.T) {
	reqMsg := api.NewTestMessage("CreateBookRequest").
		WithPackage("google.example.library.v1").
		WithSourceLocation("google/example/library/v1/library.proto", 10)
	respMsg := api.NewTestMessage("Book").
		WithPackage("google.example.library.v1").
		WithSourceLocation("google/example/library/v1/library.proto", 20)

	pt := (&api.PathTemplate{}).WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithMatch()).WithLiteral("books")
	method := makeTestRestMethod("CreateBook", "POST", "*", pt, reqMsg).WithOutput(respMsg)

	svc := api.NewTestService("LibraryService").
		WithPackage("google.example.library.v1").
		WithSourceLocation("google/example/library/v1/library.proto", 30).
		WithMethods(method)
	svc.DefaultHost = "library.googleapis.com"

	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc}).
		WithPackageName("google.example.library.v1")

	lib := &config.Library{
		Python: &config.PythonPackage{
			OptArgsByAPI: map[string][]string{
				"default": {"rest-numeric-enums"},
			},
		},
	}
	c := newTestCodec(t, model, lib)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	svcAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok || svcAnn.RestTransport == nil {
		t.Fatalf("svc.Codec is %T or RestTransport is nil", svc.Codec)
	}
	if diff := cmp.Diff("True", svcAnn.RestTransport.RestNumericEnumsBool); diff != "" {
		t.Errorf("RestNumericEnumsBool mismatch (-want +got):\n%s", diff)
	}
	if svcAnn.RestTransport.ShowRestBetaPreview {
		t.Errorf("ShowRestBetaPreview = true, want false when rest-numeric-enums is enabled")
	}
}

func TestAnnotateRestTransport_MixinsAndLRO(t *testing.T) {
	reqMsg := api.NewTestMessage("WriteLogRequest").
		WithPackage("google.example.logging.v1").
		WithSourceLocation("google/example/logging/v1/logging.proto", 10)
	opMsg := api.NewTestMessage("Operation").
		WithPackage("google.longrunning").
		WithSourceLocation("google/longrunning/operations.proto", 10)

	ptLog := (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("logs").WithVerb("write")
	lroMethod := makeTestRestMethod("WriteLog", "POST", "*", ptLog, reqMsg).
		WithOutput(opMsg).
		WithOperationInfo(&api.OperationInfo{ResponseTypeID: ".google.example.logging.v1.WriteLogResponse", MetadataTypeID: ".google.example.logging.v1.WriteLogMetadata"})

	cancelReq := api.NewTestMessage("CancelOperationRequest").
		WithPackage("google.longrunning").
		WithSourceLocation("google/longrunning/operations.proto", 20)
	emptyMsg := api.NewTestMessage("Empty").
		WithPackage("google.protobuf").
		WithSourceLocation("google/protobuf/empty.proto", 10)

	ptCancel := (&api.PathTemplate{}).WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithLiteral("operations").WithMatch()).WithVerb("cancel")
	cancelMethod := makeTestRestMethod("CancelOperation", "POST", "", ptCancel, cancelReq).
		WithOutput(emptyMsg)
	cancelMethod.SourceServiceID = ".google.longrunning.Operations"

	svc := api.NewTestService("LoggingService").
		WithPackage("google.example.logging.v1").
		WithSourceLocation("google/example/logging/v1/logging.proto", 30).
		WithMethods(lroMethod, cancelMethod)

	model := api.NewTestAPI([]*api.Message{reqMsg, opMsg, cancelReq, emptyMsg}, nil, []*api.Service{svc}).
		WithPackageName("google.example.logging.v1")

	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	svcAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok || svcAnn == nil {
		t.Fatalf("expected *serviceAnnotations on service.Codec, got %T", svc.Codec)
	}
	restAnn := svcAnn.RestTransport
	if restAnn == nil {
		t.Fatal("restAnn is nil")
	}

	if diff := cmp.Diff(true, restAnn.HasLRO); diff != "" {
		t.Errorf("restAnn.HasLRO mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, restAnn.HasOperationsMixin); diff != "" {
		t.Errorf("restAnn.HasOperationsMixin mismatch (-want +got):\n%s", diff)
	}
	if len(restAnn.LROOperations) == 0 {
		t.Errorf("len(restAnn.LROOperations) = 0, want > 0")
	}

	var hasOpImport bool
	for _, imp := range restAnn.TypeImports {
		if imp.From == "google.longrunning" && imp.Import == "operations_pb2" {
			hasOpImport = true
		}
	}
	if !hasOpImport {
		t.Errorf("TypeImports missing google.longrunning.operations_pb2")
	}

	if len(restAnn.MixinMethods) != 1 {
		t.Fatalf("len(restAnn.MixinMethods) = %d, want 1", len(restAnn.MixinMethods))
	}
	if diff := cmp.Diff("cancel_operation", restAnn.MixinMethods[0].Name); diff != "" {
		t.Errorf("MixinMethods[0].Name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, restAnn.MixinMethods[0].IsVoid); diff != "" {
		t.Errorf("MixinMethods[0].IsVoid mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateRestTransport_VoidMethod(t *testing.T) {
	reqMsg := api.NewTestMessage("DeleteRecordRequest").
		WithPackage("google.example.db.v1").
		WithSourceLocation("google/example/db/v1/db.proto", 10)
	emptyMsg := api.NewTestMessage("Empty").
		WithPackage("google.protobuf").
		WithSourceLocation("google/protobuf/empty.proto", 10)

	pt := (&api.PathTemplate{}).WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithMatch())
	deleteMethod := makeTestRestMethod("DeleteRecord", "DELETE", "", pt, reqMsg).
		WithOutput(emptyMsg)
	deleteMethod.ReturnsEmpty = true

	svc := api.NewTestService("DatabaseService").
		WithPackage("google.example.db.v1").
		WithSourceLocation("google/example/db/v1/db.proto", 20).
		WithMethods(deleteMethod)

	model := api.NewTestAPI([]*api.Message{reqMsg, emptyMsg}, nil, []*api.Service{svc}).
		WithPackageName("google.example.db.v1")

	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	svcAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok || svcAnn == nil {
		t.Fatalf("expected *serviceAnnotations on service.Codec, got %T", svc.Codec)
	}
	restAnn := svcAnn.RestTransport
	if restAnn == nil {
		t.Fatal("restAnn is nil")
	}

	if len(restAnn.PrimaryMethods) != 1 {
		t.Fatalf("len(restAnn.PrimaryMethods) = %d, want 1", len(restAnn.PrimaryMethods))
	}
	pm := restAnn.PrimaryMethods[0]
	if diff := cmp.Diff(true, pm.IsVoid); diff != "" {
		t.Errorf("pm.IsVoid mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("empty_pb2.Empty", pm.PropertyOutputTypeIdent); diff != "" {
		t.Errorf("pm.PropertyOutputTypeIdent mismatch (-want +got):\n%s", diff)
	}

	var hasEmptyImport bool
	for _, imp := range restAnn.TypeImports {
		if imp.From == "google.protobuf.empty_pb2" && imp.As == "empty_pb2" {
			hasEmptyImport = true
		}
	}
	if !hasEmptyImport {
		t.Errorf("TypeImports missing google.protobuf.empty_pb2 as empty_pb2")
	}
}

func TestDocSummaryForRest(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    string
		wantLead string
		wantRest string
		wantWrap bool
	}{
		{
			name:     "short method name",
			input:    "CreateBook",
			wantLead: "create book",
			wantRest: "",
			wantWrap: false,
		},
		{
			name:     "long method name that wraps",
			input:    "AnalyzeOrgPolicyGovernedAssetsResponse",
			wantLead: "analyze org policy",
			wantRest: "governed assets response",
			wantWrap: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotLead, gotRest, gotWrap := docSummaryForRest(test.input)
			if diff := cmp.Diff(test.wantLead, gotLead); diff != "" {
				t.Errorf("lead mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantRest, gotRest); diff != "" {
				t.Errorf("rest mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantWrap, gotWrap, cmpopts.EquateComparable()); diff != "" {
				t.Errorf("wrap mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateRestTransport_Error(t *testing.T) {
	tempDir := t.TempDir()
	pkgDir := filepath.Join(tempDir, "google", "cloud", "myapi", "v1")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(pkgDir, "myapi_v1.yaml")
	if err := os.WriteFile(filePath, []byte("authentication:\n  rules: [invalid yaml"), 0o644); err != nil {
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
	_, err := c.annotateRestTransport(svc, nil)
	if err == nil {
		t.Fatalf("annotateRestTransport(%s) error = nil, want %v", svc.Name, ErrLoadServiceConfig)
	}
	if !errors.Is(err, ErrLoadServiceConfig) {
		t.Errorf("annotateRestTransport(%s) error = %v, want %v", svc.Name, err, ErrLoadServiceConfig)
	}
}

func TestAnnotateRestTransport_RestAsyncIOAndPackageName(t *testing.T) {
	tempDir := t.TempDir()
	pkgDir := filepath.Join(tempDir, "google", "cloud", "redis", "v1")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	serviceConfig := `
publishing:
  library_settings:
    - version: google.cloud.redis.v1
      python_settings:
        experimental_features:
          rest_async_io_enabled: true
`
	if err := os.WriteFile(filepath.Join(pkgDir, "redis_v1.yaml"), []byte(serviceConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	method := api.NewTestMethod("GetInstance")
	svc := api.NewTestService("CloudRedis").
		WithPackage("google.cloud.redis.v1").
		WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc}).
		WithPackageName("google.cloud.redis.v1")
	lib := &config.Library{
		Name:  "google-cloud-redis",
		Roots: []string{tempDir},
		APIs: []*config.API{
			{Path: "google/cloud/redis/v1"},
		},
	}

	c := newTestCodec(t, model, lib)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	svcAnn, ok := svc.Codec.(*serviceAnnotations)
	if !ok {
		t.Fatalf("svc.Codec is %T, want *serviceAnnotations", svc.Codec)
	}
	restAnn := svcAnn.RestTransport
	if restAnn == nil {
		t.Fatal("svcAnn.RestTransport is nil, want non-nil")
	}

	if diff := cmp.Diff("google-cloud-redis", restAnn.PackageName); diff != "" {
		t.Errorf("PackageName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, restAnn.RestAsyncIOEnabled); diff != "" {
		t.Errorf("RestAsyncIOEnabled mismatch (-want +got):\n%s", diff)
	}
	if len(restAnn.WrappedMethods) != 1 {
		t.Fatalf("len(restAnn.WrappedMethods) = %d, want 1", len(restAnn.WrappedMethods))
	}
	if diff := cmp.Diff("get_instance", restAnn.WrappedMethods[0].Name); diff != "" {
		t.Errorf("WrappedMethods[0].Name mismatch (-want +got):\n%s", diff)
	}

	t.Run("fallback package name when c.PackageName is empty", func(t *testing.T) {
		libEmptyName := &config.Library{
			Roots: []string{tempDir},
			APIs: []*config.API{
				{Path: "google/cloud/redis/v1"},
			},
		}
		cEmpty := newTestCodec(t, model, libEmptyName)
		cEmpty.PackageName = ""
		if err := cEmpty.annotateModel(); err != nil {
			t.Fatal(err)
		}
		sAnn, ok := svc.Codec.(*serviceAnnotations)
		if !ok {
			t.Fatalf("svc.Codec is %T, want *serviceAnnotations", svc.Codec)
		}
		if sAnn.RestTransport == nil {
			t.Fatal("sAnn.RestTransport is nil, want non-nil")
		}
		if diff := cmp.Diff("google-cloud-redis", sAnn.RestTransport.PackageName); diff != "" {
			t.Errorf("PackageName fallback mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestAnnotateRestTransport_IAMPolicyMixin(t *testing.T) {
	reqMsg := api.NewTestMessage("SetIamPolicyRequest").WithPackage("google.iam.v1")
	respMsg := api.NewTestMessage("Policy").WithPackage("google.iam.v1")
	meth := api.NewTestMethod("SetIamPolicy").
		WithInput(reqMsg).
		WithOutput(respMsg)
	meth.SourceServiceID = ".google.iam.v1.IAMPolicy"
	svc := api.NewTestService("ExampleService").
		WithPackage("google.example.v1").
		WithMethods(meth)
	svc.DefaultHost = "example.googleapis.com"
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc}).
		WithPackageName("google.example.v1")
	c := newTestCodec(t, model, nil)
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}
	sAnn := svc.Codec.(*serviceAnnotations)
	if sAnn.RestTransport == nil || !sAnn.RestTransport.HasIAMPolicyMixin {
		t.Errorf("got HasIAMPolicyMixin=%v, want true",
			sAnn.RestTransport != nil && sAnn.RestTransport.HasIAMPolicyMixin)
	}
}
