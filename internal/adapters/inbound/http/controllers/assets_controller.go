package controllers

import (
	"log"
	"net/http"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
)

type AssetsController struct {
	useCase portsin.ListAssetsUseCase
	logger  *log.Logger
}

func NewAssetsController(useCase portsin.ListAssetsUseCase, logger *log.Logger) *AssetsController {
	return &AssetsController{useCase: useCase, logger: logger}
}

func (c *AssetsController) ListAssets(w http.ResponseWriter, r *http.Request) {
	output, appErr := c.useCase.Execute(r.Context(), dto.ListAssetsQuery{})
	if appErr != nil {
		c.logger.Printf("request error path=/v1/assets method=%s code=%s message=%s", r.Method, appErr.Code, appErr.Message)
		writeAppError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, output)
}
