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
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateTransports_LROAndMixins(t *testing.T) {
	outdir := t.TempDir()

	req := api.NewTestMessage("CreateBarRequest")
	lroMeth := api.NewTestMethod("CreateBar").
		WithInput(req).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".google.cloud.bar.v1.Bar",
			MetadataTypeID: ".google.cloud.bar.v1.OperationMetadata",
		})

	getLocMeth := api.NewTestMethod("GetLocation")
	getLocMeth.SourceServiceID = ".google.cloud.location.Locations"
	getLocMeth.InputTypeID = ".google.cloud.location.GetLocationRequest"
	getLocMeth.OutputTypeID = ".google.cloud.location.Location"

	getOpMeth := api.NewTestMethod("GetOperation")
	getOpMeth.SourceServiceID = ".google.longrunning.Operations"
	getOpMeth.InputTypeID = ".google.longrunning.GetOperationRequest"
	getOpMeth.OutputTypeID = ".google.longrunning.Operation"

	svc := api.NewTestService("BarService").
		WithPackage("google.cloud.bar.v1").
		WithMethods(lroMeth, getLocMeth, getOpMeth)
	svc.DefaultHost = "bar.googleapis.com"

	req.WithSourceLocation("bar.proto", 1)
	svc.WithSourceLocation("bar.proto", 10)
	model := api.NewTestAPI([]*api.Message{req}, nil, []*api.Service{svc})
	model.PackageName = "google.cloud.bar.v1"
	model.Name = "google-cloud-bar"

	library := &config.Library{
		Name:          "google-cloud-bar",
		Version:       "1.0.0",
		CopyrightYear: "2026",
	}

	if err := Generate(t.Context(), model, outdir, library); err != nil {
		t.Fatal(err)
	}

	transportsDir := filepath.Join(outdir, "google", "cloud", "bar_v1", "services", "bar_service", "transports")

	t.Run("grpc.py operations_client", func(t *testing.T) {
		grpcBytes, err := os.ReadFile(filepath.Join(transportsDir, "grpc.py"))
		if err != nil {
			t.Fatal(err)
		}
		gotLRO := extractBlock(t, string(grpcBytes), "    @property\n    def operations_client(", "return self._operations_client")
		if !strings.Contains(gotLRO, "operations_v1.OperationsClient") {
			t.Errorf("got operations_client:\n%s\nwant to contain operations_v1.OperationsClient", gotLRO)
		}
	})

	t.Run("grpc.py get_location", func(t *testing.T) {
		grpcBytes, err := os.ReadFile(filepath.Join(transportsDir, "grpc.py"))
		if err != nil {
			t.Fatal(err)
		}
		gotLoc := extractBlock(t, string(grpcBytes), "    @property\n    def get_location(", "return self._stubs[\"get_location\"]")
		if !strings.Contains(gotLoc, "/google.cloud.location.Locations/GetLocation") {
			t.Errorf("got get_location:\n%s\nwant to contain /google.cloud.location.Locations/GetLocation", gotLoc)
		}
	})

	t.Run("grpc.py get_operation", func(t *testing.T) {
		grpcBytes, err := os.ReadFile(filepath.Join(transportsDir, "grpc.py"))
		if err != nil {
			t.Fatal(err)
		}
		gotOp := extractBlock(t, string(grpcBytes), "    @property\n    def get_operation(", "return self._stubs[\"get_operation\"]")
		if !strings.Contains(gotOp, "/google.longrunning.Operations/GetOperation") {
			t.Errorf("got get_operation:\n%s\nwant to contain /google.longrunning.Operations/GetOperation", gotOp)
		}
	})

	t.Run("grpc_asyncio.py operations_client", func(t *testing.T) {
		grpcAsyncBytes, err := os.ReadFile(filepath.Join(transportsDir, "grpc_asyncio.py"))
		if err != nil {
			t.Fatal(err)
		}
		gotLRO := extractBlock(t, string(grpcAsyncBytes), "    @property\n    def operations_client(", "return self._operations_client")
		if !strings.Contains(gotLRO, "operations_v1.OperationsAsyncClient") {
			t.Errorf("got operations_client:\n%s\nwant to contain operations_v1.OperationsAsyncClient", gotLRO)
		}
	})

	t.Run("grpc_asyncio.py get_location", func(t *testing.T) {
		grpcAsyncBytes, err := os.ReadFile(filepath.Join(transportsDir, "grpc_asyncio.py"))
		if err != nil {
			t.Fatal(err)
		}
		gotLoc := extractBlock(t, string(grpcAsyncBytes), "    @property\n    def get_location(", "return self._stubs[\"get_location\"]")
		if !strings.Contains(gotLoc, "/google.cloud.location.Locations/GetLocation") {
			t.Errorf("got get_location:\n%s\nwant to contain /google.cloud.location.Locations/GetLocation", gotLoc)
		}
	})

	t.Run("grpc_asyncio.py get_operation", func(t *testing.T) {
		grpcAsyncBytes, err := os.ReadFile(filepath.Join(transportsDir, "grpc_asyncio.py"))
		if err != nil {
			t.Fatal(err)
		}
		gotOp := extractBlock(t, string(grpcAsyncBytes), "    @property\n    def get_operation(", "return self._stubs[\"get_operation\"]")
		if !strings.Contains(gotOp, "/google.longrunning.Operations/GetOperation") {
			t.Errorf("got get_operation:\n%s\nwant to contain /google.longrunning.Operations/GetOperation", gotOp)
		}
	})
}
