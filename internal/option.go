package internal

import "log/slog"

// Option is a functional option for configuring the application.
type Option func(*application)

type application struct {
	config *Config
	logger *slog.Logger
}

// WithConfig sets the application configuration.
func WithConfig(cfg *Config) Option {
	return func(a *application) {
		a.config = cfg
	}
}

// WithLogger sets the application logger.
func WithLogger(logger *slog.Logger) Option {
	return func(a *application) {
		a.logger = logger
	}
}
