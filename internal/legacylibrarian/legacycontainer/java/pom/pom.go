// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package pom handles the generation of Maven pom.xml files for a Java library.
package pom

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed template/*.tmpl
var templatesFS embed.FS

var templates *template.Template

func init() {
	templates = template.Must(template.New("").ParseFS(templatesFS, "template/*.tmpl"))
}

// Module represents a Maven module.
type Module struct {
	GroupId    string
	ArtifactId string
	Version    string
}

// Generate generates the pom.xml files for a library.
// Precondition: module directories exist except for for the *-bom.
func Generate(libraryPath, libraryID string) error {
	// 1. Create main module from libraryID.
	mainModule := &Module{
		GroupId:    "com.google.cloud",
		ArtifactId: fmt.Sprintf("google-cloud-%s", libraryID),
		Version:    "0.0.1-SNAPSHOT", // Default version
	}

	// 2. Find other modules (proto, grpc).
	modules, protoModules, grpcModules, err := findModules(libraryPath, mainModule)
	if err != nil {
		return fmt.Errorf("could not find modules: %w", err)
	}

	// 3. Render templates
	if err := renderTemplates(libraryPath, mainModule, modules, protoModules, grpcModules, libraryID); err != nil {
		return fmt.Errorf("could not render templates: %w", err)
	}

	return nil
}

func findModules(libraryPath string, mainModule *Module) (map[string]*Module, []*Module, []*Module, error) {
	modules := make(map[string]*Module)
	protoModules := []*Module{}
	grpcModules := []*Module{}

	modules[mainModule.ArtifactId] = mainModule

	files, err := os.ReadDir(libraryPath)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, f := range files {
		if f.IsDir() {
			if strings.HasPrefix(f.Name(), "proto-") {
				module := &Module{
					GroupId:    "com.google.api.grpc",
					ArtifactId: f.Name(),
					Version:    mainModule.Version,
				}
				modules[f.Name()] = module
				protoModules = append(protoModules, module)
			} else if strings.HasPrefix(f.Name(), "grpc-") {
				module := &Module{
					GroupId:    "com.google.api.grpc",
					ArtifactId: f.Name(),
					Version:    mainModule.Version,
				}
				modules[f.Name()] = module
				grpcModules = append(grpcModules, module)
			}
		}
	}
	return modules, protoModules, grpcModules, nil
}

func renderTemplates(libraryPath string, mainModule *Module, modules map[string]*Module, protoModules, grpcModules []*Module, libraryID string) error {
	// Render the parent pom.xml
	if err := renderParentPom(libraryPath, mainModule, modules, libraryID); err != nil {
		return err
	}

	for path, module := range modules {
		if strings.HasPrefix(path, "proto-") {
			if err := renderProtoPom(filepath.Join(libraryPath, path), mainModule, module); err != nil {
				return err
			}
		}
		if strings.HasPrefix(path, "grpc-") {
			protoArtifactId := strings.Replace(module.ArtifactId, "grpc-", "proto-", 1)
			protoModule, ok := modules[protoArtifactId]
			if !ok {
				return fmt.Errorf("grpc module %s exists without a corresponding proto module", module.ArtifactId)
			}
			if err := renderGrpcPom(filepath.Join(libraryPath, path), mainModule, module, protoModule); err != nil {
				return err
			}
		}
	}
	mainArtifactDir := filepath.Join(libraryPath, mainModule.ArtifactId)
	if err := renderCloudPom(mainArtifactDir, mainModule, protoModules, grpcModules, libraryID); err != nil {
		return err
	}
	bomDir := filepath.Join(libraryPath, mainModule.ArtifactId+"-bom")
	if err := renderBomPom(bomDir, mainModule, modules, libraryID); err != nil {
		return err
	}
	return nil
}

func renderParentPom(libraryPath string, mainModule *Module, modules map[string]*Module, libraryID string) error {
	var moduleList []*Module
	for _, m := range modules {
		moduleList = append(moduleList, m)
	}
	sort.Slice(moduleList, func(i, j int) bool {
		return moduleList[i].ArtifactId < moduleList[j].ArtifactId
	})

	data := struct {
		MainModule *Module
		Name       string
		Modules    []*Module
	}{
		MainModule: mainModule,
		Name:       fmt.Sprintf("Google Cloud %s", libraryID),
		Modules:    moduleList,
	}
	return renderPom(filepath.Join(libraryPath, "pom.xml"), "parent_pom.xml.tmpl", data)
}

// renderPom executes a template with the given data and writes the output to the outputPath.
func renderPom(outputPath, templateName string, data interface{}) error {
	pomFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer pomFile.Close()

	return templates.ExecuteTemplate(pomFile, templateName, data)
}

func renderProtoPom(modulePath string, mainModule, module *Module) error {
	parentModule := &Module{
		GroupId:    mainModule.GroupId,
		ArtifactId: mainModule.ArtifactId + "-parent",
		Version:    mainModule.Version,
	}

	data := struct {
		MainModule   *Module
		Module       *Module
		ParentModule *Module
	}{
		MainModule:   mainModule,
		Module:       module,
		ParentModule: parentModule,
	}
	return renderPom(filepath.Join(modulePath, "pom.xml"), "proto_pom.xml.tmpl", data)
}

func renderGrpcPom(modulePath string, mainModule, module, protoModule *Module) error {
	parentModule := &Module{
		GroupId:    mainModule.GroupId,
		ArtifactId: mainModule.ArtifactId + "-parent",
		Version:    mainModule.Version,
	}

	data := struct {
		MainModule   *Module
		Module       *Module
		ParentModule *Module
		ProtoModule  *Module
	}{
		MainModule:   mainModule,
		Module:       module,
		ParentModule: parentModule,
		ProtoModule:  protoModule,
	}
	return renderPom(filepath.Join(modulePath, "pom.xml"), "grpc_pom.xml.tmpl", data)
}

func renderCloudPom(modulePath string, mainModule *Module, protoModules, grpcModules []*Module, libraryID string) error {
	parentModule := &Module{
		GroupId:    mainModule.GroupId,
		ArtifactId: mainModule.ArtifactId + "-parent",
		Version:    mainModule.Version,
	}

	data := struct {
		Module       *Module
		Name         string
		Description  string
		ParentModule *Module
		ProtoModules []*Module
		GrpcModules  []*Module
		Repo         string
	}{
		Module:       mainModule,
		Name:         fmt.Sprintf("Google Cloud %s", libraryID),
		Description:  fmt.Sprintf("Google Cloud %s client", libraryID),
		ParentModule: parentModule,
		ProtoModules: protoModules,
		GrpcModules:  grpcModules,
		Repo:         "googleapis/google-cloud-java",
	}

	return renderPom(filepath.Join(modulePath, "pom.xml"), "cloud_pom.xml.tmpl", data)
}

func renderBomPom(modulePath string, mainModule *Module, modules map[string]*Module, libraryID string) error {
	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		if err := os.MkdirAll(modulePath, 0755); err != nil {
			return err
		}
	}
	var moduleList []*Module
	for _, m := range modules {
		moduleList = append(moduleList, m)
	}
	sort.Slice(moduleList, func(i, j int) bool {
		return moduleList[i].ArtifactId < moduleList[j].ArtifactId
	})

	data := struct {
		MainModule *Module
		Name       string
		Modules    []*Module
	}{
		MainModule: mainModule,
		Name:       fmt.Sprintf("Google Cloud %s", libraryID),
		Modules:    moduleList,
	}
	return renderPom(filepath.Join(modulePath, "pom.xml"), "bom_pom.xml.tmpl", data)
}
