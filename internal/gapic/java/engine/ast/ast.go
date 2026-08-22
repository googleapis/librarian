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

// Package ast defines the Java abstract syntax tree for code generation.
package ast

import (
	"fmt"
	"strings"
)

// Scope represents Java member visibility.
type Scope string

const (
	// Public indicates public visibility.
	Public Scope = "public"
	// Protected indicates protected visibility.
	Protected Scope = "protected"
	// Private indicates private visibility.
	Private Scope = "private"
	// PackagePrivate indicates package-private (default) visibility.
	PackagePrivate Scope = ""
)

// Modifier represents Java modifiers.
type Modifier string

const (
	// Static indicates the static modifier.
	Static Modifier = "static"
	// Final indicates the final modifier.
	Final Modifier = "final"
	// Abstract indicates the abstract modifier.
	Abstract Modifier = "abstract"
	// Synchronized indicates the synchronized modifier.
	Synchronized Modifier = "synchronized"
	// Transient indicates the transient modifier.
	Transient Modifier = "transient"
	// Volatile indicates the volatile modifier.
	Volatile Modifier = "volatile"
	// Native indicates the native modifier.
	Native Modifier = "native"
	// Strictfp indicates the strictfp modifier.
	Strictfp Modifier = "strictfp"
)

// TypeKind defines the category of a TypeNode.
type TypeKind int

const (
	// KindPrimitive indicates a primitive type.
	KindPrimitive TypeKind = iota
	// KindObject indicates a reference or object type.
	KindObject
	// KindVoid indicates the void type.
	KindVoid
)

// TypeNode represents a Java type (primitive, class, interface, generic, array).
type TypeNode struct {
	Kind           TypeKind
	Name           string
	Package        string
	Generics       []*TypeNode
	IsArray        bool
	Wildcard       bool
	ExtendsBound   *TypeNode
	SuperBound     *TypeNode
	IsStaticImport bool
}

// Common primitive & object types.
var (
	TypeVoid    = &TypeNode{Kind: KindVoid, Name: "void"}
	TypeBoolean = &TypeNode{Kind: KindPrimitive, Name: "boolean"}
	TypeByte    = &TypeNode{Kind: KindPrimitive, Name: "byte"}
	TypeShort   = &TypeNode{Kind: KindPrimitive, Name: "short"}
	TypeInt     = &TypeNode{Kind: KindPrimitive, Name: "int"}
	TypeLong    = &TypeNode{Kind: KindPrimitive, Name: "long"}
	TypeFloat   = &TypeNode{Kind: KindPrimitive, Name: "float"}
	TypeDouble  = &TypeNode{Kind: KindPrimitive, Name: "double"}
	TypeChar    = &TypeNode{Kind: KindPrimitive, Name: "char"}

	TypeBoxedBoolean  = ObjectType("Boolean", "java.lang")
	TypeBoxedInteger  = ObjectType("Integer", "java.lang")
	TypeBoxedLong     = ObjectType("Long", "java.lang")
	TypeBoxedDouble   = ObjectType("Double", "java.lang")
	TypeBoxedFloat    = ObjectType("Float", "java.lang")
	TypeString        = ObjectType("String", "java.lang")
	TypeObject        = ObjectType("Object", "java.lang")
	TypeException     = ObjectType("Exception", "java.lang")
	TypeIOException   = ObjectType("IOException", "java.io")
	TypeAutoCloseable = ObjectType("AutoCloseable", "java.lang")
)

// ObjectType constructs an object type node.
func ObjectType(name, pkg string, generics ...*TypeNode) *TypeNode {
	return &TypeNode{
		Kind:     KindObject,
		Name:     name,
		Package:  pkg,
		Generics: generics,
	}
}

// ArrayType constructs an array type of the given element type.
func ArrayType(elem *TypeNode) *TypeNode {
	return &TypeNode{
		Kind:     elem.Kind,
		Name:     elem.Name,
		Package:  elem.Package,
		Generics: elem.Generics,
		IsArray:  true,
	}
}

// WildcardType constructs a ? wildcard type with optional upper bound.
func WildcardType(extendsBound *TypeNode) *TypeNode {
	return &TypeNode{
		Kind:         KindObject,
		Name:         "?",
		Wildcard:     true,
		ExtendsBound: extendsBound,
	}
}

// ListType returns a List<E> type.
func ListType(elem *TypeNode) *TypeNode {
	return ObjectType("List", "java.util", elem)
}

