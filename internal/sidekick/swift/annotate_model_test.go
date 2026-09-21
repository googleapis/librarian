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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestModelAnnotations(t *testing.T) {
	service := api.NewTestService("Workflows").WithPackage("google.cloud.workflows.v1")
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	codec := newTestCodec(t, model, &config.Library{CopyrightYear: "2038"})
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}
	want := &modelAnnotations{
		LibraryName:     "GoogleCloudWorkflowsV1",
		PackageName:     "google-cloud-workflows-v1",
		PackageRepoName: "swift-google-cloud-workflows-v1",
		PackageVersion:  "0.0.0",
		CopyrightYear:   "2038",
		MonorepoRoot:    ".",
		WktPackage:      "GoogleWKT",
	}
	if diff := cmp.Diff(want, model.Codec, cmpopts.IgnoreFields(modelAnnotations{}, "BoilerPlate", "DependsOn")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestModelAnnotations_MessagesWithWkt(t *testing.T) {
	enum := api.NewTestEnum("SomeEnum").
		WithID(".test.SomeSnum").
		WithValues(api.NewTestEnumValue("UNSPECIFIED", 0))
	for _, test := range []struct {
		name  string
		model *api.API
		want  []string
	}{
		{
			name: "Messages with wkt",
			model: api.NewTestAPI(
				[]*api.Message{api.NewTestMessage("Request")}, nil, nil),
			want: []string{"GoogleWKT"},
		},
		{
			name:  "Enum with wkt",
			model: api.NewTestAPI(nil, []*api.Enum{enum}, nil),
			want:  []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			codec := newTestCodec(t, test.model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann := test.model.Codec.(*modelAnnotations)
			got := []string{}
			for _, d := range ann.Dependencies() {
				got = append(got, d.Name)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestModelAnnotations_WithExternalDependencies(t *testing.T) {
	externalMessage := api.NewTestMessage("ExternalMessage").
		WithPackage("google.cloud.external.v1")

	message := api.NewTestMessage("LocalMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("ext_field").WithMessageType(externalMessage),
		)

	service := api.NewTestService("TestService").
		WithPackage("google.cloud.test.v1")

	model := api.NewTestAPI(
		[]*api.Message{message}, nil, []*api.Service{service})
	model.AddMessage(externalMessage)
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "google.cloud.external.v1", Name: "GoogleCloudExternalWithOverrideV1"},
		{ApiPackage: "google.cloud.unused.v1", Name: "GoogleUnusedPackage"},
		{Name: "GoogleGax", RequiredByServices: true},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	want := map[string]bool{
		"GoogleCloudExternalWithOverrideV1": true,
		"GoogleGax":                         true, // required by the service
		"GoogleWKT":                         true,
		"GoogleUnusedPackage":               false,
	}
	got := map[string]bool{}
	for _, dep := range codec.Dependencies {
		_, got[dep.Name] = ann.DependsOn[dep.Name]
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	msg := model.Messages[0]
	msgAnn, ok := msg.Codec.(*messageAnnotations)
	if !ok {
		t.Fatalf("expected message.Codec to be *messageAnnotations, got %T", msg.Codec)
	}
	if msgAnn.Model != ann {
		t.Errorf("expected msgAnn.Model to be %p, got %p", ann, msgAnn.Model)
	}
}

func TestModelAnnotations_IgnoreSelfDependency(t *testing.T) {
	service := api.NewTestService("DummyService").WithPackage("google.cloud.placeholder.v1")
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	model.PackageName = "google.cloud.placeholder.v1"
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "google.cloud.placeholder.v1", Name: "GoogleCloudPlaceholderV1"},
		{ApiPackage: "google.cloud.other.v1", Name: "GoogleCloudOtherV1", RequiredByServices: true},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}
	dep, err := codec.addPackageDependency("GoogleCloudPlaceholderV1")
	if err != nil {
		t.Fatal(err)
	}
	if dep != nil {
		t.Errorf("expected to not set ignore self dependency, got %v", dep)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	// Self dependency should be ignored, other should be present.
	want := map[string]bool{
		"GoogleGax":                true,  // always required by services
		"GoogleCloudOtherV1":       true,  // required by the service
		"GoogleCloudPlaceholderV1": false, // this is the current package and should not be included as a dependency
		"GoogleWKT":                true,  // always required by message types
	}
	got := map[string]bool{}
	for _, dep := range codec.Dependencies {
		_, got[dep.Name] = ann.DependsOn[dep.Name]
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestModelAnnotations_Pagination(t *testing.T) {
	pageSizeField := api.NewTestField("page_size").WithType(api.TypezInt32)
	pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
	inputType := api.NewTestMessage("ListSecretsRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(pageSizeField, pageTokenField)

	secretType := api.NewTestMessage("Secret").
		WithPackage("google.cloud.secretmanager.v1")

	itemField := api.NewTestField("secrets").
		WithMessageType(secretType).
		WithRepeated()
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	outputType := api.NewTestMessage("ListSecretsResponse").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(itemField, nextPageTokenField).
		WithPagination(nextPageTokenField, itemField)

	method := api.NewTestMethod("ListSecrets").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("secrets")).
		WithPagination(pageTokenField)

	iam := api.NewTestService("SecretManagerService").
		WithPackage("google.cloud.secretmanager.v1").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{inputType, outputType, secretType}, nil, []*api.Service{iam})
	model.PackageName = "google.cloud.secretmanager.v1"

	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{Name: "GoogleGax", RequiredByServices: true},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	if _, ok := ann.DependsOn["GoogleGax"]; !ok {
		t.Errorf("expected GoogleGax dependency to be present in DependsOn")
	}
}

func TestModelAnnotations_Gating(t *testing.T) {
	model := makeRequiredServicesTestModel()
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "external", Name: "GoogleCloudExternal"},
	})
	codec.PerServiceTraits = true
	codec.DefaultTraits = []string{"TestService"}

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	wantAllTraits := []*traitDefinition{
		{Name: "TestService", EnabledTraits: []string{"ZoneOperations"}},
		{Name: "ZoneOperations"},
	}
	if diff := cmp.Diff(wantAllTraits, ann.AllTraits); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if !ann.HasTraits() {
		t.Error("expected HasTraits() to be true")
	}

	if diff := cmp.Diff([]string{"TestService"}, ann.DefaultTraits); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
