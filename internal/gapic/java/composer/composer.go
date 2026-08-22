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

// Package composer constructs Java AST definitions for GAPIC client libraries.
package composer

import (
	"fmt"

	"github.com/googleapis/librarian/internal/gapic/java/engine/ast"
	"github.com/googleapis/librarian/internal/gapic/java/model"
	"github.com/iancoleman/strcase"
)

// ComposeServiceClasses generates all Java class ASTs for a GapicContext.
func ComposeServiceClasses(ctx *model.GapicContext) []*model.GapicClass {
	var classes []*model.GapicClass

	for _, svc := range ctx.Services {
		// 1. Service Client
		clientClass := ComposeClientClass(svc, ctx)
		classes = append(classes, &model.GapicClass{
			Kind:            model.KindMain,
			ClassDefinition: clientClass,
		})

		// 2. Service Settings
		settingsClass := ComposeSettingsClass(svc, ctx)
		classes = append(classes, &model.GapicClass{
			Kind:            model.KindMain,
			ClassDefinition: settingsClass,
		})

		// 3. Service Stub
		stubClass := ComposeServiceStubClass(svc, ctx)
		classes = append(classes, &model.GapicClass{
			Kind:            model.KindStub,
			ClassDefinition: stubClass,
		})

		// 4. Service Stub Settings
		stubSettingsClass := ComposeServiceStubSettingsClass(svc, ctx)
		classes = append(classes, &model.GapicClass{
			Kind:            model.KindStub,
			ClassDefinition: stubSettingsClass,
		})

		// 5. Grpc Service Stub (if gRPC enabled)
		if ctx.Transport == model.TransportGRPC || ctx.Transport == model.TransportGRPCRest {
			grpcStub := ComposeGrpcServiceStubClass(svc, ctx)
			classes = append(classes, &model.GapicClass{
				Kind:            model.KindStub,
				ClassDefinition: grpcStub,
			})
			grpcFactory := ComposeGrpcCallableFactoryClass(svc, ctx)
			classes = append(classes, &model.GapicClass{
				Kind:            model.KindStub,
				ClassDefinition: grpcFactory,
			})
		}

		// 6. HttpJson Service Stub (if REST enabled)
		if ctx.Transport == model.TransportREST || ctx.Transport == model.TransportGRPCRest {
			httpJsonStub := ComposeHttpJsonServiceStubClass(svc, ctx)
			classes = append(classes, &model.GapicClass{
				Kind:            model.KindStub,
				ClassDefinition: httpJsonStub,
			})
			httpJsonFactory := ComposeHttpJsonCallableFactoryClass(svc, ctx)
			classes = append(classes, &model.GapicClass{
				Kind:            model.KindStub,
				ClassDefinition: httpJsonFactory,
			})
		}
	}

	// 7. Resource Name helper classes
	for _, res := range ctx.HelperResourceNames {
		resClass := ComposeResourceNameHelperClass(res)
		classes = append(classes, &model.GapicClass{
			Kind:            model.KindMain,
			ClassDefinition: resClass,
		})
	}

	// 8. Version.java if flag is enabled
	if ctx.HasGenerateVersionJava && len(ctx.Services) > 0 {
		versionClass := ComposeLibraryVersionClass(ctx.Services[0].PackageName)
		classes = append(classes, &model.GapicClass{
			Kind:            model.KindMain,
			ClassDefinition: versionClass,
		})
	}

	return classes
}

