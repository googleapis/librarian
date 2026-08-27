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

package rust

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateServerStreaming(t *testing.T) {
	outDir := t.TempDir()

	request := api.NewTestMessage("ExpandRequest").WithPackage("test.v1")
	requestId := &api.Field{
		Name:          "request_id",
		JSONName:      "requestId",
		ID:            ".test.v1.ExpandRequest.request_id",
		Typez:         api.TypezString,
		AutoPopulated: true,
	}
	request.Fields = []*api.Field{
		{
			Name:     "content",
			JSONName: "content",
			ID:       ".test.v1.ExpandRequest.content",
			Typez:    api.TypezString,
		},
		requestId,
	}
	response := api.NewTestMessage("EchoResponse").WithPackage("test.v1")

	serverMethod := api.NewTestMethod("Expand").WithInput(request).WithOutput(response).WithServerSideStreaming()
	serverMethod.AutoPopulated = []*api.Field{requestId}
	serverMethod.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{{Verb: "POST", PathTemplate: &api.PathTemplate{}}},
	}
	service := api.NewTestService("Echo").WithPackage("test.v1").WithMethods(serverMethod)

	model := api.NewTestAPI([]*api.Message{request, response}, []*api.Enum{}, []*api.Service{service})
	model.PackageName = "test.v1"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec: map[string]string{
			"package:wkt":                      "source=google.protobuf,package=google-cloud-wkt",
			"package:prost":                    "package=prost,used-if=streaming",
			"package:prost-types":              "package=prost-types,used-if=streaming",
			"include-server-streaming-methods": "true",
		},
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	files := make(map[string]string)
	readFile := func(relPath string) string {
		if content, ok := files[relPath]; ok {
			return content
		}
		b, err := os.ReadFile(filepath.Join(outDir, relPath))
		if err != nil {
			t.Fatal(err)
		}
		files[relPath] = string(b)
		return files[relPath]
	}

	for _, test := range []struct {
		name     string
		file     string
		startStr string
		endStr   string
		want     string
	}{
		{
			name:     "client: method definition",
			file:     "src/client.rs",
			startStr: "    pub fn expand(&self) -> super::builder::echo::Expand",
			endStr:   "    }",
			want: `    pub fn expand(&self) -> super::builder::echo::Expand
    {
        super::builder::echo::Expand::new(self.inner.clone())
    }`,
		},
		{
			name:     "builder: struct definition",
			file:     "src/builder.rs",
			startStr: "    #[derive(Clone, Debug)]\n    pub struct Expand(",
			endStr:   ");",
			want:     "    #[derive(Clone, Debug)]\n    pub struct Expand(RequestBuilder<crate::model::ExpandRequest>);",
		},
		{
			name:     "builder: send method",
			file:     "src/builder.rs",
			startStr: "        /// Initiates the server stream.\n        pub async fn send(",
			endStr:   "        }",
			want: `        /// Initiates the server stream.
        pub async fn send(
            self,
        ) -> Result<google_cloud_gax::streaming::ResponseReceiver<crate::model::EchoResponse>> {
            (*self.0.stub).expand(self.0.request, self.0.options).await
        }`,
		},
		{
			name:     "builder: RequestBuilder trait impl",
			file:     "src/builder.rs",
			startStr: "    #[doc(hidden)]\n    impl crate::RequestBuilder for Expand {",
			endStr:   "        }\n    }",
			want: `    #[doc(hidden)]
    impl crate::RequestBuilder for Expand {
        fn request_options(&mut self) -> &mut crate::RequestOptions {
            &mut self.0.options
        }
    }`,
		},
		{
			name:     "stub: method signature",
			file:     "src/stub.rs",
			startStr: "    fn expand(",
			endStr:   "    }",
			want: `    fn expand(
        &self,
        _req: crate::model::ExpandRequest,
        _options: crate::RequestOptions,
    ) -> impl std::future::Future<Output = crate::Result<
        google_cloud_gax::streaming::ResponseReceiver<crate::model::EchoResponse>,
    >> + Send {
        gaxi::unimplemented::unimplemented_server_streaming_stub()
    }`,
		},
		{
			name:     "stub/dynamic: trait method",
			file:     "src/stub/dynamic.rs",
			startStr: "    async fn expand(",
			endStr:   ">;",
			want: `    async fn expand(
        &self,
        req: crate::model::ExpandRequest,
        options: crate::RequestOptions,
    ) -> crate::Result<
        google_cloud_gax::streaming::ResponseReceiver<crate::model::EchoResponse>,
    >;`,
		},
		{
			name:     "stub dynamic: blanket forwarding impl",
			file:     "src/stub/dynamic.rs",
			startStr: "    /// Forwards the call to the implementation provided by `T`.\n    async fn expand(",
			endStr:   "    }",
			want: `    /// Forwards the call to the implementation provided by ` + "`T`." + `
    async fn expand(
        &self,
        req: crate::model::ExpandRequest,
        options: crate::RequestOptions,
    ) -> crate::Result<
        google_cloud_gax::streaming::ResponseReceiver<crate::model::EchoResponse>,
    > {
        T::expand(self, req, options).await
    }`,
		},
		{
			name:     "tracing: forwarder method",
			file:     "src/tracing.rs",
			startStr: "    async fn expand(",
			endStr:   "    }",
			want: `    async fn expand(
        &self,
        req: crate::model::ExpandRequest,
        options: crate::RequestOptions,
    ) -> Result<google_cloud_gax::streaming::ResponseReceiver<crate::model::EchoResponse>> {
        self.inner.expand(req, options).await
    }`,
		},
		{
			name:     "transport: method implementation",
			file:     "src/transport.rs",
			startStr: "    async fn expand(",
			endStr:   "            .await\n    }",
			want: `    async fn expand(
        &self,
        req: crate::model::ExpandRequest,
        options: crate::RequestOptions,
    ) -> Result<google_cloud_gax::streaming::ResponseReceiver<crate::model::EchoResponse>> {
        let x_goog_request_params = [
            None::<String>; 0
        ]
        .into_iter()
        .flatten()
        .fold(String::new(), |b, p| b + "&" + &p);

        let extensions = {
            let mut e = gaxi::grpc::tonic::Extensions::new();
            e.insert(gaxi::grpc::tonic::GrpcMethod::new(
                "test.v1.Echo",
                "Expand",
            ));
            e
        };
        let path = http::uri::PathAndQuery::from_static(
            "/test.v1.Echo/Expand"
        );

        self.grpc_inner
            .execute_server_streaming::<
                crate::model::ExpandRequest,
                crate::model::EchoResponse,
                crate::prost::test::v1::ExpandRequest,
                crate::prost::test::v1::EchoResponse,
            >(
                extensions,
                path,
                req,
                options,
                &crate::info::X_GOOG_API_CLIENT_HEADER,
                &x_goog_request_params,
            )
            .await
    }`,
		},
		{
			name:     "builder: field setter",
			file:     "src/builder.rs",
			startStr: "        /// Sets the value of [request_id][crate::model::ExpandRequest::request_id].\n        pub fn set_request_id<T: Into<std::string::String>>(mut self, v: T) -> Self {",
			endStr:   "        }",
			want: `        /// Sets the value of [request_id][crate::model::ExpandRequest::request_id].
        pub fn set_request_id<T: Into<std::string::String>>(mut self, v: T) -> Self {
            self.0.request.request_id = v.into();
            self
        }`,
		},
		{
			name:     "cargo.toml: dependencies block",
			file:     "Cargo.toml",
			startStr: "[dependencies]\n",
			endStr:   "prost.workspace      = true\n",
			want: `[dependencies]
prost-types.workspace = true
prost.workspace      = true
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := readFile(test.file)
			got := extractBlock(t, content, test.startStr, test.endStr)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("builder: does not generate auto_populate for server streaming", func(t *testing.T) {
		builderContent := readFile("src/builder.rs")
		if strings.Contains(builderContent, "auto_populate") {
			t.Errorf("expected builder.rs not to contain auto_populate helper for server streaming method")
		}
	})
}

func TestGenerateGrpcClientServerStreaming(t *testing.T) {
	outDir := t.TempDir()

	request := api.NewTestMessage("ExpandRequest").WithPackage("test.v1")
	request.Fields = []*api.Field{
		{
			Name:     "content",
			JSONName: "content",
			ID:       ".test.v1.ExpandRequest.content",
			Typez:    api.TypezString,
		},
	}
	response := api.NewTestMessage("EchoResponse").WithPackage("test.v1")

	serverMethod := api.NewTestMethod("Expand").WithInput(request).WithOutput(response).WithServerSideStreaming()
	serverMethod.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{{Verb: "POST", PathTemplate: &api.PathTemplate{}}},
	}
	service := api.NewTestService("Echo").WithPackage("test.v1").WithMethods(serverMethod)

	model := api.NewTestAPI([]*api.Message{request, response}, []*api.Enum{}, []*api.Service{service})
	model.PackageName = "test.v1"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec: map[string]string{
			"template-override":                "templates/grpc-client",
			"package:wkt":                      "source=google.protobuf,package=google-cloud-wkt",
			"include-server-streaming-methods": "true",
		},
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	files := make(map[string]string)
	readFile := func(relPath string) string {
		if content, ok := files[relPath]; ok {
			return content
		}
		b, err := os.ReadFile(filepath.Join(outDir, relPath))
		if err != nil {
			t.Fatal(err)
		}
		files[relPath] = string(b)
		return files[relPath]
	}

	for _, test := range []struct {
		name     string
		file     string
		startStr string
		endStr   string
		want     string
	}{
		{
			name:     "grpc-client transport: execute_server_streaming call",
			file:     "transport.rs",
			startStr: "        self.inner\n            .execute_server_streaming::<",
			endStr:   "&x_goog_request_params,\n            )\n            .await\n    }",
			want: `        self.inner
            .execute_server_streaming::<
                crate::model::ExpandRequest,
                crate::model::EchoResponse,
                crate::test::v1::ExpandRequest,
                crate::test::v1::EchoResponse,
            >(
                extensions,
                path,
                req,
                options,
                &info::X_GOOG_API_CLIENT_HEADER,
                &x_goog_request_params,
            )
            .await
    }`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := readFile(test.file)
			got := extractBlock(t, content, test.startStr, test.endStr)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
