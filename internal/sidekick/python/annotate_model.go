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
	"maps"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// modelAnnotations decorates api.API with Python-specific metadata.
type modelAnnotations struct {
	CopyrightYear   string
	BoilerPlate     []string
	PackageName     string
	PackageVersion  string
	Messages        []*messageAnnotations
	Enums           []*enumAnnotations
	Services        []*serviceAnnotations
	TypeFiles       map[string]*fileAnnotations
	AllTypeFiles    []*fileAnnotations
	SortedTypeFiles []*fileAnnotations
	AllTypeSymbols  []string
}

func (c *codec) annotateModel() error {
	ann := &modelAnnotations{
		CopyrightYear:  c.GenerationYear,
		BoilerPlate:    license.HeaderBulk(),
		PackageName:    c.PackageName,
		PackageVersion: c.PackageVersion,
		TypeFiles:      make(map[string]*fileAnnotations),
	}
	c.Model.Codec = ann

	// Annotate messages first so services can reference message types.
	for _, message := range c.Model.Messages {
		if err := c.annotateMessage(message, ann); err != nil {
			return err
		}
		if mAnn, ok := message.Codec.(*messageAnnotations); ok {
			ann.Messages = append(ann.Messages, mAnn)
		}
	}

	// Annotate enums.
	for _, enum := range c.Model.Enums {
		if err := c.annotateEnum(enum, ann); err != nil {
			return err
		}
		if eAnn, ok := enum.Codec.(*enumAnnotations); ok {
			ann.Enums = append(ann.Enums, eAnn)
		}
	}

	// Group messages, enums, and services into type files.
	fileMessages := make(map[string][]*api.Message)
	fileEnums := make(map[string][]*api.Enum)
	allFiles := make(map[string]bool)

	for _, msg := range c.Model.Messages {
		src := c.sourceFileForLocation(msg.SourceLocation)
		fileMessages[src] = append(fileMessages[src], msg)
		allFiles[src] = true
	}

	for _, enum := range c.Model.Enums {
		src := c.sourceFileForLocation(enum.SourceLocation)
		fileEnums[src] = append(fileEnums[src], enum)
		allFiles[src] = true
	}

	for _, svc := range c.Model.Services {
		src := c.sourceFileForLocation(svc.SourceLocation)
		allFiles[src] = true
	}

	for _, src := range slices.Sorted(maps.Keys(allFiles)) {
		fAnn := c.annotateFile(src, fileMessages[src], fileEnums[src])
		ann.TypeFiles[src] = fAnn
		ann.AllTypeFiles = append(ann.AllTypeFiles, fAnn)
		if fAnn.HasMessagesOrEnums {
			ann.SortedTypeFiles = append(ann.SortedTypeFiles, fAnn)
		}
	}

	slices.SortFunc(ann.AllTypeFiles, func(a, b *fileAnnotations) int {
		return strings.Compare(a.Stem, b.Stem)
	})
	slices.SortFunc(ann.SortedTypeFiles, func(a, b *fileAnnotations) int {
		return strings.Compare(a.Stem, b.Stem)
	})

	for _, fAnn := range ann.SortedTypeFiles {
		ann.AllTypeSymbols = append(ann.AllTypeSymbols, fAnn.SortedSymbols...)
	}

	// Annotate services after messages, enums, and type files.
	for _, service := range c.Model.Services {
		if err := c.annotateService(service, ann); err != nil {
			return err
		}
		if sAnn, ok := service.Codec.(*serviceAnnotations); ok {
			ann.Services = append(ann.Services, sAnn)
		}
	}

	return nil
}

func (c *codec) sourceFileForLocation(loc *api.SourceLocation) string {
	if loc != nil && loc.File != "" {
		return loc.File
	}
	if c.Library != nil && c.Library.Name != "" {
		return c.Library.Name + ".proto"
	}
	if c.GAPICName != "" {
		return c.GAPICName + ".proto"
	}
	if c.PackageName != "" {
		return c.PackageName + ".proto"
	}
	return "model.proto"
}
