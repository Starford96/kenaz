// Package config provides YAML-based configuration loading with environment variable expansion.
package config

import (
	"errors"
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

// LoadWithDefaults loads configuration with fallback to a default file.
func LoadWithDefaults[T any](filename, defaultFile string, target *T) error {
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		if defaultFile != "" {
			return Load(defaultFile, target)
		}
		return fmt.Errorf("config file not found: %s", filename)
	}
	return Load(filename, target)
}

// MustLoad loads configuration and panics on failure.
func MustLoad[T any](filename string, target *T) {
	if err := Load(filename, target); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
}
