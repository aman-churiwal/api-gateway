package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port     string          `json:"port"`
	Services []ServiceConfig `json: "services"`
}

type ServiceConfig struct {
	Path   string `json: "path"`
	Target string `json: "target"`
}

func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
