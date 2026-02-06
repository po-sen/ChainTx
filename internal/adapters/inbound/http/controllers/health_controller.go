package controllers

import (
	"log"
	"net/http"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
)

type HealthController struct {
	useCase portsin.GetHealthUseCase
	logger  *log.Logger
}

func NewHealthController(useCase portsin.GetHealthUseCase, logger *log.Logger) *HealthController {
	return &HealthController{
		useCase: useCase,
		logger:  logger,
	}
}

func (c *HealthController) GetHealth(w http.ResponseWriter, r *http.Request) {
	output, appErr := c.useCase.Execute(r.Context(), dto.GetHealthCommand{})
	if appErr != nil {
		c.logger.Printf("request error path=/healthz method=%s code=%s message=%s", r.Method, appErr.Code, appErr.Message)
		writeAppError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, output)
}
