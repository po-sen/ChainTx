package router

import (
	"net/http"

	"chaintx/internal/adapters/inbound/http/controllers"
)

type Dependencies struct {
	HealthController          *controllers.HealthController
	SwaggerController         *controllers.SwaggerController
	AssetsController          *controllers.AssetsController
	PaymentRequestsController *controllers.PaymentRequestsController
}

func New(deps Dependencies) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", deps.HealthController.GetHealth)
	mux.HandleFunc("GET /swagger", deps.SwaggerController.RedirectToIndex)
	mux.HandleFunc("GET /swagger/openapi.yaml", deps.SwaggerController.GetOpenAPISpec)
	mux.HandleFunc("GET /swagger/", deps.SwaggerController.ServeUI)
	mux.HandleFunc("GET /v1/assets", deps.AssetsController.ListAssets)
	mux.HandleFunc("POST /v1/payment-requests", deps.PaymentRequestsController.CreatePaymentRequest)
	mux.HandleFunc("GET /v1/payment-requests/{id}", deps.PaymentRequestsController.GetPaymentRequest)

	return mux
}
