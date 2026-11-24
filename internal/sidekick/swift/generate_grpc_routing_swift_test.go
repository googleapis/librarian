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

package swift

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerateStub_GRPCClient_RoutingFieldTypes_SwiftTest(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: compiles and runs swift test")
	}
	testhelper.RequireCommand(t, "swift")

	pkgDir := t.TempDir()
	genDir := filepath.Join(pkgDir, "Sources", "RoutingTestLib", "generated")
	model := makeRoutingFieldTypesModel(t)

	module := &config.SwiftModule{
		Output:     genDir,
		ModuleType: "grpc-client",
		ModulePath: "RoutingTestProtos",
	}
	swiftPkg := swiftConfig(t, []config.SwiftDependency{
		{Name: "GoogleType", ApiPackage: "google.type"},
	})
	swiftPkg.PackageNameOverride = "RoutingTestLib"
	swiftPkg.LibraryNameOverride = "RoutingTestLib"
	library := &config.Library{
		Name:  "routing-test-lib",
		Swift: swiftPkg,
	}
	if err := Generate(t.Context(), model, genDir, library, module); err != nil {
		t.Fatal(err)
	}

	writeTestFile := func(relPath, content string) {
		t.Helper()
		fullPath := filepath.Join(pkgDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile("Package.swift", `// swift-tools-version: 6.2
import PackageDescription

let strictSwiftSettings: [SwiftSetting] = [
  .enableUpcomingFeature("ExistentialAny"),
  .enableUpcomingFeature("InternalImportsByDefault"),
  .enableUpcomingFeature("MemberImportVisibility"),
  .enableUpcomingFeature("NonisolatedNonsendingByDefault"),
  .strictMemorySafety(),
]

let package = Package(
  name: "RoutingTestPackage",
  platforms: [.macOS(.v15)],
  targets: [
    .target(name: "GoogleAuth"),
    .target(name: "GoogleWKT"),
    .target(name: "GoogleWKTConvert", dependencies: ["GoogleWKT"]),
    .target(name: "SwiftProtobuf"),
    .target(name: "Logging"),
    .target(name: "GoogleType", dependencies: ["GoogleWKT"]),
    .target(name: "GoogleGax", swiftSettings: strictSwiftSettings),
    .target(
      name: "GoogleGaxGRPC",
      dependencies: ["GoogleGax", "SwiftProtobuf"],
      swiftSettings: strictSwiftSettings
    ),
    .target(name: "RoutingTestProtos", dependencies: ["SwiftProtobuf"]),
    .target(
      name: "RoutingTestLib",
      dependencies: [
        "GoogleAuth",
        "GoogleGax",
        "GoogleGaxGRPC",
        "GoogleType",
        "GoogleWKT",
        "GoogleWKTConvert",
        "SwiftProtobuf",
        "Logging",
        "RoutingTestProtos",
      ],
      swiftSettings: strictSwiftSettings
    ),
    .testTarget(
      name: "RoutingTestLibTests",
      dependencies: ["RoutingTestLib", "GoogleGax", "GoogleGaxGRPC", "GoogleType", "GoogleWKT"],
      swiftSettings: [
        .enableUpcomingFeature("ExistentialAny"),
        .enableUpcomingFeature("InternalImportsByDefault"),
        .enableUpcomingFeature("MemberImportVisibility"),
        .enableUpcomingFeature("NonisolatedNonsendingByDefault"),
      ]
    ),
  ]
)
`)

	writeTestFile("Sources/GoogleAuth/GoogleAuth.swift", `public enum _GoogleAuthStub {}
`)

	writeTestFile("Sources/SwiftProtobuf/SwiftProtobuf.swift", `import Foundation

public struct UnknownStorage: Sendable, Equatable {
  public var data: Data = Data()
  public init() {}
}

public protocol Message: Sendable {
  init()
}

public struct Google_Protobuf_Empty: Message {
  public init() {}
}

extension Message {
  public mutating func merge(serializedBytes: Data) throws {}
}
`)

	writeTestFile("Sources/Logging/Logging.swift", `public struct Logger: Sendable {
  public enum MetadataValue: Sendable, ExpressibleByStringLiteral, ExpressibleByStringInterpolation {
    case string(String)
    public init(stringLiteral value: String) { self = .string(value) }
  }
  public init() {}
  public subscript(metadataKey key: String) -> MetadataValue? {
    get { nil }
    set {}
  }
  public func debug(_ message: @autoclosure () -> String) {}
}
`)

	writeTestFile("Sources/GoogleWKT/GoogleWKT.swift", `public import Foundation

public typealias WKTStruct = [String: WKTValue]
public typealias WKTStringValue = Swift.String
public typealias WKTInt64Value = Swift.Int64

public struct WKTValue: Codable, Equatable, Sendable {
  public init() {}
}

public struct WKTAny: Sendable {
  public init() {}
}

public struct _UnknownFields: Equatable, Sendable {
  public var json: [String: WKTValue] = [:]
  @_spi(GoogleCloudInternal) public var proto: Data = Data()
  public init() {}
}

public protocol _AnyPackable: Sendable {
  static var _anyTypeUrl: String { get }
  init(fromAny any: WKTAny) throws
  func _pack() throws -> WKTStruct
}

public func _anyDeserialize<T>(_ type: T.Type, from any: WKTAny) throws -> T {
  fatalError("unused")
}

public func _anySerialize<T>(message: T) throws -> WKTStruct {
  [:]
}

extension Swift.String: _AnyPackable {
  public static var _anyTypeUrl: String { "type.googleapis.com/google.protobuf.StringValue" }
  public init(fromAny any: WKTAny) throws { self = "" }
  public func _pack() throws -> WKTStruct { [:] }
}

extension Swift.Int64: _AnyPackable {
  public static var _anyTypeUrl: String { "type.googleapis.com/google.protobuf.Int64Value" }
  public init(fromAny any: WKTAny) throws { self = 0 }
  public func _pack() throws -> WKTStruct { [:] }
}

public struct WKTRecursive<T: Codable & Sendable>: Codable, Sendable {
  private final class Storage: @unchecked Sendable {
    var value: T
    init(_ value: T) { self.value = value }
  }
  private var storage: Storage
  public var value: T { storage.value }
  public init(value: T) { self.storage = Storage(value) }
  public init(from decoder: any Decoder) throws { self.storage = Storage(try T(from: decoder)) }
  public func encode(to encoder: any Encoder) throws { try storage.value.encode(to: encoder) }
}

extension WKTRecursive: Equatable where T: Equatable {
  public static func == (lhs: WKTRecursive<T>, rhs: WKTRecursive<T>) -> Bool {
    lhs.value == rhs.value
  }
}
`)

	writeTestFile("Sources/GoogleWKTConvert/GoogleWKTConvert.swift", `public enum _GoogleWKTConvertStub {}
`)

	writeTestFile("Sources/GoogleType/ExtDetail.swift", `public struct ExtDetail: Codable, Equatable, Sendable {
  public var code: Swift.Int32 = 0
  public init() {}
  public func with(_ config: (inout Self) throws -> Swift.Void) rethrows -> Self {
    var copy = self
    try config(&copy)
    return copy
  }
}
`)

	writeTestFile("Sources/GoogleGax/GoogleGax.swift", `internal import Foundation

public struct ClientOptions: Sendable {
  public init() {}
}

public struct RequestOptions: Sendable {
  public var attemptTimeout: Duration?
  public init() {}
}

@_spi(GoogleCloudInternal)
public struct _RetryLoop: Sendable {
  public init(options: RequestOptions, withDefault: ClientOptions, idempotent: Bool) {}
  public func run<Output>(attempt: (Duration?) async throws -> Output) async throws -> Output {
    try await attempt(nil)
  }
}

@_spi(GoogleCloudInternal)
public enum _RoutingSegment: Sendable, Equatable {
  case literal(String)
  case singleWildcard
  case multiWildcard
  case trailingMultiWildcard

  func matchLength(in haystack: Substring) -> Int? {
    switch self {
    case .literal(let lit):
      return haystack.hasPrefix(lit) ? lit.count : nil
    case .singleWildcard:
      if haystack.isEmpty || haystack.hasPrefix("/") {
        return nil
      }
      if let slashIndex = haystack.firstIndex(of: "/") {
        return haystack.distance(from: haystack.startIndex, to: slashIndex)
      }
      return haystack.count
    case .multiWildcard:
      return haystack.count
    case .trailingMultiWildcard:
      if haystack.isEmpty { return 0 }
      if haystack.hasPrefix("/") || haystack.hasPrefix(":") { return haystack.count }
      return nil
    }
  }
}

@_spi(GoogleCloudInternal)
public enum _RoutingMatcher {
  private static let unreservedCharacterSet = CharacterSet(
    charactersIn: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
  )

  public static func encode(_ value: String) -> String {
    value.addingPercentEncoding(withAllowedCharacters: unreservedCharacterSet) ?? value
  }

  public static func format(_ matches: [(key: String, value: String?)]) -> [String] {
    matches.compactMap { key, value in
      guard let value, !value.isEmpty else { return nil }
      return "\(encode(key))=\(encode(value))"
    }
  }

  public static func value(
    _ haystack: String?,
    prefix: [_RoutingSegment] = [],
    matching: [_RoutingSegment],
    suffix: [_RoutingSegment] = []
  ) -> String? {
    guard let haystack, !haystack.isEmpty else { return nil }
    var remains = Substring(haystack)
    var startOffset = 0
    var endOffset = 0
    for needle in prefix {
      guard let count = needle.matchLength(in: remains) else { return nil }
      startOffset += count
      endOffset += count
      remains = remains.dropFirst(count)
    }
    for needle in matching {
      guard let count = needle.matchLength(in: remains) else { return nil }
      endOffset += count
      remains = remains.dropFirst(count)
    }
    for needle in suffix {
      guard let count = needle.matchLength(in: remains) else { return nil }
      remains = remains.dropFirst(count)
    }
    guard remains.isEmpty, startOffset < endOffset else { return nil }
    let start = haystack.index(haystack.startIndex, offsetBy: startOffset)
    let end = haystack.index(haystack.startIndex, offsetBy: endOffset)
    return String(haystack[start..<end])
  }
}
`)

	writeTestFile("Sources/GoogleGaxGRPC/GoogleGaxGRPC.swift", `@_spi(GoogleCloudInternal) public import GoogleGax
public import SwiftProtobuf

@_spi(GoogleCloudInternal)
public typealias _RoutingSegment = GoogleGax._RoutingSegment

@_spi(GoogleCloudInternal)
public typealias _RoutingMatcher = GoogleGax._RoutingMatcher

public actor RecordedCallStore {
  public private(set) var calls: [(path: String, routingParams: [String])] = []
  public init() {}
  public func record(path: String, routingParams: [String]) {
    calls.append((path, routingParams))
  }
  public func lastRoutingParams() -> [String]? {
    calls.last?.routingParams
  }
}

@_spi(GoogleCloudInternal)
public final class _GRPCClient: Sendable {
  public let store = RecordedCallStore()

  public init(from options: ClientOptions, withDefaultEndpoint: String, clientHeader: String) throws {}
  public init() {}

  public func execute<Request: Sendable, Response: SwiftProtobuf.Message>(
    path: String,
    request: Request,
    options: RequestOptions,
    routingParams: [String] = []
  ) async throws -> Response {
    await store.record(path: path, routingParams: routingParams)
    return Response()
  }
}
`)

	writeTestFile("Sources/RoutingTestProtos/Protos.swift", `public import SwiftProtobuf

public struct Google_Cloud_Test_V1_RouteRequest: SwiftProtobuf.Message {
  public init() {}
}

public struct Google_Cloud_Test_V1_RouteResponse: SwiftProtobuf.Message {
  public init() {}
}
`)

	writeTestFile("Sources/RoutingTestLib/Clients.swift", `internal import RoutingTestProtos

public enum Clients {
  static let clientHeader: Swift.String = "test-client"
}

extension RouteRequest {
  internal func toProto() throws -> RoutingTestProtos.Google_Cloud_Test_V1_RouteRequest {
    .init()
  }
}

extension RouteResponse {
  internal init(proto: RoutingTestProtos.Google_Cloud_Test_V1_RouteResponse) throws {
    self.init()
  }
}
`)

	writeTestFile("Tests/RoutingTestLibTests/RoutingTransportTests.swift", `@_spi(GoogleCloudInternal) import GoogleGax
@_spi(GoogleCloudInternal) import GoogleGaxGRPC
import GoogleType
import Testing
@testable import RoutingTestLib

@Suite struct RoutingTransportTests {
  @Test func scalarsAndEnums() async throws {
    let client = GoogleGaxGRPC._GRPCClient()
    let transport = Clients.RoutingTestServiceTransport(client)

    let populated = RouteRequest().with {
      $0.pageSize = 42
      $0.optCount = 7
      $0.active = true
      $0.state = .active
      $0.optState = .unknownIntValue(99)
      $0.info = Inner().with {
        $0.count = 123
        $0.optState = .active
      }
      $0.extDetail = ExtDetail().with {
        $0.code = 505
      }
    }
    _ = try await transport.explicitScalarsAndEnums(request: populated, options: .init())
    let got = await client.store.lastRoutingParams()
    #expect(
      got == [
        "page_size=42",
        "opt_count=7",
        "active=true",
        "state=STATE_ACTIVE",
        "opt_state=99",
        "info_count=123",
        "info_opt_state=STATE_ACTIVE",
        "ext_code=505",
      ]
    )

    let defaults = RouteRequest()
    _ = try await transport.explicitScalarsAndEnums(request: defaults, options: .init())
    let gotDefaults = await client.store.lastRoutingParams()
    #expect(
      gotDefaults == [
        "page_size=0",
        "active=false",
        "state=STATE_UNSPECIFIED",
      ]
    )

    let httpReq = RouteRequest().with {
      $0.name = "projects/p1/buckets/b1"
      $0.pageSize = 321
      $0.state = .active
    }
    _ = try await transport.implicitHttpBindings(request: httpReq, options: .init())
    let gotHTTP = await client.store.lastRoutingParams()
    #expect(
      gotHTTP == [
        "name=projects%2Fp1%2Fbuckets%2Fb1",
        "page_size=321",
        "state=STATE_ACTIVE",
      ]
    )
  }
}
`)

	args := []string{"test", "--package-path", pkgDir, "-Xswiftc", "-warnings-as-errors"}
	if _, err := os.Stat("/usr/bin/gcc"); err == nil {
		args = append(args, "-Xcc", "--gcc-toolchain=/usr", "-Xcxx", "--gcc-toolchain=/usr")
	}
	cmd := exec.CommandContext(t.Context(), "swift", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("swift %v failed: %v\n%s", args, err, out)
	}
}
