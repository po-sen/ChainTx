package di

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"

	"chaintx/internal/adapters/inbound/http/controllers"
	httpRouter "chaintx/internal/adapters/inbound/http/router"
	chainobserverdevtest "chaintx/internal/adapters/outbound/chainobserver/devtest"
	"chaintx/internal/adapters/outbound/docs"
	postgresqlassetcatalog "chaintx/internal/adapters/outbound/persistence/postgresql/assetcatalog"
	postgresqlbootstrap "chaintx/internal/adapters/outbound/persistence/postgresql/bootstrap"
	postgresqlpaymentrequest "chaintx/internal/adapters/outbound/persistence/postgresql/paymentrequest"
	postgresqlshared "chaintx/internal/adapters/outbound/persistence/postgresql/shared"
	postgresqlwebhookoutbox "chaintx/internal/adapters/outbound/persistence/postgresql/webhookoutbox"
	devtestwallet "chaintx/internal/adapters/outbound/wallet/devtest"
	prodwallet "chaintx/internal/adapters/outbound/wallet/prod"
	webhookhttp "chaintx/internal/adapters/outbound/webhook/http"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	"chaintx/internal/application/use_cases"
	"chaintx/internal/infrastructure/config"
	"chaintx/internal/infrastructure/httpserver"
	"chaintx/internal/infrastructure/reconciler"
	"chaintx/internal/infrastructure/webhook"
	"chaintx/internal/infrastructure/webhookalert"
)

type ServerContainer struct {
	Database                     *sql.DB
	Server                       *httpserver.Server
	InitializePersistenceUseCase portsin.InitializePersistenceUseCase
	ReconcilerWorker             *reconciler.Worker
}

type ReconcilerContainer struct {
	Database                     *sql.DB
	InitializePersistenceUseCase portsin.InitializePersistenceUseCase
	ReconcilerWorker             *reconciler.Worker
}

type WebhookDispatcherContainer struct {
	Database                     *sql.DB
	InitializePersistenceUseCase portsin.InitializePersistenceUseCase
	WebhookWorker                *webhook.Worker
}

type WebhookAlertWorkerContainer struct {
	Database                     *sql.DB
	InitializePersistenceUseCase portsin.InitializePersistenceUseCase
	WebhookAlertWorker           *webhookalert.Worker
}

type runtimeDependencies struct {
	databasePool                 *sql.DB
	initializePersistenceUseCase portsin.InitializePersistenceUseCase
}

type WalletGatewayBuilder func(cfg config.Config, logger *log.Logger) portsout.WalletAllocationGateway

var walletGatewayBuilders = map[string]WalletGatewayBuilder{
	"devtest": func(cfg config.Config, logger *log.Logger) portsout.WalletAllocationGateway {
		return devtestwallet.NewGateway(devtestwallet.Config{
			AllowMainnet: cfg.DevtestAllowMainnet,
			Keysets:      cfg.DevtestKeysets,
		}, logger)
	},
	"prod": func(_ config.Config, _ *log.Logger) portsout.WalletAllocationGateway {
		return prodwallet.NewGateway()
	},
}

var walletGatewayBuildersMu sync.RWMutex

func RegisterWalletGatewayBuilder(mode string, builder WalletGatewayBuilder) {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" || builder == nil {
		return
	}

	walletGatewayBuildersMu.Lock()
	defer walletGatewayBuildersMu.Unlock()
	walletGatewayBuilders[normalizedMode] = builder
}

