package router

import (
	"net/http"

	"chaintx/internal/adapters/inbound/http/controllers"
)

type Dependencies struct {
	HealthController  *controllers.HealthController
	SwaggerController *controllers.SwaggerController
}

func New(deps Dependencies) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", deps.HealthController.GetHealth)
	mux.HandleFunc("GET /swagger", deps.SwaggerController.RedirectToIndex)
	mux.HandleFunc("GET /swagger/openapi.yaml", deps.SwaggerController.GetOpenAPISpec)
	mux.HandleFunc("GET /swagger/", deps.SwaggerController.ServeUI)

	return mux
}
