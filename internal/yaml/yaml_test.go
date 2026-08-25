// Copyright 2025 Google LLC
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

package yaml

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.yaml.in/yaml/v3"
)

type testConfig struct {
	Name    string `yaml:"name,omitempty"`
	Version string `yaml:"version,omitempty"`
}

func TestUnmarshal(t *testing.T) {
	got, err := Unmarshal[testConfig]([]byte("name: test\nversion: v1.0.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := &testConfig{Name: "test", Version: "v1.0.0"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUnmarshalError(t *testing.T) {
	_, err := Unmarshal[testConfig]([]byte("name: [invalid"))
	if err == nil {
		t.Error("Unmarshal() expected error for invalid YAML")
	}
}

func TestMarshal(t *testing.T) {
	input := &testConfig{Name: "test", Version: "v1.0.0"}
	data, err := Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Unmarshal[testConfig](data)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(input, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMarshal_Comments(t *testing.T) {
	const input = `# Header comment
name: test # Inline comment
# Version comment
version: v1.0.0
`
	t.Run("struct drops comments", func(t *testing.T) {
		cfg, err := Unmarshal[testConfig]([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		const want = `name: test
version: v1.0.0
`
		if diff := cmp.Diff(want, string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("yaml.Node preserves comments", func(t *testing.T) {
		node, err := Unmarshal[yaml.Node]([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Marshal(node)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(input, string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("yaml.Node preserves comments after mutation", func(t *testing.T) {
		node, err := Unmarshal[yaml.Node]([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		mapping := node.Content[0]
		for i := 0; i < len(mapping.Content); i += 2 {
			if mapping.Content[i].Value == "version" {
				mapping.Content[i+1].Value = "v2.0.0"
			}
		}
		got, err := Marshal(node)
		if err != nil {
			t.Fatal(err)
		}
		const want = `# Header comment
name: test # Inline comment
# Version comment
version: v2.0.0
`
		if diff := cmp.Diff(want, string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestMarshalWithComments(t *testing.T) {
	type item struct {
		Name string `yaml:"name"`
		Path string `yaml:"path,omitempty"`
	}
	type itemConfig struct {
		Items []item   `yaml:"items,omitempty"`
		Roots []string `yaml:"roots,omitempty"`
	}

	for _, test := range []struct {
		name     string
		original string
		value    any
		want     string
	}{
		{
			name: "struct with modified fields preserves comments",
			original: `# Header comment
name: test # Inline comment
# Version comment
version: v1.0.0
`,
			value: &testConfig{Name: "test", Version: "v2.0.0"},
			want: `# Header comment
name: test # Inline comment
# Version comment
version: v2.0.0
`,
		},
		{
			name: "reordered sequence items preserve comments",
			original: `items:
  - # Comment A
    name: a # Inline A
    path: path/a
  - # Comment B
    name: b
    path: path/b
`,
			value: &itemConfig{
				Items: []item{
					{Name: "b", Path: "path/b"},
					{Name: "a", Path: "path/a"},
				},
			},
			want: `items:
  - # Comment B
    name: b
    path: path/b
  - # Comment A
    name: a # Inline A
    path: path/a
`,
		},
		{
			name: "added items do not inherit deleted item comments",
			original: `items:
  - # Comment old
    name: old
`,
			value: &itemConfig{
				Items: []item{
					{Name: "new"},
				},
			},
			want: `items:
  - name: new
`,
		},
		{
			name: "scalar sequence items preserve comments",
			original: `roots:
  - googleapis # inline googleapis
  - other
`,
			value: &itemConfig{
				Roots: []string{"googleapis", "newroot"},
			},
			want: `roots:
  - googleapis # inline googleapis
  - newroot
`,
		},
		{
			name:     "empty original YAML",
			original: "",
			value:    &testConfig{Name: "test", Version: "v1.0.0"},
			want: `name: test
version: v1.0.0
`,
		},
		{
			name:     "invalid original YAML falls back to marshal",
			original: "[invalid yaml",
			value:    &testConfig{Name: "test", Version: "v1.0.0"},
			want: `name: test
version: v1.0.0
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := MarshalWithComments(test.value, []byte(test.original))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadWrite(t *testing.T) {
	want := &testConfig{Name: "test", Version: "v1.0.0"}
	path := filepath.Join(t.TempDir(), "test.yaml")
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read[testConfig](path)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

const copyright = `# Copyright %s Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
`

func TestWrite(t *testing.T) {
	header := fmt.Sprintf(copyright, strconv.Itoa(time.Now().Year()))

	for _, test := range []struct {
		name       string
		existing   string
		value      any
		want       string
		writeTwice bool
	}{
		{
			name:  "new file adds license header",
			value: &testConfig{Name: "test", Version: "v1.0.0"},
			want: header + `name: test
version: v1.0.0
`,
		},
		{
			name: "existing file preserves comments",
			existing: header + `# Custom section comment
name: test # Inline comment
# Version comment
version: v1.0.0
`,
			value: &testConfig{Name: "test", Version: "v2.0.0"},
			want: header + `# Custom section comment
name: test # Inline comment
# Version comment
version: v2.0.0
`,
			writeTwice: true,
		},
		{
			name: "existing file without license header adds header",
			existing: `# Custom comment
name: test
version: v1.0.0
`,
			value: &testConfig{Name: "test", Version: "v2.0.0"},
			want: header + `# Custom comment
name: test
version: v2.0.0
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.yaml")
			if test.existing != "" {
				if err := os.WriteFile(path, []byte(test.existing), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := Write(path, test.value); err != nil {
				t.Fatal(err)
			}
			if test.writeTwice {
				if err := Write(path, test.value); err != nil {
					t.Fatal(err)
				}
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadError(t *testing.T) {
	_, err := Read[testConfig]("/nonexistent/path/file.yaml")
	if err == nil {
		t.Error("Read() expected error for nonexistent file")
	}
}

func TestWriteError(t *testing.T) {
	err := Write("/nonexistent/path/file.yaml", &testConfig{Name: "test"})
	if err == nil {
		t.Error("Write() expected error for invalid path")
	}
}

func TestStringSlice_EmptySlice(t *testing.T) {
	strSlice := StringSlice{}
	got := strSlice.IsZero()
	if diff := cmp.Diff(false, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestStringSlice_NilSlice(t *testing.T) {
	var strSlice StringSlice
	got := strSlice.IsZero()
	if diff := cmp.Diff(true, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestEmpty(t *testing.T) {
	for _, test := range []struct {
		name  string
		value testConfig
		want  bool
	}{
		{
			name: "empty",
			value: testConfig{
				Name: "",
			},
			want: true,
		},
		{
			name: "not empty",
			value: testConfig{
				Name: "name",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Empty(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
