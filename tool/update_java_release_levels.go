package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func main() {
	// 1. Load Java config.
	javaLibData, err := os.ReadFile("../google-cloud-java/librarian.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read librarian.yaml: %v\n", err)
		os.Exit(1)
	}
	javaCfgPtr, err := yaml.Unmarshal[config.Config](javaLibData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to unmarshal librarian.yaml: %v\n", err)
		os.Exit(1)
	}
	javaCfg := *javaCfgPtr

	// 2. Identify the required release level for every API path.
	// We mimic Java's deriveRepoMetadata logic: sort library APIs and pick the first one.
	needsOverride := make(map[string]bool)
	for _, lib := range javaCfg.Libraries {
		if len(lib.APIs) == 0 {
			continue
		}
		// Mimic internal/librarian/java/repometadata.go:deriveRepoMetadata
		serviceconfig.SortAPIs(lib.APIs)
		primaryAPIPath := lib.APIs[0].Path

		tempAPI := serviceconfig.API{Path: primaryAPIPath}
		if tempAPI.ReleaseLevel("java", lib.Version) == "preview" {
			// Historically Java defaulted to "stable". If heuristic says "preview", we need an override.
			// This override applies to the primary API path of this library.
			needsOverride[primaryAPIPath] = true
		}
	}

	// 3. Load sdk.yaml.
	sdkData, err := os.ReadFile("internal/serviceconfig/sdk.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read sdk.yaml: %v\n", err)
		os.Exit(1)
	}
	apisPtr, err := yaml.Unmarshal[[]serviceconfig.API](sdkData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to unmarshal sdk.yaml: %v\n", err)
		os.Exit(1)
	}
	apis := *apisPtr

	apiMap := make(map[string]int)
	for i := range apis {
		apiMap[apis[i].Path] = i
	}

	// 4. Add missing API paths to the slice first.
	var toAdd []serviceconfig.API
	for path := range needsOverride {
		if _, ok := apiMap[path]; !ok {
			if strings.HasPrefix(path, "google/cloud/") {
				toAdd = append(toAdd, serviceconfig.API{
					Path:      path,
					Languages: []string{"all"},
					ReleaseLevels: map[string]string{"java": "stable"},
				})
			}
		}
	}
	apis = append(apis, toAdd...)

	// 5. Re-index and apply overrides.
	addedCount := 0
	for i := range apis {
		api := &apis[i]
		if needsOverride[api.Path] {
			if api.ReleaseLevels["java"] == "" && api.ReleaseLevels["all"] == "" {
				if api.ReleaseLevels == nil {
					api.ReleaseLevels = make(map[string]string)
				}
				api.ReleaseLevels["java"] = "stable"
				addedCount++
			}
		}
	}

	// 6. Final sort and write.
	slices.SortFunc(apis, func(a, b serviceconfig.API) int {
		return strings.Compare(a.Path, b.Path)
	})
	if err := yaml.Write("internal/serviceconfig/sdk.yaml", apis); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write sdk.yaml: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Java release levels added (overrides): %d\n", addedCount)
	fmt.Printf("New API paths added to sdk.yaml: %d\n", len(toAdd))
}
