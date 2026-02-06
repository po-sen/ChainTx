package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"chaintx/internal/bootstrap"
	"chaintx/internal/bootstrap/di"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	cfg := bootstrap.LoadConfig()

	container := di.Build(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- container.Server.Start()
	}()

	select {
	case err := <-serverErrCh:
		if err != nil {
			logger.Printf("server startup failed: %v", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := container.Server.Shutdown(shutdownCtx); err != nil {
			logger.Printf("graceful shutdown failed: %v", err)
			os.Exit(1)
		}

		if err := <-serverErrCh; err != nil {
			logger.Printf("server stopped with error: %v", err)
			os.Exit(1)
		}

		logger.Printf("server stopped")
	}
}
