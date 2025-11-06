package config

type AutomationConfig struct {
	Build    bool
	Command  string
	ForceRun bool
	Project  string
	Push     bool
}
