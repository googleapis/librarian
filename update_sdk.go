package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

type LibrarianConfig struct {
	Libraries []struct {
		Apis []struct {
			Path string `yaml:"path"`
		} `yaml:"apis"`
	} `yaml:"libraries"`
}

func main() {
	// 1. Read google-cloud-java/librarian.yaml
	libData, err := os.ReadFile("../google-cloud-java/librarian.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var libCfg LibrarianConfig
	if err := yaml.Unmarshal(libData, &libCfg); err != nil {
		log.Fatal(err)
	}

	javaPaths := make(map[string]bool)
	for _, lib := range libCfg.Libraries {
		for _, api := range lib.Apis {
			if api.Path != "" {
				javaPaths[api.Path] = true
			}
		}
	}

	// 2. Read librarian/internal/serviceconfig/sdk.yaml
	sdkData, err := os.ReadFile("internal/serviceconfig/sdk.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(sdkData, &root); err != nil {
		log.Fatal(err)
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 || root.Content[0].Kind != yaml.SequenceNode {
		log.Fatal("unexpected sdk.yaml structure")
	}

	sequence := root.Content[0]
	updatedCount := 0

	for _, entry := range sequence.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}

		var path string
		var languagesNode *yaml.Node
		
		for i := 0; i < len(entry.Content); i += 2 {
			key := entry.Content[i].Value
			if key == "path" {
				path = entry.Content[i+1].Value
			} else if key == "languages" {
				languagesNode = entry.Content[i+1]
			}
		}

		if path != "" && javaPaths[path] {
			if languagesNode != nil && languagesNode.Kind == yaml.SequenceNode && len(languagesNode.Content) > 0 {
				hasJava := false
				for _, lang := range languagesNode.Content {
					if lang.Value == "java" {
						hasJava = true
						break
					}
				}

				if !hasJava {
					newNode := &yaml.Node{
						Kind:  yaml.ScalarNode,
						Value: "java",
						Tag:   "!!str",
					}
					languagesNode.Content = append(languagesNode.Content, newNode)
					sort.Slice(languagesNode.Content, func(i, j int) bool {
						return languagesNode.Content[i].Value < languagesNode.Content[j].Value
					})
					updatedCount++
					fmt.Printf("Added java to path: %s\n", path)
				}
			}
		}
	}

	if updatedCount == 0 {
		fmt.Println("No updates needed.")
		return
	}

	// 4. Write back
	f, err := os.Create("internal/serviceconfig/sdk.yaml")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated %d entries.\n", updatedCount)
}