// ComposeClientClass generates [Service]Client.java.
func ComposeClientClass(svc *model.Service, ctx *model.GapicContext) *ast.ClassDefinition {
	clientName := svc.Name + "Client"
	settingsType := ast.ObjectType(svc.Name+"Settings", svc.PackageName)
	stubType := ast.ObjectType(svc.Name+"Stub", svc.PackageName+".stub")

	c := &ast.ClassDefinition{
		PackageName: svc.PackageName,
		Scope:       ast.Public,
		Name:        clientName,
		ImplementsTypes: []*ast.TypeNode{
			ast.ObjectType("BackgroundResource", "com.google.api.gax.core"),
		},
		JavaDoc: &ast.JavaDocComment{
			Paragraphs: []string{
				fmt.Sprintf("Service Description: %s", svc.Name),
				fmt.Sprintf("This class provides the client business logic for %s.", svc.Name),
			},
		},
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
		Fields: []*ast.VariableExpr{
			{
				Scope:   ast.Private,
				IsFinal: true,
				Variable: &ast.Variable{
					Name: "settings",
					Type: settingsType,
				},
			},
			{
				Scope:   ast.Private,
				IsFinal: true,
				Variable: &ast.Variable{
					Name: "stub",
					Type: stubType,
				},
			},
		},
	}

	// create() factory methods
	c.Methods = append(c.Methods,
		&ast.MethodDefinition{
			Scope:            ast.Public,
			Modifiers:        []ast.Modifier{ast.Static, ast.Final},
			ReturnType:       ast.ObjectType(clientName, svc.PackageName),
			Name:             "create",
			ThrowsExceptions: []*ast.TypeNode{ast.TypeIOException},
			Statements: []ast.Statement{
				&ast.ReturnExpr{
					Expr: &ast.MethodInvocationExpr{
						TargetType: ast.ObjectType(clientName, svc.PackageName),
						MethodName: "create",
						Arguments: []ast.Expr{
							&ast.MethodInvocationExpr{
								TargetType: settingsType,
								MethodName: "newBuilder",
							},
						},
					},
				},
			},
		},
		&ast.MethodDefinition{
			Scope:      ast.Public,
			Modifiers:  []ast.Modifier{ast.Static, ast.Final},
			ReturnType: ast.ObjectType(clientName, svc.PackageName),
			Name:       "create",
			Arguments: []*ast.VariableExpr{
				{Variable: &ast.Variable{Name: "settings", Type: settingsType}},
			},
			ThrowsExceptions: []*ast.TypeNode{ast.TypeIOException},
			Statements: []ast.Statement{
				&ast.ReturnExpr{
					Expr: &ast.NewObjectExpr{
						Type: ast.ObjectType(clientName, svc.PackageName),
						Arguments: []ast.Expr{
							&ast.VariableExpr{Variable: &ast.Variable{Name: "settings"}},
						},
					},
				},
			},
		},
		&ast.MethodDefinition{
			Scope:      ast.Public,
			Modifiers:  []ast.Modifier{ast.Static, ast.Final},
			ReturnType: ast.ObjectType(clientName, svc.PackageName),
			Name:       "create",
			Arguments: []*ast.VariableExpr{
				{Variable: &ast.Variable{Name: "stub", Type: stubType}},
			},
			Statements: []ast.Statement{
				&ast.ReturnExpr{
					Expr: &ast.NewObjectExpr{
						Type: ast.ObjectType(clientName, svc.PackageName),
						Arguments: []ast.Expr{
							&ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
						},
					},
				},
			},
		},
	)

	// Protected Constructors
	c.Methods = append(c.Methods,
		&ast.MethodDefinition{
			Scope:         ast.Protected,
			IsConstructor: true,
			Name:          clientName,
			Arguments: []*ast.VariableExpr{
				{Variable: &ast.Variable{Name: "settings", Type: settingsType}},
			},
			ThrowsExceptions: []*ast.TypeNode{ast.TypeIOException},
			Statements: []ast.Statement{
				&ast.AssignmentExpr{
					Variable: &ast.VariableExpr{Variable: &ast.Variable{Name: "this.settings"}},
					Value:    &ast.VariableExpr{Variable: &ast.Variable{Name: "settings"}},
				},
				&ast.AssignmentExpr{
					Variable: &ast.VariableExpr{Variable: &ast.Variable{Name: "this.stub"}},
					Value: &ast.MethodInvocationExpr{
						TargetExpr: &ast.MethodInvocationExpr{
							TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "settings"}},
							MethodName: "getStubSettings",
						},
						MethodName: "createStub",
					},
				},
			},
		},
		&ast.MethodDefinition{
			Scope:         ast.Protected,
			IsConstructor: true,
			Name:          clientName,
			Arguments: []*ast.VariableExpr{
				{Variable: &ast.Variable{Name: "stub", Type: stubType}},
			},
			Statements: []ast.Statement{
				&ast.AssignmentExpr{
					Variable: &ast.VariableExpr{Variable: &ast.Variable{Name: "this.settings"}},
					Value:    ast.NullVal(),
				},
				&ast.AssignmentExpr{
					Variable: &ast.VariableExpr{Variable: &ast.Variable{Name: "this.stub"}},
					Value:    &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
				},
			},
		},
	)

	// Settings & Stub getters
	c.Methods = append(c.Methods,
		&ast.MethodDefinition{
			Scope:      ast.Public,
			ReturnType: settingsType,
			Name:       "getSettings",
			Statements: []ast.Statement{
				&ast.ReturnExpr{Expr: &ast.VariableExpr{Variable: &ast.Variable{Name: "settings"}}},
			},
		},
		&ast.MethodDefinition{
			Scope:      ast.Public,
			ReturnType: stubType,
			Name:       "getStub",
			Statements: []ast.Statement{
				&ast.ReturnExpr{Expr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}}},
			},
		},
	)

	// Service RPC methods
	for _, m := range svc.Methods {
		methodName := strcase.ToLowerCamel(m.Name)

		// 1. Direct Request method: public final Response method(Request request)
		c.Methods = append(c.Methods, &ast.MethodDefinition{
			Scope:      ast.Public,
			Modifiers:  []ast.Modifier{ast.Final},
			ReturnType: m.OutputType,
			Name:       methodName,
			Arguments: []*ast.VariableExpr{
				{Variable: &ast.Variable{Name: "request", Type: m.InputType}},
			},
			Statements: []ast.Statement{
				&ast.ReturnExpr{
					Expr: &ast.MethodInvocationExpr{
						TargetExpr: &ast.MethodInvocationExpr{
							TargetExpr: &ast.MethodInvocationExpr{
								TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
								MethodName: methodName + "Callable",
							},
							MethodName: "call",
							Arguments: []ast.Expr{
								&ast.VariableExpr{Variable: &ast.Variable{Name: "request"}},
							},
						},
						MethodName: "",
					},
				},
			},
		})

		// 2. Signature overloads if present
		for _, sig := range m.MethodSignatures {
			if len(sig) == 0 {
				continue
			}
			var sigArgs []*ast.VariableExpr
			for _, fieldName := range sig {
				argType := ast.TypeString
				inMsg := ctx.FindMessage(m.InputType.Name)
				if inMsg != nil && inMsg.Fields[fieldName] != nil {
					argType = inMsg.Fields[fieldName].Type
				}
				sigArgs = append(sigArgs, &ast.VariableExpr{
					Variable: &ast.Variable{
						Name: strcase.ToLowerCamel(fieldName),
						Type: argType,
					},
				})
			}

			var builderExpr ast.Expr = &ast.MethodInvocationExpr{
				TargetType: m.InputType,
				MethodName: "newBuilder",
			}
			for _, fieldName := range sig {
				builderExpr = &ast.MethodInvocationExpr{
					TargetExpr: builderExpr,
					MethodName: "set" + strcase.ToCamel(fieldName),
					Arguments: []ast.Expr{
						&ast.VariableExpr{
							Variable: &ast.Variable{
								Name: strcase.ToLowerCamel(fieldName),
								Type: nil,
							},
						},
					},
				}
			}
			initExpr := &ast.MethodInvocationExpr{
				TargetExpr: builderExpr,
				MethodName: "build",
			}

			c.Methods = append(c.Methods, &ast.MethodDefinition{
				Scope:      ast.Public,
				Modifiers:  []ast.Modifier{ast.Final},
				ReturnType: m.OutputType,
				Name:       methodName,
				Arguments:  sigArgs,
				Statements: []ast.Statement{
					&ast.VariableExpr{
						IsDecl: true,
						Variable: &ast.Variable{
							Name: "request",
							Type: m.InputType,
						},
						InitExpr: initExpr,
					},
					&ast.ReturnExpr{
						Expr: &ast.MethodInvocationExpr{
							TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "this"}},
							MethodName: methodName,
							Arguments: []ast.Expr{
								&ast.VariableExpr{Variable: &ast.Variable{Name: "request"}},
							},
						},
					},
				},
			})
		}

		// 3. Callable getter: public UnaryCallable<Request, Response> methodCallable()
		callableType := ast.ObjectType("UnaryCallable", "com.google.api.gax.rpc", m.InputType, m.OutputType)
		c.Methods = append(c.Methods, &ast.MethodDefinition{
			Scope:      ast.Public,
			ReturnType: callableType,
			Name:       methodName + "Callable",
			Statements: []ast.Statement{
				&ast.ReturnExpr{
					Expr: &ast.MethodInvocationExpr{
						TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
						MethodName: methodName + "Callable",
					},
				},
			},
		})
	}

	// BackgroundResource methods
	c.Methods = append(c.Methods,
		&ast.MethodDefinition{
			Scope:      ast.Public,
			IsOverride: true,
			ReturnType: ast.TypeVoid,
			Name:       "close",
			Statements: []ast.Statement{
				&ast.ExprStatement{
					Expr: &ast.MethodInvocationExpr{
						TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
						MethodName: "close",
					},
				},
			},
		},
		&ast.MethodDefinition{
			Scope:      ast.Public,
			IsOverride: true,
			ReturnType: ast.TypeVoid,
			Name:       "shutdown",
			Statements: []ast.Statement{
				&ast.ExprStatement{
					Expr: &ast.MethodInvocationExpr{
						TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
						MethodName: "shutdown",
					},
				},
			},
		},
		&ast.MethodDefinition{
			Scope:      ast.Public,
			IsOverride: true,
			ReturnType: ast.TypeBoolean,
			Name:       "isShutdown",
			Statements: []ast.Statement{
				&ast.ReturnExpr{
					Expr: &ast.MethodInvocationExpr{
						TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
						MethodName: "isShutdown",
					},
				},
			},
		},
		&ast.MethodDefinition{
			Scope:      ast.Public,
			IsOverride: true,
			ReturnType: ast.TypeBoolean,
			Name:       "isTerminated",
			Statements: []ast.Statement{
				&ast.ReturnExpr{
					Expr: &ast.MethodInvocationExpr{
						TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
						MethodName: "isTerminated",
					},
				},
			},
		},
		&ast.MethodDefinition{
			Scope:      ast.Public,
			IsOverride: true,
			ReturnType: ast.TypeVoid,
			Name:       "shutdownNow",
			Statements: []ast.Statement{
				&ast.ExprStatement{
					Expr: &ast.MethodInvocationExpr{
						TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "stub"}},
						MethodName: "shutdownNow",
					},
				},
			},
		},
	)

	return c
}

