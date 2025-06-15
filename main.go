package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
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
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")
			return startServer(port)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func startServer(port int) error {
	// Create context that listens for interrupt signals (記事の推奨パターン)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server := NewServer(port)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for either interrupt signal or API shutdown request
	select {
	case err := <-serverErr:
		// Server stopped due to error
		if err != nil {
			log.Printf("Server error: %v", err)
		}
		return err
	case <-ctx.Done():
		// OS interrupt signal received
		log.Println("Received interrupt signal, shutting down server gracefully...")
	case <-server.ShutdownChannel():
		// API shutdown request received
		log.Println("Received API shutdown request, shutting down server gracefully...")
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Perform graceful shutdown
	return server.Shutdown(shutdownCtx)
}
