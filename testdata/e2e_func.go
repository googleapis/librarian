package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	configureRequest       = "configure-request.json"
	configureResponse      = "configure-response.json"
	generateRequest        = "generate-request.json"
	generateResponse       = "generate-response.json"
	releaseInitRequest     = "release-stage-request.json"
	releaseInitResponse    = "release-stage-response.json"
	id                     = "id"
	inputDir               = "input"
	librarian              = "librarian"
	outputDir              = "output"
	repoDir                = "repo"
	simulateCommandErrorID = "simulate-command-error-id"
	source                 = "source"
)

var (
	scopeRegex = regexp.MustCompile(`\[(.*?)\]`)
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if len(os.Args) <= 1 {
		log.Fatal(errors.New("no command-line arguments provided"))
	}

	slog.Info("received command", "args", os.Args[1:])
	switch os.Args[1] {
	case "configure":
		if err := doConfigure(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "generate":
		if err := doGenerate(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "release-stage":
		if err := doReleaseStage(os.Args[2:]); err != nil {
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

func doReleaseStage(args []string) error {
	slog.Debug("doReleaseStage received args", "args", args)
	request, err := parseReleaseStageRequest(args)
	if err != nil {
		return err
	}
	slog.Debug("doReleaseStage received request", "request", request)
	if err := validateLibrarianDir(request.librarianDir, releaseInitRequest); err != nil {
		return err
	}

	state, err := readReleaseStageRequestJSON(filepath.Join(request.librarianDir, releaseInitRequest))
	if err != nil {
		return err
	}

	for i, library := range state.Libraries {
		slog.Debug("Library has SourceRoots", "index", i, "id", library.ID, "source_roots", library.SourceRoots)
	}

	// Update the version of the library.
	for _, library := range state.Libraries {
		if !library.ReleaseTriggered {
			continue
		}
		slog.Info("Found library to update", "id", library.ID)
		slog.Info("Version from request", "version", library.Version)

		// Create a changelog.
		var changelog strings.Builder
		changelog.WriteString(fmt.Sprintf("## %s\n\n", library.Version))
		for _, change := range library.Changes {
			changelog.WriteString(fmt.Sprintf("- %s: %s\n", change.Type, change.Subject))
		}
		for _, sourceRoot := range library.SourceRoots {
			changelogPath := filepath.Join(request.outputDir, sourceRoot, "CHANGELOG.md")
			if err := os.MkdirAll(filepath.Dir(changelogPath), 0755); err != nil {
				return fmt.Errorf("failed to create changelog directory: %w", err)
			}
			if err := os.WriteFile(changelogPath, []byte(changelog.String()), 0644); err != nil {
				return fmt.Errorf("failed to write changelog: %w", err)
			}
			slog.Info("Wrote changelog", "path", changelogPath)
		}

		// After processing, clear the fields for the final state.yaml.
		library.Changes = nil
		library.ReleaseTriggered = false
	}

	slog.Debug("State after update", "state", state)
	updatedStateBytes, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal updated state: %w", err)
	}
	slog.Debug("Marshalled updated state (YAML)", "yaml", string(updatedStateBytes))

	outputStateDir := filepath.Join(request.outputDir, ".librarian")
	if err := os.MkdirAll(outputStateDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputStatePath := filepath.Join(outputStateDir, "state.yaml")
	if err := os.WriteFile(outputStatePath, updatedStateBytes, 0644); err != nil {
		return fmt.Errorf("failed to write updated state.yaml to output: %w", err)
	}

	slog.Info("Wrote updated state.yaml", "path", outputStatePath)

	return writeReleaseStageResponseJSON(request)
}

// readReleaseStageRequestJSON reads the release init request file and creates a librarianState
// object.
func readReleaseStageRequestJSON(path string) (*librarianState, error) {
	state := &librarianState{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	slog.Debug("readReleaseStageRequest: File content", "content", string(data))
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	slog.Debug("readReleaseStageRequest: Unmarshalled state", "state", state)

	// Validate the unmarshalled JSON content.
	if err := validateReleaseStageRequestJSON(state); err != nil {
		slog.Error("readReleaseStageRequest: Validation failed", "error", err)
		return nil, err
	}

	return state, nil
}

type releaseInitOption struct {
	librarianDir string
	repoDir      string
	outputDir    string
}

func parseReleaseStageRequest(args []string) (*releaseInitOption, error) {
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

func writeReleaseStageResponseJSON(option *releaseInitOption) error {
	jsonFilePath := filepath.Join(option.librarianDir, releaseInitResponse)
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
	slog.Debug("about to write to file", "path", jsonFilePath, "data", string(data))
	_, err = jsonFile.Write(data)
	slog.Info("wrote release init response", "path", jsonFilePath)

	return err
}

// validateReleaseStageRequestJSON validates the structure and content of the
// data unmarshalled from the release-stage-request.json file.
// This file is generated by the librarian tool and consumed by this mock container.
// It checks if the structure matches the expected contract and validates that
// commits are correctly associated with their libraries.
func validateReleaseStageRequestJSON(state *librarianState) error {
	if state.Image == "" {
		return errors.New("validation error: missing 'image' in release-stage-request.json")
	}
	if len(state.Libraries) == 0 {
		return errors.New("validation error: no libraries found in release-stage-request.json")
	}

	foundTriggered := false
	for i, lib := range state.Libraries {
		if lib.ID == "" {
			return fmt.Errorf("validation error: library %d missing 'id'", i)
		}
		if !lib.ReleaseTriggered {
			continue
		}
		foundTriggered = true
		slog.Debug("Validating triggered library", "id", lib.ID)

		if lib.Version == "" {
			return fmt.Errorf("validation error: library %s missing 'version'", lib.ID)
		}
		if len(lib.SourceRoots) == 0 {
			return fmt.Errorf("validation error: library %s missing 'source_roots'", lib.ID)
		}

		if len(lib.Changes) == 0 {
			return fmt.Errorf("validation error: no changes found for library %s", lib.ID)
		}
		for j, change := range lib.Changes {
			// Validate fields based on the contract example in language-onboarding.md
			if change.Type == "" {
				return fmt.Errorf("validation error: library %s, change %d missing 'type'", lib.ID, j)
			}
			if change.Subject == "" {
				return fmt.Errorf("validation error: library %s, change %d missing 'subject'", lib.ID, j)
			}
			// body is optional

			// These fields are expected based on the contract
			if change.PiperCLNumber == "" {
				return fmt.Errorf("validation error: library %s, change %d missing 'piper_cl_number'", lib.ID, j)
			}
			if change.CommitHash == "" {
				return fmt.Errorf("validation error: library %s, change %d missing 'commit_hash'", lib.ID, j)
			}

			// Validation logic to check for incorrect commit association.
			matches := scopeRegex.FindStringSubmatch(change.Subject)
			if len(matches) > 1 {
				commitScope := matches[1]
				isAssociated := strings.Contains(lib.ID, commitScope)

				slog.Info("Running validation check", "library_id", lib.ID, "commit_scope", commitScope, "is_correctly_associated", isAssociated)

				if !isAssociated {
					// If the commit scope does not appear in the library ID, it's a mismatch.
					return fmt.Errorf("validation failed: commit with scope [%s] incorrectly associated with library %s", commitScope, lib.ID)
				}
			}
		}
	}

	if !foundTriggered {
		slog.Warn("No library was marked with release_triggered: true in request")
	}
	slog.Debug("validateReleaseStageRequestJSON: Validation passed")
	return nil
}

func parseConfigureRequest(args []string) (*configureOption, error) {
	configureOption := &configureOption{}
	for _, arg := range args {
		option, _ := strings.CutPrefix(arg, "--")
		strs := strings.Split(option, "=")
		switch strs[0] {
		case inputDir:
			configureOption.inputDir = strs[1]
		case outputDir:
			configureOption.outputDir = strs[1]
		case librarian:
			configureOption.librarianDir = strs[1]
		case repoDir:
			configureOption.repoDir = strs[1]
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
			generateOption.inputDir = strs[1]
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
		slog.Info("write configure response", "path", jsonFilePath)
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
	slog.Info("write generate response", "path", jsonFilePath)

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
		slog.Info("create file in", "path", srcPath)
	}

	return nil
}

type configureOption struct {
	inputDir     string
	outputDir    string
	librarianDir string
	repoDir      string
	sourceDir    string
}

type generateOption struct {
	inputDir     string
	outputDir    string
	librarianDir string
	sourceDir    string
}

type librarianState struct {
	Image     string          `json:"image" yaml:"image"`
	Libraries []*libraryState `json:"libraries" yaml:"libraries"`
}

type libraryState struct {
	ID                  string    `json:"id" yaml:"id"`
	Version             string    `json:"version" yaml:"version"`
	APIs                []*api    `json:"apis,omitempty" yaml:"apis,omitempty"`
	SourceRoots         []string  `json:"source_roots" yaml:"source_roots"`
	PreserveRegex       []string  `json:"preserve_regex,omitempty" yaml:"preserve_regex,omitempty"`
	RemoveRegex         []string  `json:"remove_regex,omitempty" yaml:"remove_regex,omitempty"`
	ReleaseTriggered    bool      `json:"release_triggered,omitempty" yaml:"release_triggered,omitempty"`
	Changes             []*change `json:"changes,omitempty" yaml:"changes,omitempty"`
	LastGeneratedCommit string    `json:"last_generated_commit,omitempty" yaml:"last_generated_commit,omitempty"`
	TagFormat           string    `json:"tag_format,omitempty" yaml:"tag_format"`
	ReleaseExcludePaths []string  `json:"release_exclude_paths,omitempty" yaml:"release_exclude_paths,omitempty"`
}

type api struct {
	Path          string `json:"path" yaml:"path"`
	ServiceConfig string `json:"service_config" yaml:"service_config"`
	Status        string `json:"status" yaml:"status"`
}

type change struct {
	Type          string `json:"type" yaml:"type"`
	Subject       string `json:"subject" yaml:"subject"`
	Body          string `json:"body" yaml:"body"`
	PiperCLNumber string `json:"piper_cl_number" yaml:"piper_cl_number"`
	CommitHash    string `json:"commit_hash" yaml:"commit_hash"`
}
