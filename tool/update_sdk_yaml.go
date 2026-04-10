package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

type LibrarianConfig struct {
	Libraries []struct {
		APIs []struct {
			Path string `yaml:"path"`
		} `yaml:"apis"`
	} `yaml:"libraries"`
}

func main() {
	librarianYAML := "../google-cloud-java/librarian.yaml"
	sdkYAML := "internal/serviceconfig/sdk.yaml"

	// 1. Read Librarian YAML
	libConfig, err := yaml.Read[LibrarianConfig](librarianYAML)
	if err != nil {
		log.Fatalf("failed to read %s: %v", librarianYAML, err)
	}

	javaPaths := make(map[string]bool)
	for _, lib := range libConfig.Libraries {
		for _, api := range lib.APIs {
			if api.Path != "" {
				javaPaths[api.Path] = true
			}
		}
	}

	// 2. Read SDK YAML using the official API struct
	apis, err := yaml.Read[[]serviceconfig.API](sdkYAML)
	if err != nil {
		log.Fatalf("failed to read %s: %v", sdkYAML, err)
	}

	updatedCount := 0
	foundPaths := make(map[string]bool)

	for i := range *apis {
		api := &(*apis)[i]
		if javaPaths[api.Path] {
			foundPaths[api.Path] = true
			if !containsLanguage(api.Languages, "java") {
				api.Languages = append(api.Languages, "java")
				sort.Strings(api.Languages)
				updatedCount++
				fmt.Printf("Added java to %s\n", api.Path)
			}
		}
	}

	// 3. Report missing paths
	var missingPaths []string
	for path := range javaPaths {
		if !foundPaths[path] {
			missingPaths = append(missingPaths, path)
		}
	}
	sort.Strings(missingPaths)
	if len(missingPaths) > 0 {
		fmt.Printf("\nFound %d paths in librarian.yaml that are missing in sdk.yaml:\n", len(missingPaths))
		for _, path := range missingPaths {
			fmt.Printf("  - %s\n", path)
		}
	}

	// 4. Write back if updated
	if updatedCount > 0 {
		fmt.Printf("\nUpdating %s with %d changes...\n", sdkYAML, updatedCount)
		if err := yaml.Write(sdkYAML, *apis); err != nil {
			log.Fatalf("failed to write %s: %v", sdkYAML, err)
		}
		fmt.Println("Done.")
	} else {
		fmt.Println("\nNo changes needed.")
	}
}

func containsLanguage(languages []string, lang string) bool {
	for _, l := range languages {
		if l == lang || l == "all" {
			return true
		}
	}
	return false
}
