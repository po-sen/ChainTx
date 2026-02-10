package out

import (
	"context"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type PaymentAddressAllocationInput struct {
	Chain           string
	AddressScheme   string
	WalletAccountID string
	DerivationIndex int64
}

type PaymentAddressAllocation struct {
	AddressCanonical string
	Address          string
}

type PaymentAddressAllocator interface {
	Allocate(ctx context.Context, input PaymentAddressAllocationInput) (PaymentAddressAllocation, *apperrors.AppError)
}
