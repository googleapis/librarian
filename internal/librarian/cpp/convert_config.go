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

package cpp

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

// ConvertConfigOption configures ConvertConfig behavior.
type ConvertConfigOption func(*convertConfigOptions)

type convertConfigOptions struct {
	libraryNameOverrides map[string]string
}

// WithLibraryNameOverrides provides explicit library name overrides keyed by service_proto_path.
func WithLibraryNameOverrides(overrides map[string]string) ConvertConfigOption {
	return func(o *convertConfigOptions) {
		o.libraryNameOverrides = overrides
	}
}

// ConvertConfig parses generator_config.textproto content and translates it
// into a native *config.Config structure with C++ library definitions.
// It performs strict validation and fails loudly with descriptive error messages
// on any unknown field names or syntax errors.
func ConvertConfig(input []byte, opts ...ConvertConfigOption) (*config.Config, error) {
	options := &convertConfigOptions{}
	for _, opt := range opts {
		opt(options)
	}
	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	p := &parserState{tokens: tokens}
	var libraries []*config.Library
	for p.peek().typ != tokenEOF {
		tok := p.peek()
		if tok.typ != tokenIdent {
			return nil, fmt.Errorf("unexpected token %q on line %d", tok.val, tok.line)
		}
		switch tok.val {
		case "service":
			p.next()
			p.match(tokenColon)
			lib, err := parseService(p, options)
			if err != nil {
				return nil, err
			}
			libraries = append(libraries, lib)
		case "discovery_products":
			p.next()
			p.match(tokenColon)
			libs, err := parseDiscoveryProducts(p, options)
			if err != nil {
				return nil, err
			}
			libraries = append(libraries, libs...)
		default:
			return nil, fmt.Errorf("unknown top-level field %q on line %d", tok.val, tok.line)
		}
	}
	return &config.Config{
		Language:  config.LanguageCpp,
		Libraries: libraries,
	}, nil
}

// ConvertConfigToYAML parses generator_config.textproto bytes and converts them
// to formatted librarian.yaml content.
func ConvertConfigToYAML(input []byte, opts ...ConvertConfigOption) ([]byte, error) {
	cfg, err := ConvertConfig(input, opts...)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(cfg)
}

// DeriveLibraryName derives the librarian library name for a C++ service configuration.
func DeriveLibraryName(serviceProtoPath, productPath, forwardingProductPath string) string {
	if forwardingProductPath != "" {
		cleaned := strings.TrimPrefix(forwardingProductPath, "google/cloud/")
		cleaned = strings.TrimPrefix(cleaned, "google/")
		return strings.ReplaceAll(cleaned, "/", "_")
	}
	if productPath != "" {
		cleaned := strings.TrimPrefix(productPath, "google/cloud/")
		cleaned = strings.TrimPrefix(cleaned, "google/")
		parts := strings.Split(cleaned, "/")
		if len(parts) > 1 && strings.HasPrefix(parts[len(parts)-1], "v") {
			parts = parts[:len(parts)-1]
		}
		prodName := strings.Join(parts, "_")
		protoBase := strings.TrimSuffix(path.Base(serviceProtoPath), ".proto")
		if protoBase == "service" || slices.Contains(parts, protoBase) {
			return prodName
		}
		return prodName + "_" + protoBase
	}
	base := path.Base(serviceProtoPath)
	return strings.TrimSuffix(base, ".proto")
}

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenIdent
	tokenString
	tokenColon
	tokenLBrace
	tokenRBrace
	tokenLBracket
	tokenRBracket
	tokenComma
)

type token struct {
	typ  tokenType
	val  string
	line int
}

