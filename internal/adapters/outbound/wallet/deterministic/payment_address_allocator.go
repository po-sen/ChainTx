package deterministic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	portsout "chaintx/internal/application/ports/out"
	valueobjects "chaintx/internal/domain/value_objects"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type PaymentAddressAllocator struct{}

var _ portsout.PaymentAddressAllocator = (*PaymentAddressAllocator)(nil)

func NewPaymentAddressAllocator() *PaymentAddressAllocator {
	return &PaymentAddressAllocator{}
}

func (a *PaymentAddressAllocator) Allocate(_ context.Context, input portsout.PaymentAddressAllocationInput) (portsout.PaymentAddressAllocation, *apperrors.AppError) {
	seed := fmt.Sprintf("%s|%s|%d", input.Chain, input.WalletAccountID, input.DerivationIndex)
	hash := sha256.Sum256([]byte(seed))

	switch input.Chain {
	case "ethereum":
		canonical := "0x" + hex.EncodeToString(hash[12:])
		normalized, appErr := valueobjects.NormalizeAddressForStorage("ethereum", canonical)
		if appErr != nil {
			return portsout.PaymentAddressAllocation{}, appErr
		}

		responseAddress, appErr := valueobjects.FormatAddressForResponse("ethereum", normalized)
		if appErr != nil {
			return portsout.PaymentAddressAllocation{}, appErr
		}

		return portsout.PaymentAddressAllocation{
			AddressCanonical: normalized,
			Address:          responseAddress,
		}, nil
	case "bitcoin":
		candidate := deriveBitcoinAddress(hash, input.AddressScheme)
		normalized, appErr := valueobjects.NormalizeAddressForStorage("bitcoin", candidate)
		if appErr != nil {
			return portsout.PaymentAddressAllocation{}, appErr
		}

		return portsout.PaymentAddressAllocation{
			AddressCanonical: normalized,
			Address:          normalized,
		}, nil
	default:
		return portsout.PaymentAddressAllocation{}, apperrors.NewValidation(
			"unsupported_network",
			"unsupported chain",
			map[string]any{"chain": input.Chain},
		)
	}
}

func deriveBitcoinAddress(hash [32]byte, addressScheme string) string {
	if strings.Contains(strings.ToLower(addressScheme), "bip84") || strings.HasPrefix(strings.ToLower(addressScheme), "bech32") {
		return deriveBitcoinBech32(hash)
	}

	return deriveBitcoinBase58(hash)
}

func deriveBitcoinBech32(hash [32]byte) string {
	const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

	builder := strings.Builder{}
	builder.Grow(42)
	builder.WriteString("bc1q")
	for i := 0; i < 38; i++ {
		index := hash[i%len(hash)] % byte(len(charset))
		builder.WriteByte(charset[index])
	}

	return builder.String()
}

func deriveBitcoinBase58(hash [32]byte) string {
	const charset = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	builder := strings.Builder{}
	builder.Grow(34)
	builder.WriteByte('1')
	for i := 0; i < 33; i++ {
		index := hash[i%len(hash)] % byte(len(charset))
		builder.WriteByte(charset[index])
	}

	return builder.String()
}