func BuildServer(cfg config.Config, logger *log.Logger) (ServerContainer, error) {
	runtimeDeps := buildRuntimeDependencies(cfg, logger)

	walletGateway, buildErr := buildWalletGateway(cfg, logger)
	if buildErr != nil {
		return ServerContainer{}, buildErr
	}

	healthUseCase := use_cases.NewGetHealthUseCase()
	openAPIReadModel := docs.NewFileOpenAPISpecReadModel(cfg.OpenAPISpecPath)
	openAPIUseCase := use_cases.NewGetOpenAPISpecUseCase(openAPIReadModel)
	assetCatalogReadModel := postgresqlassetcatalog.NewReadModel(runtimeDeps.databasePool)
	paymentRequestRepository := newPaymentRequestRepository(runtimeDeps.databasePool, cfg, logger)
	paymentRequestReadModel := postgresqlpaymentrequest.NewReadModel(runtimeDeps.databasePool)
	webhookOutboxRepository := postgresqlwebhookoutbox.NewRepository(runtimeDeps.databasePool)
	chainObserverGateway := buildChainObserverGateway(cfg)

	listAssetsUseCase := use_cases.NewListAssetsUseCase(assetCatalogReadModel)
	createPaymentRequestUseCase := use_cases.NewCreatePaymentRequestUseCase(
		assetCatalogReadModel,
		paymentRequestRepository,
		walletGateway,
		use_cases.NewSystemClock(),
		cfg.WebhookURLAllowList,
	)
	getPaymentRequestUseCase := use_cases.NewGetPaymentRequestUseCase(paymentRequestReadModel)
	getWebhookOutboxOverviewUseCase := use_cases.NewGetWebhookOutboxOverviewUseCase(
		webhookOutboxRepository,
	)
	listWebhookDLQEventsUseCase := use_cases.NewListWebhookDLQEventsUseCase(
		webhookOutboxRepository,
	)
	requeueWebhookDLQEventUseCase := use_cases.NewRequeueWebhookDLQEventUseCase(
		webhookOutboxRepository,
	)
	cancelWebhookOutboxEventUseCase := use_cases.NewCancelWebhookOutboxEventUseCase(
		webhookOutboxRepository,
	)
	reconcilePaymentRequestsUseCase := use_cases.NewReconcilePaymentRequestsUseCase(
		paymentRequestRepository,
		chainObserverGateway,
	)
	reconcilerWorker := buildReconcilerWorker(cfg, reconcilePaymentRequestsUseCase, logger)

	healthController := controllers.NewHealthController(healthUseCase, logger)
	swaggerController := controllers.NewSwaggerController(openAPIUseCase, logger)
	assetsController := controllers.NewAssetsController(listAssetsUseCase, logger)
	paymentRequestsController := controllers.NewPaymentRequestsController(
		createPaymentRequestUseCase,
		getPaymentRequestUseCase,
		logger,
	)
	webhookOutboxController := controllers.NewWebhookOutboxController(
		getWebhookOutboxOverviewUseCase,
		listWebhookDLQEventsUseCase,
		requeueWebhookDLQEventUseCase,
		cancelWebhookOutboxEventUseCase,
		cfg.WebhookOpsAdminKeys,
		logger,
	)

	router := httpRouter.New(httpRouter.Dependencies{
		HealthController:          healthController,
		SwaggerController:         swaggerController,
		AssetsController:          assetsController,
		PaymentRequestsController: paymentRequestsController,
		WebhookOutboxController:   webhookOutboxController,
	})

	server := httpserver.New(cfg.Address(), router, logger)

	return ServerContainer{
		Database:                     runtimeDeps.databasePool,
		Server:                       server,
		InitializePersistenceUseCase: runtimeDeps.initializePersistenceUseCase,
		ReconcilerWorker:             reconcilerWorker,
	}, nil
}

func BuildReconciler(cfg config.Config, logger *log.Logger) (ReconcilerContainer, error) {
	runtimeDeps := buildRuntimeDependencies(cfg, logger)
	paymentRequestRepository := newPaymentRequestRepository(runtimeDeps.databasePool, cfg, logger)
	chainObserverGateway := buildChainObserverGateway(cfg)
	reconcilePaymentRequestsUseCase := use_cases.NewReconcilePaymentRequestsUseCase(
		paymentRequestRepository,
		chainObserverGateway,
	)
	reconcilerWorker := buildReconcilerWorker(cfg, reconcilePaymentRequestsUseCase, logger)

	return ReconcilerContainer{
		Database:                     runtimeDeps.databasePool,
		InitializePersistenceUseCase: runtimeDeps.initializePersistenceUseCase,
		ReconcilerWorker:             reconcilerWorker,
	}, nil
}

func BuildWebhookDispatcher(cfg config.Config, logger *log.Logger) (WebhookDispatcherContainer, error) {
	runtimeDeps := buildRuntimeDependencies(cfg, logger)
	webhookOutboxRepository := postgresqlwebhookoutbox.NewRepository(runtimeDeps.databasePool)
	webhookEventGateway := webhookhttp.NewGateway(webhookhttp.Config{
		HMACSecret: cfg.WebhookHMACSecret,
		Timeout:    cfg.WebhookTimeout,
	})
	dispatchWebhookEventsUseCase := use_cases.NewDispatchWebhookEventsUseCase(
		webhookOutboxRepository,
		webhookEventGateway,
	)
	webhookWorker := webhook.NewWorker(
		cfg.WebhookEnabled,
		cfg.WebhookPollInterval,
		cfg.WebhookBatchSize,
		cfg.WebhookWorkerID,
		cfg.WebhookLeaseDuration,
		cfg.WebhookInitialBackoff,
		cfg.WebhookMaxBackoff,
		cfg.WebhookRetryJitterBPS,
		cfg.WebhookRetryBudget,
		dispatchWebhookEventsUseCase,
		logger,
	)

	return WebhookDispatcherContainer{
		Database:                     runtimeDeps.databasePool,
		InitializePersistenceUseCase: runtimeDeps.initializePersistenceUseCase,
		WebhookWorker:                webhookWorker,
	}, nil
}

