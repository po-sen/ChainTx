package controllers

import (
	"encoding/json"
	"net/http"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type errorResponse struct {
	Error errorEnvelope `json:"error"`
}

type errorEnvelope struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeAppError(w http.ResponseWriter, appErr *apperrors.AppError) {
	status := http.StatusInternalServerError
	switch appErr.Type {
	case apperrors.TypeValidation:
		status = http.StatusBadRequest
	case apperrors.TypeNotFound:
		status = http.StatusNotFound
	case apperrors.TypeConflict:
		status = http.StatusConflict
	}

	writeJSON(w, status, errorResponse{
		Error: errorEnvelope{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
	})
}
