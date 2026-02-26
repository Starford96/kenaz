package internal

// Option is a functional option for configuring the application.
type Option func(*application)

type application struct {
	config *Config
}

// WithConfig sets the application configuration.
func WithConfig(cfg *Config) Option {
	return func(a *application) {
		a.config = cfg
	}
}
