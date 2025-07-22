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
	inputDir         = "input"
	librarian        = "librarian"
	libraryID        = "library-id"
	outputDir        = "output"
	source           = "source"
	generateRequest  = "generate-request.json"
	generateResponse = "generate-response.json"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatal(errors.New("no command-line arguments provided"))
	}

	log.Print("received command: ", os.Args[1:])
	switch os.Args[1] {
	case "generate":
		if err := doGenerate(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unrecognized command: ", os.Args[1])
	}
}

func doGenerate(args []string) error {
	request, err := parseRequest(args)
	if err != nil {
		return err
	}
	if err := validateLibrarianDir(request.librarianDir); err != nil {
		return err
	}
	if err := writeToOutput(request.outputDir); err != nil {
		return err
	}
	return nil
}

func parseRequest(args []string) (*generateOption, error) {
	request := &generateOption{}
	for _, arg := range args {
		option, _ := strings.CutPrefix(arg, "--")
		strs := strings.Split(option, "=")
		switch strs[0] {
		case inputDir:
			request.intputDir = strs[1]
		case librarian:
			request.librarianDir = strs[1]
		case libraryID:
			request.libraryID = strs[1]
		case outputDir:
			request.outputDir = strs[1]
		case source:
			request.sourceDir = strs[1]
		default:
			return nil, errors.New("unrecognized option: " + option)
		}
	}

	return request, nil
}

func validateLibrarianDir(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, generateRequest)); err != nil {
		return err
	}

	return nil
}

func writeToOutput(dir string) error {
	jsonFilePath := filepath.Join(dir, generateResponse)
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

type generateOption struct {
	intputDir    string
	outputDir    string
	librarianDir string
	libraryID    string
	sourceDir    string
}