func tokenize(input []byte) ([]token, error) {
	var tokens []token
	runes := []rune(string(input))
	n := len(runes)
	i := 0
	line := 1

	for i < n {
		r := runes[i]
		switch {
		case r == '\n':
			line++
			i++
		case unicode.IsSpace(r):
			i++
		case r == '#':
			for i < n && runes[i] != '\n' {
				i++
			}
		case r == ':':
			tokens = append(tokens, token{typ: tokenColon, val: ":", line: line})
			i++
		case r == '{':
			tokens = append(tokens, token{typ: tokenLBrace, val: "{", line: line})
			i++
		case r == '}':
			tokens = append(tokens, token{typ: tokenRBrace, val: "}", line: line})
			i++
		case r == '[':
			tokens = append(tokens, token{typ: tokenLBracket, val: "[", line: line})
			i++
		case r == ']':
			tokens = append(tokens, token{typ: tokenRBracket, val: "]", line: line})
			i++
		case r == ',':
			tokens = append(tokens, token{typ: tokenComma, val: ",", line: line})
			i++
		case r == '"':
			startLine := line
			i++ // skip opening quote
			var sb strings.Builder
			closed := false
			for i < n {
				cur := runes[i]
				if cur == '\n' {
					line++
				}
				if cur == '"' {
					closed = true
					i++
					break
				}
				if cur == '\\' && i+1 < n {
					nextRune := runes[i+1]
					switch nextRune {
					case 'n':
						sb.WriteRune('\n')
					case 'r':
						sb.WriteRune('\r')
					case 't':
						sb.WriteRune('\t')
					case '"':
						sb.WriteRune('"')
					case '\\':
						sb.WriteRune('\\')
					default:
						sb.WriteRune('\\')
						sb.WriteRune(nextRune)
					}
					i += 2
					continue
				}
				sb.WriteRune(cur)
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unclosed string literal on line %d", startLine)
			}
			tokens = append(tokens, token{typ: tokenString, val: sb.String(), line: startLine})
		case isIdentStart(r):
			startLine := line
			start := i
			for i < n && isIdentChar(runes[i]) {
				i++
			}
			tokens = append(tokens, token{typ: tokenIdent, val: string(runes[start:i]), line: startLine})
		default:
			return nil, fmt.Errorf("unexpected character %q on line %d", r, line)
		}
	}
	tokens = append(tokens, token{typ: tokenEOF, val: "", line: line})
	return tokens, nil
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.'
}

func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '/'
}

type parserState struct {
	tokens []token
	pos    int
}

func (p *parserState) peek() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *parserState) next() token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *parserState) match(typ tokenType) bool {
	if p.peek().typ == typ {
		p.next()
		return true
	}
	return false
}

func (p *parserState) expect(typ tokenType) error {
	tok := p.peek()
	if tok.typ != typ {
		return fmt.Errorf("unexpected token %q on line %d", tok.val, tok.line)
	}
	p.next()
	return nil
}

