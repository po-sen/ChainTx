package di

import (
	"database/sql"
	"log"

	"chaintx/internal/adapters/inbound/http/controllers"
	httpRouter "chaintx/internal/adapters/inbound/http/router"
	"chaintx/internal/adapters/outbound/docs"
	postgresqlassetcatalog "chaintx/internal/adapters/outbound/persistence/postgresql/assetcatalog"
	postgresqlbootstrap "chaintx/internal/adapters/outbound/persistence/postgresql/bootstrap"
	postgresqlpaymentrequest "chaintx/internal/adapters/outbound/persistence/postgresql/paymentrequest"
	postgresqlshared "chaintx/internal/adapters/outbound/persistence/postgresql/shared"
	deterministicwallet "chaintx/internal/adapters/outbound/wallet/deterministic"
	portsin "chaintx/internal/application/ports/in"
	"chaintx/internal/application/use_cases"
	"chaintx/internal/bootstrap"
	"chaintx/internal/infrastructure/httpserver"
)

type Container struct {
	Database                     *sql.DB
	Server                       *httpserver.Server
	InitializePersistenceUseCase portsin.InitializePersistenceUseCase
}

func Build(cfg bootstrap.Config, logger *log.Logger) Container {
	healthUseCase := use_cases.NewGetHealthUseCase()
	openAPIReadModel := docs.NewFileOpenAPISpecReadModel(cfg.OpenAPISpecPath)
	openAPIUseCase := use_cases.NewGetOpenAPISpecUseCase(openAPIReadModel)
	persistenceGateway := postgresqlbootstrap.NewGateway(
		cfg.DatabaseURL,
		cfg.DatabaseTarget,
		cfg.MigrationsPath,
		logger,
	)
	initializePersistenceUseCase := use_cases.NewInitializePersistenceUseCase(persistenceGateway)
	databasePool := postgresqlshared.NewDatabasePool(cfg.DatabaseURL, logger)

	assetCatalogReadModel := postgresqlassetcatalog.NewReadModel(databasePool)
	paymentAddressAllocator := deterministicwallet.NewPaymentAddressAllocator()
	paymentRequestRepository := postgresqlpaymentrequest.NewRepository(databasePool, paymentAddressAllocator, logger)
	paymentRequestReadModel := postgresqlpaymentrequest.NewReadModel(databasePool)

	listAssetsUseCase := use_cases.NewListAssetsUseCase(assetCatalogReadModel)
	createPaymentRequestUseCase := use_cases.NewCreatePaymentRequestUseCase(
		assetCatalogReadModel,
		paymentRequestRepository,
		use_cases.NewSystemClock(),
	)
	getPaymentRequestUseCase := use_cases.NewGetPaymentRequestUseCase(paymentRequestReadModel)

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
	}
}
