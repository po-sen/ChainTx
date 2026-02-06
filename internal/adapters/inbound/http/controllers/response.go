package controllers

import (
	"encoding/json"
	"net/http"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type errorResponse struct {
	Type     string            `json:"type"`
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata,omitempty"`
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
	}

	writeJSON(w, status, errorResponse{
		Type:     string(appErr.Type),
		Code:     appErr.Code,
		Message:  appErr.Message,
		Metadata: appErr.Metadata,
	})
}