// ComposeSettingsClass generates [Service]Settings.java.
func ComposeSettingsClass(svc *model.Service, ctx *model.GapicContext) *ast.ClassDefinition {
	settingsName := svc.Name + "Settings"

	c := &ast.ClassDefinition{
		PackageName: svc.PackageName,
		Scope:       ast.Public,
		Name:        settingsName,
		ExtendsType: ast.ObjectType("ClientSettings", "com.google.api.gax.rpc", ast.ObjectType(settingsName, svc.PackageName)),
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}

	// Methods for settings
	for _, m := range svc.Methods {
		methodName := strcase.ToLowerCamel(m.Name)
		callSettingsType := ast.ObjectType("UnaryCallSettings", "com.google.api.gax.rpc", m.InputType, m.OutputType)
		c.Methods = append(c.Methods, &ast.MethodDefinition{
			Scope:      ast.Public,
			ReturnType: callSettingsType,
			Name:       methodName + "Settings",
			Statements: []ast.Statement{
				&ast.ReturnExpr{
					Expr: &ast.MethodInvocationExpr{
						TargetExpr: &ast.MethodInvocationExpr{
							TargetExpr: &ast.MethodInvocationExpr{
								TargetExpr: &ast.VariableExpr{Variable: &ast.Variable{Name: "this"}},
								MethodName: "getStubSettings",
							},
							MethodName: "getStubSettings",
						},
						MethodName: methodName + "Settings",
					},
				},
			},
		})
	}

	return c
}

