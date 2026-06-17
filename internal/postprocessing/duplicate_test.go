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
)

func TestDuplicateMethod(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		content  string
		funcName string
		newName  string
		want     string
	}{
		{
			name: "simple java method",
			content: `package com.example;

public class TestClass {
	public void myMethod() {
		System.out.println("Hello");
	}
}
`,
			funcName: "public void myMethod()",
			newName:  "myMethodCopy",
			want: `package com.example;

public class TestClass {
	public void myMethod() {
		System.out.println("Hello");
	}

	public void myMethodCopy() {
		System.out.println("Hello");
	}
}
`,
		},
		{
			name: "spaces and comments",
			content: `package com.example;

public class TestClass {
	public void myMethod   (   ) {
		System.out.println("Hello");
	}
}
// Some trailing comment with brace }
`,
			funcName: "public void myMethod   (   )",
			newName:  "myMethodCopy",
			want: `package com.example;

public class TestClass {
	public void myMethod   (   ) {
		System.out.println("Hello");
	}

	public void myMethodCopy(   ) {
		System.out.println("Hello");
	}
}
// Some trailing comment with brace }
`,
		},
		{
			name: "insert between methods",
			content: `package com.example;

public class TestClass {
	public void myMethod() {
		System.out.println("Hello");
	}

	public void otherMethod() {
		System.out.println("Other");
	}
}
`,
			funcName: "public void myMethod()",
			newName:  "myMethodCopy",
			want: `package com.example;

public class TestClass {
	public void myMethod() {
		System.out.println("Hello");
	}

	public void myMethodCopy() {
		System.out.println("Hello");
	}

	public void otherMethod() {
		System.out.println("Other");
	}
}
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "TestClass.java")
			if err := os.WriteFile(path, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			if err := DuplicateMethod(t.Context(), path, test.funcName, test.newName, "java"); err != nil {
				t.Fatal(err)
			}
			gotBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got := string(gotBytes)
			if got != test.want {
				t.Errorf("DuplicateMethod() =\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
}

func TestDuplicateMethod_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		content  string
		funcName string
		newName  string
		language string
		wantErr  error
	}{
		{
			name:     "unsupported language",
			content:  "def myMethod(): pass",
			funcName: "def myMethod()",
			newName:  "myMethodCopy",
			language: "python",
			wantErr:  errUnsupportedLanguage,
		},
		{
			name:     "invalid signature (no closing parenthesis)",
			content:  "class T { void myMethod( }",
			funcName: "void myMethod(",
			newName:  "myMethodCopy",
			language: "java",
			wantErr:  errInvalidSignature,
		},
		{
			name: "duplicate already exists",
			content: `package com.example;
public class TestClass {
	public void myMethod() {}
	public void myMethodCopy() {}
}`,
			funcName: "public void myMethod()",
			newName:  "myMethodCopy",
			language: "java",
			wantErr:  errMethodAlreadyExists,
		},
		{
			name:     "method not found",
			content:  "class T { void otherMethod() {} }",
			funcName: "void myMethod()",
			newName:  "myMethodCopy",
			language: "java",
			wantErr:  errMethodNotFound,
		},
		{
			name:     "method not found (target is substring of existing method)",
			content:  "class T { void foobar() {} }",
			funcName: "void foo()",
			newName:  "myMethodCopy",
			language: "java",
			wantErr:  errMethodNotFound,
		},
		{
			name:     "method not found (existing method is substring of target)",
			content:  "class T { void foo() {} }",
			funcName: "void foobar()",
			newName:  "myMethodCopy",
			language: "java",
			wantErr:  errMethodNotFound,
		},
		{
			name:     "opening brace not found",
			content:  "class T { void myMethod(); void otherMethod() {} }",
			funcName: "void myMethod()",
			newName:  "myMethodCopy",
			language: "java",
			wantErr:  errOpeningBraceNotFound,
		},
		{
			name:     "closing brace not found",
			content:  "class T { void myMethod() { System.out.println();",
			funcName: "void myMethod()",
			newName:  "myMethodCopy",
			language: "java",
			wantErr:  errClosingBraceNotFound,
		},
		{
			name: "ambiguous duplication (same signature in nested class)",
			content: `package com.example;
public class TestClass {
	public void myMethod() {}
	public static class Inner {
		public void myMethod() {}
	}
}`,
			funcName: "public void myMethod()",
			newName:  "myMethodCopy",
			language: "java",
			wantErr:  errAmbiguousDuplication,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "TestClass.java")
			if err := os.WriteFile(path, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			err := DuplicateMethod(t.Context(), path, test.funcName, test.newName, test.language)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("DuplicateMethod() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
