# Multi-Language GAPIC Generator Framework Design for Librarian

## 1. Overview and Executive Summary

Librarian manages the lifecycle, generation, onboarding, and release of Google Cloud client libraries across multiple programming languages. Client generation is historically performed by language-specific GAPIC generators, each maintaining its own repository, dependencies, runtime environments, and plugin invocation semantics.

This document presents:
1. An architectural analysis and comparison of GAPIC generators across major languages (`Go`, `PHP`, `Ruby`, `TypeScript`, `Python`, and `Java`).
2. The design of Librarian's unified multi-language GAPIC Generator invocation framework.
3. The design and implementation of the Java GAPIC Generator natively ported to Go as part of Librarian.

---

## 2. Analysis of Existing Language GAPIC Generators

Protobuf compiler (`protoc`) code generator plugins communicate via standard input/output with serialized protocol buffer messages (`google.protobuf.compiler.CodeGeneratorRequest` $\rightarrow$ `CodeGeneratorResponse`). However, each language ecosystem has adopted distinct architectural choices:

| Language | Repository / Package | Implementation Language | Execution Model | Staging & Post-Processing |
| :--- | :--- | :--- | :--- | :--- |
| **Go** | `gapic-generator-go` | Go | `protoc-gen-go_gapic` binary | Direct output to destination package; `go mod tidy`, version generation, repo metadata |
| **PHP** | `gapic-generator-php` | PHP | `protoc-gen-gapic` (wrapper script) | Staged to `owl-bot-staging/<component>/<subdir>`, unzipped, formatted via php-cs-fixer |
| **Ruby** | `gapic-generator-ruby` | Ruby | `protoc-gen-ruby_cloud` (gem binary) | Staged to temp directory, merged with `keep` filter, `common_resources` proto cleanup |
| **TypeScript** | `gapic-generator-typescript` | TypeScript | CLI / `compileProtos` & `gapic-node-processing` | Staged to `owl-bot-staging/<library>/<index_slug>`, postprocessed via templates & prettier |
| **Python** | `google-cloud-python/packages/gapic-generator` | Python | `protoc-gen-python_gapic` | Staged to `owl-bot-staging`, postprocessed via synthtool (`owlbot_main`), string replacements |
| **Java** (Legacy) | `google-cloud-java/sdk-platform-java/gapic-generator-java` | Java | `protoc-gen-java_gapic` (JAR wrapper) | Generated to `srcjar`/dir, postprocessed via Maven POM generation, google-java-format |
| **Java** (Librarian Native) | `librarian/internal/gapic/java` | Go | In-process or `protoc-gen-java_gapic` binary | Native Go AST composer, high performance, zero JVM dependency overhead |

### Key Generator Patterns
1. **Request & Option Ingestion**:
   - `grpc-service-config`: Path to gRPC service config JSON containing retry policies, timeouts, and retryable status codes.
   - `gapic-config`: Path to GAPIC YAML containing batching settings, LRO polling delays, and interface name overrides.
   - `api-service-config`: Path to API service YAML (e.g. `logging.yaml`) containing documentation, HTTP REST rules, and API definitions.
   - `transport`: `grpc`, `rest`, or `grpc+rest`.
   - `metadata`: Flag controlling generation of `gapic_metadata.json` and GraalVM `reflect-config.json`.
2. **Intermediate Representation**:
   - Parsed protobuf descriptors (`FileDescriptorProto`, `ServiceDescriptorProto`, `MethodDescriptorProto`, `FieldDescriptorProto`) coupled with annotations (`google.api.http`, `google.api.method_signature`, `google.api.resource`, `google.api.routing`, `google.longrunning.operation_info`).
3. **AST Composition & Code Emission**:
   - Constructing idiomatic client classes, stub interfaces, transport implementations, settings builders, and helper classes.

---

## 3. Librarian's Unified GAPIC Generator Framework

### 3.1 Architecture Diagram

