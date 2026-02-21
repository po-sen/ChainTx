package controllers

import (
	"log"
	"net/http"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type SwaggerController struct {
	useCase         portsin.GetOpenAPISpecUseCase
	logger          *log.Logger
	swaggerUIHandle http.Handler
}

func NewSwaggerController(useCase portsin.GetOpenAPISpecUseCase, logger *log.Logger) *SwaggerController {
	return &SwaggerController{
		useCase: useCase,
		logger:  logger,
		swaggerUIHandle: httpSwagger.Handler(
			httpSwagger.URL("/swagger/openapi.yaml"),
			httpSwagger.PersistAuthorization(true),
		),
	}
}

func (c *SwaggerController) RedirectToIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger/index.html", http.StatusTemporaryRedirect)
}

func (c *SwaggerController) ServeUI(w http.ResponseWriter, r *http.Request) {
	c.swaggerUIHandle.ServeHTTP(w, r)
}

func (c *SwaggerController) GetOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	output, appErr := c.useCase.Execute(r.Context(), dto.GetOpenAPISpecQuery{})
	if appErr != nil {
		c.logger.Printf("request error path=/swagger/openapi.yaml method=%s code=%s message=%s", r.Method, appErr.Code, appErr.Message)
		writeAppError(w, appErr)
		return
	}

	w.Header().Set("Content-Type", output.ContentType)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(output.Content); err != nil {
		c.logger.Printf("response write error path=/swagger/openapi.yaml method=%s error=%v", r.Method, err)
	}
}
