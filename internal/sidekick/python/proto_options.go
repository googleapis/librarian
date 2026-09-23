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

package python

import (
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

var (
	reProtoOAuthScopes = regexp.MustCompile(`option\s*\(\s*google\.api\.oauth_scopes\s*\)\s*=\s*([^;]+);`)
	reQuotedString     = regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
	reProtoOpService   = regexp.MustCompile(`option\s*\(\s*google\.cloud\.operation_service\s*\)\s*=\s*"([^"]+)";`)
	reRPCHeader        = regexp.MustCompile(`\brpc\s+([A-Za-z0-9_]+)\s*\(`)
)

// protoServiceOptions stores annotations extracted directly from proto definitions
// for a service, including custom OAuth scopes and extended operations options.
type protoServiceOptions struct {
	OAuthScopes                []string
	MethodOperationServices    map[string]string // methodName -> operationService (e.g. "ZoneOperations")
	ExtendedOperationsServices []*extendedOperationService
}

// extendedOperationService represents an extended operations service referenced by a service's RPCs.
type extendedOperationService struct {
	ServiceNameSnake  string
	ServiceNamePascal string
}

// loadProtoServiceOptions loads and parses proto service options (OAuth scopes and extended operations)
// for the given service, caching results on the codec.
func (c *codec) loadProtoServiceOptions(service *api.Service) *protoServiceOptions {
	if service == nil {
		return &protoServiceOptions{MethodOperationServices: make(map[string]string)}
	}
	if c.protoOptionsCache == nil {
		c.protoOptionsCache = make(map[string]*protoServiceOptions)
	}
	cacheKey := service.ID
	if cacheKey == "" {
		cacheKey = service.Package + "." + service.Name
	}
	if cached, ok := c.protoOptionsCache[cacheKey]; ok {
		return cached
	}

	opts := &protoServiceOptions{
		MethodOperationServices: make(map[string]string),
	}
	c.protoOptionsCache[cacheKey] = opts

	protoPath := c.findConfigFileForService(service, "*.proto")
	if protoPath == "" {
		return opts
	}

	data, err := os.ReadFile(protoPath)
	if err != nil {
		return opts
	}

	parseProtoServiceBlock(string(data), service.Name, opts)
	return opts
}

// parseProtoServiceBlock parses the service block for serviceName from protoContent and populates opts.
func parseProtoServiceBlock(protoContent, serviceName string, opts *protoServiceOptions) {
	serviceHeaderRe := regexp.MustCompile(`\bservice\s+` + regexp.QuoteMeta(serviceName) + `\s*\{`)
	loc := serviceHeaderRe.FindStringIndex(protoContent)
	if loc == nil {
		return
	}

	openBraceIdx := loc[1] - 1
	serviceBody := extractBraceBlock(protoContent[openBraceIdx:])
	if serviceBody == "" {
		return
	}

	// 1. Extract OAuth scopes from service body
	if scopeMatch := reProtoOAuthScopes.FindStringSubmatch(serviceBody); len(scopeMatch) > 1 {
		rawStrings := reQuotedString.FindAllStringSubmatch(scopeMatch[1], -1)
		var combined strings.Builder
		for _, q := range rawStrings {
			if len(q) > 1 {
				combined.WriteString(q[1])
			}
		}
		rawParts := strings.FieldsFunc(combined.String(), func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r'
		})
		for _, s := range rawParts {
			s = strings.TrimSpace(s)
			if s != "" && !slices.Contains(opts.OAuthScopes, s) {
				opts.OAuthScopes = append(opts.OAuthScopes, s)
			}
		}
	}

	// 2. Extract RPC method operation services from service body
	extServicesMap := make(map[string]bool)
	rpcMatches := reRPCHeader.FindAllStringSubmatchIndex(serviceBody, -1)
	for _, idxs := range rpcMatches {
		if len(idxs) < 4 {
			continue
		}
		methName := serviceBody[idxs[2]:idxs[3]]
		rpcRest := serviceBody[idxs[0]:]
		braceIdx := findNextBraceOrSemicolon(rpcRest)
		if braceIdx < 0 || rpcRest[braceIdx] != '{' {
			continue
		}
		rpcBody := extractBraceBlock(rpcRest[braceIdx:])
		if rpcBody == "" {
			continue
		}

		if opMatch := reProtoOpService.FindStringSubmatch(rpcBody); len(opMatch) > 1 {
			opService := opMatch[1]
			opts.MethodOperationServices[methName] = opService
			if !extServicesMap[opService] {
				extServicesMap[opService] = true
				opts.ExtendedOperationsServices = append(opts.ExtendedOperationsServices, &extendedOperationService{
					ServiceNameSnake:  snakeCase(opService),
					ServiceNamePascal: opService,
				})
			}
		}
	}

	slices.SortFunc(opts.ExtendedOperationsServices, func(a, b *extendedOperationService) int {
		return strings.Compare(a.ServiceNameSnake, b.ServiceNameSnake)
	})
}

// extractBraceBlock extracts the content enclosed by the outermost braces starting at s[0].
func extractBraceBlock(s string) string {
	if len(s) == 0 || s[0] != '{' {
		return ""
	}
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			if depth == 0 {
				start = i + 1
			}
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start:i]
			}
		}
	}
	return ""
}

// findNextBraceOrSemicolon finds the index of the first '{' or ';' in s.
func findNextBraceOrSemicolon(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '{' || s[i] == ';' {
			return i
		}
	}
	return -1
}
