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
	if !cfg.ReconcilerEnabled {
		logger.Printf("reconciler config error code=CONFIG_RECONCILER_DISABLED message=PAYMENT_REQUEST_RECONCILER_ENABLED must be true for reconciler runtime")
		os.Exit(1)
	}

	container, buildErr := di.Build(cfg, logger)
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

	logger.Printf("reconciler persistence initialization starting database_target=%s", cfg.DatabaseTarget)
	persistenceErr := container.InitializePersistenceUseCase.Execute(ctx, dto.InitializePersistenceCommand{
		ReadinessTimeout:       cfg.DBReadinessTimeout,
		ReadinessRetryInterval: cfg.DBReadinessRetryInterval,
	})
	if persistenceErr != nil {
		logger.Printf(
			"reconciler persistence initialization failed code=%s message=%s metadata=%v",
			persistenceErr.Code,
			persistenceErr.Message,
			persistenceErr.Details,
		)
		os.Exit(1)
	}
	logger.Printf("reconciler persistence initialization completed database_target=%s", cfg.DatabaseTarget)

	if container.ReconcilerWorker == nil || !container.ReconcilerWorker.Enabled() {
		logger.Printf("reconciler startup failed code=RECONCILER_WORKER_NOT_ENABLED message=reconciler worker is not enabled")
		os.Exit(1)
	}

	container.ReconcilerWorker.Start(ctx)
	logger.Printf("reconciler stopped")
}
