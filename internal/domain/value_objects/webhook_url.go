package valueobjects

import (
	"net"
	"net/url"
	"strings"

	apperrors "chaintx/internal/shared_kernel/errors"
)

func NormalizeWebhookURL(raw string) (string, string, *apperrors.AppError) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", apperrors.NewValidation(
			"invalid_request",
			"webhook_url is required",
			map[string]any{"field": "webhook_url"},
		)
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || !parsed.IsAbs() || strings.TrimSpace(parsed.Host) == "" {
		return "", "", apperrors.NewValidation(
			"invalid_request",
			"webhook_url must be a valid absolute URL",
			map[string]any{"field": "webhook_url"},
		)
	}

	if parsed.User != nil {
		return "", "", apperrors.NewValidation(
			"invalid_request",
			"webhook_url must not contain user info",
			map[string]any{"field": "webhook_url"},
		)
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", "", apperrors.NewValidation(
			"invalid_request",
			"webhook_url must use http or https",
			map[string]any{"field": "webhook_url"},
		)
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return "", "", apperrors.NewValidation(
			"invalid_request",
			"webhook_url host is required",
			map[string]any{"field": "webhook_url"},
		)
	}

	port := strings.TrimSpace(parsed.Port())
	canonicalHost := host
	if port != "" {
		canonicalHost = net.JoinHostPort(host, port)
	}

	parsed.Scheme = scheme
	parsed.Host = canonicalHost
	parsed.Fragment = ""

	return parsed.String(), host, nil
}

func NormalizeWebhookHostPattern(raw string) (string, *apperrors.AppError) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	trimmed = strings.TrimSuffix(trimmed, ".")
	if trimmed == "" {
		return "", apperrors.NewValidation(
			"invalid_request",
			"webhook host pattern must not be empty",
			nil,
		)
	}

	if strings.Contains(trimmed, "://") ||
		strings.ContainsAny(trimmed, "/?#") ||
		strings.Contains(trimmed, " ") {
		return "", apperrors.NewValidation(
			"invalid_request",
			"webhook host pattern must be a host pattern",
			nil,
		)
	}

	if strings.HasPrefix(trimmed, "*.") {
		suffix := strings.TrimPrefix(trimmed, "*.")
		if suffix == "" || strings.Contains(suffix, "*") {
			return "", apperrors.NewValidation(
				"invalid_request",
				"webhook host wildcard pattern is invalid",
				nil,
			)
		}
		return "*." + suffix, nil
	}

	if strings.Contains(trimmed, "*") {
		return "", apperrors.NewValidation(
			"invalid_request",
			"webhook host wildcard pattern is invalid",
			nil,
		)
	}

	return trimmed, nil
}

func IsWebhookHostAllowed(host string, allowlist []string) bool {
	normalizedHost := strings.ToLower(strings.TrimSpace(host))
	normalizedHost = strings.TrimSuffix(normalizedHost, ".")
	if normalizedHost == "" {
		return false
	}

	for _, pattern := range allowlist {
		normalizedPattern := strings.ToLower(strings.TrimSpace(pattern))
		normalizedPattern = strings.TrimSuffix(normalizedPattern, ".")
		if normalizedPattern == "" {
			continue
		}

		if strings.HasPrefix(normalizedPattern, "*.") {
			suffix := strings.TrimPrefix(normalizedPattern, "*.")
			if suffix == "" {
				continue
			}
			if normalizedHost != suffix && strings.HasSuffix(normalizedHost, "."+suffix) {
				return true
			}
			continue
		}

		if normalizedHost == normalizedPattern {
			return true
		}
	}

	return false
}
