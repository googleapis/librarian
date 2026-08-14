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

package maven

import (
	"slices"
	"strings"
	"testing"
)

func TestDownloadArgs(t *testing.T) {
	online := downloadArgs("g:a:1.0:jar", false)
	want := []string{"dependency:get", "-B", "-ntp", "-Dartifact=g:a:1.0:jar"}
	if !slices.Equal(online, want) {
		t.Errorf("downloadArgs(online) = %v, want %v", online, want)
	}
	offline := downloadArgs("g:a:1.0:jar", true)
	if !slices.Contains(offline, "-o") {
		t.Errorf("downloadArgs(offline) missing -o: %v", offline)
	}
	// -Dartifact must remain last so the coordinate is unambiguous.
	if offline[len(offline)-1] != "-Dartifact=g:a:1.0:jar" {
		t.Errorf("downloadArgs(offline) does not end with -Dartifact: %v", offline)
	}
}

func TestPackageArgs(t *testing.T) {
	online := packageArgs("sdk-platform-java/gapic-generator-java", false)
	joined := strings.Join(online, " ")
	for _, want := range []string{"package", "-B", "-ntp", "-T 1.5C", "-DskipTests", "--also-make"} {
		if !strings.Contains(joined, want) {
			t.Errorf("packageArgs missing %q: %v", want, online)
		}
	}
	if slices.Contains(online, "-o") {
		t.Errorf("packageArgs(online) should not contain -o: %v", online)
	}
	// Module path and --also-make must remain the final two args.
	if got := online[len(online)-2:]; !slices.Equal(got, []string{"-pl", "sdk-platform-java/gapic-generator-java"}) &&
		online[len(online)-1] != "--also-make" {
		t.Errorf("packageArgs tail = %v, want ... -pl <path> --also-make", online)
	}

	offline := packageArgs("x", true)
	if !slices.Contains(offline, "-o") {
		t.Errorf("packageArgs(offline) missing -o: %v", offline)
	}
	if offline[len(offline)-1] != "--also-make" {
		t.Errorf("packageArgs(offline) must still end with --also-make: %v", offline)
	}
}

func TestMavenOffline(t *testing.T) {
	for _, test := range []struct {
		val  string
		want bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"1", true},
		{"true", true},
		{"YES", true},
	} {
		t.Setenv("LIBRARIAN_MAVEN_OFFLINE", test.val)
		if got := mavenOffline(); got != test.want {
			t.Errorf("mavenOffline() with %q = %v, want %v", test.val, got, test.want)
		}
	}
}
