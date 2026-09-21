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
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGeneratePagers(t *testing.T) {
	t.Run("emitted for service with paged methods", func(t *testing.T) {
		outDir := t.TempDir()

		instanceMsg := api.NewTestMessage("Instance").
			WithPackage("google.cloud.redis.v1").
			WithSourceLocation("google/cloud/redis/v1/cloud_redis.proto", 1)
		instancesField := api.NewTestField("instances").
			WithMessageType(instanceMsg).
			WithRepeated()
		nextPageTokenField := api.NewTestField("next_page_token").
			WithType(api.TypezString)
		respMsg := api.NewTestMessage("ListInstancesResponse").
			WithPackage("google.cloud.redis.v1").
			WithSourceLocation("google/cloud/redis/v1/cloud_redis.proto", 10).
			WithFields(instancesField, nextPageTokenField).
			WithPagination(nextPageTokenField, instancesField)

		pageTokenField := api.NewTestField("page_token").
			WithType(api.TypezString)
		reqMsg := api.NewTestMessage("ListInstancesRequest").
			WithPackage("google.cloud.redis.v1").
			WithSourceLocation("google/cloud/redis/v1/cloud_redis.proto", 20).
			WithFields(pageTokenField)

		method := api.NewTestMethod("ListInstances").
			WithInput(reqMsg).
			WithOutput(respMsg).
			WithPagination(pageTokenField)

		svc := api.NewTestService("CloudRedis").
			WithPackage("google.cloud.redis.v1").
			WithSourceLocation("google/cloud/redis/v1/cloud_redis.proto", 30).
			WithMethods(method)

		model := api.NewTestAPI([]*api.Message{instanceMsg, reqMsg, respMsg}, nil, []*api.Service{svc}).
			WithPackageName("google.cloud.redis.v1")
		model.Name = "google-cloud-redis"

		lib := &config.Library{
			Name:          "google-cloud-redis",
			CopyrightYear: "2026",
		}

		if err := Generate(t.Context(), model, outDir, lib); err != nil {
			t.Fatal(err)
		}

		pagersPath := filepath.Join(outDir, "google", "cloud", "redis_v1", "services", "cloud_redis", "pagers.py")
		contentBytes, err := os.ReadFile(pagersPath)
		if err != nil {
			t.Fatal(err)
		}
		content := string(contentBytes)

		t.Run("type imports header", func(t *testing.T) {
			got := extractBlock(t, content, "from google.cloud.redis_v1.types import cloud_redis", "from google.cloud.redis_v1.types import cloud_redis")
			want := "from google.cloud.redis_v1.types import cloud_redis"
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("OptionalRetry and OptionalAsyncRetry compatibility block", func(t *testing.T) {
			got := extractBlock(t, content, "try:\n    OptionalRetry = Union[retries.Retry, gapic_v1.method._MethodDefault, None]", "OptionalAsyncRetry = Union[retries_async.AsyncRetry, object, None]  # type: ignore")
			want := `try:
    OptionalRetry = Union[retries.Retry, gapic_v1.method._MethodDefault, None]
    OptionalAsyncRetry = Union[retries_async.AsyncRetry, gapic_v1.method._MethodDefault, None]
except AttributeError:  # pragma: NO COVER
    OptionalRetry = Union[retries.Retry, object, None]  # type: ignore
    OptionalAsyncRetry = Union[retries_async.AsyncRetry, object, None]  # type: ignore`
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("sync pager docstring and sphinx class references", func(t *testing.T) {
			gotDoc := extractBlock(t, content, "class ListInstancesPager:\n    \"\"\"", "attribute lookup.\n    \"\"\"")
			wantDoc := "class ListInstancesPager:\n" +
				"    \"\"\"A pager for iterating through ``list_instances`` requests.\n" +
				"\n" +
				"    This class thinly wraps an initial\n" +
				"    :class:`google.cloud.redis_v1.types.ListInstancesResponse` object, and\n" +
				"    provides an ``__iter__`` method to iterate through its\n" +
				"    ``instances`` field.\n" +
				"\n" +
				"    If there are more pages, the ``__iter__`` method will make additional\n" +
				"    ``ListInstances`` requests and continue to iterate\n" +
				"    through the ``instances`` field on the\n" +
				"    corresponding responses.\n" +
				"\n" +
				"    All the usual :class:`google.cloud.redis_v1.types.ListInstancesResponse`\n" +
				"    attributes are available on the pager. If multiple requests are made, only\n" +
				"    the most recent response is retained, and thus used for attribute lookup.\n" +
				"    \"\"\""
			if diff := cmp.Diff(wantDoc, gotDoc); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("sync pager __init__", func(t *testing.T) {
			gotInit := extractBlock(t, content, "    def __init__(self,\n            method: Callable[..., cloud_redis.ListInstancesResponse],", "self._metadata = metadata")
			wantInit := `    def __init__(self,
            method: Callable[..., cloud_redis.ListInstancesResponse],
            request: cloud_redis.ListInstancesRequest,
            response: cloud_redis.ListInstancesResponse,
            *,
            retry: OptionalRetry = gapic_v1.method.DEFAULT,
            timeout: Union[float, object] = gapic_v1.method.DEFAULT,
            metadata: Sequence[Tuple[str, Union[str, bytes]]] = ()):
        """Instantiate the pager.

        Args:
            method (Callable): The method that was originally called, and
                which instantiated this pager.
            request (google.cloud.redis_v1.types.ListInstancesRequest):
                The initial request object.
            response (google.cloud.redis_v1.types.ListInstancesResponse):
                The initial response object.
            retry (google.api_core.retry.Retry): Designation of what errors,
                if any, should be retried.
            timeout (float): The timeout for this request.
            metadata (Sequence[Tuple[str, Union[str, bytes]]]): Key/value pairs which should be
                sent along with the request as metadata. Normally, each value must be of type ` + "`str`" + `,
                but for metadata keys ending with the suffix ` + "`-bin`" + `, the corresponding values must
                be of type ` + "`bytes`" + `.
        """
        self._method = method
        self._request = cloud_redis.ListInstancesRequest(request)
        self._response = response
        self._retry = retry
        self._timeout = timeout
        self._metadata = metadata`
			if diff := cmp.Diff(wantInit, gotInit); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("sync pager pages", func(t *testing.T) {
			gotPages := extractBlock(t, content, "    @property\n    def pages(self) -> Iterator[cloud_redis.ListInstancesResponse]:", "metadata=self._metadata)\n            yield self._response")
			wantPages := `    @property
    def pages(self) -> Iterator[cloud_redis.ListInstancesResponse]:
        yield self._response
        while self._response.next_page_token:
            self._request.page_token = self._response.next_page_token
            self._response = self._method(self._request, retry=self._retry, timeout=self._timeout, metadata=self._metadata)
            yield self._response`
			if diff := cmp.Diff(wantPages, gotPages); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("sync pager __iter__", func(t *testing.T) {
			gotIter := extractBlock(t, content, "    def __iter__(self) -> Iterator[cloud_redis.Instance]:", "yield from page.instances")
			wantIter := `    def __iter__(self) -> Iterator[cloud_redis.Instance]:
        for page in self.pages:
            yield from page.instances`
			if diff := cmp.Diff(wantIter, gotIter); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("__getattr__ implementation", func(t *testing.T) {
			got := extractBlock(t, content, "    def __getattr__(self, name: str) -> Any:", "return getattr(self._response, name)")
			want := `    def __getattr__(self, name: str) -> Any:
        return getattr(self._response, name)`
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("__repr__ implementation", func(t *testing.T) {
			got := extractBlock(t, content, "    def __repr__(self) -> str:", "return '{0}<{1!r}>'.format(self.__class__.__name__, self._response)")
			want := `    def __repr__(self) -> str:
        return '{0}<{1!r}>'.format(self.__class__.__name__, self._response)`
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("async pager docstring and sphinx class references", func(t *testing.T) {
			gotDoc := extractBlock(t, content, "class ListInstancesAsyncPager:\n    \"\"\"", "attribute lookup.\n    \"\"\"")
			wantDoc := "class ListInstancesAsyncPager:\n" +
				"    \"\"\"A pager for iterating through ``list_instances`` requests.\n" +
				"\n" +
				"    This class thinly wraps an initial\n" +
				"    :class:`google.cloud.redis_v1.types.ListInstancesResponse` object, and\n" +
				"    provides an ``__aiter__`` method to iterate through its\n" +
				"    ``instances`` field.\n" +
				"\n" +
				"    If there are more pages, the ``__aiter__`` method will make additional\n" +
				"    ``ListInstances`` requests and continue to iterate\n" +
				"    through the ``instances`` field on the\n" +
				"    corresponding responses.\n" +
				"\n" +
				"    All the usual :class:`google.cloud.redis_v1.types.ListInstancesResponse`\n" +
				"    attributes are available on the pager. If multiple requests are made, only\n" +
				"    the most recent response is retained, and thus used for attribute lookup.\n" +
				"    \"\"\""
			if diff := cmp.Diff(wantDoc, gotDoc); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("async pager __init__", func(t *testing.T) {
			gotInit := extractBlock(t, content, "    def __init__(self,\n            method: Callable[..., Awaitable[cloud_redis.ListInstancesResponse]],", "self._metadata = metadata")
			wantInit := `    def __init__(self,
            method: Callable[..., Awaitable[cloud_redis.ListInstancesResponse]],
            request: cloud_redis.ListInstancesRequest,
            response: cloud_redis.ListInstancesResponse,
            *,
            retry: OptionalAsyncRetry = gapic_v1.method.DEFAULT,
            timeout: Union[float, object] = gapic_v1.method.DEFAULT,
            metadata: Sequence[Tuple[str, Union[str, bytes]]] = ()):
        """Instantiates the pager.

        Args:
            method (Callable): The method that was originally called, and
                which instantiated this pager.
            request (google.cloud.redis_v1.types.ListInstancesRequest):
                The initial request object.
            response (google.cloud.redis_v1.types.ListInstancesResponse):
                The initial response object.
            retry (google.api_core.retry.AsyncRetry): Designation of what errors,
                if any, should be retried.
            timeout (float): The timeout for this request.
            metadata (Sequence[Tuple[str, Union[str, bytes]]]): Key/value pairs which should be
                sent along with the request as metadata. Normally, each value must be of type ` + "`str`" + `,
                but for metadata keys ending with the suffix ` + "`-bin`" + `, the corresponding values must
                be of type ` + "`bytes`" + `.
        """
        self._method = method
        self._request = cloud_redis.ListInstancesRequest(request)
        self._response = response
        self._retry = retry
        self._timeout = timeout
        self._metadata = metadata`
			if diff := cmp.Diff(wantInit, gotInit); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("async pager pages and __aiter__", func(t *testing.T) {
			gotAiter := extractBlock(t, content, "    @property\n    async def pages(self) -> AsyncIterator[cloud_redis.ListInstancesResponse]:", "return async_generator()")
			wantAiter := `    @property
    async def pages(self) -> AsyncIterator[cloud_redis.ListInstancesResponse]:
        yield self._response
        while self._response.next_page_token:
            self._request.page_token = self._response.next_page_token
            self._response = await self._method(self._request, retry=self._retry, timeout=self._timeout, metadata=self._metadata)
            yield self._response
    def __aiter__(self) -> AsyncIterator[cloud_redis.Instance]:
        async def async_generator():
            async for page in self.pages:
                for response in page.instances:
                    yield response

        return async_generator()`
			if diff := cmp.Diff(wantAiter, gotAiter); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("not emitted for service without paged methods", func(t *testing.T) {
		outDir := t.TempDir()

		req := api.NewTestMessage("GetSecretRequest").
			WithPackage("google.iam.credentials.v1")
		resp := api.NewTestMessage("Secret").
			WithPackage("google.iam.credentials.v1")
		svc := api.NewTestService("IAMCredentials").
			WithPackage("google.iam.credentials.v1").
			WithMethods(api.NewTestMethod("GenerateAccessToken").WithInput(req).WithOutput(resp))

		model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
			WithPackageName("google.iam.credentials.v1")
		model.Name = "google-iam-credentials"

		lib := &config.Library{
			Name:          "google-iam-credentials",
			CopyrightYear: "2026",
		}

		if err := Generate(t.Context(), model, outDir, lib); err != nil {
			t.Fatal(err)
		}

		pagersPath := filepath.Join(outDir, "google", "iam", "credentials_v1", "services", "iam_credentials", "pagers.py")
		_, err := os.Stat(pagersPath)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected pagers.py to not exist, got err: %v", err)
		}
	})
}
