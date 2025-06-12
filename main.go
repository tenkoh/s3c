package main

import (
	"log"
	"os"

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
	server := NewServer(port)
	return server.Start()
}
