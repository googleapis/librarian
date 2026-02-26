package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/googleapis/librarian/internal/yaml"
)

func main() {
	path := "internal/serviceconfig/sdk.yaml"
	config, err := yaml.Read[[]map[string]any](path)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range *config {
		langs, ok := entry["languages"].([]any)
		// If languages is missing (not ok) or only contains "dart"
		if !ok || (len(langs) == 1 && langs[0] == "dart") {
			langSet := make(map[string]bool)
			if ok {
				for _, l := range langs {
					langSet[l.(string)] = true
				}
			}
			langSet["go"] = true
			langSet["python"] = true
			langSet["rust"] = true

			var newLangs []string
			for l := range langSet {
				newLangs = append(newLangs, l)
			}
			sort.Strings(newLangs)
			entry["languages"] = newLangs
		}
	}

	if err := yaml.Write(path, *config); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully updated sdk.yaml")
}
