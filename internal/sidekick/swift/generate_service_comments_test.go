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

package swift

import (
	"github.com/googleapis/librarian/internal/config"

	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateService_DocComments(t *testing.T) {
	outDir := t.TempDir()

	req := api.NewTestMessage("GetSecretRequest").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("project").WithType(api.TypezString),
			api.NewTestField("secret").WithType(api.TypezString),
		)
	res := api.NewTestMessage("Secret").
		WithPackage("google.cloud.test.v1")

	method := api.NewTestMethod("GetSecret").
		WithDocumentation("Documentation for GetSecret method.").
		WithInput(req).
		WithOutput(res).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("projects").WithVariableNamed("project").WithLiteral("secrets").WithVariableNamed("secret"))

	service := api.NewTestService("SecretManager").
		WithDocumentation("Documentation for SecretManager service.").
		WithPackage("google.cloud.test.v1").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{req, res}, nil, []*api.Service{service})

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "SecretManager.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify service documentation
	want := "/// Documentation for SecretManager service.\n///\n/// @Snippet(path: \"SecretManagerQuickstart\")\npublic final class SecretManagerClient: "
	got := extractBlock(t, contentStr, "/// Documentation for SecretManager service.", "public final class SecretManagerClient: ")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify method documentation
	want = "  /// Documentation for GetSecret method.\n  ///\n  /// @Snippet(path: \"SecretManager_GetSecret\")\n  public func getSecret"
	got = extractBlock(t, contentStr, "  /// Documentation for GetSecret method.", "public func getSecret")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateService_DiscoveryLRODocComments(t *testing.T) {
	outDir := t.TempDir()

	req := api.NewTestMessage("InsertInstanceRequest").
		WithPackage("google.cloud.compute.v1").
		WithFields(
			api.NewTestField("project").WithType(api.TypezString),
			api.NewTestField("zone").WithType(api.TypezString),
		)
	op := api.NewTestMessage("Operation").
		WithPackage("google.cloud.compute.v1")

	method := api.NewTestMethod("Insert").
		WithDocumentation("Creates an instance resource.").
		WithInput(req).
		WithOutput(op).
		WithVerb("POST").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("compute").WithLiteral("v1").WithLiteral("projects").WithVariableNamed("project").WithLiteral("zones").WithVariableNamed("zone").WithLiteral("instances")).
		WithDiscoveryLro(&api.DiscoveryLro{
			PollingPathParameters: []string{"project", "zone"},
		})

	service := api.NewTestService("Instances").
		WithDocumentation("The Instances API.").
		WithPackage("google.cloud.compute.v1").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{req, op}, nil, []*api.Service{service})

	library := &config.Library{
		Swift: swiftConfig(t, []config.SwiftDependency{
			{Name: "GoogleGax", RequiredByServices: true},
			{ApiPackage: "google.cloud.compute.v1", Name: "GoogleCloudComputeV1"},
		}),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudComputeV1", "Instances.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify Discovery LRO method documentation includes the - Note: callout
	wantMethod := `  /// Creates an instance resource.
  ///
  /// - Note: Unlike standard gRPC long-running operations where polling returns the resulting
  ///   resource, Discovery-based operations return the terminal ` + "`GoogleCloudComputeV1.Operation`" + `
  ///   status object. To obtain the created or modified resource, invoke the corresponding ` + "`get`" + `
  ///   method once the operation completes.
  ///
  /// @Snippet(path: "Instances_Insert")
  public func insertPollingUntilDone`
	gotMethod := extractBlock(t, contentStr, "  /// Creates an instance resource.\n  ///\n  /// - Note: Unlike standard gRPC long-running operations where polling returns the resulting", "public func insertPollingUntilDone")
	if diff := cmp.Diff(wantMethod, gotMethod); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify protocol documentation includes the - Note: callout
	wantProtocol := `    /// See ` + "`InstancesClient.insert`" + `.
    ///
    /// - Note: Unlike standard gRPC long-running operations where polling returns the resulting
    ///   resource, Discovery-based operations return the terminal ` + "`GoogleCloudComputeV1.Operation`" + `
    ///   status object. To obtain the created or modified resource, invoke the corresponding ` + "`get`" + `
    ///   method once the operation completes.
    func insertPollingUntilDone`
	gotProtocol := extractBlock(t, contentStr, "    /// See `InstancesClient.insert`.\n    ///\n    /// - Note: Unlike standard gRPC long-running operations where polling returns the resulting", "func insertPollingUntilDone")
	if diff := cmp.Diff(wantProtocol, gotProtocol); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