```
                 +---------------------------------------+
                 |            Librarian CLI              |
                 |      (librarian generate <lib>)       |
                 +---------------------------------------+
                                     |
                                     v
                 +---------------------------------------+
                 |       Source & Config Discovery       |
                 |  (Sources, Proto Gathering, Metadata) |
                 +---------------------------------------+
                                     |
                                     v
                 +---------------------------------------+
                 |   Unified Generator Engine Interface  |
                 +---------------------------------------+
                                     |
                   +-----------------+-----------------+
                   |                                   |
                   v                                   v
    [In-Process Go Execution]               [External Tool / Protoc Execution]
    - Go (`gapic-generator-go`)             - PHP (`gapic-generator-php`)
    - Java (`gapic/java`)                   - Ruby (`gapic-generator-ruby`)
    - Dart (`sidekick/dart`)                - TypeScript (`gapic-gen-typescript`)
    - Rust (`sidekick/rust`)                - Python (`gapic-generator-python`)
    - Swift (`sidekick/swift`)
                   |                                   |
                   +-----------------+-----------------+
                                     |
                                     v
                 +---------------------------------------+
                 |         Staging & Keep Merging        |
                 |   (filesystem.MoveAndMergeWithKeep)   |
                 +---------------------------------------+
                                     |
                                     v
                 +---------------------------------------+
                 | Language Post-Processing & Formatting |
                 | (google-java-format, gofmt, rubocop)  |
                 +---------------------------------------+
```

### 3.2 Five-Stage Lifecycle Pipeline
Every language generator in Librarian follows a standardized five-stage pipeline:

1. **Discovery & Validation**:
   - Collects protos from primary API directories and configured additional protos (`common_resources.proto`, etc.).
   - Discovers companion service configs (`service.yaml`, `grpc_service_config.json`, `gapic.yaml`).
2. **Option Synthesis**:
   - Derives transport parameters (`grpc`, `rest`, `grpc+rest`).
   - Extracts numeric enum options, package names, warehouse package names, and release levels.
3. **Execution**:
   - For Go-native engines (`Java`, `Go`, `Dart`, `Rust`, `Swift`): Invokes the generator directly in-process or passes request to the compiled plugin binary.
   - For interpreted engines (`PHP`, `Ruby`, `Python`, `Node.js`): Uses cached binary wrappers managed by `librarian install`.
4. **Staging & Preservation**:
   - Staging to isolated temporary sandboxes (`librarian-gen-*` or `owl-bot-staging`).
   - Merging with library root using keep predicates (`library.Keep`) to prevent overwriting handwritten customizations.
5. **Post-Processing**:
   - Code formatting (`gofmt`, `google-java-format`, `dart format`, `rustfmt`, `rubocop`, `prettier`).
   - Generating metadata (`.repo-metadata.json`, `gapic_metadata.json`, GraalVM `reflect-config.json`, `README.md`, `pom.xml`, `package.json`).

---

## 4. Design and Implementation of GAPIC Generator Java in Go

### 4.1 Package Structure in Librarian

```
librarian/
├── cmd/
│   └── protoc-gen-java_gapic/
│       └── main.go                  # Standalone protoc plugin binary
└── internal/
    └── gapic/
        └── java/
            ├── generator.go          # Top-level GenerateGapic API
            ├── generator_test.go     # End-to-end integration test
            ├── engine/
            │   ├── ast/              # Java AST nodes, types, and expressions
            │   │   ├── ast.go
            │   │   └── ast_test.go
            │   ├── lexicon/          # Java keywords, operators, and identifier validators
            │   │   ├── lexicon.go
            │   │   └── lexicon_test.go
            │   ├── escaper/          # Keyword escaping utilities
            │   └── writer/           # Java AST to formatted source code emitter
            │       ├── writer.go
            │       └── writer_test.go
            ├── model/                # Data models (Service, Method, Message, Field, etc.)
            │   ├── model.go
            │   └── model_test.go
            ├── protoparser/          # Protobuf descriptor & config parsers
            │   ├── argument_parser.go
            │   ├── argument_parser_test.go
            │   ├── service_config_parser.go
            │   ├── gapic_yaml_parser.go
            │   ├── service_yaml_parser.go
            │   ├── parser.go
            │   └── parser_test.go
            ├── composer/             # Class, client, stub, settings composers
            │   ├── composer.go
            │   └── composer_test.go
            └── protowriter/          # CodeGeneratorResponse serializer
                ├── writer.go
                └── writer_test.go
```

### 4.2 Core Components & Subsystems

