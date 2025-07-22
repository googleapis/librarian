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
	inputDir          = "input"
	librarian         = "librarian"
	libraryID         = "library-id"
	outputDir         = "output"
	source            = "source"
	generateRequest   = "generate-request.json"
	generateResponse  = "generate-response.json"
	nonExistedLibrary = "non-existed-library"
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
	if err := writeToOutput(request); err != nil {
		return err
	}
	return nil
}

func parseRequest(args []string) (*generateOption, error) {
	generateOption := &generateOption{}
	for _, arg := range args {
		option, _ := strings.CutPrefix(arg, "--")
		strs := strings.Split(option, "=")
		switch strs[0] {
		case inputDir:
			generateOption.intputDir = strs[1]
		case librarian:
			generateOption.librarianDir = strs[1]
		case libraryID:
			generateOption.libraryID = strs[1]
		case outputDir:
			generateOption.outputDir = strs[1]
		case source:
			generateOption.sourceDir = strs[1]
		default:
			return nil, errors.New("unrecognized option: " + option)
		}
	}

	return generateOption, nil
}

func validateLibrarianDir(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, generateRequest)); err != nil {
		return err
	}

	return nil
}

func writeToOutput(option *generateOption) error {
	jsonFilePath := filepath.Join(option.outputDir, generateResponse)
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	dataMap := map[string]string{}
	if option.libraryID == nonExistedLibrary {
		dataMap["error"] = "simulated generation error"
	}
	data, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		return err
	}
	if _, err := jsonFile.Write(data); err != nil {
		return err
	}
	log.Print("write generate response to " + jsonFilePath)
	if option.libraryID == nonExistedLibrary {
		return errors.New("generation failed due to invalid library id")
	}
	return nil
}

type generateOption struct {
	intputDir    string
	outputDir    string
	librarianDir string
	libraryID    string
	sourceDir    string
}