func parseService(p *parserState, options *convertConfigOptions) (*config.Library, error) {
	startTok := p.peek()
	if err := p.expect(tokenLBrace); err != nil {
		return nil, err
	}
	cppLib := &config.CppLibrary{}
	var serviceProtoPath string

	for p.peek().typ != tokenRBrace && p.peek().typ != tokenEOF {
		fieldTok := p.next()
		if fieldTok.typ != tokenIdent {
			return nil, fmt.Errorf("expected field name on line %d, got %q", fieldTok.line, fieldTok.val)
		}
		p.match(tokenColon)

		switch fieldTok.val {
		case "service_proto_path":
			val, err := parseString(p)
			if err != nil {
				return nil, err
			}
			serviceProtoPath = val
		case "product_path":
			val, err := parseString(p)
			if err != nil {
				return nil, err
			}
			cppLib.ProductPath = val
		case "initial_copyright_year":
			val, err := parseString(p)
			if err != nil {
				return nil, err
			}
			cppLib.InitialCopyrightYear = val
		case "forwarding_product_path":
			val, err := parseString(p)
			if err != nil {
				return nil, err
			}
			cppLib.ForwardingProductPath = val
		case "service_endpoint_env_var":
			val, err := parseString(p)
			if err != nil {
				return nil, err
			}
			cppLib.ServiceEndpointEnvVar = val
		case "emulator_endpoint_env_var":
			val, err := parseString(p)
			if err != nil {
				return nil, err
			}
			cppLib.EmulatorEndpointEnvVar = val
		case "endpoint_location_style":
			val, err := parseIdentOrString(p)
			if err != nil {
				return nil, err
			}
			cppLib.EndpointLocationStyle = val
		case "override_service_config_yaml_name":
			val, err := parseString(p)
			if err != nil {
				return nil, err
			}
			cppLib.OverrideServiceConfigYAMLName = val
		case "generate_round_robin_decorator":
			val, err := parseBool(p)
			if err != nil {
				return nil, err
			}
			cppLib.GenerateRoundRobinDecorator = val
		case "generate_rest_transport":
			val, err := parseBool(p)
			if err != nil {
				return nil, err
			}
			cppLib.GenerateRestTransport = val
		case "generate_grpc_transport":
			val, err := parseBool(p)
			if err != nil {
				return nil, err
			}
			cppLib.GenerateGrpcTransport = &val
		case "backwards_compatibility_namespace_alias":
			val, err := parseBool(p)
			if err != nil {
				return nil, err
			}
			cppLib.BackwardsCompatibilityNamespace = val
		case "omit_client":
			val, err := parseBool(p)
			if err != nil {
				return nil, err
			}
			cppLib.OmitClient = val
		case "omit_connection":
			val, err := parseBool(p)
			if err != nil {
				return nil, err
			}
			cppLib.OmitConnection = val
		case "omit_stub_factory":
			val, err := parseBool(p)
			if err != nil {
				return nil, err
			}
			cppLib.OmitStubFactory = val
		case "omit_repo_metadata":
			val, err := parseBool(p)
			if err != nil {
				return nil, err
			}
			cppLib.OmitRepoMetadata = val
		case "omitted_rpcs":
			items, err := parseStringList(p)
			if err != nil {
				return nil, err
			}
			cppLib.OmittedRPCs = append(cppLib.OmittedRPCs, items...)
		case "gen_async_rpcs":
			items, err := parseStringList(p)
			if err != nil {
				return nil, err
			}
			cppLib.GenAsyncRPCs = append(cppLib.GenAsyncRPCs, items...)
		case "omitted_services":
			items, err := parseStringList(p)
			if err != nil {
				return nil, err
			}
			cppLib.OmittedServices = append(cppLib.OmittedServices, items...)
		case "retryable_status_codes":
			items, err := parseStringList(p)
			if err != nil {
				return nil, err
			}
			cppLib.RetryableStatusCodes = append(cppLib.RetryableStatusCodes, items...)
		case "additional_proto_files":
			items, err := parseStringList(p)
			if err != nil {
				return nil, err
			}
			cppLib.AdditionalProtoFiles = append(cppLib.AdditionalProtoFiles, items...)
		case "idempotency_overrides":
			rules, err := parseIdempotencyOverrides(p)
			if err != nil {
				return nil, err
			}
			cppLib.IdempotencyOverrides = append(cppLib.IdempotencyOverrides, rules...)
		case "service_name_mapping":
			if cppLib.ServiceNameMapping == nil {
				cppLib.ServiceNameMapping = make(map[string]string)
			}
			k, v, err := parseKeyValuePair(p, "service_name_mapping")
			if err != nil {
				return nil, err
			}
			cppLib.ServiceNameMapping[k] = v
		case "service_name_to_comment":
			if cppLib.ServiceNameToComment == nil {
				cppLib.ServiceNameToComment = make(map[string]string)
			}
			k, v, err := parseKeyValuePair(p, "service_name_to_comment")
			if err != nil {
				return nil, err
			}
			cppLib.ServiceNameToComment[k] = v
		case "experimental":
			if _, err := parseBool(p); err != nil {
				return nil, err
			}
		case "proto_file_source":
			if _, err := parseIdentOrString(p); err != nil {
				return nil, err
			}
		case "preserve_proto_field_names_in_json":
			if _, err := parseBool(p); err != nil {
				return nil, err
			}
		case "omit_streaming_updater":
			if _, err := parseBool(p); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown field %q in service on line %d", fieldTok.val, fieldTok.line)
		}
	}

	if err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}

	if serviceProtoPath == "" {
		return nil, fmt.Errorf("service missing required service_proto_path on line %d", startTok.line)
	}

	name := ""
	if options != nil && options.libraryNameOverrides != nil {
		name = options.libraryNameOverrides[serviceProtoPath]
	}
	if name == "" {
		name = DeriveLibraryName(serviceProtoPath, cppLib.ProductPath, cppLib.ForwardingProductPath)
	}

	apiDir := path.Dir(serviceProtoPath)
	return &config.Library{
		Name: name,
		APIs: []*config.API{
			{Path: apiDir},
		},
		Cpp: cppLib,
	}, nil
}

func parseDiscoveryProducts(p *parserState, options *convertConfigOptions) ([]*config.Library, error) {
	if err := p.expect(tokenLBrace); err != nil {
		return nil, err
	}
	var libs []*config.Library
	for p.peek().typ != tokenRBrace && p.peek().typ != tokenEOF {
		fieldTok := p.next()
		if fieldTok.typ != tokenIdent {
			return nil, fmt.Errorf("expected field name on line %d, got %q", fieldTok.line, fieldTok.val)
		}
		p.match(tokenColon)

		switch fieldTok.val {
		case "discovery_document_url":
			if _, err := parseString(p); err != nil {
				return nil, err
			}
		case "operation_services":
			if _, err := parseStringList(p); err != nil {
				return nil, err
			}
		case "rest_services":
			lib, err := parseService(p, options)
			if err != nil {
				return nil, err
			}
			libs = append(libs, lib)
		default:
			return nil, fmt.Errorf("unknown field %q in discovery_products on line %d", fieldTok.val, fieldTok.line)
		}
	}
	if err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}
	return libs, nil
}

