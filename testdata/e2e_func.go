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
	inputDir                 = "input"
	librarian                = "librarian"
	outputDir                = "output"
	source                   = "source"
	configureRequest         = "configure-request.json"
	configureResponse        = "configure-response.json"
	generateRequest          = "generate-request.json"
	generateResponse         = "generate-response.json"
	simulateConfigureErrorID = "simulate-configure-error-id"
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

	library, err := readConfigureRequest(filepath.Join(request.librarianDir, configureRequest))
	if err != nil {
		return err
	}

	return writeConfigureResponse(request, library)
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

func readConfigureRequest(path string) (*libraryState, error) {
	library := &libraryState{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, library); err != nil {
		return nil, err
	}

	if library.ID == simulateConfigureErrorID {
		// Simulate a configure command error
		return nil, errors.New("simulate configure command error")
	}

	return library, nil
}

func writeConfigureResponse(option *configureOption, library *libraryState) error {
	library = populateAdditionalFields(library)
	data, err := json.MarshalIndent(library, "", "  ")
	if err != nil {
		return err
	}

	jsonFilePath := filepath.Join(option.librarianDir, configureResponse)
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return err
	}

	if _, err := jsonFile.Write(data); err != nil {
		return err
	}
	log.Print("write configure response to " + jsonFilePath)

	return nil
}

func writeToOutput(option *generateOption) (err error) {
	jsonFilePath := filepath.Join(option.outputDir, generateResponse)
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, jsonFile.Close())
	}()

	dataMap := map[string]string{}
	if option.libraryID == simulateConfigureErrorID {
		dataMap["error"] = "simulated generation error"
	}
	data, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		return err
	}
	if _, err := jsonFile.Write(data); err != nil {
		return err
	}
	if option.libraryID == simulateConfigureErrorID {
		return errors.New("generation failed due to invalid library id")
	}
	return nil
}

func populateAdditionalFields(library *libraryState) *libraryState {
	library.Version = "1.0.0"
	library.SourceRoots = []string{"example-source-path", "example-source-path-2"}
	library.PreserveRegex = []string{"example-preserve-regex", "example-preserve-regex-2"}
	library.RemoveRegex = []string{"example-remove-regex", "example-remove-regex-2"}

	return library
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

type libraryState struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	APIs          []*api   `json:"apis"`
	SourceRoots   []string `json:"source_roots"`
	PreserveRegex []string `json:"preserve_regex"`
	RemoveRegex   []string `json:"remove_regex"`
}

type api struct {
	Path          string `json:"path"`
	ServiceConfig string `json:"service_config"`
}
