package libconfig

import (
	"encoding/json"
	"os"
)

type LibraryConfig struct {
	Paths []string `json:"paths"`
}

func LoadLibraryConfig(configFile string) (map[string]LibraryConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var libMap map[string]LibraryConfig
	err = json.Unmarshal(data, &libMap)
	if err != nil {
		return nil, err
	}

	return libMap, nil
}
