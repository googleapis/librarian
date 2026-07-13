package main

import (
	"fmt"
	"log"
	"os/exec"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func main() {
	// 1. Read google-cloud-php/librarian.yaml
	targetConfigPath := "../google-cloud-php/librarian.yaml"
	cfgPtr, err := yaml.Read[config.Config](targetConfigPath)
	if err != nil {
		log.Fatalf("failed to read librarian.yaml from %s: %v", targetConfigPath, err)
	}
	cfg := *cfgPtr

	// 2. Extract all API paths from the configuration
	var targetPaths []string
	for _, lib := range cfg.Libraries {
		for _, api := range lib.APIs {
			if api.Path != "" {
				targetPaths = append(targetPaths, api.Path)
			}
		}
	}
	fmt.Printf("Found %d API paths in google-cloud-php/librarian.yaml\n", len(targetPaths))

	// 3. Read internal/serviceconfig/sdk.yaml
	sdkPath := "internal/serviceconfig/sdk.yaml"
	apisPtr, err := yaml.Read[[]serviceconfig.API](sdkPath)
	if err != nil {
		log.Fatalf("failed to read %s: %v", sdkPath, err)
	}
	apis := *apisPtr

	// 4. Update the allowlist
	updatedCount := 0
	for i, api := range apis {
		if slices.Contains(targetPaths, api.Path) {
			// If languages restriction exists, check if php is in it
			if len(api.Languages) > 0 {
				if !slices.Contains(api.Languages, "php") && !slices.Contains(api.Languages, "all") {
					oldLanguages := make([]string, len(apis[i].Languages))
					copy(oldLanguages, apis[i].Languages)
					apis[i].Languages = append(apis[i].Languages, "php")
					slices.Sort(apis[i].Languages)
					fmt.Printf("Adding 'php' to allowed languages for %s (existing: %v -> %v)\n", api.Path, oldLanguages, apis[i].Languages)
					updatedCount++
				}
			}
		}
	}

	if updatedCount == 0 {
		fmt.Println("No API paths needed updating in sdk.yaml")
		return
	}

	// 5. Write back to internal/serviceconfig/sdk.yaml
	if err := yaml.Write(sdkPath, apis); err != nil {
		log.Fatalf("failed to write updated sdk.yaml: %v", err)
	}
	fmt.Printf("Successfully updated %d APIs in sdk.yaml\n", updatedCount)

	// 6. Format the yaml file
	fmt.Println("Formatting sdk.yaml...")
	cmd := exec.Command("go", "run", "github.com/google/yamlfmt/cmd/yamlfmt", sdkPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to format sdk.yaml: %v\nOutput:\n%s", err, string(output))
	}
	fmt.Println("Successfully formatted sdk.yaml")
}
