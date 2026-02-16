package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/starford/kenaz/internal"
	pkgconfig "github.com/starford/kenaz/pkg/config"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v3"
)

func run(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")

	cfg := internal.NewDefaultConfig()
	if err := pkgconfig.Load(configPath, cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	opts := []internal.Option{
		internal.WithConfig(cfg),
	}

	if err := internal.Run(ctx, opts...); err != nil {
		return fmt.Errorf("app run error: %w", err)
	}

	return nil
}

func main() {
	cmd := &cli.Command{
		Name:   "kenaz",
		Usage:  "Local-first knowledge base with Markdown storage, full-text search, and graph visualization",
		Action: run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Path to config file",
				DefaultText: "config/config.yaml",
				Value:       "config/config.yaml",
				Sources:     cli.EnvVars("APP_CONFIG_FILE"),
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("application error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
