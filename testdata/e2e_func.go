package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	inputDir  = "input"
	librarian = "librarian"
	libraryID = "library-id"
	outputDir = "output"
	source    = "source"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatal(errors.New("no command-line arguments provided"))
	}

	log.Print("received command: ", os.Args[1:])
	switch os.Args[1] {
	case "generate":
		if err := validateGenerate(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unrecognized command: ", os.Args[1])
	}
}

func validateGenerate(args []string) error {
	for _, arg := range args {
		option, _ := strings.CutPrefix(arg, "--")
		strs := strings.Split(option, "=")
		switch strs[0] {
		case librarian:
			if err := validateLibrarianDir(strs[1]); err != nil {
				return err
			}
		case inputDir:
			continue
		case outputDir:
			if err := writeToOutput(strs[1]); err != nil {
				return err
			}
		case source:
			continue
		case libraryID:
			continue
		default:
			return errors.New("unrecognized option: " + option)
		}
	}

	return nil
}

func validateLibrarianDir(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "generate-request.json")); err != nil {
		return err
	}

	return nil
}

func writeToOutput(dir string) error {
	jsonFilePath := filepath.Join(dir, "generate-response.json")
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	dataMap := map[string]int{
		"a": 1,
		"b": 2,
	}
	data, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		return err
	}
	if _, err := jsonFile.Write(data); err != nil {
		return err
	}
	log.Print("write generate response to " + jsonFilePath)
	return nil
}
