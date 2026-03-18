package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: split_yaml <file_path> <prefix>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	prefix := os.Args[2]

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		os.Exit(1)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		fmt.Printf("Failed to unmarshal yaml: %v\n", err)
		os.Exit(1)
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		fmt.Println("Invalid YAML document")
		os.Exit(1)
	}

	mainMap := root.Content[0]
	var apisNode *yaml.Node
	for i := 0; i < len(mainMap.Content); i += 2 {
		if mainMap.Content[i].Value == "apis" {
			apisNode = mainMap.Content[i+1]
			break
		}
	}

	if apisNode == nil {
		fmt.Println("No 'apis' node found")
		os.Exit(0)
	}

	tracks := make(map[string]bool)
	for _, api := range apisNode.Content {
		for i := 0; i < len(api.Content); i += 2 {
			if api.Content[i].Value == "release_tracks" {
				trackSeq := api.Content[i+1]
				for _, trackNode := range trackSeq.Content {
					tracks[trackNode.Value] = true
				}
			}
		}
	}

	for track := range tracks {
		// Deep copy the root node
		var trackRoot yaml.Node
		yaml.Unmarshal(data, &trackRoot)

		trackMainMap := trackRoot.Content[0]
		var trackApisNode *yaml.Node
		for i := 0; i < len(trackMainMap.Content); i += 2 {
			if trackMainMap.Content[i].Value == "apis" {
				trackApisNode = trackMainMap.Content[i+1]
				break
			}
		}

		var filteredApis []*yaml.Node
		for _, api := range trackApisNode.Content {
			hasTrack := false
			var trackSeq *yaml.Node
			for i := 0; i < len(api.Content); i += 2 {
				if api.Content[i].Value == "release_tracks" {
					trackSeq = api.Content[i+1]
					for _, trackNode := range trackSeq.Content {
						if trackNode.Value == track {
							hasTrack = true
							break
						}
					}
				}
			}
			if hasTrack {
				// filter the release tracks list to just this track
				var filteredTracks []*yaml.Node
				for _, t := range trackSeq.Content {
					if t.Value == track {
						filteredTracks = append(filteredTracks, t)
					}
				}
				trackSeq.Content = filteredTracks
				filteredApis = append(filteredApis, api)
			}
		}

		trackApisNode.Content = filteredApis

		outData, err := yaml.Marshal(&trackRoot)
		if err != nil {
			fmt.Printf("Failed to marshal track %s: %v\n", track, err)
			continue
		}

		outFile := fmt.Sprintf("%s_%s.yaml", prefix, strings.ToLower(track))
		if err := os.WriteFile(outFile, outData, 0644); err != nil {
			fmt.Printf("Failed to write file %s: %v\n", outFile, err)
			continue
		}
		fmt.Printf("Created %s\n", outFile)
	}
}