// ComposeServiceStubClass generates stub/[Service]Stub.java.
func ComposeServiceStubClass(svc *model.Service, ctx *model.GapicContext) *ast.ClassDefinition {
	stubName := svc.Name + "Stub"

	c := &ast.ClassDefinition{
		PackageName: svc.PackageName + ".stub",
		Scope:       ast.Public,
		Modifiers:   []ast.Modifier{ast.Abstract},
		Name:        stubName,
		ImplementsTypes: []*ast.TypeNode{
			ast.ObjectType("BackgroundResource", "com.google.api.gax.core"),
		},
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}

	for _, m := range svc.Methods {
		methodName := strcase.ToLowerCamel(m.Name)
		callType := ast.ObjectType("UnaryCallable", "com.google.api.gax.rpc", m.InputType, m.OutputType)
		c.Methods = append(c.Methods, &ast.MethodDefinition{
			Scope:      ast.Public,
			ReturnType: callType,
			Name:       methodName + "Callable",
			Statements: []ast.Statement{
				&ast.ThrowExpr{
					Expr: &ast.NewObjectExpr{
						Type: ast.ObjectType("UnsupportedOperationException", "java.lang"),
						Arguments: []ast.Expr{
							ast.StringVal("Not implemented: " + methodName + "Callable()"),
						},
					},
				},
			},
		})
	}

	c.Methods = append(c.Methods, &ast.MethodDefinition{
		Scope:      ast.Public,
		IsAbstract: true,
		IsOverride: true,
		ReturnType: ast.TypeVoid,
		Name:       "close",
	})

	return c
}

