package devtest

import (
	"context"
	"fmt"
	"log"
	"strings"

	portsout "chaintx/internal/application/ports/out"
	"chaintx/internal/infrastructure/walletkeys"
	apperrors "chaintx/internal/shared_kernel/errors"
)

const (
	bitcoinChain  = "bitcoin"
	ethereumChain = "ethereum"

	bitcoinBIP84Scheme = "bip84_p2wpkh"
	evmBIP44Scheme     = "evm_bip44"
)

type Config struct {
	AllowMainnet bool
	Keysets      map[string]string
}

type Gateway struct {
	allowMainnet bool
	keysets      map[string]string
	logger       *log.Logger
}

var _ portsout.WalletAllocationGateway = (*Gateway)(nil)

func NewGateway(cfg Config, logger *log.Logger) *Gateway {
	keysets := make(map[string]string, len(cfg.Keysets))
	for keysetID, value := range cfg.Keysets {
		trimmedKeysetID := strings.TrimSpace(keysetID)
		if trimmedKeysetID == "" {
			continue
		}
		keysets[trimmedKeysetID] = strings.TrimSpace(value)
	}

	return &Gateway{
		allowMainnet: cfg.AllowMainnet,
		keysets:      keysets,
		logger:       logger,
	}
}

func (g *Gateway) Mode() string {
	return "devtest"
}

func (g *Gateway) DeriveAddress(_ context.Context, input portsout.DeriveAddressInput) (portsout.DerivedAddress, *apperrors.AppError) {
	chain := strings.ToLower(strings.TrimSpace(input.Chain))
	network := strings.ToLower(strings.TrimSpace(input.Network))
	addressScheme := strings.ToLower(strings.TrimSpace(input.AddressScheme))

	if network == "mainnet" && !g.allowMainnet {
		return portsout.DerivedAddress{}, apperrors.NewValidation(
			"mainnet_allocation_blocked",
			"devtest mode blocks mainnet allocations by default",
			map[string]any{
				"chain":   chain,
				"network": network,
			},
		)
	}

	rawKey, ok := g.keysets[strings.TrimSpace(input.KeysetID)]
	if !ok || strings.TrimSpace(rawKey) == "" {
		return portsout.DerivedAddress{}, apperrors.NewInternal(
			"invalid_configuration",
			"keyset_id is not configured for devtest wallet allocation",
			map[string]any{
				"keyset_id": input.KeysetID,
				"chain":     chain,
				"network":   network,
			},
		)
	}

	switch chain {
	case bitcoinChain:
		return g.deriveBitcoin(rawKey, network, addressScheme, input)
	case ethereumChain:
		return g.deriveEVM(rawKey, network, addressScheme, input)
	default:
		return portsout.DerivedAddress{}, apperrors.NewValidation(
			"unsupported_allocator_target",
			"unsupported chain for wallet allocation",
			map[string]any{"chain": chain},
		)
	}
}

func (g *Gateway) deriveBitcoin(
	rawKey string,
	network string,
	addressScheme string,
	input portsout.DeriveAddressInput,
) (portsout.DerivedAddress, *apperrors.AppError) {
	if addressScheme != bitcoinBIP84Scheme {
		return portsout.DerivedAddress{}, apperrors.NewInternal(
			"invalid_configuration",
			"bitcoin address scheme is not allowed for devtest allocator",
			map[string]any{
				"address_scheme": input.AddressScheme,
				"chain":          input.Chain,
			},
		)
	}

	if network != "regtest" && network != "testnet" && network != "mainnet" {
		return portsout.DerivedAddress{}, apperrors.NewValidation(
			"unsupported_allocator_target",
			"unsupported bitcoin network for devtest allocator",
			map[string]any{"network": network},
		)
	}

	key, _, keyErr := walletkeys.NormalizeBitcoinKeyset(rawKey)
	if keyErr != nil {
		return portsout.DerivedAddress{}, mapKeyError(keyErr)
	}
	if keyErr := walletkeys.ValidateAccountLevelPolicy(key); keyErr != nil {
		return portsout.DerivedAddress{}, mapKeyError(keyErr)
	}

	address, keyErr := walletkeys.DeriveBitcoinP2WPKHAddress(key, network, input.DerivationPathTemplate, input.DerivationIndex)
	if keyErr != nil {
		return portsout.DerivedAddress{}, mapKeyError(keyErr)
	}

	return portsout.DerivedAddress{
		AddressRaw:    address,
		AddressScheme: input.AddressScheme,
	}, nil
}

func (g *Gateway) deriveEVM(
	rawKey string,
	network string,
	addressScheme string,
	input portsout.DeriveAddressInput,
) (portsout.DerivedAddress, *apperrors.AppError) {
	if addressScheme != evmBIP44Scheme {
		return portsout.DerivedAddress{}, apperrors.NewInternal(
			"invalid_configuration",
			"ethereum address scheme is not allowed for devtest allocator",
			map[string]any{
				"address_scheme": input.AddressScheme,
				"chain":          input.Chain,
			},
		)
	}

	if network != "sepolia" && network != "mainnet" {
		return portsout.DerivedAddress{}, apperrors.NewValidation(
			"unsupported_allocator_target",
			"unsupported ethereum network for devtest allocator",
			map[string]any{"network": network},
		)
	}

	key, _, keyErr := walletkeys.NormalizeEVMKeyset(rawKey)
	if keyErr != nil {
		return portsout.DerivedAddress{}, mapKeyError(keyErr)
	}
	if keyErr := walletkeys.ValidateAccountLevelPolicy(key); keyErr != nil {
		return portsout.DerivedAddress{}, mapKeyError(keyErr)
	}

	address, keyErr := walletkeys.DeriveEVMAddress(key, input.DerivationPathTemplate, input.DerivationIndex)
	if keyErr != nil {
		return portsout.DerivedAddress{}, mapKeyError(keyErr)
	}

	chainID := int64(11155111)
	if network == "mainnet" {
		chainID = 1
	}

	return portsout.DerivedAddress{
		AddressRaw:    address,
		AddressScheme: input.AddressScheme,
		ChainID:       &chainID,
	}, nil
}

func mapKeyError(keyErr *walletkeys.KeyError) *apperrors.AppError {
	if keyErr == nil {
		return nil
	}

	code := string(keyErr.Code)
	if code == "" {
		code = "address_derivation_failed"
	}
	details := map[string]any{
		"reason": keyErr.Message,
	}
	if keyErr.Cause != nil {
		details["cause"] = keyErr.Cause.Error()
	}

	switch keyErr.Code {
	case walletkeys.CodeUnsupportedTarget:
		return apperrors.NewValidation(code, keyErr.Message, details)
	case walletkeys.CodeInvalidConfiguration:
		return apperrors.NewInternal(code, keyErr.Message, details)
	case walletkeys.CodeInvalidKeyMaterialFormat:
		return apperrors.NewInternal(code, keyErr.Message, details)
	case walletkeys.CodeDerivationFailed:
		return apperrors.NewInternal(code, keyErr.Message, details)
	default:
		return apperrors.NewInternal("address_derivation_failed", fmt.Sprintf("address derivation failed: %s", keyErr.Message), details)
	}
}
