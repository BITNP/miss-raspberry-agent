// Command server is the service's main entry point: it loads configuration and
// delegates wiring and lifecycle to the internal/app composition root.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"miss-raspberry-agent/internal/app"
	"miss-raspberry-agent/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load .env if it exists; ignore it otherwise (env vars are injected by the deployment environment).
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return app.Run(ctx, cfg)
}
