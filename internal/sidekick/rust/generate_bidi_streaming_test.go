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

	for _, tc := range []struct {
		name     string
		file     string
		startStr string
		endStr   string
		want     string
	}{
		{
			name:     "builder: struct definition",
			file:     "src/builder.rs",
			startStr: "    #[derive(Clone, Debug)]\n    pub struct Chat(",
			endStr:   ");",
			want:     "    #[derive(Clone, Debug)]\n    pub struct Chat(RequestBuilder<crate::model::Request>);",
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
			name:     "builder: with_request method",
			file:     "src/builder.rs",
			startStr: "        /// Sets the full request, replacing any prior values.\n        pub fn with_request<",
			endStr:   "        }",
			want: `        /// Sets the full request, replacing any prior values.
        pub fn with_request<V: Into<crate::model::Request>>(mut self, v: V) -> Self {
            self.0.request = v.into();
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
            self.0.request.query = v.into();
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
        _req: crate::model::Request,
        _options: crate::RequestOptions,
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
        req: crate::model::Request,
        options: crate::RequestOptions,
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
        req: crate::model::Request,
        options: crate::RequestOptions,
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
        req: crate::model::Request,
        options: crate::RequestOptions,
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
        req: crate::model::Request,
        options: crate::RequestOptions,
    ) -> Result<(
        google_cloud_gax::streaming::RequestSender<crate::model::Request>,
        google_cloud_gax::streaming::ResponseReceiver<crate::model::Response>,
    )> {`,
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
                options,
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
                            google_cloud_gax::error::Error::io("cannot send request: stream is closed")
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
		t.Run(tc.name, func(t *testing.T) {
			content := readFile(tc.file)
			got := extractBlock(t, content, tc.startStr, tc.endStr)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
