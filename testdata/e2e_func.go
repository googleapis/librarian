package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	configureRequest       = "configure-request.json"
	configureResponse      = "configure-response.json"
	generateRequest        = "generate-request.json"
	generateResponse       = "generate-response.json"
	id                     = "id"
	inputDir               = "input"
	librarian              = "librarian"
	outputDir              = "output"
	simulateCommandErrorID = "simulate-command-error-id"
	source                 = "source"
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
	case "release-init":
		if err := doReleaseInit(os.Args[2:]); err != nil {
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

	state, err := readConfigureRequest(filepath.Join(request.librarianDir, configureRequest))
	if err != nil {
		return err
	}

	return writeConfigureResponse(request, state)
}

func doGenerate(args []string) error {
	request, err := parseGenerateOption(args)
	if err != nil {
		return err
	}
	if err := validateLibrarianDir(request.librarianDir, generateRequest); err != nil {
		return err
	}

	library, err := readGenerateRequest(filepath.Join(request.librarianDir, generateRequest))
	if err != nil {
		return err
	}

	if err := generateLibrary(library, request.outputDir); err != nil {
		return fmt.Errorf("failed to generate library %s: %w", library.ID, err)
	}

	return writeGenerateResponse(request)
}

func doReleaseInit(args []string) error {
	request, err := parseReleaseInitRequest(args)
	if err != nil {
		return err
	}
	if err := validateLibrarianDir(request.librarianDir, "release-init-request.json"); err != nil {
		// The request file is not created for release-init, so we don't validate it.
		// return err
	}

	// Read the state.yaml file.
	stateFilePath := filepath.Join(request.repoDir, ".librarian", "state.yaml")
	stateBytes, err := os.ReadFile(stateFilePath)
	if err != nil {
		return fmt.Errorf("failed to read state.yaml: %w", err)
	}
	var state librarianState
	if err := yaml.Unmarshal(stateBytes, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state.yaml: %w", err)
	}

	// Update the version of the library.
	for _, library := range state.Libraries {
		if library.ID == request.libraryID {
			library.Version = request.libraryVersion
			break
		}
	}

	// Write the updated state back to state.yaml.
	updatedStateBytes, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal updated state: %w", err)
	}
	if err := os.WriteFile(stateFilePath, updatedStateBytes, 0644); err != nil {
		return fmt.Errorf("failed to write updated state.yaml: %w", err)
	}

	// The release-init command in the fake image doesn't need to do anything,
	// as the e2e test only checks if the command is called correctly.
	// We just need to create an empty response file.
	return writeReleaseInitResponse(request)
}

type releaseInitOption struct {
	librarianDir string
	repoDir      string
	outputDir    string
}

func parseReleaseInitRequest(args []string) (*releaseInitOption, error) {
	option := &releaseInitOption{}
	for _, arg := range args {
		opt, _ := strings.CutPrefix(arg, "--")
		strs := strings.Split(opt, "=")
		switch strs[0] {
		case "librarian":
			option.librarianDir = strs[1]
		case "repo":
			option.repoDir = strs[1]
		case "output":
			option.outputDir = strs[1]
		default:
			return nil, errors.New("unrecognized option: " + opt)
		}
	}
	return option, nil
}

func writeReleaseInitResponse(option *releaseInitOption) error {
	jsonFilePath := filepath.Join(option.librarianDir, "release-init-response.json")
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	dataMap := map[string]string{}
	data, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		return err
	}
	_, err = jsonFile.Write(data)
	log.Print("write release init response to " + jsonFilePath)

	return err
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

func parseGenerateOption(args []string) (*generateOption, error) {
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

// readConfigureRequest reads the configure request file and creates a librarianState
// object.
func readConfigureRequest(path string) (*librarianState, error) {
	state := &librarianState{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, err
	}

	for _, library := range state.Libraries {
		if library.ID == simulateCommandErrorID {
			return nil, errors.New("simulate command error")
		}
	}

	return state, nil
}

func writeConfigureResponse(option *configureOption, state *librarianState) error {
	for _, library := range state.Libraries {
		needConfigure := false
		for _, oneAPI := range library.APIs {
			if oneAPI.Status == "new" {
				needConfigure = true
			}
		}

		if !needConfigure {
			continue
		}

		populateAdditionalFields(library)
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
	}

	return nil
}

// readGenerateRequest reads the generate request file and creates a libraryState
// object.
func readGenerateRequest(path string) (*libraryState, error) {
	library := &libraryState{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, library); err != nil {
		return nil, err
	}

	if library.ID == simulateCommandErrorID {
		// Simulate a command error
		return nil, errors.New("simulate command error")
	}

	return library, nil
}

func writeGenerateResponse(option *generateOption) (err error) {
	jsonFilePath := filepath.Join(option.librarianDir, generateResponse)
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, jsonFile.Close())
	}()

	dataMap := map[string]string{}
	data, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		return err
	}
	_, err = jsonFile.Write(data)
	log.Print("write generate response to " + jsonFilePath)

	return err
}

func populateAdditionalFields(library *libraryState) {
	library.Version = "1.0.0"
	library.SourceRoots = []string{"example-source-path", "example-source-path-2"}
	library.PreserveRegex = []string{"example-preserve-regex", "example-preserve-regex-2"}
	library.RemoveRegex = []string{"example-remove-regex", "example-remove-regex-2"}
	for _, oneAPI := range library.APIs {
		oneAPI.Status = "existing"
	}
}

// generateLibrary creates files in sourceDir.
func generateLibrary(library *libraryState, outputDir string) error {
	for _, src := range library.SourceRoots {
		srcPath := filepath.Join(outputDir, src)
		if err := os.MkdirAll(srcPath, 0755); err != nil {
			return err
		}
		if _, err := os.Create(filepath.Join(srcPath, "example.txt")); err != nil {
			return err
		}
		log.Print("create file in " + srcPath)
	}

	return nil
}

type configureOption struct {
	intputDir    string
	librarianDir string
	sourceDir    string
}

type generateOption struct {
	intputDir    string
	outputDir    string
	librarianDir string
	sourceDir    string
}

type librarianState struct {
	Image     string          `json:"image"`
	Libraries []*libraryState `json:"libraries"`
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
	Status        string `json:"status"`
}
