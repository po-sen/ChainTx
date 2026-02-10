package router

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chaintx/internal/adapters/inbound/http/controllers"
	"chaintx/internal/adapters/outbound/docs"
	"chaintx/internal/application/dto"
	"chaintx/internal/application/use_cases"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestRouterHealthAndSwaggerRoutes(t *testing.T) {
	openAPISpecPath := writeTempOpenAPISpec(t)
	mux := newTestRouter(openAPISpecPath)

	t.Run("healthz returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
			t.Fatalf("expected body to contain status ok, got %s", rec.Body.String())
		}
	})

	t.Run("swagger root redirects to index", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusTemporaryRedirect {
			t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "/swagger/index.html" {
			t.Fatalf("expected redirect location /swagger/index.html, got %q", location)
		}
	})

	t.Run("swagger UI index is served", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Fatalf("expected text/html content type, got %q", contentType)
		}
	})

	t.Run("openapi spec is served with version 3.0.3", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/swagger/openapi.yaml", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "openapi: 3.0.3") {
			t.Fatalf("expected openapi version 3.0.3 in body, got %s", rec.Body.String())
		}
	})

	t.Run("assets route returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/assets", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"assets"`) {
			t.Fatalf("expected assets payload, got %s", rec.Body.String())
		}
	})

	t.Run("create payment request route returns 201", func(t *testing.T) {
		body := bytes.NewBufferString(`{"chain":"bitcoin","network":"mainnet","asset":"BTC"}`)
		req := httptest.NewRequest(http.MethodPost, "/v1/payment-requests", body)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Location") == "" {
			t.Fatalf("expected Location header")
		}
	})

	t.Run("get payment request route returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/payment-requests/pr_test", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"id":"pr_test"`) {
			t.Fatalf("expected payment request id in body, got %s", rec.Body.String())
		}
	})
}

func TestRouterHealthzRejectsNonGET(t *testing.T) {
	openAPISpecPath := writeTempOpenAPISpec(t)
	mux := newTestRouter(openAPISpecPath)

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected non-200 status for POST /healthz, got %d", rec.Code)
	}
}

func newTestRouter(openAPISpecPath string) *http.ServeMux {
	logger := log.New(io.Discard, "", 0)

	healthUseCase := use_cases.NewGetHealthUseCase()
	openAPIReadModel := docs.NewFileOpenAPISpecReadModel(openAPISpecPath)
	openAPIUseCase := use_cases.NewGetOpenAPISpecUseCase(openAPIReadModel)

	assetsController := controllers.NewAssetsController(stubListAssetsUseCase{}, logger)
	paymentRequestsController := controllers.NewPaymentRequestsController(
		stubCreatePaymentRequestUseCase{},
		stubGetPaymentRequestUseCase{},
		logger,
	)

	return New(Dependencies{
		HealthController:          controllers.NewHealthController(healthUseCase, logger),
		SwaggerController:         controllers.NewSwaggerController(openAPIUseCase, logger),
		AssetsController:          assetsController,
		PaymentRequestsController: paymentRequestsController,
	})
}

func writeTempOpenAPISpec(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "openapi.yaml")

	content := []byte("openapi: 3.0.3\ninfo:\n  title: test\n  version: 1.0.0\npaths:\n  /healthz:\n    get:\n      responses:\n        '200':\n          description: ok\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("failed to write temp openapi file: %v", err)
	}

	return path
}

type stubListAssetsUseCase struct{}

func (stubListAssetsUseCase) Execute(_ context.Context, _ dto.ListAssetsQuery) (dto.ListAssetsOutput, *apperrors.AppError) {
	return dto.ListAssetsOutput{Assets: []dto.AssetCatalogEntry{}}, nil
}

type stubCreatePaymentRequestUseCase struct{}

func (stubCreatePaymentRequestUseCase) Execute(_ context.Context, _ dto.CreatePaymentRequestCommand) (dto.CreatePaymentRequestOutput, *apperrors.AppError) {
	createdAt := time.Unix(0, 0).UTC()
	expiresAt := createdAt.Add(1 * time.Hour)
	return dto.CreatePaymentRequestOutput{
		Resource: dto.PaymentRequestResource{
			ID:        "pr_test",
			Status:    "pending",
			Chain:     "bitcoin",
			Network:   "mainnet",
			Asset:     "BTC",
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			PaymentInstructions: dto.PaymentInstructions{
				Address:         "bc1qexample",
				AddressScheme:   "bip84_p2wpkh",
				DerivationIndex: 1,
			},
		},
	}, nil
}

type stubGetPaymentRequestUseCase struct{}

func (stubGetPaymentRequestUseCase) Execute(_ context.Context, query dto.GetPaymentRequestQuery) (dto.PaymentRequestResource, *apperrors.AppError) {
	createdAt := time.Unix(0, 0).UTC()
	expiresAt := createdAt.Add(1 * time.Hour)
	return dto.PaymentRequestResource{
		ID:        query.ID,
		Status:    "pending",
		Chain:     "bitcoin",
		Network:   "mainnet",
		Asset:     "BTC",
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		PaymentInstructions: dto.PaymentInstructions{
			Address:         "bc1qexample",
			AddressScheme:   "bip84_p2wpkh",
			DerivationIndex: 1,
		},
	}, nil
}
