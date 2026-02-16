package internal

import (
	"fmt"
	"log/slog"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Config represents the application configuration.
type Config struct {
	App    ApplicationConfig `yaml:"app"`
	Vault  VaultConfig       `yaml:"vault"`
	SQLite SQLiteConfig      `yaml:"sqlite"`
	Auth   AuthConfig        `yaml:"auth"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if err := c.App.Validate(); err != nil {
		return err
	}
	if err := c.Vault.Validate(); err != nil {
		return err
	}
	return c.SQLite.Validate()
}

// ApplicationConfig holds application-level configuration.
type ApplicationConfig struct {
	LogLevel slog.Level `yaml:"log_level"`
	HTTP     HTTPConfig `yaml:"http"`
}

// Validate validates the application configuration.
func (c *ApplicationConfig) Validate() error {
	return c.HTTP.Validate()
}

// HTTPConfig holds HTTP server configuration.
type HTTPConfig struct {
	Port int `yaml:"port"`
}

// Address returns HTTP server address.
func (c *HTTPConfig) Address() string {
	return fmt.Sprintf(":%d", c.Port)
}

// Validate validates the HTTP configuration.
func (c *HTTPConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Port, validation.Required, validation.Min(1), validation.Max(65535)),
	)
}

// VaultConfig holds the path to the Markdown vault directory.
type VaultConfig struct {
	Path string `yaml:"path"`
}

// Validate validates the vault configuration.
func (c *VaultConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Path, validation.Required),
	)
}

// SQLiteConfig holds SQLite database configuration.
type SQLiteConfig struct {
	Path string `yaml:"path"`
}

// Validate validates the SQLite configuration.
func (c *SQLiteConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Path, validation.Required),
	)
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Token string `yaml:"token"`
}

// NewDefaultConfig returns a new Config with sensible default values.
func NewDefaultConfig() *Config {
	return &Config{
		App: ApplicationConfig{
			LogLevel: slog.LevelInfo,
			HTTP: HTTPConfig{
				Port: 8080,
			},
		},
		Vault: VaultConfig{
			Path: "./vault",
		},
		SQLite: SQLiteConfig{
			Path: "./kenaz.db",
		},
	}
}