// MapType returns a Map<K, V> type.
func MapType(key, val *TypeNode) *TypeNode {
	return ObjectType("Map", "java.util", key, val)
}

// SetType returns a Set<E> type.
func SetType(elem *TypeNode) *TypeNode {
	return ObjectType("Set", "java.util", elem)
}

// FullName returns package.Name.
func (t *TypeNode) FullName() string {
	if t.Package == "" || t.Package == "java.lang" {
		return t.Name
	}
	return t.Package + "." + t.Name
}

// String renders the type name for code output.
func (t *TypeNode) String() string {
	if t == nil {
		return ""
	}
	if t.Wildcard {
		if t.ExtendsBound != nil {
			return "? extends " + t.ExtendsBound.String()
		}
		if t.SuperBound != nil {
			return "? super " + t.SuperBound.String()
		}
		return "?"
	}
	var sb strings.Builder
	sb.WriteString(t.Name)
	if len(t.Generics) > 0 {
		sb.WriteString("<")
		for i, g := range t.Generics {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(g.String())
		}
		sb.WriteString(">")
	}
	if t.IsArray {
		sb.WriteString("[]")
	}
	return sb.String()
}

// ASTNode is the base interface for all Java AST nodes.
type ASTNode interface {
	isASTNode()
}

// Expr is the interface for expression nodes.
type Expr interface {
	ASTNode
	isExpr()
}

// Statement is the interface for statement nodes.
type Statement interface {
	ASTNode
	isStatement()
}

// AnnotationNode represents a Java annotation e.g. @Override or @Generated("...").
type AnnotationNode struct {
	Type       *TypeNode
	Attributes map[string]Expr
	Value      Expr
}

func (a *AnnotationNode) isASTNode() {}

// JavaDocComment represents a structured Javadoc block.
type JavaDocComment struct {
	Paragraphs []string
	Params     []ParamDoc
	Returns    string
	Throws     []ThrowsDoc
	Deprecated string
	SeeAlso    []string
}

// ParamDoc represents a @param tag in Javadoc.
type ParamDoc struct {
	Name        string
	Description string
}

// ThrowsDoc represents a @throws or @exception tag in Javadoc.
type ThrowsDoc struct {
	Exception   string
	Description string
}

func (j *JavaDocComment) isASTNode() {}

// LineComment represents a // line comment.
type LineComment struct {
	Comment string
}

func (l *LineComment) isASTNode()   {}
func (l *LineComment) isStatement() {}

// BlockComment represents a /* ... */ comment.
type BlockComment struct {
	Comment string
}

func (b *BlockComment) isASTNode()   {}
func (b *BlockComment) isStatement() {}

// Variable represents a variable declaration or reference.
type Variable struct {
	Name string
	Type *TypeNode
}

// VariableExpr represents variable access or declaration.
type VariableExpr struct {
	Variable *Variable
	Scope    Scope
	IsDecl   bool
	IsFinal  bool
	IsStatic bool
	InitExpr Expr
}

func (v *VariableExpr) isASTNode()   {}
func (v *VariableExpr) isExpr()      {}
func (v *VariableExpr) isStatement() {}

// ValueExpr represents a literal value (number, boolean, string, null, this, super).
type ValueExpr struct {
	ValueType *TypeNode
	Literal   string
	IsString  bool
	IsNull    bool
	IsThis    bool
	IsSuper   bool
}

func (v *ValueExpr) isASTNode() {}
func (v *ValueExpr) isExpr()    {}

// IntVal creates an integer literal ValueExpr.
func IntVal(val int) *ValueExpr {
	return &ValueExpr{ValueType: TypeInt, Literal: fmt.Sprintf("%d", val)}
}

// LongVal creates a long integer literal ValueExpr.
func LongVal(val int64) *ValueExpr {
	return &ValueExpr{ValueType: TypeLong, Literal: fmt.Sprintf("%dL", val)}
}

// DoubleVal creates a double floating-point literal ValueExpr.
func DoubleVal(val float64) *ValueExpr {
	return &ValueExpr{ValueType: TypeDouble, Literal: fmt.Sprintf("%f", val)}
}

// BoolVal creates a boolean literal ValueExpr.
func BoolVal(val bool) *ValueExpr {
	if val {
		return &ValueExpr{ValueType: TypeBoolean, Literal: "true"}
	}
	return &ValueExpr{ValueType: TypeBoolean, Literal: "false"}
}

// StringVal creates a string literal ValueExpr.
func StringVal(val string) *ValueExpr {
	return &ValueExpr{ValueType: TypeString, Literal: val, IsString: true}
}

