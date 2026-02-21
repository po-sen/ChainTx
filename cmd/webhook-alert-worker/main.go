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
	if alertCfgErr := validateWebhookAlertWorkerConfig(cfg); alertCfgErr != nil {
		logger.Printf(
			"webhook alert worker config error code=%s message=%s metadata=%v",
			alertCfgErr.Code,
			alertCfgErr.Message,
			alertCfgErr.Metadata,
		)
		os.Exit(1)
	}

	container, buildErr := di.BuildWebhookAlertWorker(cfg, logger)
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

	logger.Printf("webhook alert worker persistence initialization starting database_target=%s", cfg.DatabaseTarget)
	persistenceErr := container.InitializePersistenceUseCase.Execute(ctx, dto.InitializePersistenceCommand{
		ReadinessTimeout:       cfg.DBReadinessTimeout,
		ReadinessRetryInterval: cfg.DBReadinessRetryInterval,
	})
	if persistenceErr != nil {
		logger.Printf(
			"webhook alert worker persistence initialization failed code=%s message=%s metadata=%v",
			persistenceErr.Code,
			persistenceErr.Message,
			persistenceErr.Details,
		)
		os.Exit(1)
	}
	logger.Printf("webhook alert worker persistence initialization completed database_target=%s", cfg.DatabaseTarget)

	if container.WebhookAlertWorker == nil || !container.WebhookAlertWorker.Enabled() {
		logger.Printf("webhook alert worker startup failed code=WEBHOOK_ALERT_WORKER_NOT_ENABLED message=webhook alert worker is not enabled")
		os.Exit(1)
	}

	go container.WebhookAlertWorker.Start(ctx)
	<-ctx.Done()
	logger.Printf("webhook alert worker stopped")
}

func validateWebhookAlertWorkerConfig(cfg config.Config) *config.ConfigError {
	if !cfg.WebhookAlertEnabled {
		return &config.ConfigError{
			Code:    "CONFIG_WEBHOOK_ALERT_DISABLED",
			Message: "PAYMENT_REQUEST_WEBHOOK_ALERT_ENABLED must be true for webhook alert worker runtime",
		}
	}

	return nil
}
