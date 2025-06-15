package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/tenkoh/s3c/pkg/logger"
	"github.com/urfave/cli/v2"
)

func main() {
	// Initialize logger early
	log := logger.NewDefaultLogger()
	mainLogger := logger.WithComponent(log, "main")

	app := &cli.App{
		Name:  "s3c",
		Usage: "S3 and S3 compatible object storage Client working locally",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   8080,
				Usage:   "Port to serve the web interface",
			},
			&cli.StringFlag{
				Name:  "log-level",
				Value: "info",
				Usage: "Log level (debug, info, warn, error)",
			},
			&cli.StringFlag{
				Name:  "log-format",
				Value: "json",
				Usage: "Log format (text, json)",
			},
		},
		Action: func(c *cli.Context) error {
			// Create logger with CLI options
			config := logger.LoggerConfig{
				Level:  c.String("log-level"),
				Format: c.String("log-format"),
				Output: "stdout",
			}
			appLogger := logger.NewLogger(config)

			port := c.Int("port")
			mainLogger.Info("Starting s3c application",
				"port", port,
				"logLevel", config.Level,
				"logFormat", config.Format,
			)

			return startServer(port, appLogger)
		},
	}

	if err := app.Run(os.Args); err != nil {
		mainLogger.Error("Application failed to start", "error", err)
		os.Exit(1)
	}
}

func startServer(port int, appLogger *slog.Logger) error {
	serverLogger := logger.WithComponent(appLogger, "server")

	// Create context that listens for interrupt signals (記事の推奨パターン)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server := NewServer(port, appLogger)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	serverLogger.Info("Server startup initiated", "port", port)

	// Wait for either interrupt signal or API shutdown request
	select {
	case err := <-serverErr:
		// Server stopped due to error
		if err != nil {
			serverLogger.Error("Server stopped with error", "error", err)
		}
		return err
	case <-ctx.Done():
		// OS interrupt signal received
		serverLogger.Info("Received interrupt signal, initiating graceful shutdown")
	case <-server.ShutdownChannel():
		// API shutdown request received
		serverLogger.Info("Received API shutdown request, initiating graceful shutdown")
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverLogger.Info("Starting graceful shutdown", "timeout", "5s")

	// Perform graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		serverLogger.Error("Error during graceful shutdown", "error", err)
		return err
	}

	serverLogger.Info("Server shutdown completed successfully")
	return nil
}
