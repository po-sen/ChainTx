package di

import (
	"log"

	"chaintx/internal/adapters/inbound/http/controllers"
	httpRouter "chaintx/internal/adapters/inbound/http/router"
	"chaintx/internal/adapters/outbound/docs"
	"chaintx/internal/application/use_cases"
	"chaintx/internal/bootstrap"
	"chaintx/internal/infrastructure/httpserver"
)

type Container struct {
	Server *httpserver.Server
}

func Build(cfg bootstrap.Config, logger *log.Logger) Container {
	healthUseCase := use_cases.NewGetHealthUseCase()
	openAPIReadModel := docs.NewFileOpenAPISpecReadModel(cfg.OpenAPISpecPath)
	openAPIUseCase := use_cases.NewGetOpenAPISpecUseCase(openAPIReadModel)

	healthController := controllers.NewHealthController(healthUseCase, logger)
	swaggerController := controllers.NewSwaggerController(openAPIUseCase, logger)

	router := httpRouter.New(httpRouter.Dependencies{
		HealthController:  healthController,
		SwaggerController: swaggerController,
	})

	server := httpserver.New(cfg.Address(), router, logger)

	return Container{
		Server: server,
	}
}
