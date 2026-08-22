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

package writer

import (
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/gapic/java/engine/ast"
)

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestWriteClassBasic(t *testing.T) {
	clazz := &ast.ClassDefinition{
		PackageName: "com.google.example.v1",
		Scope:       ast.Public,
		Name:        "EchoClient",
		ImplementsTypes: []*ast.TypeNode{
			ast.TypeAutoCloseable,
		},
		JavaDoc: &ast.JavaDocComment{
			Paragraphs: []string{"Service Description: This service is used for testing."},
		},
		Fields: []*ast.VariableExpr{
			{
				Scope:   ast.Private,
				IsFinal: true,
				Variable: &ast.Variable{
					Name: "settings",
					Type: ast.ObjectType("EchoSettings", "com.google.example.v1"),
				},
			},
		},
		Methods: []*ast.MethodDefinition{
			{
				Scope:      ast.Public,
				Name:       "getSettings",
				ReturnType: ast.ObjectType("EchoSettings", "com.google.example.v1"),
				Statements: []ast.Statement{
					&ast.ReturnExpr{
						Expr: &ast.VariableExpr{
							Variable: &ast.Variable{Name: "settings"},
						},
					},
				},
			},
			{
				Scope:      ast.Public,
				IsOverride: true,
				Name:       "close",
				ReturnType: ast.TypeVoid,
				Statements: []ast.Statement{
					&ast.ExprStatement{
						Expr: &ast.MethodInvocationExpr{
							TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
							MethodName: "close",
						},
					},
				},
			},
		},
	}

	code := WriteClass(clazz)
	if !strings.Contains(code, "package com.google.example.v1;") {
		t.Errorf("missing package declaration, got:\n%s", code)
	}
	if !strings.Contains(code, "public class EchoClient implements AutoCloseable {") {
		t.Errorf("missing class header, got:\n%s", code)
	}
	if !strings.Contains(code, "private final EchoSettings settings;") {
		t.Errorf("missing field declaration, got:\n%s", code)
	}
	if !strings.Contains(code, "public EchoSettings getSettings() {") {
		t.Errorf("missing method getSettings, got:\n%s", code)
	}
	if !strings.Contains(code, "return settings;") {
		t.Errorf("missing return statement, got:\n%s", code)
	}
	if !strings.Contains(code, "@Override\n  public void close() {") {
		t.Errorf("missing close method override, got:\n%s", code)
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestWriteClassEnum(t *testing.T) {
	clazz := &ast.ClassDefinition{
		PackageName: "com.google.example.v1",
		Scope:       ast.Public,
		IsEnum:      true,
		Name:        "DayOfWeek",
		EnumValues: []*ast.EnumValueDefinition{
			{Name: "MONDAY"},
			{Name: "TUESDAY"},
			{Name: "WEDNESDAY"},
		},
	}

	code := WriteClass(clazz)
	if !strings.Contains(code, "public enum DayOfWeek {") {
		t.Errorf("missing enum header, got:\n%s", code)
	}
	if !strings.Contains(code, "MONDAY,") || !strings.Contains(code, "WEDNESDAY;") {
		t.Errorf("missing enum values, got:\n%s", code)
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestWriteClassWithAnnotations(t *testing.T) {
	clazz := &ast.ClassDefinition{
		PackageName: "com.google.example.v1",
		Scope:       ast.Public,
		Name:        "AnnotatedService",
		Annotations: []*ast.AnnotationNode{
			{Type: ast.ObjectType("Generated", "javax.annotation")},
		},
		Methods: []*ast.MethodDefinition{
			{
				Scope:      ast.Public,
				Name:       "annotatedMethod",
				ReturnType: ast.TypeVoid,
				Annotations: []*ast.AnnotationNode{
					{Type: ast.ObjectType("Beta", "com.google.api.core")},
				},
				Statements: []ast.Statement{},
			},
		},
	}

	code := WriteClass(clazz)
	if !strings.Contains(code, "import com.google.api.core.Beta;") {
		t.Errorf("missing method annotation import in code:\n%s", code)
	}
	if !strings.Contains(code, "import javax.annotation.Generated;") {
		t.Errorf("missing class annotation import in code:\n%s", code)
	}
}