// NullVal creates a null literal ValueExpr.
func NullVal() *ValueExpr {
	return &ValueExpr{ValueType: TypeObject, Literal: "null", IsNull: true}
}

// ThisVal creates a 'this' reference ValueExpr.
func ThisVal() *ValueExpr {
	return &ValueExpr{Literal: "this", IsThis: true}
}

// SuperVal creates a 'super' reference ValueExpr.
func SuperVal() *ValueExpr {
	return &ValueExpr{Literal: "super", IsSuper: true}
}

// MethodInvocationExpr represents method calls e.g. obj.method(arg1, arg2).
type MethodInvocationExpr struct {
	TargetExpr Expr
	TargetType *TypeNode
	MethodName string
	Arguments  []Expr
	Generics   []*TypeNode
	ReturnType *TypeNode
	IsStatic   bool
}

func (m *MethodInvocationExpr) isASTNode()   {}
func (m *MethodInvocationExpr) isExpr()      {}
func (m *MethodInvocationExpr) isStatement() {}

// NewObjectExpr represents object instantiation: new MyClass(arg1, arg2).
type NewObjectExpr struct {
	Type      *TypeNode
	Arguments []Expr
	Generics  []*TypeNode
}

func (n *NewObjectExpr) isASTNode()   {}
func (n *NewObjectExpr) isExpr()      {}
func (n *NewObjectExpr) isStatement() {}

// AssignmentExpr represents assignment: target = value.
type AssignmentExpr struct {
	Variable *VariableExpr
	Value    Expr
}

func (a *AssignmentExpr) isASTNode()   {}
func (a *AssignmentExpr) isExpr()      {}
func (a *AssignmentExpr) isStatement() {}

// AssignmentOperationExpr represents compound assignment: target += value.
type AssignmentOperationExpr struct {
	Left     Expr
	Right    Expr
	Operator string // "+=", "-=", "*=", "/=", "%="
}

func (a *AssignmentOperationExpr) isASTNode()   {}
func (a *AssignmentOperationExpr) isExpr()      {}
func (a *AssignmentOperationExpr) isStatement() {}

// BinaryOperationExpr represents binary operations (arithmetic, relational, logical).
type BinaryOperationExpr struct {
	Left     Expr
	Right    Expr
	Operator string
}

func (b *BinaryOperationExpr) isASTNode() {}
func (b *BinaryOperationExpr) isExpr()    {}

// UnaryOperationExpr represents unary operations (!x, -x, ++x, x++).
type UnaryOperationExpr struct {
	Expr     Expr
	Operator string
	Postfix  bool
}

func (u *UnaryOperationExpr) isASTNode()   {}
func (u *UnaryOperationExpr) isExpr()      {}
func (u *UnaryOperationExpr) isStatement() {}

// TernaryExpr represents condition ? then : else.
type TernaryExpr struct {
	Condition Expr
	ThenExpr  Expr
	ElseExpr  Expr
}

func (t *TernaryExpr) isASTNode() {}
func (t *TernaryExpr) isExpr()    {}

// CastExpr represents (Type) expr.
type CastExpr struct {
	Type *TypeNode
	Expr Expr
}

func (c *CastExpr) isASTNode() {}
func (c *CastExpr) isExpr()    {}

// InstanceofExpr represents expr instanceof Type.
type InstanceofExpr struct {
	Expr      Expr
	CheckType *TypeNode
}

func (i *InstanceofExpr) isASTNode() {}
func (i *InstanceofExpr) isExpr()    {}

// LambdaExpr represents (a, b) -> body.
type LambdaExpr struct {
	Arguments  []*Variable
	BodyExpr   Expr
	Statements []Statement
}

func (l *LambdaExpr) isASTNode() {}
func (l *LambdaExpr) isExpr()    {}

// ReferenceConstructorExpr represents MyClass::new or obj::method.
type ReferenceConstructorExpr struct {
	Type       *TypeNode
	MethodName string // "new" or method name
}

func (r *ReferenceConstructorExpr) isASTNode() {}
func (r *ReferenceConstructorExpr) isExpr()    {}

// EnumRefExpr represents EnumType.VALUE.
type EnumRefExpr struct {
	Type  *TypeNode
	Value string
}

func (e *EnumRefExpr) isASTNode() {}
func (e *EnumRefExpr) isExpr()    {}

// ArrayExpr represents new Type[]{elem1, elem2} or new Type[size].
type ArrayExpr struct {
	Type     *TypeNode
	Elements []Expr
	Size     Expr
}

