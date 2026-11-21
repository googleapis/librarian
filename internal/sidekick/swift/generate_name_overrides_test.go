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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateService_NameOverrides(t *testing.T) {
	outDir := t.TempDir()

	request := api.NewTestMessage("CreateTopicRequest").
		WithPackage("google.pubsub.v1").
		WithFields(api.NewTestField("name").WithType(api.TypezString))
	response := api.NewTestMessage("Topic").WithPackage("google.pubsub.v1")
	method := api.NewTestMethod("CreateTopic").
		WithInput(request).
		WithOutput(response).
		WithVerb("PUT").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("name").Build())
	service := api.NewTestService("Publisher").
		WithPackage("google.pubsub.v1").
		WithMethods(method).
		WithQuickstartMethod(method)

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service}).
		WithPackageName("google.pubsub.v1")

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{Name: "GoogleGax", RequiredByServices: true},
	})
	swiftCfg.PerServiceTraits = true
	swiftCfg.NameOverrides = map[string]string{
		".google.pubsub.v1.Publisher": "TopicAdmin",
	}
	library := &config.Library{
		Name:  "google-pubsub-v1",
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	wantFiles := map[string][]string{
		filepath.Join("Sources", "GooglePubsubV1", "TopicAdmin.swift"): {
			"#if TopicAdmin",
			"public final class TopicAdminClient: Clients.TopicAdminProtocol, Sendable",
			`@Snippet(path: "TopicAdminQuickstart")`,
			`@Snippet(path: "TopicAdmin_CreateTopic")`,
			"Clients.TopicAdminTransport(options)",
			"Clients.TopicAdminRetry(inner, options: options)",
			"Clients.TopicAdminLogging(inner, logger: logger)",
		},
		filepath.Join("Sources", "GooglePubsubV1", "TopicAdmin+Stub.swift"): {
			"#if TopicAdmin",
			"protocol TopicAdminStub: Sendable",
		},
		filepath.Join("Sources", "GooglePubsubV1", "TopicAdmin+Transport.swift"): {
			"#if TopicAdmin",
			"final class TopicAdminTransport: TopicAdminStub",
		},
		filepath.Join("Sources", "GooglePubsubV1", "TopicAdmin+Logging.swift"): {
			"#if TopicAdmin",
			"final class TopicAdminLogging: TopicAdminStub",
			`"gcp.experimental.swift.client"] = "TopicAdmin"`,
		},
		filepath.Join("Sources", "GooglePubsubV1", "TopicAdmin+Retry.swift"): {
			"#if TopicAdmin",
			"final class TopicAdminRetry: TopicAdminStub",
		},
		filepath.Join("Sources", "GooglePubsubV1", "GooglePubsubV1.docc", "Index.md"): {
			"``TopicAdminClient``",
			"(enabled by the `TopicAdmin` trait)",
			`@Snippet(path: "TopicAdminQuickstart")`,
		},
		filepath.Join("Sources", "GooglePubsubV1", "GooglePubsubV1.docc", "PackageTraits.md"): {
			"| `TopicAdmin` |",
			"``TopicAdminClient``",
		},
		filepath.Join("Snippets", "TopicAdminQuickstart.swift"): {
			"#if TopicAdmin",
			"GooglePubsubV1.TopicAdminClient()",
		},
		filepath.Join("Snippets", "TopicAdmin_CreateTopic.swift"): {
			"#if TopicAdmin",
			"func sample(client: TopicAdminClient",
			"GooglePubsubV1.TopicAdminClient()",
		},
		"README.md": {
			"`TopicAdminClient`",
			"| `TopicAdmin` |",
		},
		"Package.swift": {
			`name: "TopicAdmin"`,
		},
	}
	for rel, wants := range wantFiles {
		bytes, err := os.ReadFile(filepath.Join(outDir, rel))
		if err != nil {
			t.Errorf("expected generated file %s: %v", rel, err)
			continue
		}
		got := string(bytes)
		for _, want := range wants {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q\ngot:\n%s", rel, want, got)
			}
		}
	}

	unwantFiles := []string{
		filepath.Join("Sources", "GooglePubsubV1", "Publisher.swift"),
		filepath.Join("Snippets", "PublisherQuickstart.swift"),
		filepath.Join("Snippets", "Publisher_CreateTopic.swift"),
	}
	for _, rel := range unwantFiles {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err == nil {
			t.Errorf("unexpected un-overridden file %s", rel)
		}
	}

	topicAdminBytes, err := os.ReadFile(filepath.Join(outDir, "Sources", "GooglePubsubV1", "TopicAdmin.swift"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(topicAdminBytes), "Publisher_") {
		t.Errorf("TopicAdmin.swift still references un-overridden snippet name Publisher_:\n%s", string(topicAdminBytes))
	}
}

