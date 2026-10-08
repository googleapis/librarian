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

package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T, ask string) *server {
	t.Helper()
	src, root := writeFixture(t, samplePage)
	s, err := newServer(options{src: src, root: root, sha: "abc", repo: "example/librarian", ask: ask})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(s.out) })
	return s
}

func postAsk(t *testing.T, h http.Handler, token, prompt string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(askRequest{Token: token, Prompt: prompt})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(string(body))))
	return rec
}

func TestServerPagesRebuild(t *testing.T) {
	s := newTestServer(t, "cat")
	h := s.handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sample.html", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /sample.html: %d %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	html := rec.Body.String()
	for _, want := range []string{
		`<meta name="walkthrough-token" content="` + s.token + `">`,
		`<meta name="walkthrough-ask" content="1">`,
		`<details class="ask"`,
		`>Ask</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("sample.html does not contain %q", want)
		}
	}
	// Edits show up on the next page load.
	src := s.opts.src
	page := strings.Replace(samplePage, "title: Sample page", "title: Edited page", 1)
	if err := os.WriteFile(filepath.Join(src, "sample.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Edited page") {
		t.Errorf("GET / after edit: %d, body has edited title: %v", rec.Code, strings.Contains(rec.Body.String(), "Edited page"))
	}
	// Non-page assets are served without a rebuild.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET /style.css: %d", rec.Code)
	}
}

func TestServerRebuildError(t *testing.T) {
	s := newTestServer(t, "")
	if err := os.WriteFile(filepath.Join(s.opts.src, "broken.md"), []byte("---\ntitle: [\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sample.html", nil))
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "build failed") {
		t.Errorf("GET with broken page: %d %s", rec.Code, rec.Body)
	}
}

func TestServerAskDisabled(t *testing.T) {
	s := newTestServer(t, "")
	h := s.handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sample.html", nil))
	html := rec.Body.String()
	if strings.Contains(html, `walkthrough-ask`) || !strings.Contains(html, `>Copy prompt</button>`) {
		t.Errorf("page without -ask should offer to copy the prompt")
	}
	if rec := postAsk(t, h, s.token, "hi"); rec.Code != http.StatusNotImplemented {
		t.Errorf("POST /ask without -ask: %d, want 501", rec.Code)
	}
}

func TestServerAsk(t *testing.T) {
	s := newTestServer(t, "cat")
	h := s.handler()
	rec := postAsk(t, h, s.token, "What does Greet do?")
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /ask: %d %s", rec.Code, rec.Body)
	}
	root, _ := filepath.Abs(s.opts.root)
	got := rec.Body.String()
	if !strings.Contains(got, "repository at "+root) || !strings.HasSuffix(got, "What does Greet do?") {
		t.Errorf("answer = %q, want preamble with root and the question", got)
	}
	for _, tc := range []struct {
		name   string
		token  string
		prompt string
		want   int
	}{
		{"bad token", "nope", "hi", http.StatusForbidden},
		{"empty prompt", s.token, " \n", http.StatusBadRequest},
	} {
		if rec := postAsk(t, h, tc.token, tc.prompt); rec.Code != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, rec.Code, tc.want)
		}
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader("{")))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("malformed JSON: %d, want 400", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ask", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /ask: %d, want 404 (only POST is routed)", rec.Code)
	}
}

func TestServerAskCommandFails(t *testing.T) {
	s := newTestServer(t, "echo agent output; exit 3")
	rec := postAsk(t, s.handler(), s.token, "hi")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("POST /ask with failing command: %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "agent output") || !strings.Contains(string(body), "exit status 3") {
		t.Errorf("body = %q, want command output and exit status", body)
	}
}

func TestServe(t *testing.T) {
	src, root := writeFixture(t, samplePage)
	base := options{src: src, root: root, sha: "abc", repo: "example/librarian"}
	for _, addr := range []string{"0.0.0.0:0", "[::]:0"} {
		opts := base
		opts.serve = addr
		if err := serve(t.Context(), opts); err == nil || !strings.Contains(err.Error(), "not loopback") {
			t.Errorf("serve(%s) err = %v, want not loopback", addr, err)
		}
	}
	opts := base
	opts.serve = "127.0.0.1:0"
	opts.src = filepath.Join(src, "missing")
	if err := serve(t.Context(), opts); err == nil {
		t.Error("serve with missing src: want initial build error")
	}
	opts.src = src
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serve(ctx, opts) }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop after context cancellation")
	}
}