// ComposeServiceStubSettingsClass generates stub/[Service]StubSettings.java.
func ComposeServiceStubSettingsClass(svc *model.Service, ctx *model.GapicContext) *ast.ClassDefinition {
	stubSettingsName := svc.Name + "StubSettings"

	c := &ast.ClassDefinition{
		PackageName: svc.PackageName + ".stub",
		Scope:       ast.Public,
		Name:        stubSettingsName,
		ExtendsType: ast.ObjectType("StubSettings", "com.google.api.gax.rpc", ast.ObjectType(stubSettingsName, svc.PackageName+".stub")),
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}

	// Host and Port default constants
	c.Fields = append(c.Fields,
		&ast.VariableExpr{
			Scope:    ast.Private,
			IsStatic: true,
			IsFinal:  true,
			Variable: &ast.Variable{
				Name: "DEFAULT_SERVICE_ENDPOINT",
				Type: ast.TypeString,
			},
			InitExpr: ast.StringVal(svc.HostName + ":" + svc.DefaultPort),
		},
	)

	return c
}

// ComposeGrpcServiceStubClass generates stub/Grpc[Service]Stub.java.
func ComposeGrpcServiceStubClass(svc *model.Service, ctx *model.GapicContext) *ast.ClassDefinition {
	grpcStubName := "Grpc" + svc.Name + "Stub"
	stubType := ast.ObjectType(svc.Name+"Stub", svc.PackageName+".stub")

	c := &ast.ClassDefinition{
		PackageName: svc.PackageName + ".stub",
		Scope:       ast.Public,
		Name:        grpcStubName,
		ExtendsType: stubType,
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}

	return c
}

// ComposeHttpJsonServiceStubClass generates stub/HttpJson[Service]Stub.java.
func ComposeHttpJsonServiceStubClass(svc *model.Service, ctx *model.GapicContext) *ast.ClassDefinition {
	httpStubName := "HttpJson" + svc.Name + "Stub"
	stubType := ast.ObjectType(svc.Name+"Stub", svc.PackageName+".stub")

	c := &ast.ClassDefinition{
		PackageName: svc.PackageName + ".stub",
		Scope:       ast.Public,
		Name:        httpStubName,
		ExtendsType: stubType,
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}

	return c
}

