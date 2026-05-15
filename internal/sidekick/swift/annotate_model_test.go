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
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestModelAnnotations(t *testing.T) {
	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{},
		[]*api.Service{{Name: "Workflows", Package: "google.cloud.workflows.v1"}})
	codec := newTestCodec(t, model, map[string]string{"copyright-year": "2038"})
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}
	want := &modelAnnotations{
		PackageName:   "GoogleCloudWorkflowsV1",
		CopyrightYear: "2038",
		MonorepoRoot:  ".",
		WktPackage:    "GoogleCloudWkt",
	}
	if diff := cmp.Diff(want, model.Codec, cmpopts.IgnoreFields(modelAnnotations{}, "BoilerPlate", "DependsOn")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestModelAnnotations_MessagesWithWkt(t *testing.T) {
	enum := &api.Enum{
		Name: "SomeEnum", ID: ".test.SomeSnum", Package: "test",
		Values: []*api.EnumValue{{Name: "UNSPECIFIED", Number: 0}},
	}
	enum.UniqueNumberValues = enum.Values
	for _, test := range []struct {
		name  string
		model *api.API
		want  map[string]bool
	}{
		{
			name: "Messages with wkt",
			model: api.NewTestAPI(
				[]*api.Message{{Name: "Request", ID: ".test.Request", Package: "test"}}, nil, nil),
			want: map[string]bool{"GoogleCloudWkt": true},
		},
		{
			name:  "Enum with wkt",
			model: api.NewTestAPI(nil, []*api.Enum{enum}, nil),
			want:  map[string]bool{"GoogleCloudWkt": false},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			codec := newTestCodec(t, test.model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann := test.model.Codec.(*modelAnnotations)
			got := map[string]bool{}
			for _, d := range codec.Dependencies {
				_, ok := ann.DependsOn[d.Name]
				got[d.Name] = ok
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestModelAnnotations_WithExternalDependencies(t *testing.T) {
	externalMessage := &api.Message{
		Name:    "ExternalMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.ExternalMessage",
	}

	message := &api.Message{
		Name:    "LocalMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.LocalMessage",
		Fields: []*api.Field{
			{
				Name:    "ext_field",
				Typez:   api.TypezMessage,
				TypezID: ".google.cloud.external.v1.ExternalMessage",
			},
		},
	}

	service := &api.Service{
		Name:    "TestService",
		ID:      ".google.cloud.test.v1.TestService",
		Package: "google.cloud.test.v1",
	}

	model := api.NewTestAPI(
		[]*api.Message{message}, []*api.Enum{}, []*api.Service{service})
	model.AddMessage(externalMessage)
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "google.cloud.external.v1", Name: "GoogleCloudExternalWithOverrideV1"},
		{ApiPackage: "google.cloud.unused.v1", Name: "GoogleUnusedPackage"},
		{Name: "GoogleCloudGax", RequiredByServices: true},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	want := []string{"GoogleCloudExternalWithOverrideV1", "GoogleCloudGax", "GoogleCloudWkt"}
	var got []string
	for name := range ann.DependsOn {
		got = append(got, name)
	}
	slices.Sort(got)
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
	model := api.NewTestAPI(
		[]*api.Message{{
			Name:    "LocalMessage",
			Package: "google.cloud.placeholder.v1",
			ID:      ".google.cloud.placeholder.v1.LocalMessage",
		}},
		[]*api.Enum{},
		[]*api.Service{{Name: "DummyService", Package: "google.cloud.placeholder.v1"}},
	)
	model.PackageName = "google.cloud.placeholder.v1"
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "google.cloud.placeholder.v1", Name: "GoogleCloudPlaceholderV1"},
		{ApiPackage: "google.cloud.other.v1", Name: "GoogleCloudOtherV1", RequiredByServices: true},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	want := []string{"GoogleCloudOtherV1", "GoogleCloudWkt"}
	var got []string
	for name := range ann.DependsOn {
		got = append(got, name)
	}
	slices.Sort(got)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestModelAnnotations_Pagination(t *testing.T) {
	pageSizeField := &api.Field{Name: "page_size", JSONName: "pageSize", Typez: api.TypezInt32}
	pageTokenField := &api.Field{Name: "page_token", JSONName: "pageToken", Typez: api.TypezString}
	inputType := &api.Message{
		Name:    "ListSecretsRequest",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.ListSecretsRequest",
		Fields:  []*api.Field{pageSizeField, pageTokenField},
	}
	pageSizeField.Parent = inputType
	pageTokenField.Parent = inputType

	itemField := &api.Field{Name: "secrets", JSONName: "secrets", Typez: api.TypezMessage, TypezID: ".google.cloud.secretmanager.v1.Secret", Repeated: true}
	nextPageTokenField := &api.Field{Name: "next_page_token", JSONName: "nextPageToken", Typez: api.TypezString}
	outputType := &api.Message{
		Name:    "ListSecretsResponse",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.ListSecretsResponse",
		Fields:  []*api.Field{itemField, nextPageTokenField},
		Pagination: &api.PaginationInfo{
			NextPageToken: nextPageTokenField,
			PageableItem:  itemField,
		},
	}
	itemField.Parent = outputType
	nextPageTokenField.Parent = outputType

	secretType := &api.Message{
		Name:    "Secret",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.Secret",
	}

	iam := &api.Service{
		Name:    "SecretManagerService",
		ID:      ".google.cloud.secretmanager.v1.SecretManagerService",
		Package: "google.cloud.secretmanager.v1",
		Methods: []*api.Method{
			{
				Name:         "ListSecrets",
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "GET",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("secrets"),
					}},
				},
				Pagination: pageTokenField,
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{inputType, outputType, secretType}, nil, []*api.Service{iam})
	model.PackageName = "google.cloud.secretmanager.v1"

	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{Name: "GoogleCloudGax", RequiredByServices: true},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}

	// Verify GoogleCloudGax is in DependsOn because we have a service.
	if _, ok := ann.DependsOn["GoogleCloudGax"]; !ok {
		t.Errorf("expected GoogleCloudGax dependency to be in DependsOn")
	}
}

func TestModelAnnotations_ConditionalLro(t *testing.T) {
	emptyType := &api.Message{
		Name:    "Empty",
		Package: "google.protobuf",
		ID:      ".google.protobuf.Empty",
	}
	operationType := &api.Message{
		Name:    "Operation",
		Package: "google.longrunning",
		ID:      ".google.longrunning.Operation",
	}

	lroMethod := &api.Method{
		Name:         "CreateWorkflow",
		ID:           ".google.cloud.workflows.v1.Workflows.CreateWorkflow",
		InputTypeID:  emptyType.ID,
		InputType:    emptyType,
		OutputTypeID: operationType.ID,
		OutputType:   operationType,
		IsLRO:        true,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{{
				Verb:         "POST",
				PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("workflows"),
			}},
		},
	}
	nonLroMethod := &api.Method{
		Name:         "GetWorkflow",
		ID:           ".google.cloud.workflows.v1.Workflows.GetWorkflow",
		InputTypeID:  emptyType.ID,
		InputType:    emptyType,
		OutputTypeID: emptyType.ID,
		OutputType:   emptyType,
		IsLRO:        false,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{{
				Verb:         "GET",
				PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("workflows"),
			}},
		},
	}

	serviceWithLro := &api.Service{
		Name:    "WorkflowsWithLro",
		ID:      ".google.cloud.workflows.v1.WorkflowsWithLro",
		Package: "google.cloud.workflows.v1",
		Methods: []*api.Method{lroMethod},
	}
	lroMethod.Service = serviceWithLro
	lroMethod.SourceService = serviceWithLro

	serviceWithoutLro := &api.Service{
		Name:    "WorkflowsWithoutLro",
		ID:      ".google.cloud.workflows.v1.WorkflowsWithoutLro",
		Package: "google.cloud.workflows.v1",
		Methods: []*api.Method{nonLroMethod},
	}
	nonLroMethod.Service = serviceWithoutLro
	nonLroMethod.SourceService = serviceWithoutLro

	model := api.NewTestAPI([]*api.Message{emptyType, operationType}, nil, []*api.Service{serviceWithLro, serviceWithoutLro})
	model.PackageName = "google.cloud.workflows.v1"

	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{Name: "GoogleCloudLro", RequiredByServices: true},
		{Name: "GoogleCloudGax", RequiredByServices: true},
		{ApiPackage: "google.longrunning", Name: "GoogleLongrunning"},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	// Verify package-level DependsOn contains GoogleCloudLro because at least one service needs it.
	ann, ok := model.Codec.(*modelAnnotations)
	if !ok {
		t.Fatalf("expected model.Codec to be *modelAnnotations, got %T", model.Codec)
	}
	if _, ok := ann.DependsOn["GoogleCloudLro"]; !ok {
		t.Errorf("expected GoogleCloudLro to be in package-level DependsOn")
	}

	// Verify serviceWithLro imports GoogleCloudLro
	gotLroServiceImports := serviceWithLro.Codec.(*serviceAnnotations).ServiceImports
	if !slices.Contains(gotLroServiceImports, "GoogleCloudLro") {
		t.Errorf("expected ServiceImports for serviceWithLro to contain GoogleCloudLro, got %v", gotLroServiceImports)
	}

	// Verify serviceWithoutLro does NOT import GoogleCloudLro
	gotNonLroServiceImports := serviceWithoutLro.Codec.(*serviceAnnotations).ServiceImports
	if slices.Contains(gotNonLroServiceImports, "GoogleCloudLro") {
		t.Errorf("expected ServiceImports for serviceWithoutLro to NOT contain GoogleCloudLro, got %v", gotNonLroServiceImports)
	}
}
