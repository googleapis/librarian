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

package postprocessing

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestDeprecateMethod(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		inputContent       string
		funcName           string
		deprecationMessage string
		wantContent        string
	}{
		{
			name: "simple method without javadoc and annotations",
			inputContent: `package com.google.example;

public class TestClass {
  public static String formatName(String name) {
    return name;
  }
}
`,
			funcName:           "public static String formatName(String name)",
			deprecationMessage: "Please use {@link #anotherMethod()} instead",
			wantContent: `package com.google.example;

public class TestClass {
  /**
   * @deprecated Please use {@link #anotherMethod()} instead
   */
  @Deprecated
  public static String formatName(String name) {
    return name;
  }
}
`,
		},
		{
			name: "method with existing annotations but no javadoc",
			inputContent: `package com.google.example;

public class TestClass {
  @BetaApi
  @SuppressWarnings("unchecked")
  public static String formatName(String name) {
    return name;
  }
}
`,
			funcName:           "public static String formatName(String name)",
			deprecationMessage: "Please use {@link #anotherMethod()} instead",
			wantContent: `package com.google.example;

public class TestClass {
  /**
   * @deprecated Please use {@link #anotherMethod()} instead
   */
  @BetaApi
  @SuppressWarnings("unchecked")
  @Deprecated
  public static String formatName(String name) {
    return name;
  }
}
`,
		},
		{
			name: "method with existing javadoc but no annotations",
			inputContent: `package com.google.example;

public class TestClass {
  /**
   * Formats a name.
   *
   * @param name the name to format
   * @return the formatted name
   */
  public static String formatName(String name) {
    return name;
  }
}
`,
			funcName:           "public static String formatName(String name)",
			deprecationMessage: "Please use {@link #anotherMethod()} instead",
			wantContent: `package com.google.example;

public class TestClass {
  /**
   * Formats a name.
   *
   * @param name the name to format
   * @return the formatted name
   * @deprecated Please use {@link #anotherMethod()} instead
   */
  @Deprecated
  public static String formatName(String name) {
    return name;
  }
}
`,
		},
		{
			name: "method with existing javadoc and existing annotations",
			inputContent: `package com.google.example;

public class TestClass {
  /**
   * Formats a name.
   */
  @BetaApi
  public static String formatName(String name) {
    return name;
  }
}
`,
			funcName:           "public static String formatName(String name)",
			deprecationMessage: "Please use {@link #anotherMethod()} instead",
			wantContent: `package com.google.example;

public class TestClass {
  /**
   * Formats a name.
   * @deprecated Please use {@link #anotherMethod()} instead
   */
  @BetaApi
  @Deprecated
  public static String formatName(String name) {
    return name;
  }
}
`,
		},
		{
			name: "method with single-line javadoc",
			inputContent: `package com.google.example;

public class TestClass {
  /** Formats a name. */
  public static String formatName(String name) {
    return name;
  }
}
`,
			funcName:           "public static String formatName(String name)",
			deprecationMessage: "Please use {@link #anotherMethod()} instead",
			wantContent: `package com.google.example;

public class TestClass {
  /**
   * Formats a name.
   * @deprecated Please use {@link #anotherMethod()} instead
   */
  @Deprecated
  public static String formatName(String name) {
    return name;
  }
}
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "TestClass.java")
			if err := os.WriteFile(filePath, []byte(test.inputContent), 0644); err != nil {
				t.Fatal(err)
			}
			if err := DeprecateMethod(filePath, test.funcName, test.deprecationMessage, config.LanguageJava); err != nil {
				t.Fatal(err)
			}
			gotBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantContent, string(gotBytes)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeprecateMethod_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		inputContent       string
		funcName           string
		deprecationMessage string
		language           string
		wantErr            error
	}{
		{
			name:               "unsupported language",
			inputContent:       "public void foo() {}",
			funcName:           "public void foo()",
			deprecationMessage: "use bar",
			language:           "python",
			wantErr:            errUnsupportedLanguage,
		},
		{
			name:               "invalid method signature",
			inputContent:       "public void foo() {}",
			funcName:           "public void foo",
			deprecationMessage: "use bar",
			language:           config.LanguageJava,
			wantErr:            errInvalidSignature,
		},
		{
			name:               "empty deprecation message",
			inputContent:       "public void foo() {}",
			funcName:           "public void foo()",
			deprecationMessage: "   ",
			language:           config.LanguageJava,
			wantErr:            errEmptyDeprecationMessage,
		},
		{
			name: "method not found",
			inputContent: `public class TestClass {
  public void bar() {}
}
`,
			funcName:           "public void foo()",
			deprecationMessage: "use bar",
			language:           config.LanguageJava,
			wantErr:            errMethodNotFound,
		},
		{
			name: "ambiguous matches in a file",
			inputContent: `package com.google.example;

public class TestClass {
  public static String formatName(String name) {
    return name;
  }

  public static class Inner {
    public static String formatName(String name) {
      return "inner-" + name;
    }
  }
}
`,
			funcName:           "public static String formatName(String name)",
			deprecationMessage: "Please use {@link #anotherMethod()} instead",
			language:           config.LanguageJava,
			wantErr:            errAmbiguousDeprecation,
		},
		{
			name: "method already deprecated with tag and annotation",
			inputContent: `package com.google.example;

public class TestClass {
  /**
   * Formats a name.
   * @deprecated Please use {@link #anotherMethod()} instead
   */
  @Deprecated
  public static String formatName(String name) {
    return name;
  }
}
`,
			funcName:           "public static String formatName(String name)",
			deprecationMessage: "Please use {@link #anotherMethod()} instead",
			language:           config.LanguageJava,
			wantErr:            errMethodAlreadyDeprecated,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "TestClass.java")
			if err := os.WriteFile(filePath, []byte(test.inputContent), 0644); err != nil {
				t.Fatal(err)
			}
			err := DeprecateMethod(filePath, test.funcName, test.deprecationMessage, test.language)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("DeprecateMethod(%q, %q, %q, %q) error = %v, wantErr %v", filePath, test.funcName, test.deprecationMessage, test.language, err, test.wantErr)
			}
		})
	}
}

func TestAddJavadocTag(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		lines       []string
		indentation string
		header      methodHeader
		tagName     string
		tagMessage  string
		wantLines   []string
	}{
		{
			name: "no javadoc",
			lines: []string{
				"  public void foo() {",
				"  }",
			},
			indentation: "  ",
			header: methodHeader{
				hasJavadoc:         false,
				firstAnnotationIdx: 0,
			},
			tagName:    "deprecated",
			tagMessage: "use bar",
			wantLines: []string{
				"  /**",
				"   * @deprecated use bar",
				"   */",
				"  public void foo() {",
				"  }",
			},
		},
		{
			name: "existing Javadoc without tag",
			lines: []string{
				"  /**",
				"   * Hello world",
				"   */",
				"  public void foo() {",
				"  }",
			},
			indentation: "  ",
			header: methodHeader{
				hasJavadoc:       true,
				javadocEndIdx:    2,
				hasDeprecatedTag: false,
			},
			tagName:    "deprecated",
			tagMessage: "use bar",
			wantLines: []string{
				"  /**",
				"   * Hello world",
				"   * @deprecated use bar",
				"   */",
				"  public void foo() {",
				"  }",
			},
		},
		{
			name: "existing Javadoc with tag",
			lines: []string{
				"  /**",
				"   * @deprecated use bar",
				"   */",
				"  public void foo() {",
				"  }",
			},
			indentation: "  ",
			header: methodHeader{
				hasJavadoc:       true,
				javadocEndIdx:    2,
				hasDeprecatedTag: true,
			},
			tagName:    "deprecated",
			tagMessage: "use bar",
			wantLines: []string{
				"  /**",
				"   * @deprecated use bar",
				"   */",
				"  public void foo() {",
				"  }",
			},
		},
		{
			name: "single-line Javadoc",
			lines: []string{
				"  /** Hello world */",
				"  public void foo() {",
				"  }",
			},
			indentation: "  ",
			header: methodHeader{
				hasJavadoc:       true,
				javadocEndIdx:    0,
				hasDeprecatedTag: false,
			},
			tagName:    "deprecated",
			tagMessage: "use bar",
			wantLines: []string{
				"  /**",
				"   * Hello world",
				"   * @deprecated use bar",
				"   */",
				"  public void foo() {",
				"  }",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			inputBytes := make([][]byte, len(test.lines))
			for i, l := range test.lines {
				inputBytes[i] = []byte(l)
			}
			gotBytes := addJavadocTag(inputBytes, []byte(test.indentation), test.header, test.tagName, test.tagMessage)
			gotLines := make([]string, len(gotBytes))
			for i, b := range gotBytes {
				gotLines[i] = string(b)
			}
			if diff := cmp.Diff(test.wantLines, gotLines); diff != "" {
				t.Errorf("addJavadocTag() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
