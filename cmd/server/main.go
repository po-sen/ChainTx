package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"chaintx/internal/application/dto"
	"chaintx/internal/infrastructure/config"
	"chaintx/internal/infrastructure/di"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	cfg, cfgErr := config.LoadConfig()
	if cfgErr != nil {
		logger.Printf("startup config error code=%s message=%s metadata=%v", cfgErr.Code, cfgErr.Message, cfgErr.Metadata)
		os.Exit(1)
	}
	logger.Printf(
		"wallet allocation config mode=%s devtest_allow_mainnet=%t",
		cfg.AllocationMode,
		cfg.DevtestAllowMainnet,
	)

	container, buildErr := di.BuildServer(cfg, logger)
	if buildErr != nil {
		logger.Printf("dependency wiring error: %v", buildErr)
		os.Exit(1)
	}
	defer func() {
		if container.Database == nil {
			return
		}
		if err := container.Database.Close(); err != nil {
			logger.Printf("database close warning error=%v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Printf("persistence initialization starting database_target=%s", cfg.DatabaseTarget)
	persistenceErr := container.InitializePersistenceUseCase.Execute(ctx, dto.InitializePersistenceCommand{
		ReadinessTimeout:       cfg.DBReadinessTimeout,
		ReadinessRetryInterval: cfg.DBReadinessRetryInterval,
	})
	if persistenceErr != nil {
		logger.Printf(
			"persistence initialization failed code=%s message=%s metadata=%v",
			persistenceErr.Code,
			persistenceErr.Message,
			persistenceErr.Details,
		)
		os.Exit(1)
	}
	logger.Printf("persistence initialization completed database_target=%s", cfg.DatabaseTarget)

	if container.ReconcilerWorker != nil && container.ReconcilerWorker.Enabled() {
		go container.ReconcilerWorker.Start(ctx)
	}

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
