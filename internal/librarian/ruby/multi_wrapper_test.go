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

package ruby

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPascalize(t *testing.T) {
	for _, test := range []struct {
		in   string
		want string
	}{
		{"workflows", "Workflows"},
		{"beyond_corp", "BeyondCorp"},
		{"app_connections", "AppConnections"},
		{"iam", "Iam"},
		{"v1", "V1"},
		{"", ""},
	} {
		got := pascalize(test.in)
		if got != test.want {
			t.Errorf("pascalize(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

func TestMakePath(t *testing.T) {
	for _, test := range []struct {
		in   string
		want string
	}{
		{"google-cloud-workflows", "google/cloud/workflows"},
		{"google-cloud-workflows-executions", "google/cloud/workflows/executions"},
		{"google-cloud-beyond_corp", "google/cloud/beyond_corp"},
	} {
		got := makePath(test.in)
		if got != test.want {
			t.Errorf("makePath(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

func TestMakeConstant(t *testing.T) {
	for _, test := range []struct {
		in   string
		want string
	}{
		{"google-cloud-workflows", "Google::Cloud::Workflows"},
		{"google-cloud-workflows-executions", "Google::Cloud::Workflows::Executions"},
		{"google-cloud-beyond_corp-app_connections", "Google::Cloud::BeyondCorp::AppConnections"},
		{"google-cloud-policy_troubleshooter-iam", "Google::Cloud::PolicyTroubleshooter::Iam"},
	} {
		got := makeConstant(test.in)
		if got != test.want {
			t.Errorf("makeConstant(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

func TestPrepareMultiWrapper_SameMainGem(t *testing.T) {
	stagingDir := t.TempDir()
	mainGem := "google-cloud-workflows"
	otherGem := "google-cloud-workflows-executions"
	mainDir := filepath.Join(stagingDir, mainGem)
	otherDir := filepath.Join(stagingDir, otherGem)

	// Create mainGem staged files
	mustWrite(t, filepath.Join(mainDir, ".repo-metadata.json"), `{"name_pretty":"Workflows API","distribution_name":"google-cloud-workflows","client_documentation":"https://cloud.google.com/ruby/docs/reference/google-cloud-workflows/latest"}`)
	mustWrite(t, filepath.Join(mainDir, "Gemfile"), "local_dependencies = [\"google-cloud-workflows-v1\"]\n")
	mustWrite(t, filepath.Join(mainDir, "google-cloud-workflows.gemspec"), "Gem::Specification.new do |gem|\n  gem.name = \"google-cloud-workflows\"\n  gem.summary = \"API client library for Workflows\"\n  gem.add_dependency \"google-cloud-workflows-v1\", \"~> 2.0\"\nend\n")
	mustWrite(t, filepath.Join(mainDir, "README.md"), "# Workflows\n\nReference documentation:\n[google-cloud-workflows-v1](https://cloud.google.com/ruby/docs/reference/google-cloud-workflows-v1/latest).\n\nDebug logging:\n[google-cloud-workflows-v1](https://cloud.google.com/ruby/docs/reference/google-cloud-workflows-v1/latest).\n")
	mustWrite(t, filepath.Join(mainDir, "lib", "google-cloud-workflows.rb"), "require \"google/cloud/workflows\" unless defined? Google::Cloud::Workflows::VERSION\n")
	mustWrite(t, filepath.Join(mainDir, "lib", "google", "cloud", "workflows.rb"), "module Google; module Cloud; module Workflows; end; end; end\n")
	mustWrite(t, filepath.Join(mainDir, "lib", "google", "cloud", "workflows", "version.rb"), "\n  VERSION = \"3.2.1\"\n")
	mustWrite(t, filepath.Join(mainDir, "test", "google", "cloud", "workflows", "client_test.rb"), "# client test\n")

	// Create otherGem staged files
	mustWrite(t, filepath.Join(otherDir, ".repo-metadata.json"), `{"name_pretty":"Workflows Executions API"}`)
	mustWrite(t, filepath.Join(otherDir, "Gemfile"), "local_dependencies = [\"google-cloud-workflows-executions-v1\"]\n")
	mustWrite(t, filepath.Join(otherDir, "google-cloud-workflows-executions.gemspec"), "Gem::Specification.new do |gem|\n  gem.name = \"google-cloud-workflows-executions\"\n  gem.add_dependency \"google-cloud-workflows-executions-v1\", \"~> 1.2\"\nend\n")
	mustWrite(t, filepath.Join(otherDir, "README.md"), "# Executions\n\n[google-cloud-workflows-executions-v1](https://cloud.google.com/ruby/docs/reference/google-cloud-workflows-executions-v1/latest).\n")
	mustWrite(t, filepath.Join(otherDir, "lib", "google-cloud-workflows-executions.rb"), "require \"google/cloud/workflows/executions\"\n")
	mustWrite(t, filepath.Join(otherDir, "lib", "google", "cloud", "workflows", "executions.rb"), "module Google; module Cloud; module Workflows; module Executions; end; end; end; end\n")
	mustWrite(t, filepath.Join(otherDir, "lib", "google", "cloud", "workflows", "executions", "version.rb"), "\n  VERSION = \"1.6.1\"\n")
	mustWrite(t, filepath.Join(otherDir, "test", "google", "cloud", "workflows", "executions", "client_test.rb"), "# executions client test\n")

	if err := prepareMultiWrapper(stagingDir, []string{mainGem, otherGem}, "", mainGem); err != nil {
		t.Fatal(err)
	}

	// Verify entrypoint expanded
	entrypoint := mustRead(t, filepath.Join(stagingDir, "lib", "google-cloud-workflows.rb"))
	if !strings.Contains(entrypoint, `require "google/cloud/workflows/executions" unless defined? Google::Cloud::Workflows::Executions::VERSION`) {
		t.Errorf("entrypoint missing executions require: %s", entrypoint)
	}

	// Verify gemspec expanded
	gemspec := mustRead(t, filepath.Join(stagingDir, "google-cloud-workflows.gemspec"))
	if !strings.Contains(gemspec, `gem.add_dependency "google-cloud-workflows-v1", "~> 2.0"`) || !strings.Contains(gemspec, `gem.add_dependency "google-cloud-workflows-executions-v1", "~> 1.2"`) {
		t.Errorf("gemspec missing dependency: %s", gemspec)
	}

	// Verify README expanded in both places
	readme := mustRead(t, filepath.Join(stagingDir, "README.md"))
	expectedSnippet := "[google-cloud-workflows-v1](https://cloud.google.com/ruby/docs/reference/google-cloud-workflows-v1/latest),\n[google-cloud-workflows-executions-v1](https://cloud.google.com/ruby/docs/reference/google-cloud-workflows-executions-v1/latest)."
	if strings.Count(readme, expectedSnippet) != 2 {
		t.Errorf("readme did not expand all links correctly: %s", readme)
	}

	// Verify Gemfile expanded
	gemfile := mustRead(t, filepath.Join(stagingDir, "Gemfile"))
	if !strings.Contains(gemfile, `local_dependencies = ["google-cloud-workflows-v1", "google-cloud-workflows-executions-v1"]`) {
		t.Errorf("gemfile missing local deps: %s", gemfile)
	}

	// Verify secondary version disabled
	executionsVersion := mustRead(t, filepath.Join(stagingDir, "lib", "google", "cloud", "workflows", "executions", "version.rb"))
	if !strings.Contains(executionsForTesting(executionsVersion), "# @private Unused\n  VERSION = \"\"") {
		t.Errorf("executions version not disabled: %s", executionsVersion)
	}

	// Verify files moved
	if _, err := os.Stat(filepath.Join(stagingDir, "lib", "google", "cloud", "workflows", "executions.rb")); err != nil {
		t.Errorf("executions.rb not moved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stagingDir, "test", "google", "cloud", "workflows", "executions", "client_test.rb")); err != nil {
		t.Errorf("executions client_test.rb not moved: %v", err)
	}

	// Verify subdirs removed
	if _, err := os.Stat(mainDir); !os.IsNotExist(err) {
		t.Errorf("mainDir was not removed")
	}
	if _, err := os.Stat(otherDir); !os.IsNotExist(err) {
		t.Errorf("otherDir was not removed")
	}
}

func TestPrepareMultiWrapper_DifferentMainGem(t *testing.T) {
	stagingDir := t.TempDir()
	mainGem := "google-cloud-beyond_corp-app_connections"
	otherGem := "google-cloud-beyond_corp-app_connectors"
	finalGem := "google-cloud-beyond_corp"
	mainDir := filepath.Join(stagingDir, mainGem)
	otherDir := filepath.Join(stagingDir, otherGem)

	// Create mainGem staged files
	mustWrite(t, filepath.Join(mainDir, ".repo-metadata.json"), `{"name_pretty":"BeyondCorp AppConnections","distribution_name":"google-cloud-beyond_corp-app_connections","client_documentation":"https://cloud.google.com/ruby/docs/reference/google-cloud-beyond_corp-app_connections/latest"}`)
	mustWrite(t, filepath.Join(mainDir, ".rubocop.yml"), "gemspec: \"google-cloud-beyond_corp-app_connections.gemspec\"\nlib: \"lib/google-cloud-beyond_corp-app_connections.rb\"\n")
	mustWrite(t, filepath.Join(mainDir, ".yardopts"), "\"BeyondCorp AppConnections\"\n")
	mustWrite(t, filepath.Join(mainDir, "AUTHENTICATION.md"), "Gem `google-cloud-beyond_corp-app_connections` authentication. Require \"google/cloud/beyond_corp/app_connections\".\n")
	mustWrite(t, filepath.Join(mainDir, "Gemfile"), "gemspec name: \"google-cloud-beyond_corp-app_connections.gemspec\"\nlocal_dependencies = [\"google-cloud-beyond_corp-app_connections-v1\"]\n")
	mustWrite(t, filepath.Join(mainDir, "google-cloud-beyond_corp-app_connections.gemspec"), "require File.expand_path(\"lib/google/cloud/beyond_corp/app_connections/version\", __dir__)\nGem::Specification.new do |gem|\n  gem.name = \"google-cloud-beyond_corp-app_connections\"\n  gem.version = Google::Cloud::BeyondCorp::AppConnections::VERSION\n  gem.summary = \"Old summary\"\n  gem.add_dependency \"google-cloud-beyond_corp-app_connections-v1\", \"~> 0.4\"\nend\n")
	mustWrite(t, filepath.Join(mainDir, "README.md"), "# Ruby Client for BeyondCorp AppConnections\n\nThe gem `google-cloud-beyond_corp-app_connections` is main.\n[google-cloud-beyond_corp-app_connections-v1](https://cloud.google.com/ruby/docs/reference/google-cloud-beyond_corp-app_connections-v1/latest).\n")
	mustWrite(t, filepath.Join(mainDir, "lib", "google-cloud-beyond_corp-app_connections.rb"), "require \"google/cloud/beyond_corp/app_connections\" unless defined? Google::Cloud::BeyondCorp::AppConnections::VERSION\n")
	mustWrite(t, filepath.Join(mainDir, "lib", "google", "cloud", "beyond_corp", "app_connections.rb"), "module Google; module Cloud; module BeyondCorp; module AppConnections; end; end; end; end\n")
	mustWrite(t, filepath.Join(mainDir, "lib", "google", "cloud", "beyond_corp", "app_connections", "version.rb"), "\n  VERSION = \"0.13.1\"\n")
	mustWrite(t, filepath.Join(mainDir, "test", "google", "cloud", "beyond_corp", "app_connections", "client_test.rb"), "# client test\n")

	// Create otherGem staged files
	mustWrite(t, filepath.Join(otherDir, ".repo-metadata.json"), `{"name_pretty":"BeyondCorp AppConnectors"}`)
	mustWrite(t, filepath.Join(otherDir, "Gemfile"), "local_dependencies = [\"google-cloud-beyond_corp-app_connectors-v1\"]\n")
	mustWrite(t, filepath.Join(otherDir, "google-cloud-beyond_corp-app_connectors.gemspec"), "Gem::Specification.new do |gem|\n  gem.name = \"google-cloud-beyond_corp-app_connectors\"\n  gem.add_dependency \"google-cloud-beyond_corp-app_connectors-v1\", \"~> 0.4\"\nend\n")
	mustWrite(t, filepath.Join(otherDir, "README.md"), "# Connectors\n\n[google-cloud-beyond_corp-app_connectors-v1](https://cloud.google.com/ruby/docs/reference/google-cloud-beyond_corp-app_connectors-v1/latest).\n")
	mustWrite(t, filepath.Join(otherDir, "lib", "google", "cloud", "beyond_corp", "app_connectors.rb"), "# app connectors\n")
	mustWrite(t, filepath.Join(otherDir, "lib", "google", "cloud", "beyond_corp", "app_connectors", "version.rb"), "\n  VERSION = \"0.13.1\"\n")

	if err := prepareMultiWrapper(stagingDir, []string{mainGem, otherGem}, "BeyondCorp API", finalGem); err != nil {
		t.Fatal(err)
	}

	// Verify repo metadata updated
	var meta map[string]any
	metaData := mustRead(t, filepath.Join(stagingDir, ".repo-metadata.json"))
	if err := json.Unmarshal([]byte(metaData), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["distribution_name"] != "google-cloud-beyond_corp" || meta["name_pretty"] != "BeyondCorp API" {
		t.Errorf("unexpected repo metadata: %v", meta)
	}

	// Verify adjusted rubocop & yardopts
	rubocop := mustRead(t, filepath.Join(stagingDir, ".rubocop.yml"))
	if !strings.Contains(rubocop, `"google-cloud-beyond_corp.gemspec"`) || !strings.Contains(rubocop, `"lib/google-cloud-beyond_corp.rb"`) {
		t.Errorf("rubocop not adjusted: %s", rubocop)
	}
	yardopts := mustRead(t, filepath.Join(stagingDir, ".yardopts"))
	if !strings.Contains(yardopts, `"BeyondCorp API"`) {
		t.Errorf("yardopts not adjusted: %s", yardopts)
	}

	// Verify synthetic version and main
	syntheticVersion := mustRead(t, filepath.Join(stagingDir, "lib", "google", "cloud", "beyond_corp", "version.rb"))
	if !strings.Contains(syntheticVersion, "module Google\n  module Cloud\n    module BeyondCorp\n      VERSION = \"0.0.1\"\n    end\n  end\nend") {
		t.Errorf("synthetic version unexpected: %s", syntheticVersion)
	}
	syntheticMain := mustRead(t, filepath.Join(stagingDir, "lib", "google", "cloud", "beyond_corp.rb"))
	if !strings.Contains(syntheticMain, `require "lib/google-cloud-beyond_corp"`) {
		t.Errorf("synthetic main unexpected: %s", syntheticMain)
	}

	// Verify adjusted entrypoint
	entrypoint := mustRead(t, filepath.Join(stagingDir, "lib", "google-cloud-beyond_corp.rb"))
	if !strings.Contains(entrypoint, `require "google/cloud/beyond_corp" unless defined? Google::Cloud::BeyondCorp::VERSION`) || !strings.Contains(entrypoint, `require "google/cloud/beyond_corp/app_connectors" unless defined? Google::Cloud::BeyondCorp::AppConnectors::VERSION`) {
		t.Errorf("entrypoint unexpected: %s", entrypoint)
	}

	// Verify gemspec adjusted
	gemspec := mustRead(t, filepath.Join(stagingDir, "google-cloud-beyond_corp.gemspec"))
	if !strings.Contains(gemspec, `gem.name = "google-cloud-beyond_corp"`) || !strings.Contains(gemspec, `gem.summary = "API client library for the BeyondCorp API"`) || !strings.Contains(gemspec, `gem.version = Google::Cloud::BeyondCorp::VERSION`) {
		t.Errorf("gemspec unexpected: %s", gemspec)
	}
	if !strings.Contains(gemspec, `gem.add_dependency "google-cloud-beyond_corp-app_connections-v1", "~> 0.4"`) || !strings.Contains(gemspec, `gem.add_dependency "google-cloud-beyond_corp-app_connectors-v1", "~> 0.4"`) {
		t.Errorf("gemspec missing dependencies: %s", gemspec)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func executionsForTesting(s string) string {
	return strings.TrimSpace(s)
}
