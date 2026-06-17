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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDeleteMethod(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		content  string
		funcName string
		want     string
	}{
		{
			name: "simple method",
			content: `package com.example;
class Test {
    public void myMethod() {
        System.out.println("hello");
    }
    public void otherMethod() {
    }
}`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
    public void otherMethod() {
    }
}`,
		},
		{
			name: "nested braces",
			content: `package com.example;
class Test {
    public void myMethod() {
        if (true) {
            System.out.println("nested");
        }
    }
    public void otherMethod() {
    }
}`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
    public void otherMethod() {
    }
}`,
		},
		// This test has an unbalanced '{' in a string.
		// It passes because cleanJavaCode strips strings before brace counting.
		{
			name: "braces in strings",
			content: `package com.example;
class Test {
    public void myMethod() {
        System.out.println(" { ");
    }
    public void otherMethod() {
    }
}`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
    public void otherMethod() {
    }
}`,
		},
		{
			name: "braces in comments",
			content: `package com.example;
class Test {
    public void myMethod() {
        // {
        /* } */
    }
    public void otherMethod() {
    }
}`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
    public void otherMethod() {
    }
}`,
		},
		{
			name: "escaped backslash before quote",
			content: `package com.example;
class Test {
    public void myMethod() {
        System.out.println("value \\");
        if (true) {
             System.out.println("inside if");
        }
    }
    public void otherMethod() {
    }
}`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
    public void otherMethod() {
    }
}`,
		},
		{
			name: "annotation with braces",
			content: `package com.example;
class Test {
    @MyAnnotation(keys = {"a", "b"})
    public void myMethod() {
        System.out.println("hello");
    }
    public void otherMethod() {
    }
}`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
    @MyAnnotation(keys = {"a", "b"})
    public void otherMethod() {
    }
}`,
		},
		{
			name: "java text block",
			content: `package com.example;
class Test {
    public void myMethod() {
        String block = """
            {
            }
            """;
    }
    public void otherMethod() {
    }
}`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
    public void otherMethod() {
    }
}`,
		},
		{
			name: "signature in comment first",
			content: `package com.example;
class Test {
    // Deprecated: public void myMethod() is no longer used
    public void myMethod() {
        System.out.println("hello");
    }
    public void otherMethod() {
    }
}`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
    // Deprecated: public void myMethod() is no longer used
    public void otherMethod() {
    }
}`,
		},
		{
			name:     "same line delete exact range only",
			content:  "class Test { public void myMethod() { System.out.println(\"hello\"); } public void otherMethod() {} }",
			funcName: "public void myMethod()",
			want:     "class Test {  public void otherMethod() {} }",
		},
		{
			name: "method followed by class closing brace on same line",
			content: `package com.example;
class Test {
    public void myMethod() {} }`,
			funcName: "public void myMethod()",
			want: `package com.example;
class Test {
     }`,
		},
		{
			name: "complete signature deletes method cleanly",
			content: `class Test {
    public int afoo() { return 1; }
    public int foo() { return 2; }
}`,
			funcName: "public int foo()",
			want: `class Test {
    public int afoo() { return 1; }
}`,
		},
		{
			name: "delete multiple matching methods",
			content: `class A {
    public void foo() {
        System.out.println("A");
    }
}
class B {
    public void foo() {
        System.out.println("B");
    }
}`,
			funcName: "public void foo()",
			want: `class A {
}
class B {
}`,
		},
		{
			name: "nested method with same signature",
			content: `class Test {
    public void foo() {
        Runnable r = new Runnable() {
            public void foo() {
                System.out.println("inner");
            }
        };
    }
}`,
			funcName: "public void foo()",
			want: `class Test {
}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpFile := filepath.Join(t.TempDir(), "test.java")
			if err := os.WriteFile(tmpFile, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			if err := DeleteMethod(tmpFile, test.funcName, "java"); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeleteMethod_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		content  string
		funcName string
		language string
		wantErr  error
	}{
		{
			name:     "function not found",
			content:  "class T { void o() {} }",
			funcName: "void m()",
			language: "java",
			wantErr:  errMethodNotFound,
		},
		{
			name:     "unsupported language",
			content:  "def m(): pass",
			funcName: "def m()",
			language: "python",
			wantErr:  errUnsupportedLanguage,
		},
		{
			name:     "opening brace not found",
			content:  "class T { void m(); void o() {} }",
			funcName: "void m()",
			language: "java",
			wantErr:  errOpeningBraceNotFound,
		},
		{
			name:     "invalid signature",
			content:  "class T { void m() {} }",
			funcName: "void m",
			language: "java",
			wantErr:  errInvalidSignature,
		},
		{
			name:     "closing brace not found",
			content:  "class T { void m() { System.out.println();",
			funcName: "void m()",
			language: "java",
			wantErr:  errClosingBraceNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpFile := filepath.Join(t.TempDir(), "test.java")
			if err := os.WriteFile(tmpFile, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			err := DeleteMethod(tmpFile, test.funcName, test.language)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("DeleteMethod(%q, %q, %q) error = %v, wantErr %v", tmpFile, test.funcName, test.language, err, test.wantErr)
			}
		})
	}
}

func TestAdjustDeleteBounds(t *testing.T) {
	for _, test := range []struct {
		name        string
		content     string
		startSub    string
		endSub      string
		wantDeleted string
	}{
		{
			name: "method on its own lines",
			content: `class T {
    void m() {
    }
}`,
			startSub:    "void m()",
			endSub:      "}",
			wantDeleted: "    void m() {\n    }\n",
		},
		{
			name:        "inline method",
			content:     "class T { void m() {} void o() {} }",
			startSub:    "void m()",
			endSub:      "}",
			wantDeleted: "void m() {}",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := []byte(test.content)
			start := strings.Index(test.content, test.startSub)
			endRelative := strings.Index(test.content[start:], test.endSub)
			end := start + endRelative + len(test.endSub)
			gotStart, gotEnd := adjustDeleteBounds(content, start, end)
			got := string(content[gotStart:gotEnd])
			if diff := cmp.Diff(test.wantDeleted, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