func TestGenerateService_NameOverrides_GRPCWirePathUnchanged(t *testing.T) {
	outDir := t.TempDir()

	request := api.NewTestMessage("CreateTopicRequest").
		WithPackage("google.pubsub.v1").
		WithFields(api.NewTestField("name").WithType(api.TypezString))
	response := api.NewTestMessage("Topic").WithPackage("google.pubsub.v1")
	method := api.NewTestMethod("CreateTopic").
		WithInput(request).
		WithOutput(response).
		WithVerb("PUT").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("name").Build())
	service := api.NewTestService("Publisher").
		WithPackage("google.pubsub.v1").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service}).
		WithPackageName("google.pubsub.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	module := &config.SwiftModule{
		Output:     outDir,
		ModuleType: "grpc-client",
		ModulePath: "PubSubProtos",
		NameOverrides: map[string]string{
			".google.pubsub.v1.Publisher": "TopicAdmin",
		},
	}
	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{Name: "GoogleGax", RequiredByServices: true},
	})
	library := &config.Library{
		Name:  "google-pubsub-v1",
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	transportFile := filepath.Join(outDir, "TopicAdmin+Transport.swift")
	bytes, err := os.ReadFile(transportFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(bytes)
	for _, want := range []string{
		"final class TopicAdminTransport: TopicAdminStub",
		`path: "/google.pubsub.v1.Publisher/CreateTopic"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: missing %q\ngot:\n%s", transportFile, want, got)
		}
	}
}

func TestGenerateService_NameOverrides_ServicePlaceholder(t *testing.T) {
	outDir := t.TempDir()

	placeholder := api.NewTestMessage("Advice").
		WithPackage("google.cloud.test.v1").
		WithServicePlaceholder()
	nestedReq := api.NewTestMessage("GetAdviceRequest").
		WithFields(api.NewTestField("project").WithType(api.TypezString))
	placeholder.WithMessages(nestedReq)
	resp := api.NewTestMessage("AdviceResponse").WithPackage("google.cloud.test.v1")

	method := api.NewTestMethod("GetAdvice").
		WithInput(nestedReq).
		WithOutput(resp).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("project").Build())
	service := api.NewTestService("Advice").
		WithPackage("google.cloud.test.v1").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{placeholder, resp}, nil, []*api.Service{service}).
		WithPackageName("google.cloud.test.v1")

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{Name: "GoogleGax", RequiredByServices: true},
	})
	swiftCfg.PerServiceTraits = true
	swiftCfg.NameOverrides = map[string]string{
		".google.cloud.test.v1.Advice": "CustomAdvice",
	}
	library := &config.Library{
		Name:  "google-cloud-test-v1",
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	reqFile := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "CustomAdvice+Requests.swift")
	reqBytes, err := os.ReadFile(reqFile)
	if err != nil {
		t.Fatal(err)
	}
	reqStr := string(reqBytes)
	for _, want := range []string{
		"#if CustomAdvice",
		"extension CustomAdviceClient {",
		"public struct GetAdviceRequest:",
	} {
		if !strings.Contains(reqStr, want) {
			t.Errorf("%s: missing %q\ngot:\n%s", reqFile, want, reqStr)
		}
	}

	svcFile := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "CustomAdvice.swift")
	svcBytes, err := os.ReadFile(svcFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svcBytes), "CustomAdviceClient.GetAdviceRequest") {
		t.Errorf("%s: expected nested placeholder request type CustomAdviceClient.GetAdviceRequest\ngot:\n%s", svcFile, string(svcBytes))
	}
}