func BuildWebhookAlertWorker(cfg config.Config, logger *log.Logger) (WebhookAlertWorkerContainer, error) {
	runtimeDeps := buildRuntimeDependencies(cfg, logger)
	webhookOutboxRepository := postgresqlwebhookoutbox.NewRepository(runtimeDeps.databasePool)
	getWebhookOutboxOverviewUseCase := use_cases.NewGetWebhookOutboxOverviewUseCase(
		webhookOutboxRepository,
	)
	alertWorker := webhookalert.NewWorker(
		cfg.WebhookAlertEnabled,
		cfg.WebhookPollInterval,
		cfg.WebhookWorkerID,
		getWebhookOutboxOverviewUseCase,
		webhookalert.AlertConfig{
			Enabled:                  cfg.WebhookAlertEnabled,
			Cooldown:                 cfg.WebhookAlertCooldown,
			FailedCountThreshold:     cfg.WebhookAlertFailedCount,
			PendingReadyThreshold:    cfg.WebhookAlertPendingReady,
			OldestPendingAgeSecLimit: cfg.WebhookAlertOldestAgeSec,
		},
		logger,
	)
	return WebhookAlertWorkerContainer{
		Database:                     runtimeDeps.databasePool,
		InitializePersistenceUseCase: runtimeDeps.initializePersistenceUseCase,
		WebhookAlertWorker:           alertWorker,
	}, nil
}

func buildRuntimeDependencies(cfg config.Config, logger *log.Logger) runtimeDependencies {
	persistenceGateway := postgresqlbootstrap.NewGateway(
		cfg.DatabaseURL,
		cfg.DatabaseTarget,
		cfg.MigrationsPath,
		postgresqlbootstrap.ValidationRules{
			AllocationMode:         cfg.AllocationMode,
			DevtestAllowMainnet:    cfg.DevtestAllowMainnet,
			DevtestKeysets:         cfg.DevtestKeysets,
			DevtestKeysetPreflight: mapPreflightEntries(cfg.DevtestKeysetPreflights),
			KeysetHashAlgorithm:    cfg.KeysetHashAlgorithm,
			KeysetHashHMACSecret:   cfg.KeysetHashHMACSecret,
			KeysetHashHMACLegacy:   cfg.KeysetHashHMACLegacyKeys,
			AddressSchemeAllowList: cfg.AddressSchemeAllowList,
		},
		logger,
	)
	initializePersistenceUseCase := use_cases.NewInitializePersistenceUseCase(persistenceGateway)
	databasePool := postgresqlshared.NewDatabasePool(cfg.DatabaseURL, logger)
	return runtimeDependencies{
		databasePool:                 databasePool,
		initializePersistenceUseCase: initializePersistenceUseCase,
	}
}

func buildChainObserverGateway(cfg config.Config) portsout.PaymentChainObserverGateway {
	return chainobserverdevtest.NewGateway(chainobserverdevtest.Config{
		BTCExploraBaseURL:  cfg.BTCExploraBaseURL,
		EVMRPCURLs:         cfg.EVMRPCURLs,
		DetectedBPS:        cfg.ReconcilerDetectedBPS,
		ConfirmedBPS:       cfg.ReconcilerConfirmedBPS,
		BTCMinConf:         cfg.ReconcilerBTCMinConf,
		BTCFinalityMinConf: cfg.ReconcilerBTCFinalityMinConf,
		EVMMinConf:         cfg.ReconcilerEVMMinConf,
		EVMFinalityMinConf: cfg.ReconcilerEVMFinalityMinConf,
	})
}

func newPaymentRequestRepository(
	db *sql.DB,
	cfg config.Config,
	logger *log.Logger,
) *postgresqlpaymentrequest.Repository {
	return postgresqlpaymentrequest.NewRepositoryWithConfig(
		db,
		logger,
		postgresqlpaymentrequest.Config{
			WebhookOutboxEnabled: cfg.WebhookEnabled,
			WebhookMaxAttempts:   cfg.WebhookMaxAttempts,
		},
	)
}

func buildReconcilerWorker(
	cfg config.Config,
	useCase portsin.ReconcilePaymentRequestsUseCase,
	logger *log.Logger,
) *reconciler.Worker {
	return reconciler.NewWorker(
		cfg.ReconcilerEnabled,
		cfg.ReconcilerPollInterval,
		cfg.ReconcilerBatchSize,
		cfg.ReconcilerWorkerID,
		cfg.ReconcilerLeaseDuration,
		cfg.ReconcilerReorgObserveWindow,
		cfg.ReconcilerStabilityCycles,
		useCase,
		logger,
	)
}

func mapPreflightEntries(entries []config.DevtestKeysetPreflightEntry) []postgresqlbootstrap.DevtestKeysetPreflightEntry {
	if len(entries) == 0 {
		return []postgresqlbootstrap.DevtestKeysetPreflightEntry{}
	}

	out := make([]postgresqlbootstrap.DevtestKeysetPreflightEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, postgresqlbootstrap.DevtestKeysetPreflightEntry{
			Chain:                 entry.Chain,
			Network:               entry.Network,
			KeysetID:              entry.KeysetID,
			ExtendedPublicKey:     entry.ExtendedPublicKey,
			ExpectedIndex0Address: entry.ExpectedIndex0Address,
		})
	}
	return out
}

func buildWalletGateway(cfg config.Config, logger *log.Logger) (portsout.WalletAllocationGateway, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.AllocationMode))

	walletGatewayBuildersMu.RLock()
	builder, exists := walletGatewayBuilders[mode]
	walletGatewayBuildersMu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unsupported allocation mode: %s", cfg.AllocationMode)
	}

	return builder(cfg, logger), nil
}
