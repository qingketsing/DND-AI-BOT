package soak

import (
	"encoding/json"
	"os"
)

// LoadConfig reads a JSON soak config and expands ${ENV} placeholders.
func LoadConfig(path string) (SoakConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SoakConfig{}, err
	}
	expanded := os.ExpandEnv(string(raw))

	var config SoakConfig
	if err := json.Unmarshal([]byte(expanded), &config); err != nil {
		return SoakConfig{}, err
	}
	return config, nil
}
