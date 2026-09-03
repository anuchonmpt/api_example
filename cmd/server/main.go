package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/api-example/internal/app"
	"github.com/example/api-example/internal/config"
	starterlogger "github.com/example/api-example/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(1)
	}
	logger, err := starterlogger.New(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		log.Printf("logging error: %v", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.RunServer(ctx, cfg, logger); err != nil {
		logger.WithError(err).Error("server stopped")
		os.Exit(1)
	}
}
