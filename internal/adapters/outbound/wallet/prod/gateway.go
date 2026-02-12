package prod

import (
	"context"

	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type Gateway struct{}

var _ portsout.WalletAllocationGateway = (*Gateway)(nil)

func NewGateway() *Gateway {
	return &Gateway{}
}

func (g *Gateway) Mode() string {
	return "prod"
}

func (g *Gateway) DeriveAddress(_ context.Context, _ portsout.DeriveAddressInput) (portsout.DerivedAddress, *apperrors.AppError) {
	return portsout.DerivedAddress{}, apperrors.NewInternal(
		"wallet_allocation_not_implemented",
		"production wallet allocation gateway is not configured",
		nil,
	)
}