func (a *ArrayExpr) isASTNode() {}
func (a *ArrayExpr) isExpr()    {}

// Statements

// ExprStatement wraps an expression as a statement.
type ExprStatement struct {
	Expr Expr
}

func (e *ExprStatement) isASTNode()   {}
func (e *ExprStatement) isStatement() {}

// BlockStatement represents a list of statements inside braces.
type BlockStatement struct {
	Statements []Statement
}

func (b *BlockStatement) isASTNode()   {}
func (b *BlockStatement) isStatement() {}

// IfStatement represents an if-elseif-else construct.
type IfStatement struct {
	Condition      Expr
	ThenStatements []Statement
	ElseIfs        []*ElseIfBlock
	ElseStatements []Statement
}

// ElseIfBlock represents an else-if conditional branch in an IfStatement.
type ElseIfBlock struct {
	Condition  Expr
	Statements []Statement
}

func (i *IfStatement) isASTNode()   {}
func (i *IfStatement) isStatement() {}

// WhileStatement represents while (condition) { body }.
type WhileStatement struct {
	Condition  Expr
	Statements []Statement
}

func (w *WhileStatement) isASTNode()   {}
func (w *WhileStatement) isStatement() {}

// ForStatement represents for (init; condition; update) { body }.
type ForStatement struct {
	Init       Statement
	Condition  Expr
	Update     Statement
	Statements []Statement
}

func (f *ForStatement) isASTNode()   {}
func (f *ForStatement) isStatement() {}

// GeneralForStatement represents for (Type item : collection) { body }.
type GeneralForStatement struct {
	ItemVar    *Variable
	Collection Expr
	Statements []Statement
}

func (g *GeneralForStatement) isASTNode()   {}
func (g *GeneralForStatement) isStatement() {}

// TryCatchStatement represents try (resources) { ... } catch (E e) { ... } finally { ... }.
type TryCatchStatement struct {
	Resources   []*VariableExpr
	TryBody     []Statement
	CatchBlocks []*CatchBlock
	FinallyBody []Statement
}

// CatchBlock represents a catch block handling one or more exception types.
type CatchBlock struct {
	Exceptions []*TypeNode
	VarName    string
	Body       []Statement
}

func (t *TryCatchStatement) isASTNode()   {}
func (t *TryCatchStatement) isStatement() {}

// SynchronizedStatement represents synchronized (lock) { body }.
type SynchronizedStatement struct {
	LockExpr   Expr
	Statements []Statement
}

func (s *SynchronizedStatement) isASTNode()   {}
func (s *SynchronizedStatement) isStatement() {}

// ReturnExpr represents return expr;.
type ReturnExpr struct {
	Expr Expr
}

func (r *ReturnExpr) isASTNode()   {}
func (r *ReturnExpr) isExpr()      {}
func (r *ReturnExpr) isStatement() {}

// ThrowExpr represents throw expr;.
type ThrowExpr struct {
	Expr Expr
}

func (t *ThrowExpr) isASTNode()   {}
func (t *ThrowExpr) isExpr()      {}
func (t *ThrowExpr) isStatement() {}

// MethodDefinition represents a method or constructor in a class/interface.
type MethodDefinition struct {
	Scope            Scope
	Modifiers        []Modifier
	ReturnType       *TypeNode
	Name             string
	Arguments        []*VariableExpr
	ThrowsExceptions []*TypeNode
	Statements       []Statement
	Annotations      []*AnnotationNode
	JavaDoc          *JavaDocComment
	IsConstructor    bool
	IsAbstract       bool
	IsOverride       bool
	TemplateGenerics []*TypeNode
}

func (m *MethodDefinition) isASTNode() {}

// ClassDefinition represents a top-level or inner Java class/interface/enum.
type ClassDefinition struct {
	PackageName     string
	Imports         []string
	StaticImports   []string
	Scope           Scope
	Modifiers       []Modifier
	Name            string
	ExtendsType     *TypeNode
	ImplementsTypes []*TypeNode
	JavaDoc         *JavaDocComment
	Annotations     []*AnnotationNode
	Fields          []*VariableExpr
	Methods         []*MethodDefinition
	InnerClasses    []*ClassDefinition
	IsInterface     bool
	IsEnum          bool
	EnumValues      []*EnumValueDefinition
}

// EnumValueDefinition represents an enum constant in an enum class.
type EnumValueDefinition struct {
	Name      string
	Arguments []Expr
	JavaDoc   *JavaDocComment
}

func (c *ClassDefinition) isASTNode() {}
