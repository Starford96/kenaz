// Package config provides YAML-based configuration loading with environment variable expansion.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Validator is an interface for configuration validation.
type Validator interface {
	Validate() error
}

// Load loads configuration from a YAML file with environment variable expansion.
func Load[T any](filename string, target *T) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	expandedData := os.ExpandEnv(string(data))

	if err := yaml.Unmarshal([]byte(expandedData), target); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}

	if validator, ok := any(target).(Validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}
	}

	return nil
}
