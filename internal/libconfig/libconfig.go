package libconfig

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
)

// ApiGenerationState represents the structure of each item in the "apiGenerationStates" array.
type ApiGenerationState struct {
	ID                  string `json:"id"`
	LastGeneratedCommit string `json:"lastGeneratedCommit"`
	AutomationLevel     string `json:"automationLevel"`
}

// Api represents the structure of each item in the "apis" array within LibraryReleaseState.
type Api struct {
	ApiID              string `json:"apiId"`
	LastReleasedCommit string `json:"lastReleasedCommit"`
}

// LibraryReleaseState represents the structure of each item in the "libraryReleaseStates" array.
type LibraryReleaseState struct {
	ID                 string   `json:"id"`
	CurrentVersion     string   `json:"currentVersion"`
	NextReleaseVersion string   `json:"nextReleaseVersion"`
	AutomationLevel    string   `json:"automationLevel"`
	ReleaseTimestamp   string   `json:"releaseTimestamp"`
	Apis               []Api    `json:"apis"`
	SourcePaths        []string `json:"sourcePaths"`
}

// Data represents the overall structure of the JSON data.
type PipelineData struct {
	ImageTag                 string                `json:"imageTag"`
	ApiGenerationStates      []ApiGenerationState  `json:"apiGenerationStates"`
	LibraryReleaseStates     []LibraryReleaseState `json:"libraryReleaseStates"`
	CommonLibrarySourcePaths []string              `json:"common_library_source_paths"`
}

func LoadLibraryConfig(configFile string) ([]LibraryReleaseState, error) {
	slog.Info("reading library %s", configFile)
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var pipelineData PipelineData
	err = json.Unmarshal(data, &pipelineData)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	return pipelineData.LibraryReleaseStates, nil
}
