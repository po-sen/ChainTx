package controllers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"chaintx/internal/application/use_cases"
)

func TestHealthControllerGetHealth(t *testing.T) {
	controller := NewHealthController(use_cases.NewGetHealthUseCase(), log.New(io.Discard, "", 0))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	controller.GetHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON body, got error: %v", err)
	}

	status, ok := payload["status"].(string)
	if !ok || status != "ok" {
		t.Fatalf("expected JSON field status=ok, got %v", payload["status"])
	}
}
