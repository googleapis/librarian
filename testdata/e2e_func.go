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
	configureRequest  = "configure-request.json"
	configureResponse = "configure-response.json"
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
	case "configure":
		if err := doConfigure(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "generate":
		if err := doGenerate(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unrecognized command: ", os.Args[1])
	}
}

func doConfigure(args []string) error {
	request, err := parseConfigureRequest(args)
	if err != nil {
		return err
	}
	if err := validateLibrarianDir(request.librarianDir, configureRequest); err != nil {
		return err
	}

	dataMap, err := readConfigureRequest(filepath.Join(request.librarianDir, configureRequest))
	if err != nil {
		return err
	}

	return writeConfigureResponse(request, dataMap)
}

func doGenerate(args []string) error {
	request, err := parseGenerateRequest(args)
	if err != nil {
		return err
	}
	if err := validateLibrarianDir(request.librarianDir, generateRequest); err != nil {
		return err
	}

	return writeToOutput(request)
}

func parseConfigureRequest(args []string) (*configureOption, error) {
	configureOption := &configureOption{}
	for _, arg := range args {
		option, _ := strings.CutPrefix(arg, "--")
		strs := strings.Split(option, "=")
		switch strs[0] {
		case inputDir:
			configureOption.intputDir = strs[1]
		case librarian:
			configureOption.librarianDir = strs[1]
		case libraryID:
			configureOption.libraryID = strs[1]
		case source:
			configureOption.sourceDir = strs[1]
		default:
			return nil, errors.New("unrecognized option: " + option)
		}
	}

	return configureOption, nil
}

func parseGenerateRequest(args []string) (*generateOption, error) {
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

func validateLibrarianDir(dir, requestFile string) error {
	if _, err := os.Stat(filepath.Join(dir, requestFile)); err != nil {
		return err
	}

	return nil
}

func readConfigureRequest(path string) (*map[string]string, error) {

}

func writeConfigureResponse(option *configureOption, dataMap *map[string]string) error {
	jsonFilePath := filepath.Join(option.librarianDir, configureResponse)
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return err
	}
	dataMap = populateMap(dataMap)
	data, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		return err
	}

	if _, err := jsonFile.Write(data); err != nil {
		return err
	}
	log.Print("write configure response to " + jsonFilePath)

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

func populateMap(initialData *map[string]string) *map[string]string {
	(*initialData)["version"] = "1.0.0"
	(*initialData)["last_generated_commit"] = "abcd123"

	return initialData
}

type configureOption struct {
	intputDir    string
	librarianDir string
	libraryID    string
	sourceDir    string
}

type generateOption struct {
	intputDir    string
	outputDir    string
	librarianDir string
	libraryID    string
	sourceDir    string
}
