package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/starford/kenaz/internal"
	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/mcpserver"
	"github.com/starford/kenaz/internal/storage"
	pkgconfig "github.com/starford/kenaz/pkg/config"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v3"
)

func loadConfig(cmd *cli.Command) (*internal.Config, error) {
	configPath := cmd.String("config")
	cfg := internal.NewDefaultConfig()
	if err := pkgconfig.Load(configPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

func runServe(ctx context.Context, cmd *cli.Command) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	opts := []internal.Option{internal.WithConfig(cfg)}
	if err := internal.Run(ctx, opts...); err != nil {
		return fmt.Errorf("app run error: %w", err)
	}
	return nil
}

func runMCP(ctx context.Context, cmd *cli.Command) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	// Ensure vault exists.
	if err := os.MkdirAll(cfg.Vault.Path, 0o755); err != nil {
		return fmt.Errorf("create vault dir: %w", err)
	}

	store, err := storage.NewFS(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	db, err := index.Open(cfg.SQLite.Path)
	if err != nil {
		return fmt.Errorf("init index: %w", err)
	}
	defer db.Close()

	// Sync index before serving MCP.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	index.Sync(db, store, logger)

	srv := mcpserver.New(store, db)
	return srv.ServeStdio()
}

var configFlag = &cli.StringFlag{
	Name:        "config",
	Aliases:     []string{"c"},
	Usage:       "Path to config file",
	DefaultText: "config/config.yaml",
	Value:       "config/config.yaml",
	Sources:     cli.EnvVars("APP_CONFIG_FILE"),
}

//	@title			Kenaz API
//	@version		1.0.0
//	@description	Local-first knowledge base with Markdown storage, full-text search, and graph visualization.
//	@BasePath		/api
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
func main() {
	cmd := &cli.Command{
		Name:   "kenaz",
		Usage:  "Local-first knowledge base with Markdown storage, full-text search, and graph visualization",
		Action: runServe,
		Flags:  []cli.Flag{configFlag},
		Commands: []*cli.Command{
			{
				Name:   "serve",
				Usage:  "Start the HTTP server (default)",
				Action: runServe,
				Flags:  []cli.Flag{configFlag},
			},
			{
				Name:   "mcp",
				Usage:  "Start the MCP server on stdio for LLM integration",
				Action: runMCP,
				Flags:  []cli.Flag{configFlag},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("application error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