1. **AST Engine (`engine/ast` & `engine/writer`)**:
   - Represents all Java language constructs: `ClassDefinition`, `MethodDefinition`, `TypeNode` (primitives, objects, generics, wildcards, arrays), `VariableExpr`, `MethodInvocationExpr`, `NewObjectExpr`, `IfStatement`, `WhileStatement`, `ForStatement`, `TryCatchStatement`, `SynchronizedStatement`, `JavaDocComment`, and `AnnotationNode`.
   - `writer.WriteClass` formats classes with license headers, package declarations, grouped imports, Javadocs, field definitions, constructors, methods, and inner classes.

2. **Protobuf & Config Parser (`protoparser`)**:
   - Ingests `CodeGeneratorRequest`.
   - Extracts Google API annotations: `google.api.http`, `google.api.method_signature`, `google.api.resource`, `google.api.resource_definition`, `google.api.field_behavior`, `google.api.routing`, `google.longrunning.operation_info`.
   - Parses gRPC service configuration (`grpc_service_config.json`), GAPIC configuration (`gapic.yaml`), and service configuration (`service.yaml`).
   - Automatically detects pagination methods (matching `page_size`/`max_results`, `page_token`, `next_page_token`, and repeated item collections).

3. **Composers (`composer`)**:
   - `ComposeClientClass`: Generates `[Service]Client.java` implementing `BackgroundResource`, with `create()` factory overloads, callable getters, unary/paged/streaming caller methods, and method signature overloads.
   - `ComposeSettingsClass`: Generates `[Service]Settings.java` extending `ClientSettings`.
   - `ComposeServiceStubClass`: Generates abstract `[Service]Stub.java`.
   - `ComposeServiceStubSettingsClass`: Generates `[Service]StubSettings.java` with endpoint defaults.
   - `ComposeGrpcServiceStubClass` & `ComposeHttpJsonServiceStubClass`: Generates transport stubs for gRPC and HTTP/JSON (REST).
   - `ComposeGrpcCallableFactoryClass` & `ComposeHttpJsonCallableFactoryClass`: Generates callable factories.
   - `ComposeResourceNameHelperClass`: Generates typed resource name classes (e.g. `TopicName.java`).
   - `ComposePackageInfo`: Generates `package-info.java`.
   - `ComposeLibraryVersionClass`: Generates `Version.java`.
   - `ComposeNativeReflectConfig`: Generates GraalVM reflection configurations and `gapic_metadata.json`.

4. **Protowriter & Plugin Entrypoint (`protowriter` & `cmd/protoc-gen-java_gapic`)**:
   - Emits `pluginpb.CodeGeneratorResponse` populated with generated files and metadata files.
   - Compiles to native executable `protoc-gen-java_gapic` for protoc plugin execution.

---

## 5. Verification & Test Suite

All ported test suites from `google-cloud-java/sdk-platform-java/gapic-generator-java` have been implemented and validated in Go:
- **Lexicon Tests**: Verifies keyword classification, operator detection, separator identification, identifier validation, and keyword escaping.
- **AST Tests**: Verifies type string rendering (primitives, boxed types, lists, maps, arrays, wildcards) and AST node composition.
- **Writer Tests**: Verifies complete Java class formatting, Javadocs, field annotations, method signatures, and enum classes.
- **Argument Parser Tests**: Verifies parsing of `--java_gapic_opt` arguments (`grpc-service-config`, `gapic-config`, `api-service-config`, `transport`, `repo`, `artifact`, `metadata`, `rest-numeric-enums`, `generate-version-java`).
- **Parser Tests**: Verifies parsing of `CodeGeneratorRequest` with Google API annotations (`HttpRule`, `MethodSignature`, `DefaultHost`, `ResourceDescriptor`).
- **Composer Tests**: Verifies composition of `Client`, `Settings`, `Stub`, `StubSettings`, `GrpcStub`, `HttpJsonStub`, `ResourceName`, `Version`, `package-info`, and reflect configs.
- **Protowriter Tests**: Verifies `CodeGeneratorResponse` file layout and `gapic_metadata.json` structure.
- **End-to-End Generator Tests**: Full roundtrip generation from `CodeGeneratorRequest` $\rightarrow$ `CodeGeneratorResponse`.
