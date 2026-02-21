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
	devtestwallet "chaintx/internal/adapters/outbound/wallet/devtest"
	prodwallet "chaintx/internal/adapters/outbound/wallet/prod"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	"chaintx/internal/application/use_cases"
	"chaintx/internal/infrastructure/config"
	"chaintx/internal/infrastructure/httpserver"
	"chaintx/internal/infrastructure/reconciler"
)

type Container struct {
	Database                     *sql.DB
	Server                       *httpserver.Server
	InitializePersistenceUseCase portsin.InitializePersistenceUseCase
	ReconcilerWorker             *reconciler.Worker
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

func Build(cfg config.Config, logger *log.Logger) (Container, error) {
	walletGateway, buildErr := buildWalletGateway(cfg, logger)
	if buildErr != nil {
		return Container{}, buildErr
	}

	healthUseCase := use_cases.NewGetHealthUseCase()
	openAPIReadModel := docs.NewFileOpenAPISpecReadModel(cfg.OpenAPISpecPath)
	openAPIUseCase := use_cases.NewGetOpenAPISpecUseCase(openAPIReadModel)
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

	assetCatalogReadModel := postgresqlassetcatalog.NewReadModel(databasePool)
	paymentRequestRepository := postgresqlpaymentrequest.NewRepository(databasePool, logger)
	paymentRequestReadModel := postgresqlpaymentrequest.NewReadModel(databasePool)
	chainObserverGateway := chainobserverdevtest.NewGateway(chainobserverdevtest.Config{
		BTCExploraBaseURL: cfg.BTCExploraBaseURL,
		EVMRPCURLs:        cfg.EVMRPCURLs,
		DetectedBPS:       cfg.ReconcilerDetectedBPS,
		ConfirmedBPS:      cfg.ReconcilerConfirmedBPS,
		BTCMinConf:        cfg.ReconcilerBTCMinConf,
		EVMMinConf:        cfg.ReconcilerEVMMinConf,
	})

	listAssetsUseCase := use_cases.NewListAssetsUseCase(assetCatalogReadModel)
	createPaymentRequestUseCase := use_cases.NewCreatePaymentRequestUseCase(
		assetCatalogReadModel,
		paymentRequestRepository,
		walletGateway,
		use_cases.NewSystemClock(),
	)
	getPaymentRequestUseCase := use_cases.NewGetPaymentRequestUseCase(paymentRequestReadModel)
	reconcilePaymentRequestsUseCase := use_cases.NewReconcilePaymentRequestsUseCase(
		paymentRequestRepository,
		chainObserverGateway,
	)
	reconcilerWorker := reconciler.NewWorker(
		cfg.ReconcilerEnabled,
		cfg.ReconcilerPollInterval,
		cfg.ReconcilerBatchSize,
		cfg.ReconcilerWorkerID,
		cfg.ReconcilerLeaseDuration,
		reconcilePaymentRequestsUseCase,
		logger,
	)

	healthController := controllers.NewHealthController(healthUseCase, logger)
	swaggerController := controllers.NewSwaggerController(openAPIUseCase, logger)
	assetsController := controllers.NewAssetsController(listAssetsUseCase, logger)
	paymentRequestsController := controllers.NewPaymentRequestsController(
		createPaymentRequestUseCase,
		getPaymentRequestUseCase,
		logger,
	)

	router := httpRouter.New(httpRouter.Dependencies{
		HealthController:          healthController,
		SwaggerController:         swaggerController,
		AssetsController:          assetsController,
		PaymentRequestsController: paymentRequestsController,
	})

	server := httpserver.New(cfg.Address(), router, logger)

	return Container{
		Database:                     databasePool,
		Server:                       server,
		InitializePersistenceUseCase: initializePersistenceUseCase,
		ReconcilerWorker:             reconcilerWorker,
	}, nil
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
