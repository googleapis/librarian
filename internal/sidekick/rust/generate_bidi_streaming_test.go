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
    ///         .build();
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
    /// let (sender, mut receiver) = builder.build();
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
			want:     "    #[derive(Clone, Debug)]\n    pub struct Chat(BidiStreamBuilder);",
		},
		{
			name:     "builder: build method",
			file:     "src/builder.rs",
			startStr: "        /// Initiates the bidirectional stream.\n        pub fn build(",
			endStr:   "            (*self.0.stub).chat(self.0.options)\n        }",
			want: `        /// Initiates the bidirectional stream.
        pub fn build(
            self,
        ) -> (
            google_cloud_gax::streaming::RequestSender<crate::model::Request>,
            google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
        ) {
            (*self.0.stub).chat(self.0.options)
        }`,
		},
		{
			name:     "builder: request builder impl",
			file:     "src/builder.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    #[doc(hidden)]\n    impl crate::RequestBuilder for Chat",
			endStr:   "\n    }",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    #[doc(hidden)]
    impl crate::RequestBuilder for Chat {
        fn request_options(&mut self) -> &mut crate::RequestOptions {
            &mut self.0.options
        }
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
        _options: crate::RequestOptions,
    ) -> (
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    ) {
        gaxi::unimplemented::unimplemented_bidi_stub()
    }`,
		},
		{
			name:     "stub dynamic: trait method",
			file:     "src/stub/dynamic.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    fn chat(",
			endStr:   ");",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    fn chat(
        &self,
        options: crate::RequestOptions,
    ) -> (
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    );`,
		},
		{
			name:     "stub dynamic: blanket forwarding impl",
			file:     "src/stub/dynamic.rs",
			startStr: "    /// Forwards the call to the implementation provided by `T`.\n    #[cfg(google_cloud_unstable_gapic_streaming)]\n    fn chat(",
			endStr:   "    }",
			want: `    /// Forwards the call to the implementation provided by ` + "`T`." + `
    #[cfg(google_cloud_unstable_gapic_streaming)]
    fn chat(
        &self,
        options: crate::RequestOptions,
    ) -> (
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    ) {
        T::chat(self, options)
    }`,
		},
		{
			name:     "tracing: method wrapper",
			file:     "src/tracing.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    fn chat(",
			endStr:   "    }",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    fn chat(
        &self,
        options: crate::RequestOptions,
    ) -> (
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    ) {
        self.inner.chat(options)
    }`,
		},
		{
			name:     "transport: method signature",
			file:     "src/transport.rs",
			startStr: "    #[cfg(google_cloud_unstable_gapic_streaming)]\n    fn chat(",
			endStr:   ") -> (",
			want: `    #[cfg(google_cloud_unstable_gapic_streaming)]
    fn chat(
        &self,
        options: crate::RequestOptions,
    ) -> (`,
		},
		{
			name:     "transport: request stream assembly and future call",
			file:     "src/transport.rs",
			startStr: "        let (req_tx, req_rx) = tokio::sync::mpsc::channel",
			endStr:   "        let request_sender = google_cloud_gax::streaming::RequestSender::from_fn(",
			want: `        let (req_tx, req_rx) = tokio::sync::mpsc::channel(
            options
                .request_stream_channel_capacity()
                .unwrap_or(google_cloud_gax::options::internal::DEFAULT_REQUEST_CHANNEL_CAPACITY),
        );
        let req_stream = tokio_stream::wrappers::ReceiverStream::new(req_rx);

        let extensions = {
            let mut e = gaxi::grpc::tonic::Extensions::new();
            e.insert(gaxi::grpc::tonic::GrpcMethod::new(
                "test.v1.Protocol",
                "Chat",
            ));
            e
        };
        let path = http::uri::PathAndQuery::from_static(
            "/test.v1.Protocol/Chat"
        );

        let (resp_tx, resp_rx) = tokio::sync::oneshot::channel();

        let grpc_inner = self.grpc_inner.clone();
        tokio::spawn(async move {
            let result = grpc_inner
                .bidi_stream::<
                    crate::prost::test::v1::Request,
                    crate::prost::test::v1::Response,
                >(
                    extensions,
                    path,
                    req_stream,
                    options,
                    &crate::info::X_GOOG_API_CLIENT_HEADER,
                    x_goog_request_params,
                )
                .await;
            let _ = resp_tx.send(result);
        });

        let request_sender = google_cloud_gax::streaming::RequestSender::from_fn(`,
		},
		{
			name:     "transport: request sender and response receiver",
			file:     "src/transport.rs",
			startStr: "        let request_sender = google_cloud_gax::streaming::RequestSender::from_fn(",
			endStr:   "        (request_sender, response_receiver)\n    }",
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
                        .map_err(|_| google_cloud_gax::streaming::SendError::StreamClosed)
                }
            },
        );
        let future = resp_rx.map(|res| match res {
            Ok(Ok(response)) => Ok(response.into_inner().map(|res| {
                res.map_err(gaxi::grpc::from_status::to_gax_error)
                    .and_then(|m| m.cnv().map_err(google_cloud_gax::error::Error::deser))
            })),
            Ok(Err(err)) => Err(err),
            Err(_) => Err(google_cloud_gax::error::Error::io(std::io::Error::new(
                std::io::ErrorKind::BrokenPipe,
                "stream initialization task cancelled",
            ))),
        });
        let response_receiver = google_cloud_gax::streaming::ResponseReceiver::from_future(future);

        (request_sender, response_receiver)
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
