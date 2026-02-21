package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
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
	if dispatcherCfgErr := validateWebhookDispatcherConfig(cfg); dispatcherCfgErr != nil {
		logger.Printf(
			"webhook dispatcher config error code=%s message=%s metadata=%v",
			dispatcherCfgErr.Code,
			dispatcherCfgErr.Message,
			dispatcherCfgErr.Metadata,
		)
		os.Exit(1)
	}

	container, buildErr := di.BuildWebhookDispatcher(cfg, logger)
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

	logger.Printf("webhook dispatcher persistence initialization starting database_target=%s", cfg.DatabaseTarget)
	persistenceErr := container.InitializePersistenceUseCase.Execute(ctx, dto.InitializePersistenceCommand{
		ReadinessTimeout:       cfg.DBReadinessTimeout,
		ReadinessRetryInterval: cfg.DBReadinessRetryInterval,
	})
	if persistenceErr != nil {
		logger.Printf(
			"webhook dispatcher persistence initialization failed code=%s message=%s metadata=%v",
			persistenceErr.Code,
			persistenceErr.Message,
			persistenceErr.Details,
		)
		os.Exit(1)
	}
	logger.Printf("webhook dispatcher persistence initialization completed database_target=%s", cfg.DatabaseTarget)

	if container.WebhookWorker == nil || !container.WebhookWorker.Enabled() {
		logger.Printf("webhook dispatcher startup failed code=WEBHOOK_WORKER_NOT_ENABLED message=webhook worker is not enabled")
		os.Exit(1)
	}

	go container.WebhookWorker.Start(ctx)
	<-ctx.Done()
	logger.Printf("webhook dispatcher stopped")
}

func validateWebhookDispatcherConfig(cfg config.Config) *config.ConfigError {
	if !cfg.WebhookEnabled {
		return &config.ConfigError{
			Code:    "CONFIG_WEBHOOK_DISABLED",
			Message: "PAYMENT_REQUEST_WEBHOOK_ENABLED must be true for webhook dispatcher runtime",
		}
	}

	if strings.TrimSpace(cfg.WebhookHMACSecret) == "" {
		return &config.ConfigError{
			Code:    "CONFIG_WEBHOOK_HMAC_SECRET_REQUIRED",
			Message: "PAYMENT_REQUEST_WEBHOOK_HMAC_SECRET is required when webhook dispatcher runtime is enabled",
		}
	}

	return nil
}
