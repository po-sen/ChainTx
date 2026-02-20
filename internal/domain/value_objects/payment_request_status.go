package valueobjects

import apperrors "chaintx/internal/shared_kernel/errors"

type PaymentRequestStatus string

const (
	PaymentRequestStatusPending   PaymentRequestStatus = "pending"
	PaymentRequestStatusDetected  PaymentRequestStatus = "detected"
	PaymentRequestStatusConfirmed PaymentRequestStatus = "confirmed"
	PaymentRequestStatusExpired   PaymentRequestStatus = "expired"
	PaymentRequestStatusFailed    PaymentRequestStatus = "failed"
)

func NewPendingPaymentRequestStatus() PaymentRequestStatus {
	return PaymentRequestStatusPending
}

func ParsePaymentRequestStatus(raw string) (PaymentRequestStatus, *apperrors.AppError) {
	switch raw {
	case string(PaymentRequestStatusPending):
		return PaymentRequestStatusPending, nil
	case string(PaymentRequestStatusDetected):
		return PaymentRequestStatusDetected, nil
	case string(PaymentRequestStatusConfirmed):
		return PaymentRequestStatusConfirmed, nil
	case string(PaymentRequestStatusExpired):
		return PaymentRequestStatusExpired, nil
	case string(PaymentRequestStatusFailed):
		return PaymentRequestStatusFailed, nil
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
