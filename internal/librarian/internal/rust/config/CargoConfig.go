package config

const (
	configName = "Cargo.toml"
)

// CargoConfig represents relevant fields from Cargo.toml.
type CargoConfig struct {
	Package struct {
		Name        string      `toml:"name"`
		Version     string      `toml:"version"`
		Publish     interface{} `toml:"publish"` // Can be bool or array of strings
		Description string      `toml:"description"`
	} `toml:"package"`

	Dependencies map[string]string `toml:"dependencies"`
}
