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
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// askTimeout bounds one agent invocation from the ask panel.
const askTimeout = 10 * time.Minute

// server rebuilds the site on every page request, so edits to the markdown
// or to the code show up on reload, and optionally forwards questions from
// the page to a local agent command.
type server struct {
	opts  options
	out   string
	token string
	mu    sync.Mutex
}

func newServer(opts options) (*server, error) {
	out, err := os.MkdirTemp("", "walkthrough-")
	if err != nil {
		return nil, err
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	return &server{opts: opts, out: out, token: hex.EncodeToString(b[:])}, nil
}

// serve runs the server on opts.serve until ctx is done. The address must
// be loopback: the ask endpoint runs a command on this machine.
func serve(ctx context.Context, opts options) error {
	s, err := newServer(opts)
	if err != nil {
		return err
	}
	defer os.RemoveAll(s.out)
	ln, err := net.Listen("tcp", opts.serve)
	if err != nil {
		return err
	}
	if !ln.Addr().(*net.TCPAddr).IP.IsLoopback() {
		ln.Close()
		return fmt.Errorf("serve address %s is not loopback; the ask endpoint runs commands locally", opts.serve)
	}
	if err := s.rebuild(ctx); err != nil {
		ln.Close()
		return err
	}
	srv := &http.Server{Handler: s.handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	fmt.Printf("serving walkthroughs at http://%s/ (rebuilt on every page load", ln.Addr())
	if opts.ask != "" {
		fmt.Printf("; questions go to %q", opts.ask)
	}
	fmt.Println(")")
	if err := srv.Serve(ln); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir(s.out))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") || strings.HasSuffix(r.URL.Path, ".html") {
			if err := s.rebuild(r.Context()); err != nil {
				http.Error(w, "build failed:\n\n"+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
	mux.HandleFunc("POST /ask", s.ask)
	return mux
}

// rebuild regenerates the site into the temp directory. Builds are quick
// (well under a second), so no change detection is needed.
func (s *server) rebuild(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	opts := s.opts
	opts.out = s.out
	opts.serveToken = s.token
	opts.askEnabled = s.opts.ask != ""
	return build(ctx, opts)
}

// askRequest is the JSON body of POST /ask. The prompt is composed by the
// page from the step the reader is on.
type askRequest struct {
	Token  string `json:"token"`
	Prompt string `json:"prompt"`
}

// ask pipes the prompt to the configured command and returns its stdout.
// The per-run token keeps other pages open in the browser from reaching
// the endpoint.
func (s *server) ask(w http.ResponseWriter, r *http.Request) {
	if s.opts.ask == "" {
		http.Error(w, "no agent command configured; start with -ask", http.StatusNotImplemented)
		return
	}
	var req askRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Token != s.token {
		http.Error(w, "bad token", http.StatusForbidden)
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		http.Error(w, "empty prompt", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), askTimeout)
	defer cancel()
	log.Printf("ask: %d bytes to %q", len(req.Prompt), s.opts.ask)
	out, err := runAsk(ctx, s.opts.root, s.opts.ask, promptPreamble(s.opts.root)+req.Prompt)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "%s\n\n%v", out, err)
		return
	}
	io.WriteString(w, out)
}

// runAsk runs the ask command through the shell with the prompt on stdin,
// in the repository root so the agent sees the same tree as the site.
func runAsk(ctx context.Context, dir, cmdline, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdline)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// promptPreamble is prepended to every question so the agent knows where
// the tree is and how to answer.
func promptPreamble(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return "You are helping a developer read the Librarian walkthroughs in the repository at " + abs + ". Answer from the source tree and doc/walkthroughs; cite files.\n\n"
}
