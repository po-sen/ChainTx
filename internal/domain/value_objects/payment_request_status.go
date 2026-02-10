package valueobjects

import apperrors "chaintx/internal/shared_kernel/errors"

type PaymentRequestStatus string

const (
	PaymentRequestStatusPending PaymentRequestStatus = "pending"
)

func NewPendingPaymentRequestStatus() PaymentRequestStatus {
	return PaymentRequestStatusPending
}

func ParsePaymentRequestStatus(raw string) (PaymentRequestStatus, *apperrors.AppError) {
	switch raw {
	case string(PaymentRequestStatusPending):
		return PaymentRequestStatusPending, nil
	default:
		return "", apperrors.NewInternal(
			"payment_request_status_invalid",
			"payment request status is invalid",
			map[string]any{"status": raw},
		)
	}
}

func (s PaymentRequestStatus) String() string {
	return string(s)
}
