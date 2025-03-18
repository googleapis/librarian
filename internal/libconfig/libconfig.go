package libconfig

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
)

type Library struct {
	ID                   string   `json:"id"`
	LatestReleaseVersion string   `json:"latestReleaseVersion"`
	NextReleaseVersion   string   `json:"nextReleaseVersion"`
	ReleaseTimestamp     string   `json:"releaseTimestamp"`
	Apis                 []API    `json:"apis"`
	SourcePaths          []string `json:"sourcePaths"`
}

type API struct {
	ID            string `json:"id"`
	ReleaseCommit string `json:"release_commit"`
}

type Libraries struct {
	Libraries   []Library `json:"libraries"`
	CommonPaths []string  `json:"commonPaths"`
}

func LoadLibraryConfig(configFile string) (*Libraries, error) {
	slog.Info("reading library %s", configFile)
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var libs Libraries
	err = json.Unmarshal(data, &libs)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	return &libs, nil
}
