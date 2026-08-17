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
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func extractBlock(t *testing.T, content, startStr, endStr string) string {
	t.Helper()
	startIdx := strings.Index(content, startStr)
	if startIdx == -1 {
		t.Fatalf("missing expected block start %q\n\n%s", startStr, content)
	}
	endIdx := strings.Index(content[startIdx:], endStr)
	if endIdx == -1 {
		t.Fatalf("missing expected block end %q\n\n%s", endStr, content)
	}
	return content[startIdx : startIdx+endIdx+len(endStr)]
}

func TestGenerateBidiStreaming(t *testing.T) {
	outDir := t.TempDir()

	request := api.NewTestMessage("Request").WithPackage("test.v1")
	request.Fields = []*api.Field{
		{
			Name:     "query",
			JSONName: "query",
			ID:       ".test.v1.Request.query",
			Typez:    api.TypezString,
		},
	}
	response := api.NewTestMessage("Response").WithPackage("test.v1")

	bidiMethod := api.NewTestMethod("Chat").WithInput(request).WithOutput(response).WithBidiStreaming()
	bidiMethod.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
	}
	service := api.NewTestService("Protocol").WithPackage("test.v1").WithMethods(bidiMethod)

	model := api.NewTestAPI([]*api.Message{request, response}, []*api.Enum{}, []*api.Service{service})
	model.PackageName = "test.v1"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec: map[string]string{
			"package:wkt":                    "source=google.protobuf,package=google-cloud-wkt",
			"include-bidi-streaming-methods": "true",
			"generate-rpc-samples":           "true",
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
			name:     "client: method docstring and example",
			file:     "src/client.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    ///\n    /// # Example",
			endStr:   "        super::builder::protocol::Chat::new(self.inner.clone())\n    }",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    ///
    /// # Example
    /// ` + "```" + `
    /// # use google_cloud_test_v1::client::Protocol;
    /// # use google_cloud_test_v1::model::Request;
    /// use google_cloud_test_v1::Result;
    /// async fn sample(
    ///    client: &Protocol
    /// ) -> Result<()> {
    ///     let (sender, mut receiver) = client.chat()
    ///         .with_request(Request::default())
    ///         .send()
    ///         .await?;
    ///
    ///     sender.send(Request::default()).await?;
    ///     drop(sender); // Half-close the stream
    ///
    ///     while let Some(response) = receiver.recv().await {
    ///         let response = response?;
    ///         println!("response {:?}", response);
    ///     }
    ///     Ok(())
    /// }
    /// ` + "```" + `
    pub fn chat(&self) -> super::builder::protocol::Chat
    {
        super::builder::protocol::Chat::new(self.inner.clone())
    }`,
		},
		{
			name:     "builder: struct docstring and example",
			file:     "src/builder.rs",
			startStr: "    /// The request builder for [Protocol::chat][crate::client::Protocol::chat] calls.\n    ///\n    /// # Example",
			endStr:   "    #[cfg(google_cloud_unstable_gapic_streaming)]",
			want: `    /// The request builder for [Protocol::chat][crate::client::Protocol::chat] calls.
    ///
    /// # Example
    /// ` + "```" + `
    /// # use google_cloud_test_v1::builder::protocol::Chat;
    /// # use google_cloud_test_v1::model::Request;
    /// # async fn sample() -> google_cloud_test_v1::Result<()> {
    /// let builder = prepare_request_builder();
    /// let (sender, mut receiver) = builder
    ///     .with_request(Request::default())
    ///     .send()
    ///     .await?;
    ///
    /// sender.send(Request::default()).await?;
    /// drop(sender); // Half-close the stream
    ///
    /// while let Some(response) = receiver.recv().await {
    ///     let response = response?;
    ///     println!("response {:?}", response);
    /// }
    /// # Ok(()) }
    ///
    /// fn prepare_request_builder() -> Chat {
    ///   # panic!();
    ///   // ... details omitted ...
    /// }
    /// ` + "```" + `
    #[cfg(google_cloud_unstable_gapic_streaming)]`,
		},
		{
			name:     "builder: struct definition",
			file:     "src/builder.rs",
			startStr: "    #[derive(Clone, Debug)]\n    pub struct Chat(",
			endStr:   ");",
			want:     "    #[derive(Clone, Debug)]\n    pub struct Chat(BidiStreamBuilder<crate::model::Request>);",
		},
		{
			name:     "builder: send method",
			file:     "src/builder.rs",
			startStr: "        /// Initiates the bidirectional stream.\n        pub async fn send(",
			endStr:   "        }",
			want: `        /// Initiates the bidirectional stream.
        pub async fn send(
            self,
        ) -> Result<(
            google_cloud_gax::streaming::RequestSender<crate::model::Request>,
            google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
        )> {
            (*self.0.stub).chat(self.0.request, self.0.options).await
        }`,
		},
		{
			name:     "builder: with_request_channel_capacity method",
			file:     "src/builder.rs",
			startStr: "        /// Sets the buffer capacity of internal request channel.\n",
			endStr:   "        }",
			want: `        /// Sets the buffer capacity of internal request channel.
        ///
        /// Valid values must be between ` + "`1`" + ` and ` + "`google_cloud_gax::options::MAX_REQUEST_CHANNEL_CAPACITY`." + `
        /// The default capacity is ` + "`16`." + `
        pub fn with_request_channel_capacity(mut self, capacity: usize) -> Self {
            self.0.options.set_request_channel_capacity(capacity);
            self
        }`,
		},
		{
			name:     "builder: with_request method",
			file:     "src/builder.rs",
			startStr: "        /// Sets the full request, replacing any prior values.\n        pub fn with_request<",
			endStr:   "        }",
			want: `        /// Sets the full request, replacing any prior values.
        pub fn with_request<V: Into<crate::model::Request>>(mut self, v: V) -> Self {
            self.0.request = std::option::Option::Some(v.into());
            self
        }`,
		},
		{
			name:     "builder: field setter",
			file:     "src/builder.rs",
			startStr: "        /// Sets the value of [query]",
			endStr:   "        }",
			want: `        /// Sets the value of [query][crate::model::Request::query].
        pub fn set_query<T: Into<std::string::String>>(mut self, v: T) -> Self {
            self.0.request.get_or_insert_with(std::default::Default::default).query = v.into();
            self
        }`,
		},
		{
			name:     "stub: method signature and default body",
			file:     "src/stub.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    fn chat(",
			endStr:   "    }",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    fn chat(
        &self,
        _req: std::option::Option<crate::model::Request>,
        _options: crate::BidiStreamOptions,
    ) -> impl std::future::Future<Output = crate::Result<(
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    )>> + Send {
        gaxi::unimplemented::unimplemented_bidi_stub()
    }`,
		},
		{
			name:     "stub dynamic: trait method",
			file:     "src/stub/dynamic.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    async fn chat(",
			endStr:   ")>;",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    async fn chat(
        &self,
        req: std::option::Option<crate::model::Request>,
        options: crate::BidiStreamOptions,
    ) -> crate::Result<(
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    )>;`,
		},
		{
			name:     "stub dynamic: blanket forwarding impl",
			file:     "src/stub/dynamic.rs",
			startStr: "    /// Forwards the call to the implementation provided by `T`.\n    #[cfg(google_cloud_unstable_gapic_streaming)]\n    async fn chat(",
			endStr:   "    }",
			want: `    /// Forwards the call to the implementation provided by ` + "`T`." + `
    #[cfg(google_cloud_unstable_gapic_streaming)]
    async fn chat(
        &self,
        req: std::option::Option<crate::model::Request>,
        options: crate::BidiStreamOptions,
    ) -> crate::Result<(
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    )> {
        T::chat(self, req, options).await
    }`,
		},
		{
			name:     "tracing: method wrapper",
			file:     "src/tracing.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    async fn chat(",
			endStr:   "    }",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    async fn chat(
        &self,
        req: std::option::Option<crate::model::Request>,
        options: crate::BidiStreamOptions,
    ) -> Result<(
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    )> {
        self.inner.chat(req, options).await
    }`,
		},
		{
			name:     "transport: method signature",
			file:     "src/transport.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    async fn chat(",
			endStr:   ")> {",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    async fn chat(
        &self,
        req: std::option::Option<crate::model::Request>,
        options: crate::BidiStreamOptions,
    ) -> Result<(
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    )> {`,
		},
		{
			name:     "transport: request params without routing",
			file:     "src/transport.rs",
			startStr: "        let req = req.ok_or_else(|| {\n",
			endStr:   "        let x_goog_request_params = \"\";",
			want: `        let req = req.ok_or_else(|| {
            google_cloud_gax::error::Error::binding(
                "a request is required"
            )
        })?;

        let x_goog_request_params = "";`,
		},
		{
			name:     "transport: eager bidi_stream call",
			file:     "src/transport.rs",
			startStr: "        let result = self.grpc_inner\n            .bidi_stream::<",
			endStr:   ".await?;",
			want: `        let result = self.grpc_inner
            .bidi_stream::<
                crate::prost::test::v1::Request,
                crate::prost::test::v1::Response,
            >(
                extensions,
                path,
                req_stream,
                options.into(),
                &crate::info::X_GOOG_API_CLIENT_HEADER,
                x_goog_request_params,
            )
            .await?;`,
		},
		{
			name:     "transport: request sender and response receiver",
			file:     "src/transport.rs",
			startStr: "        let request_sender = google_cloud_gax::streaming::RequestSender::from_fn(",
			endStr:   "        Ok((request_sender, response_receiver))\n    }",
			want: `        let request_sender = google_cloud_gax::streaming::RequestSender::from_fn(
            move |item: crate::model::Request| {
                let req_tx = req_tx.clone();
                async move {
                    let prost_item = item
                        .to_proto()
                        .map_err(google_cloud_gax::error::Error::ser)?;
                    req_tx
                        .send(prost_item)
                        .await
                        .map_err(|_| {
                            google_cloud_gax::error::Error::io(std::io::Error::new(
                                std::io::ErrorKind::BrokenPipe,
                                "cannot send request: stream is closed",
                            ))
                        })
                }
            },
        );
        let response_receiver = google_cloud_gax::streaming::ResponseReceiver::from_stream(
            result.into_inner().map(|res| {
                res.map_err(gaxi::grpc::from_status::to_gax_error)
                    .and_then(|m| m.cnv().map_err(google_cloud_gax::error::Error::deser))
            }),
        );

        Ok((request_sender, response_receiver))
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

func TestGenerateBidiStreamingWithRouting(t *testing.T) {
	for _, test := range []struct {
		name            string
		routingRequired bool
		wantSubstrings  []string
		wantAbsent      []string
	}{
		{
			name:            "without routing required",
			routingRequired: false,
			wantSubstrings: []string{
				"let x_goog_request_params = {",
				"gaxi::routing_parameter::format(&[",
				`.map(|v| ("table_name", v))`,
			},
			wantAbsent: []string{
				"BindingError",
				"PathMismatchBuilder",
			},
		},
		{
			name:            "with routing required",
			routingRequired: true,
			wantSubstrings: []string{
				"let x_goog_request_params = {",
				"if x_goog_request_params.is_empty() {",
				"use google_cloud_gax::error::binding::BindingError;",
				"use gaxi::path_parameter::PathMismatchBuilder;",
				"let builder = PathMismatchBuilder::default();",
				`"projects/*/datasets/*/tables/*"`,
				"return Err(google_cloud_gax::error::Error::binding(BindingError { paths }))",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			request := api.NewTestMessage("Request").WithPackage("test.v1")
			request.Fields = []*api.Field{
				{
					Name:     "table_name",
					JSONName: "tableName",
					ID:       ".test.v1.Request.table_name",
					Typez:    api.TypezString,
				},
			}
			response := api.NewTestMessage("Response").WithPackage("test.v1")

			bidiMethod := api.NewTestMethod("AppendRows").WithInput(request).WithOutput(response).WithBidiStreaming()
			bidiMethod.PathInfo = &api.PathInfo{
				Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
			}
			bidiMethod.Routing = []*api.RoutingInfo{
				{
					Name: "table_name",
					Variants: []*api.RoutingInfoVariant{
						{
							FieldPath: []string{"table_name"},
							Matching:  api.RoutingPathSpec{Segments: []string{"projects", "*", "datasets", "*", "tables", "*"}},
						},
					},
				},
			}
			service := api.NewTestService("WriteStream").WithPackage("test.v1").WithMethods(bidiMethod)

			model := api.NewTestAPI([]*api.Message{request, response}, []*api.Enum{}, []*api.Service{service})
			model.PackageName = "test.v1"
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}

			cfg := &parser.ModelConfig{
				SpecificationFormat: libconfig.SpecProtobuf,
				Codec: map[string]string{
					"package:wkt":                    "source=google.protobuf,package=google-cloud-wkt",
					"include-bidi-streaming-methods": "true",
					"routing-required":               strconv.FormatBool(test.routingRequired),
				},
			}
			if err := Generate(t.Context(), model, outDir, cfg); err != nil {
				t.Fatal(err)
			}

			transportContent, err := os.ReadFile(filepath.Join(outDir, "src/transport.rs"))
			if err != nil {
				t.Fatal(err)
			}
			content := string(transportContent)

			for _, sub := range test.wantSubstrings {
				if !strings.Contains(content, sub) {
					t.Errorf("missing expected substring %q in generated transport.rs", sub)
				}
			}
			for _, absent := range test.wantAbsent {
				if strings.Contains(content, absent) {
					t.Errorf("unexpected substring %q found in generated transport.rs", absent)
				}
			}
		})
	}
}
