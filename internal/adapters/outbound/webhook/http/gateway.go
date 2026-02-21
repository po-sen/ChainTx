package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

const (
	defaultHTTPTimeout = 5 * time.Second
	maxErrorBodyBytes  = 1024
	signatureVersionV1 = "v1"
	nonceByteLength    = 16
)

type Config struct {
	HMACSecret string
	Timeout    time.Duration
}

type Gateway struct {
	hmacSecret string
	client     *nethttp.Client
}

var _ portsout.WebhookEventGateway = (*Gateway)(nil)

func NewGateway(cfg Config) *Gateway {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}

	return &Gateway{
		hmacSecret: strings.TrimSpace(cfg.HMACSecret),
		client: &nethttp.Client{
			Timeout: timeout,
		},
	}
}

func (g *Gateway) SendWebhookEvent(
	ctx context.Context,
	input dto.SendWebhookEventInput,
) (dto.SendWebhookEventOutput, *apperrors.AppError) {
	if g == nil || g.client == nil {
		return dto.SendWebhookEventOutput{}, apperrors.NewInternal(
			"webhook_gateway_not_configured",
			"webhook gateway is not configured",
			nil,
		)
	}
	if g.hmacSecret == "" {
		return dto.SendWebhookEventOutput{}, apperrors.NewInternal(
			"webhook_hmac_secret_missing",
			"webhook hmac secret is missing",
			nil,
		)
	}

	body := input.Payload
	if len(body) == 0 {
		return dto.SendWebhookEventOutput{}, apperrors.NewValidation(
			"webhook_payload_missing",
			"webhook payload is required",
			nil,
		)
	}
	eventID := strings.TrimSpace(input.EventID)
	if eventID == "" {
		return dto.SendWebhookEventOutput{}, apperrors.NewValidation(
			"webhook_event_id_missing",
			"webhook event id is required",
			nil,
		)
	}
	eventType := strings.TrimSpace(input.EventType)
	if eventType == "" {
		return dto.SendWebhookEventOutput{}, apperrors.NewValidation(
			"webhook_event_type_missing",
			"webhook event type is required",
			nil,
		)
	}
	destinationURL := strings.TrimSpace(input.DestinationURL)
	if destinationURL == "" {
		return dto.SendWebhookEventOutput{}, apperrors.NewValidation(
			"webhook_destination_missing",
			"webhook destination url is required",
			map[string]any{"field": "destination_url"},
		)
	}

	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	nonce, nonceErr := webhookNonce()
	if nonceErr != nil {
		return dto.SendWebhookEventOutput{}, apperrors.NewInternal(
			"webhook_nonce_generation_failed",
			"failed to generate webhook nonce",
			map[string]any{"error": nonceErr.Error()},
		)
	}
	legacySignature := webhookLegacySignature(g.hmacSecret, timestamp, body)
	signatureV1 := webhookV1Signature(g.hmacSecret, timestamp, nonce, eventID, eventType, body)
	deliveryAttempt := input.DeliveryAttempt
	if deliveryAttempt <= 0 {
		deliveryAttempt = 1
	}

	request, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodPost, destinationURL, bytes.NewReader(body))
	if err != nil {
		return dto.SendWebhookEventOutput{}, apperrors.NewInternal(
			"webhook_request_build_failed",
			"failed to build webhook request",
			map[string]any{"error": err.Error()},
		)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-ChainTx-Event-Id", eventID)
	request.Header.Set("Idempotency-Key", eventID)
	request.Header.Set("X-ChainTx-Event-Type", eventType)
	request.Header.Set("X-ChainTx-Delivery-Attempt", strconv.Itoa(deliveryAttempt))
	request.Header.Set("X-ChainTx-Timestamp", timestamp)
	request.Header.Set("X-ChainTx-Nonce", nonce)
	request.Header.Set("X-ChainTx-Signature-Version", signatureVersionV1)
	request.Header.Set("X-ChainTx-Signature-V1", "sha256="+signatureV1)
	request.Header.Set("X-ChainTx-Signature", "sha256="+legacySignature)

	response, err := g.client.Do(request)
	if err != nil {
		return dto.SendWebhookEventOutput{}, apperrors.NewInternal(
			"webhook_delivery_failed",
			"failed to send webhook request",
			map[string]any{"error": err.Error()},
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		bodyPreview := ""
		raw, readErr := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		if readErr == nil {
			bodyPreview = strings.TrimSpace(string(raw))
		}
		return dto.SendWebhookEventOutput{StatusCode: response.StatusCode}, apperrors.NewInternal(
			"webhook_delivery_failed",
			"webhook endpoint returned non-2xx status",
			map[string]any{
				"status_code": response.StatusCode,
				"body":        bodyPreview,
			},
		)
	}

	return dto.SendWebhookEventOutput{StatusCode: response.StatusCode}, nil
}

func webhookNonce() (string, error) {
	raw := make([]byte, nonceByteLength)
	if _, err := cryptorand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func webhookLegacySignature(secret string, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func webhookV1Signature(
	secret string,
	timestamp string,
	nonce string,
	eventID string,
	eventType string,
	body []byte,
) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write([]byte(nonce))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write([]byte(strings.TrimSpace(eventID)))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write([]byte(strings.TrimSpace(eventType)))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func BuildExpectedSignatureHeader(secret string, timestamp string, body []byte) string {
	return fmt.Sprintf("sha256=%s", webhookLegacySignature(secret, timestamp, body))
}

func BuildExpectedSignatureV1Header(
	secret string,
	timestamp string,
	nonce string,
	eventID string,
	eventType string,
	body []byte,
) string {
	return fmt.Sprintf(
		"sha256=%s",
		webhookV1Signature(secret, timestamp, nonce, eventID, eventType, body),
	)
}
