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

package ast

import (
	"testing"
)

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestTypeNodeString(t *testing.T) {
	tests := []struct {
		typeNode *TypeNode
		want     string
		fullName string
	}{
		{TypeVoid, "void", "void"},
		{TypeBoolean, "boolean", "boolean"},
		{TypeByte, "byte", "byte"},
		{TypeShort, "short", "short"},
		{TypeInt, "int", "int"},
		{TypeLong, "long", "long"},
		{TypeFloat, "float", "float"},
		{TypeDouble, "double", "double"},
		{TypeChar, "char", "char"},
		{TypeString, "String", "String"},
		{TypeBoxedInteger, "Integer", "Integer"},
		{TypeBoxedBoolean, "Boolean", "Boolean"},
		{TypeBoxedLong, "Long", "Long"},
		{TypeBoxedDouble, "Double", "Double"},
		{TypeBoxedFloat, "Float", "Float"},
		{TypeObject, "Object", "Object"},
		{TypeException, "Exception", "Exception"},
		{TypeIOException, "IOException", "java.io.IOException"},
		{TypeAutoCloseable, "AutoCloseable", "AutoCloseable"},
		{ListType(TypeString), "List<String>", "java.util.List"},
		{MapType(TypeString, TypeInt), "Map<String, int>", "java.util.Map"},
		{SetType(TypeString), "Set<String>", "java.util.Set"},
		{ArrayType(TypeByte), "byte[]", "byte"},
		{WildcardType(TypeString), "? extends String", "?"},
		{ObjectType("EchoClient", "com.google.example.v1"), "EchoClient", "com.google.example.v1.EchoClient"},
	}
	for _, tt := range tests {
		if got := tt.typeNode.String(); got != tt.want {
			t.Errorf("TypeNode.String() = %q, want %q", got, tt.want)
		}
		if got := tt.typeNode.FullName(); got != tt.fullName {
			t.Errorf("TypeNode.FullName() = %q, want %q", got, tt.fullName)
		}
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestValueExprs(t *testing.T) {
	if IntVal(42).Literal != "42" {
		t.Errorf("expected 42")
	}
	if LongVal(100).Literal != "100L" {
		t.Errorf("expected 100L")
	}
	if BoolVal(true).Literal != "true" || BoolVal(false).Literal != "false" {
		t.Errorf("expected true and false")
	}
	if StringVal("hello").Literal != "hello" || !StringVal("hello").IsString {
		t.Errorf("expected string literal hello")
	}
	if NullVal().Literal != "null" || !NullVal().IsNull {
		t.Errorf("expected null")
	}
	if ThisVal().Literal != "this" || !ThisVal().IsThis {
		t.Errorf("expected this")
	}
	if SuperVal().Literal != "super" || !SuperVal().IsSuper {
		t.Errorf("expected super")
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestASTDefinitions(t *testing.T) {
	v := &Variable{Name: "client", Type: ObjectType("EchoClient", "com.google.example.v1")}
	ve := &VariableExpr{
		Variable: v,
		Scope:    Private,
		IsFinal:  true,
		IsStatic: false,
		InitExpr: &NewObjectExpr{
			Type: ObjectType("EchoClient", "com.google.example.v1"),
		},
	}
	if ve.Variable.Name != "client" || !ve.IsFinal || ve.Scope != Private {
		t.Errorf("unexpected variable expr: %+v", ve)
	}

	method := &MethodDefinition{
		Scope:            Public,
		Modifiers:        []Modifier{Static, Final},
		Name:             "echo",
		ReturnType:       TypeString,
		ThrowsExceptions: []*TypeNode{TypeIOException},
		Arguments: []*VariableExpr{
			{Variable: &Variable{Name: "content", Type: TypeString}},
		},
		JavaDoc: &JavaDocComment{
			Paragraphs: []string{"Performs an echo operation."},
			Params:     []ParamDoc{{Name: "content", Description: "the content to echo"}},
			Returns:    "the echoed content",
			Throws:     []ThrowsDoc{{Exception: "IOException", Description: "on I/O error"}},
			Deprecated: "Use newEcho() instead",
		},
		Annotations: []*AnnotationNode{
			{Type: ObjectType("Override", "java.lang")},
		},
		Statements: []Statement{
			&ReturnExpr{Expr: &VariableExpr{Variable: &Variable{Name: "content"}}},
		},
	}
	if method.Name != "echo" || len(method.Statements) != 1 || len(method.ThrowsExceptions) != 1 {
		t.Errorf("unexpected method definition: %+v", method)
	}

	clazz := &ClassDefinition{
		PackageName: "com.google.example.v1",
		Scope:       Public,
		Modifiers:   []Modifier{Final},
		Name:        "EchoClient",
		ExtendsType: ObjectType("BaseClient", "com.google.example.v1"),
		ImplementsTypes: []*TypeNode{
			TypeAutoCloseable,
		},
		Fields:  []*VariableExpr{ve},
		Methods: []*MethodDefinition{method},
	}
	if clazz.Name != "EchoClient" || len(clazz.Fields) != 1 || len(clazz.Methods) != 1 {
		t.Errorf("unexpected class definition: %+v", clazz)
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestComplexStatementsAndExpressions(t *testing.T) {
	// If statement with ElseIf and Else
	ifStmt := &IfStatement{
		Condition: &BinaryOperationExpr{
			Left:     &VariableExpr{Variable: &Variable{Name: "x"}},
			Operator: ">",
			Right:    IntVal(0),
		},
		ThenStatements: []Statement{
			&ExprStatement{Expr: &AssignmentExpr{
				Variable: &VariableExpr{Variable: &Variable{Name: "y"}},
				Value:    IntVal(1),
			}},
		},
		ElseIfs: []*ElseIfBlock{
			{
				Condition: &BinaryOperationExpr{
					Left:     &VariableExpr{Variable: &Variable{Name: "x"}},
					Operator: "==",
					Right:    IntVal(0),
				},
				Statements: []Statement{
					&ExprStatement{Expr: &AssignmentExpr{
						Variable: &VariableExpr{Variable: &Variable{Name: "y"}},
						Value:    IntVal(0),
					}},
				},
			},
		},
		ElseStatements: []Statement{
			&ExprStatement{Expr: &AssignmentExpr{
				Variable: &VariableExpr{Variable: &Variable{Name: "y"}},
				Value:    IntVal(-1),
			}},
		},
	}
	if len(ifStmt.ElseIfs) != 1 || len(ifStmt.ElseStatements) != 1 {
		t.Errorf("unexpected ifStmt: %+v", ifStmt)
	}

	// While statement
	whileStmt := &WhileStatement{
		Condition: &BinaryOperationExpr{
			Left:     &VariableExpr{Variable: &Variable{Name: "x"}},
			Operator: ">",
			Right:    IntVal(0),
		},
		Statements: []Statement{
			&ExprStatement{Expr: &UnaryOperationExpr{
				Expr:     &VariableExpr{Variable: &Variable{Name: "x"}},
				Operator: "--",
				Postfix:  true,
			}},
		},
	}
	if len(whileStmt.Statements) != 1 {
		t.Errorf("unexpected whileStmt: %+v", whileStmt)
	}

	// For statement
	forStmt := &ForStatement{
		Init: &VariableExpr{
			IsDecl:   true,
			Variable: &Variable{Name: "i", Type: TypeInt},
			InitExpr: IntVal(0),
		},
		Condition: &BinaryOperationExpr{
			Left:     &VariableExpr{Variable: &Variable{Name: "i"}},
			Operator: "<",
			Right:    IntVal(10),
		},
		Update: &ExprStatement{
			Expr: &UnaryOperationExpr{
				Expr:     &VariableExpr{Variable: &Variable{Name: "i"}},
				Operator: "++",
				Postfix:  true,
			},
		},
		Statements: []Statement{
			&ExprStatement{Expr: &AssignmentOperationExpr{
				Left:     &VariableExpr{Variable: &Variable{Name: "sum"}},
				Operator: "+=",
				Right:    &VariableExpr{Variable: &Variable{Name: "i"}},
			}},
		},
	}
	if len(forStmt.Statements) != 1 {
		t.Errorf("unexpected forStmt: %+v", forStmt)
	}

	// General (enhanced) for statement
	genFor := &GeneralForStatement{
		ItemVar:    &Variable{Name: "item", Type: TypeString},
		Collection: &VariableExpr{Variable: &Variable{Name: "items"}},
		Statements: []Statement{
			&ExprStatement{Expr: &MethodInvocationExpr{
				TargetExpr: &VariableExpr{Variable: &Variable{Name: "System.out"}},
				MethodName: "println",
				Arguments:  []Expr{&VariableExpr{Variable: &Variable{Name: "item"}}},
			}},
		},
	}
	if len(genFor.Statements) != 1 {
		t.Errorf("unexpected genFor: %+v", genFor)
	}

	// Try-catch-finally statement
	tryCatch := &TryCatchStatement{
		Resources: []*VariableExpr{
			{
				Variable: &Variable{Name: "client", Type: ObjectType("EchoClient", "com.google.example.v1")},
				InitExpr: &MethodInvocationExpr{
					TargetType: ObjectType("EchoClient", "com.google.example.v1"),
					MethodName: "create",
				},
			},
		},
		TryBody: []Statement{
			&ExprStatement{Expr: &MethodInvocationExpr{
				TargetExpr: &VariableExpr{Variable: &Variable{Name: "client"}},
				MethodName: "echo",
				Arguments:  []Expr{StringVal("test")},
			}},
		},
		CatchBlocks: []*CatchBlock{
			{
				Exceptions: []*TypeNode{TypeIOException, TypeException},
				VarName:    "e",
				Body: []Statement{
					&ThrowExpr{Expr: &NewObjectExpr{
						Type:      ObjectType("RuntimeException", "java.lang"),
						Arguments: []Expr{&VariableExpr{Variable: &Variable{Name: "e"}}},
					}},
				},
			},
		},
		FinallyBody: []Statement{
			&LineComment{Comment: "cleanup"},
		},
	}
	if len(tryCatch.Resources) != 1 || len(tryCatch.CatchBlocks) != 1 {
		t.Errorf("unexpected tryCatch: %+v", tryCatch)
	}

	// Synchronized statement
	syncStmt := &SynchronizedStatement{
		LockExpr: &VariableExpr{Variable: &Variable{Name: "lock"}},
		Statements: []Statement{
			&ExprStatement{Expr: &AssignmentExpr{
				Variable: &VariableExpr{Variable: &Variable{Name: "count"}},
				Value:    IntVal(1),
			}},
		},
	}
	if len(syncStmt.Statements) != 1 {
		t.Errorf("unexpected syncStmt: %+v", syncStmt)
	}

	// Expressions: Ternary, Cast, Instanceof, Lambda, ReferenceConstructor, EnumRef, ArrayExpr
	ternary := &TernaryExpr{
		Condition: &BinaryOperationExpr{Left: &VariableExpr{Variable: &Variable{Name: "a"}}, Operator: ">", Right: &VariableExpr{Variable: &Variable{Name: "b"}}},
		ThenExpr:  &VariableExpr{Variable: &Variable{Name: "a"}},
		ElseExpr:  &VariableExpr{Variable: &Variable{Name: "b"}},
	}
	if ternary.Condition == nil || ternary.ThenExpr == nil || ternary.ElseExpr == nil {
		t.Errorf("unexpected nil in ternary: %+v", ternary)
	}

	cast := &CastExpr{Type: TypeString, Expr: &VariableExpr{Variable: &Variable{Name: "obj"}}}
	instanceOf := &InstanceofExpr{Expr: &VariableExpr{Variable: &Variable{Name: "obj"}}, CheckType: TypeString}
	lambda := &LambdaExpr{
		Arguments: []*Variable{{Name: "x", Type: TypeInt}},
		BodyExpr:  &BinaryOperationExpr{Left: &VariableExpr{Variable: &Variable{Name: "x"}}, Operator: "*", Right: IntVal(2)},
	}
	refConst := &ReferenceConstructorExpr{Type: ObjectType("EchoClient", "com.google.example.v1"), MethodName: "create"}
	enumRef := &EnumRefExpr{Type: ObjectType("StatusCode", "com.google.api.gax.rpc"), Value: "OK"}
	arrayExpr := &ArrayExpr{Type: TypeString, Elements: []Expr{StringVal("a"), StringVal("b")}}

	if cast.Expr == nil || instanceOf.Expr == nil || len(lambda.Arguments) != 1 || refConst.Type == nil || enumRef.Type == nil || len(arrayExpr.Elements) != 2 {
		t.Errorf("unexpected nil expression")
	}
}