// ComposeGrpcCallableFactoryClass generates stub/Grpc[Service]CallableFactory.java.
func ComposeGrpcCallableFactoryClass(svc *model.Service, ctx *model.GapicContext) *ast.ClassDefinition {
	factoryName := "Grpc" + svc.Name + "CallableFactory"

	return &ast.ClassDefinition{
		PackageName: svc.PackageName + ".stub",
		Scope:       ast.Public,
		Name:        factoryName,
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}
}

// ComposeHttpJsonCallableFactoryClass generates stub/HttpJson[Service]CallableFactory.java.
func ComposeHttpJsonCallableFactoryClass(svc *model.Service, ctx *model.GapicContext) *ast.ClassDefinition {
	factoryName := "HttpJson" + svc.Name + "CallableFactory"

	return &ast.ClassDefinition{
		PackageName: svc.PackageName + ".stub",
		Scope:       ast.Public,
		Name:        factoryName,
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}
}

// ComposeResourceNameHelperClass generates [Resource]Name.java.
func ComposeResourceNameHelperClass(res *model.ResourceName) *ast.ClassDefinition {
	c := &ast.ClassDefinition{
		PackageName: res.PackageName,
		Scope:       ast.Public,
		Name:        res.ClassName,
		ImplementsTypes: []*ast.TypeNode{
			ast.ObjectType("ResourceName", "com.google.api.resourcenames"),
		},
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}

	if len(res.Patterns) > 0 {
		c.Fields = append(c.Fields, &ast.VariableExpr{
			Scope:    ast.Private,
			IsStatic: true,
			IsFinal:  true,
			Variable: &ast.Variable{
				Name: "PATH_TEMPLATE",
				Type: ast.ObjectType("PathTemplate", "com.google.api.pathtemplate"),
			},
			InitExpr: &ast.MethodInvocationExpr{
				TargetType: ast.ObjectType("PathTemplate", "com.google.api.pathtemplate"),
				MethodName: "createWithoutUrlEncoding",
				Arguments: []ast.Expr{
					ast.StringVal(res.Patterns[0]),
				},
			},
		})
	}

	return c
}

// ComposeLibraryVersionClass generates Version.java.
func ComposeLibraryVersionClass(pkg string) *ast.ClassDefinition {
	return &ast.ClassDefinition{
		PackageName: pkg,
		Scope:       ast.Public,
		Modifiers:   []ast.Modifier{ast.Final},
		Name:        "Version",
		JavaDoc: &ast.JavaDocComment{
			Paragraphs: []string{"Internal class for library versioning."},
		},
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
		Fields: []*ast.VariableExpr{
			{
				Scope:    ast.Public,
				IsStatic: true,
				IsFinal:  true,
				Variable: &ast.Variable{
					Name: "VERSION",
					Type: ast.TypeString,
				},
				InitExpr: ast.StringVal("1.0.0"),
			},
		},
	}
}

// ComposePackageInfo generates GapicPackageInfo for package-info.java.
func ComposePackageInfo(ctx *model.GapicContext) *model.GapicPackageInfo {
	if len(ctx.Services) == 0 {
		return nil
	}
	return &model.GapicPackageInfo{
		PackageName: ctx.Services[0].PackageName,
		Description: fmt.Sprintf("A client to %s", ctx.Services[0].Name),
		Annotations: []*ast.AnnotationNode{
			generatedAnnotation(),
		},
	}
}

// ComposeNativeReflectConfig generates GraalVM reflect-config.json entries.
func ComposeNativeReflectConfig(ctx *model.GapicContext) []*model.ReflectConfig {
	var configs []*model.ReflectConfig
	for _, svc := range ctx.Services {
		configs = append(configs, &model.ReflectConfig{
			Name: svc.PackageName + "." + svc.Name + "Client",
		})
		configs = append(configs, &model.ReflectConfig{
			Name: svc.PackageName + "." + svc.Name + "Settings",
		})
	}
	return configs
}

func generatedAnnotation() *ast.AnnotationNode {
	return &ast.AnnotationNode{
		Type:  ast.ObjectType("Generated", "javax.annotation"),
		Value: ast.StringVal("by gapic-generator-java"),
	}
}
