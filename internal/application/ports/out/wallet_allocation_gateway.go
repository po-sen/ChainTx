package out

import (
	"context"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type DeriveAddressInput struct {
	Chain                  string
	Network                string
	AddressScheme          string
	KeysetID               string
	DerivationPathTemplate string
	DerivationIndex        int64
	ChainID                *int64
}

type DerivedAddress struct {
	AddressRaw    string
	AddressScheme string
	ChainID       *int64
}

type WalletAllocationGateway interface {
	DeriveAddress(ctx context.Context, input DeriveAddressInput) (DerivedAddress, *apperrors.AppError)
}
