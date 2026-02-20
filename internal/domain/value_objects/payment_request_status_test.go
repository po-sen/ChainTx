//go:build !integration

package valueobjects

import "testing"

func TestParsePaymentRequestStatus(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		valid  bool
		status PaymentRequestStatus
	}{
		{name: "pending", raw: "pending", valid: true, status: PaymentRequestStatusPending},
		{name: "detected", raw: "detected", valid: true, status: PaymentRequestStatusDetected},
		{name: "confirmed", raw: "confirmed", valid: true, status: PaymentRequestStatusConfirmed},
		{name: "expired", raw: "expired", valid: true, status: PaymentRequestStatusExpired},
		{name: "failed", raw: "failed", valid: true, status: PaymentRequestStatusFailed},
		{name: "invalid", raw: "wat", valid: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, appErr := ParsePaymentRequestStatus(tc.raw)
			if tc.valid {
				if appErr != nil {
					t.Fatalf("expected no error, got %+v", appErr)
				}
				if status != tc.status {
					t.Fatalf("expected %s, got %s", tc.status, status)
				}
				return
			}

			if appErr == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
