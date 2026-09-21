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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// serviceAnnotations decorates api.Service with Python-specific metadata.
type serviceAnnotations struct {
	Model           *modelAnnotations
	Service         *api.Service
	Name            string
	ProtoName       string
	ClientName      string
	AsyncClientName string
	DirectoryName   string
	CopyrightYear   string
	DocLines        []string
	Methods         []*methodAnnotations
	Transport       *transportAnnotations
	// Pagers holds pager metadata for services with paged methods.
	Pagers []*pagerAnnotations
	// PagerTypeImports holds module imports needed by pagers.py.
	PagerTypeImports []*pagerTypeImport
	// HasPagers is true if the service defines at least one paged method.
	HasPagers bool
}

func (c *codec) annotateService(service *api.Service, model *modelAnnotations) error {
	docLines := formatDocLines(service.Documentation)
	name := service.Name
	clientName := name
	if !strings.HasSuffix(name, "Client") {
		clientName = name + "Client"
	}
	asyncClientName := strings.TrimSuffix(clientName, "Client") + "AsyncClient"
	copyrightYear := ""
	if model != nil {
		copyrightYear = model.CopyrightYear
	}
	ann := &serviceAnnotations{
		Model:           model,
		Service:         service,
		Name:            name,
		ProtoName:       service.Name,
		ClientName:      clientName,
		AsyncClientName: asyncClientName,
		DirectoryName:   snakeCase(name),
		CopyrightYear:   copyrightYear,
		DocLines:        docLines,
	}

	for _, method := range service.Methods {
		if err := c.annotateMethod(method, ann); err != nil {
			return err
		}
		if mAnn, ok := method.Codec.(*methodAnnotations); ok {
			ann.Methods = append(ann.Methods, mAnn)
		}
	}

	tAnn, err := c.annotateTransport(service)
	if err != nil {
		return err
	}
	ann.Transport = tAnn

	pAnn := c.annotatePagers(service)
	ann.Pagers = pAnn.Pagers
	ann.PagerTypeImports = pAnn.TypeImports
	ann.HasPagers = pAnn.HasPagers

	service.Codec = ann
	return nil
}
