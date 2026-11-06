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

package golang

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestRunProtoc(t *testing.T) {
	stubName := "protoc"
	if runtime.GOOS == "windows" {
		stubName += ".exe"
	}
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, recordFile string) (*config.Protoc, string)
	}{
		{
			name: "nil config uses system protoc in PATH",
			setup: func(t *testing.T, recordFile string) (*config.Protoc, string) {
				stubDir := t.TempDir()
				path := filepath.Join(stubDir, stubName)
				createStubExecutable(t, path, recordFile)
				t.Setenv(envPath, stubDir+string(filepath.ListSeparator)+os.Getenv(envPath))
				return nil, path
			},
		},
		{
			name: "configured protoc uses installed tool",
			setup: func(t *testing.T, recordFile string) (*config.Protoc, string) {
				binDir := t.TempDir()
				t.Setenv(cache.EnvLibrarianBin, binDir)
				version := "25.1"
				protocPath := filepath.Join(binDir, "protoc", "v"+version, "bin", stubName)
				createStubExecutable(t, protocPath, recordFile)
				return &config.Protoc{Version: version}, protocPath
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			recordFile := filepath.Join(t.TempDir(), "calls.txt")
			pc, want := test.setup(t, recordFile)
			if err := runProtoc(t.Context(), pc, "--version"); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(recordFile)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.TrimSpace(string(data))
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToolsEnv(t *testing.T) {
	for _, test := range []struct {
		name   string
		binDir func(t *testing.T) string
		want   func(absBin string) map[string]string
	}{
		{
			name: "returns PATH pointing to go_tools directory",
			binDir: func(t *testing.T) string {
				return t.TempDir()
			},
			want: func(absBin string) map[string]string {
				return map[string]string{
					envPath: filepath.Join(absBin, toolsDir),
				}
			},
		},
		{
			name: "handles nested bin directory path",
			binDir: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nested", "bin")
			},
			want: func(absBin string) map[string]string {
				return map[string]string{
					envPath: filepath.Join(absBin, toolsDir),
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			binDir := test.binDir(t)
			t.Setenv(cache.EnvLibrarianBin, binDir)
			got, err := toolsEnv()
			if err != nil {
				t.Fatal(err)
			}
			wantAbsBin, err := filepath.Abs(binDir)
			if err != nil {
				t.Fatal(err)
			}
			wantMap := test.want(wantAbsBin)
			if diff := cmp.Diff(wantMap, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToolsEnv_Error(t *testing.T) {
	t.Setenv(cache.EnvLibrarianBin, "")
	t.Setenv(cache.EnvLibrarianCache, "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	// This error is not easy to check, we only make sure
	// it doesn't return nil.
	if _, err := toolsEnv(); err == nil {
		t.Error("toolsEnv() error = nil, want error")
	}
}

func TestRunInDirWithEnvRetry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script stub test on Windows")
	}
	t.Setenv(cache.EnvLibrarianBin, t.TempDir())
	stubDir := t.TempDir()
	recordFile := filepath.Join(t.TempDir(), "calls.txt")
	stubPath := filepath.Join(stubDir, "stub-cmd")
	createStubExecutable(t, stubPath, recordFile)
	t.Setenv(envPath, stubDir+string(filepath.ListSeparator)+os.Getenv(envPath))

	if err := runInDirWithEnvRetry(t.Context(), t.TempDir(), nil, "stub-cmd", "arg1"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	if diff := cmp.Diff(stubPath, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunInDirWithEnvAndBackoff_RetrySucceeds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script stub test on Windows")
	}
	t.Setenv(cache.EnvLibrarianBin, t.TempDir())
	stubDir := t.TempDir()
	attemptsFile := filepath.Join(t.TempDir(), "attempts.txt")
	stubPath := filepath.Join(stubDir, "flaky-cmd")
	createFlakyStubExecutable(t, stubPath, attemptsFile, 3)
	t.Setenv(envPath, stubDir+string(filepath.ListSeparator)+os.Getenv(envPath))

	if err := runInDirWithEnvAndBackoff(t.Context(), t.TempDir(), map[string]string{"TEST_VAR": "1"}, time.Millisecond, "flaky-cmd", "arg1"); err != nil {
		t.Fatal(err)
	}
	got := countLines(t, attemptsFile)
	want := 3
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunInDirWithEnvAndBackoff_RetryExhausted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script stub test on Windows")
	}
	t.Setenv(cache.EnvLibrarianBin, t.TempDir())
	stubDir := t.TempDir()
	attemptsFile := filepath.Join(t.TempDir(), "attempts.txt")
	stubPath := filepath.Join(stubDir, "failing-cmd")
	createFlakyStubExecutable(t, stubPath, attemptsFile, 10)
	t.Setenv(envPath, stubDir+string(filepath.ListSeparator)+os.Getenv(envPath))

	err := runInDirWithEnvAndBackoff(t.Context(), t.TempDir(), nil, time.Millisecond, "failing-cmd", "arg1")
	if err == nil {
		t.Fatal("runInDirWithEnvAndBackoff() error = nil, want error")
	}
	got := countLines(t, attemptsFile)
	want := maxRetryAttempts
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunInDirWithEnvAndBackoff_ContextCanceled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script stub test on Windows")
	}
	for _, test := range []struct {
		name    string
		ctx     func(t *testing.T) context.Context
		wantErr error
	}{
		{
			name: "canceled before first attempt",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline exceeded during backoff",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			wantErr: context.DeadlineExceeded,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(cache.EnvLibrarianBin, t.TempDir())
			stubDir := t.TempDir()
			testhelper.WriteExecutable(t, filepath.Join(stubDir, "failing-cmd"), "#!/bin/sh\nexit 1\n")
			t.Setenv(envPath, stubDir+string(filepath.ListSeparator)+os.Getenv(envPath))

			gotErr := runInDirWithEnvAndBackoff(test.ctx(t), t.TempDir(), nil, time.Hour, "failing-cmd", "arg1")
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("runInDirWithEnvAndBackoff() error = %v, want %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestBackoffWait_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	gotErr := backoffWait(ctx, time.Hour)
	if !errors.Is(gotErr, context.Canceled) {
		t.Errorf("backoffWait() error = %v, want %v", gotErr, context.Canceled)
	}
}

func createStubExecutable(t *testing.T, path, recordFile string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("#!/bin/sh\necho \"$0\" >> %q\nexit 0\n", recordFile)
	if runtime.GOOS == "windows" {
		content = fmt.Sprintf("@echo off\r\necho %%0 >> %q\r\nexit /b 0\r\n", recordFile)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func createFlakyStubExecutable(t *testing.T, path, attemptsFile string, succeedOnAttempt int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(`#!/bin/sh
echo "attempt" >> %q
count=$(wc -l < %q)
if [ "$count" -lt %d ]; then
  echo "transient failure" >&2
  exit 1
fi
exit 0
`, attemptsFile, attemptsFile, succeedOnAttempt)
	testhelper.WriteExecutable(t, path, content)
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "\n"))
}