func parseIdempotencyOverrides(p *parserState) ([]config.IdempotencyRule, error) {
	if err := p.expect(tokenLBracket); err != nil {
		return nil, err
	}
	var rules []config.IdempotencyRule
	for p.peek().typ != tokenRBracket && p.peek().typ != tokenEOF {
		if err := p.expect(tokenLBrace); err != nil {
			return nil, err
		}
		var rule config.IdempotencyRule
		for p.peek().typ != tokenRBrace && p.peek().typ != tokenEOF {
			fieldTok := p.next()
			if fieldTok.typ != tokenIdent {
				return nil, fmt.Errorf("expected field name on line %d, got %q", fieldTok.line, fieldTok.val)
			}
			p.match(tokenColon)
			switch fieldTok.val {
			case "rpc_name":
				val, err := parseString(p)
				if err != nil {
					return nil, err
				}
				rule.RPCName = val
			case "idempotency":
				val, err := parseIdentOrString(p)
				if err != nil {
					return nil, err
				}
				rule.Idempotency = val
			default:
				return nil, fmt.Errorf("unknown field %q in idempotency_overrides on line %d", fieldTok.val, fieldTok.line)
			}
			p.match(tokenComma)
		}
		if err := p.expect(tokenRBrace); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
		p.match(tokenComma)
	}
	if err := p.expect(tokenRBracket); err != nil {
		return nil, err
	}
	return rules, nil
}

func parseKeyValuePair(p *parserState, blockName string) (string, string, error) {
	if err := p.expect(tokenLBrace); err != nil {
		return "", "", err
	}
	var key, val string
	for p.peek().typ != tokenRBrace && p.peek().typ != tokenEOF {
		fieldTok := p.next()
		if fieldTok.typ != tokenIdent {
			return "", "", fmt.Errorf("expected field name on line %d, got %q", fieldTok.line, fieldTok.val)
		}
		p.match(tokenColon)
		switch fieldTok.val {
		case "key":
			s, err := parseString(p)
			if err != nil {
				return "", "", err
			}
			key = s
		case "value":
			s, err := parseString(p)
			if err != nil {
				return "", "", err
			}
			val = s
		default:
			return "", "", fmt.Errorf("unknown field %q in %s on line %d", fieldTok.val, blockName, fieldTok.line)
		}
		p.match(tokenComma)
	}
	if err := p.expect(tokenRBrace); err != nil {
		return "", "", err
	}
	return key, val, nil
}

func parseStringList(p *parserState) ([]string, error) {
	if p.match(tokenLBracket) {
		var list []string
		for p.peek().typ != tokenRBracket && p.peek().typ != tokenEOF {
			tok := p.next()
			if tok.typ != tokenString && tok.typ != tokenIdent {
				return nil, fmt.Errorf("expected string or identifier in list on line %d, got %q", tok.line, tok.val)
			}
			list = append(list, tok.val)
			p.match(tokenComma)
		}
		if err := p.expect(tokenRBracket); err != nil {
			return nil, err
		}
		return list, nil
	}
	tok := p.next()
	if tok.typ != tokenString && tok.typ != tokenIdent {
		return nil, fmt.Errorf("expected string or list on line %d, got %q", tok.line, tok.val)
	}
	return []string{tok.val}, nil
}

func parseString(p *parserState) (string, error) {
	tok := p.next()
	if tok.typ != tokenString && tok.typ != tokenIdent {
		return "", fmt.Errorf("expected string on line %d, got %q", tok.line, tok.val)
	}
	return tok.val, nil
}

func parseIdentOrString(p *parserState) (string, error) {
	tok := p.next()
	if tok.typ != tokenIdent && tok.typ != tokenString {
		return "", fmt.Errorf("expected identifier or string on line %d, got %q", tok.line, tok.val)
	}
	return tok.val, nil
}

func parseBool(p *parserState) (bool, error) {
	tok := p.next()
	switch tok.val {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected boolean (true/false) on line %d, got %q", tok.line, tok.val)
	}
}
